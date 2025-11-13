package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	srvAuth "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/auth"
	coreauth "github.com/berjistech/berjis-ecosystem/shared/coreauth"
)

type Options struct {
	AllowedOrigins    string
	CoreAPIBase       string
	DB                *sqlx.DB
	UploadsPublicBase string
	TaxRatePercent    float64
	ShippingFlatCents int64
}

func New(opts Options) *fiber.App {
	app := fiber.New()
	app.Use(fiberrecover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     opts.AllowedOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Authorization,Content-Type,Accept",
		AllowCredentials: true,
	}))

	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		rid := c.Get("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
		}
		c.Set("X-Request-ID", rid)
		err := c.Next()
		status := c.Response().StatusCode()
		level := "info"
		if status >= 500 {
			level = "error"
		}
		ip := c.Get("CF-Connecting-IP")
		if ip == "" {
			ip = c.Get("X-Forwarded-For")
		}
		if ip == "" {
			ip = c.IP()
		}
		entry := map[string]any{
			"ts":        time.Now().Format(time.RFC3339Nano),
			"level":     level,
			"requestId": rid,
			"method":    c.Method(),
			"path":      c.Path(),
			"query":     string(c.Context().URI().QueryString()),
			"status":    status,
			"latencyMs": time.Since(start).Milliseconds(),
			"ip":        ip,
			"ua":        c.Get("User-Agent"),
		}
		if uid := srvAuth.UserID(c); uid != "" {
			entry["user"] = uid
		}
		if err != nil {
			entry["error"] = err.Error()
		}
		if b, mErr := json.Marshal(entry); mErr == nil {
			log.Printf("%s", b)
		} else {
			log.Printf("level=%s requestId=%s method=%s path=%s status=%d latencyMs=%d", level, rid, c.Method(), c.Path(), status, time.Since(start).Milliseconds())
		}
		return err
	})

	_ = os.MkdirAll("/data/uploads/products", 0o755)
	app.Static("/uploads", "/data/uploads")

	registerPublicRoutes(app, opts)

	httpClientAuth := &http.Client{Timeout: 8 * time.Second}
	var authVerifier *coreauth.Verifier
	if base := strings.TrimSpace(opts.CoreAPIBase); base != "" {
		if v, err := coreauth.NewVerifier(coreauth.Config{
			CoreAPIBase: base,
			HTTPClient:  httpClientAuth,
		}); err != nil {
			log.Printf("warn: coreauth verifier init failed: %v", err)
		} else {
			authVerifier = v
		}
	}

	requireAuth := srvAuth.Middleware(srvAuth.Options{
		CoreAPIBase: opts.CoreAPIBase,
		HTTPClient:  httpClientAuth,
		Verifier:    authVerifier,
	})

	registerShopProductRoutes(app, opts, requireAuth)
	registerInventoryRoutes(app, opts, requireAuth)
	registerCollectionRoutes(app, opts, requireAuth)
	registerDiscountRoutes(app, opts, requireAuth)
	registerCustomerRoutes(app, opts, requireAuth)
	registerTeamRoutes(app, opts, requireAuth)
	registerOrderRoutes(app, opts, requireAuth)
	registerMiscRoutes(app, opts, requireAuth)
	registerCartRoutes(app, opts, requireAuth)
	registerSearchRoutes(app, opts, requireAuth)
	registerProfileRoutes(app, opts, requireAuth)
	registerMarketingRoutes(app, opts, requireAuth)

	return app
}
