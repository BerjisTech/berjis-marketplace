package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppName              string
	Env                  string
	Port                 string
	DatabaseURL          string
	CoreAPIBase          string
	AllowedOrigins       string
	UploadsPublicBase    string
	TaxRatePercent       float64
	ShippingFlatCents    int64
	MaxShopsPerUser      int
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string
	EmailProvider        string
	SMTPHost             string
	SMTPPort             int
	SMTPUsername         string
	SMTPPassword         string
	SendGridAPIKey       string
	EmailFrom            string
	EmailFromName        string
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvFloat(k string, def float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getenvInt64(k string, def int64) int64 {
	if v := os.Getenv(k); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return def
}

func Load() Config {
	return Config{
		AppName:           getenv("APP_NAME", "berjis-marketplace"),
		Env:               getenv("APP_ENV", "development"),
		Port:              getenv("PORT", "8093"),
		DatabaseURL:       getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5450/berjis_marketplace?sslmode=disable"),
		CoreAPIBase:       getenv("CORE_API_BASE", "http://localhost:8080"),
		AllowedOrigins:    getenv("ALLOWED_ORIGINS", "https://marketplace.berjis.tech,https://berjis.tech"),
		UploadsPublicBase: getenv("UPLOADS_PUBLIC_BASE", "/uploads"),
		TaxRatePercent:    getenvFloat("TAX_RATE_PERCENT", 8.5),
		ShippingFlatCents: getenvInt64("SHIPPING_FLAT_CENTS", 1500),
		MaxShopsPerUser:      int(getenvInt64("MAX_SHOPS_PER_USER", 0)),
		StripeSecretKey:      getenv("STRIPE_SECRET_KEY", ""),
		StripePublishableKey: getenv("STRIPE_PUBLISHABLE_KEY", ""),
		StripeWebhookSecret:  getenv("STRIPE_WEBHOOK_SECRET", ""),
		EmailProvider:        getenv("EMAIL_PROVIDER", ""),
		SMTPHost:             getenv("SMTP_HOST", ""),
		SMTPPort:             int(getenvInt64("SMTP_PORT", 587)),
		SMTPUsername:         getenv("SMTP_USERNAME", ""),
		SMTPPassword:         getenv("SMTP_PASSWORD", ""),
		SendGridAPIKey:       getenv("SENDGRID_API_KEY", ""),
		EmailFrom:            getenv("EMAIL_FROM", ""),
		EmailFromName:        getenv("EMAIL_FROM_NAME", "Berjis Marketplace"),
	}
}
