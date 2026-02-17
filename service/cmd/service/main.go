package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"

	"github.com/berjistech/berjis-ecosystem/marketplace/service/internal/config"
	"github.com/berjistech/berjis-ecosystem/marketplace/service/internal/db"
	"github.com/berjistech/berjis-ecosystem/marketplace/service/internal/email"
	"github.com/berjistech/berjis-ecosystem/marketplace/service/internal/migrate"
	"github.com/berjistech/berjis-ecosystem/marketplace/service/internal/server"
	stripeClient "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/stripe"
	"github.com/berjistech/berjis-ecosystem/marketplace/service/internal/webhook"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	dbc, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	if err := migrate.Run(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	var sc *stripeClient.Client
	if cfg.StripeSecretKey != "" {
		sc = stripeClient.NewClient(cfg.StripeSecretKey)
		log.Println("stripe client initialized")
	}

	whDispatcher := webhook.NewDispatcher(dbc)
	whDispatcher.StartWorker(context.Background())
	log.Println("webhook dispatcher started")

	// Initialize email sender based on environment configuration
	var emailSender email.Sender
	switch cfg.EmailProvider {
	case "smtp":
		if cfg.SMTPHost != "" {
			emailSender = &email.SMTPSender{
				Host:     cfg.SMTPHost,
				Port:     cfg.SMTPPort,
				Username: cfg.SMTPUsername,
				Password: cfg.SMTPPassword,
				From:     cfg.EmailFrom,
				FromName: cfg.EmailFromName,
			}
			log.Println("email sender initialized (smtp)")
		}
	case "sendgrid":
		if cfg.SendGridAPIKey != "" {
			emailSender = &email.SendGridSender{
				APIKey:   cfg.SendGridAPIKey,
				From:     cfg.EmailFrom,
				FromName: cfg.EmailFromName,
			}
			log.Println("email sender initialized (sendgrid)")
		}
	}
	if emailSender == nil {
		emailSender = &email.NoopSender{}
		log.Println("email sender initialized (noop/development)")
	}

	// Start email worker goroutine
	emailWorker := &email.Worker{
		DB:     dbc,
		Sender: emailSender,
	}
	go emailWorker.Start(context.Background())
	log.Println("email worker started")

	app := server.New(server.Options{
		AllowedOrigins:       cfg.AllowedOrigins,
		CoreAPIBase:          cfg.CoreAPIBase,
		DB:                   dbc,
		UploadsPublicBase:    cfg.UploadsPublicBase,
		TaxRatePercent:       cfg.TaxRatePercent,
		ShippingFlatCents:    cfg.ShippingFlatCents,
		MaxShopsPerUser:      cfg.MaxShopsPerUser,
		StripeClient:         sc,
		StripeWebhookSecret:  cfg.StripeWebhookSecret,
		StripePublishableKey: cfg.StripePublishableKey,
		WebhookDispatcher:    whDispatcher,
		EmailSender:          emailSender,
	})

	addr := ":" + cfg.Port
	log.Printf("marketplace listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatal(err)
	}
}
