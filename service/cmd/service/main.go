package main

import (
	"log"

	"github.com/joho/godotenv"

	"github.com/berjistech/berjis-ecosystem/marketplace/service/internal/config"
	"github.com/berjistech/berjis-ecosystem/marketplace/service/internal/db"
	"github.com/berjistech/berjis-ecosystem/marketplace/service/internal/migrate"
	"github.com/berjistech/berjis-ecosystem/marketplace/service/internal/server"
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

	app := server.New(server.Options{
		AllowedOrigins:    cfg.AllowedOrigins,
		CoreAPIBase:       cfg.CoreAPIBase,
		DB:                dbc,
		UploadsPublicBase: cfg.UploadsPublicBase,
		TaxRatePercent:    cfg.TaxRatePercent,
		ShippingFlatCents: cfg.ShippingFlatCents,
	})

	addr := ":" + cfg.Port
	log.Printf("marketplace listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatal(err)
	}
}
