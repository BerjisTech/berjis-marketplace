package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	pq "github.com/lib/pq"

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

type Shop struct {
	UUID        uuid.UUID `db:"uuid" json:"uuid"`
	Name        string    `db:"name" json:"name"`
	Slug        string    `db:"slug" json:"slug"`
	OwnerUUID   uuid.UUID `db:"owner_uuid" json:"ownerUuid"`
	Public      bool      `db:"public" json:"public"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
	Description string    `db:"description" json:"description"`
	Meta        any       `db:"meta" json:"meta,omitempty"`
}

type Product struct {
	UUID         uuid.UUID  `db:"uuid" json:"uuid"`
	ShopUUID     uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	Title        string     `db:"title" json:"title"`
	Slug         string     `db:"slug" json:"slug"`
	Summary      string     `db:"summary" json:"summary"`
	PriceCents   int64      `db:"price_cents" json:"priceCents"`
	Currency     string     `db:"currency" json:"currency"`
	Stock        int64      `db:"stock" json:"stock"`
	ImageURL     *string    `db:"image_url" json:"imageUrl,omitempty"`
	Category     string     `db:"category" json:"category"`
	CategoryUUID *uuid.UUID `db:"category_uuid" json:"categoryUuid,omitempty"`
	Images       []string   `db:"images" json:"images"`
	Rating       float32    `db:"rating" json:"rating"`
	ReviewCount  int64      `db:"review_count" json:"reviewCount"`
	CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
	DeletedAt    *time.Time `db:"deleted_at" json:"-"`
	Published    bool       `db:"published" json:"published"`
	ShopName     string     `db:"shop_name" json:"shopName"`
	ShopSlug     string     `db:"shop_slug" json:"shopSlug"`
}

type Category struct {
	UUID        uuid.UUID  `db:"uuid" json:"uuid"`
	ShopUUID    uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	ParentUUID  *uuid.UUID `db:"parent_uuid" json:"parentUuid,omitempty"`
	Name        string     `db:"name" json:"name"`
	Slug        string     `db:"slug" json:"slug"`
	Description string     `db:"description" json:"description"`
	SortOrder   int        `db:"sort_order" json:"sortOrder"`
	IsActive    bool       `db:"is_active" json:"isActive"`
	CreatedAt   time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updatedAt"`
}

type InventoryLocation struct {
	UUID        uuid.UUID `db:"uuid" json:"uuid"`
	ShopUUID    uuid.UUID `db:"shop_uuid" json:"shopUuid"`
	Name        string    `db:"name" json:"name"`
	Code        string    `db:"code" json:"code"`
	Description string    `db:"description" json:"description"`
	IsPrimary   bool      `db:"is_primary" json:"isPrimary"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

type InventoryLevel struct {
	UUID         uuid.UUID  `db:"uuid" json:"uuid"`
	ShopUUID     uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	ProductUUID  uuid.UUID  `db:"product_uuid" json:"productUuid"`
	LocationUUID *uuid.UUID `db:"location_uuid" json:"locationUuid,omitempty"`
	Quantity     int64      `db:"quantity" json:"quantity"`
	Reserved     int64      `db:"reserved" json:"reserved"`
	SafetyStock  int64      `db:"safety_stock" json:"safetyStock"`
	CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
}

type InventoryEntry struct {
	InventoryLevel
	ProductTitle string  `db:"product_title" json:"productTitle"`
	ProductSlug  string  `db:"product_slug" json:"productSlug"`
	LocationName *string `db:"location_name" json:"locationName,omitempty"`
	LocationCode *string `db:"location_code" json:"locationCode,omitempty"`
}

type Collection struct {
	UUID        uuid.UUID       `db:"uuid" json:"uuid"`
	ShopUUID    uuid.UUID       `db:"shop_uuid" json:"shopUuid"`
	Title       string          `db:"title" json:"title"`
	Slug        string          `db:"slug" json:"slug"`
	Description string          `db:"description" json:"description"`
	IsAutomatic bool            `db:"is_automatic" json:"isAutomatic"`
	Rules       json.RawMessage `db:"rules" json:"rules,omitempty"`
	SortOrder   int             `db:"sort_order" json:"sortOrder"`
	IsActive    bool            `db:"is_active" json:"isActive"`
	CreatedAt   time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updatedAt"`
}

type Discount struct {
	UUID                  uuid.UUID       `db:"uuid" json:"uuid"`
	ShopUUID              uuid.UUID       `db:"shop_uuid" json:"shopUuid"`
	Name                  string          `db:"name" json:"name"`
	Code                  string          `db:"code" json:"code"`
	Description           string          `db:"description" json:"description"`
	DiscountType          string          `db:"discount_type" json:"discountType"`
	AmountCents           int64           `db:"amount_cents" json:"amountCents"`
	Percentage            float64         `db:"percentage" json:"percentage"`
	StartsAt              *time.Time      `db:"starts_at" json:"startsAt,omitempty"`
	EndsAt                *time.Time      `db:"ends_at" json:"endsAt,omitempty"`
	UsageLimitTotal       *int            `db:"usage_limit_total" json:"usageLimitTotal,omitempty"`
	UsageLimitPerCustomer *int            `db:"usage_limit_per_customer" json:"usageLimitPerCustomer,omitempty"`
	AutoApply             bool            `db:"auto_apply" json:"autoApply"`
	Status                string          `db:"status" json:"status"`
	AppliesTo             json.RawMessage `db:"applies_to" json:"appliesTo,omitempty"`
	CreatedAt             time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt             time.Time       `db:"updated_at" json:"updatedAt"`
}

type GiftCard struct {
	UUID                 uuid.UUID  `db:"uuid" json:"uuid"`
	ShopUUID             uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	Code                 string     `db:"code" json:"code"`
	BalanceCents         int64      `db:"balance_cents" json:"balanceCents"`
	OriginalBalanceCents int64      `db:"original_balance_cents" json:"originalBalanceCents"`
	Currency             string     `db:"currency" json:"currency"`
	IssuedToEmail        *string    `db:"issued_to_email" json:"issuedToEmail,omitempty"`
	Note                 string     `db:"note" json:"note"`
	Status               string     `db:"status" json:"status"`
	ExpiresAt            *time.Time `db:"expires_at" json:"expiresAt,omitempty"`
	IssuedAt             time.Time  `db:"issued_at" json:"issuedAt"`
	RedeemedAt           *time.Time `db:"redeemed_at" json:"redeemedAt,omitempty"`
	CreatedAt            time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt            time.Time  `db:"updated_at" json:"updatedAt"`
}

type GiftCardTransaction struct {
	UUID         uuid.UUID `db:"uuid" json:"uuid"`
	GiftCardUUID uuid.UUID `db:"gift_card_uuid" json:"giftCardUuid"`
	ChangeCents  int64     `db:"change_cents" json:"changeCents"`
	Reason       string    `db:"reason" json:"reason"`
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
}

type StoreUser struct {
	UUID      uuid.UUID `db:"uuid" json:"uuid"`
	StoreUUID uuid.UUID `db:"store_uuid" json:"storeUuid"`
	UserUUID  uuid.UUID `db:"user_uuid" json:"userUuid"`
	Role      string    `db:"role" json:"role"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

type StoreInvitation struct {
	UUID       uuid.UUID  `db:"uuid" json:"uuid"`
	StoreUUID  uuid.UUID  `db:"store_uuid" json:"storeUuid"`
	Email      string     `db:"email" json:"email"`
	Role       string     `db:"role" json:"role"`
	Token      string     `db:"token" json:"token,omitempty"`
	Status     string     `db:"status" json:"status"`
	InvitedBy  uuid.UUID  `db:"invited_by" json:"invitedBy"`
	ExpiresAt  time.Time  `db:"expires_at" json:"expiresAt"`
	CreatedAt  time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updatedAt"`
	AcceptedAt *time.Time `db:"accepted_at" json:"acceptedAt,omitempty"`
}

type CustomerSummary struct {
	UUID             uuid.UUID      `db:"uuid" json:"uuid"`
	ShopUUID         uuid.UUID      `db:"shop_uuid" json:"shopUuid"`
	UserUUID         *uuid.UUID     `db:"user_uuid" json:"userUuid,omitempty"`
	Email            string         `db:"email" json:"email"`
	FirstName        string         `db:"first_name" json:"firstName"`
	LastName         string         `db:"last_name" json:"lastName"`
	Phone            string         `db:"phone" json:"phone"`
	Tags             pq.StringArray `db:"tags" json:"tags"`
	Notes            string         `db:"notes" json:"notes"`
	MarketingOptIn   bool           `db:"marketing_opt_in" json:"marketingOptIn"`
	CreatedAt        time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt        time.Time      `db:"updated_at" json:"updatedAt"`
	TotalSpentCents  int64          `db:"total_spent_cents" json:"totalSpentCents"`
	OrdersCount      int64          `db:"orders_count" json:"ordersCount"`
	LastOrderAt      *time.Time     `db:"last_order_at" json:"lastOrderAt,omitempty"`
	CustomerFullName string         `db:"customer_name" json:"customerName"`
}

type CustomerOrder struct {
	UUID            uuid.UUID  `db:"uuid" json:"uuid"`
	TotalCents      int64      `db:"total_cents" json:"totalCents"`
	Currency        string     `db:"currency" json:"currency"`
	Status          string     `db:"status" json:"status"`
	CreatedAt       time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updatedAt"`
	TrackingNumber  *string    `db:"tracking_number" json:"trackingNumber,omitempty"`
	TrackingURL     *string    `db:"tracking_url" json:"trackingUrl,omitempty"`
	ShippingCarrier *string    `db:"shipping_carrier" json:"shippingCarrier,omitempty"`
	ShippedAt       *time.Time `db:"shipped_at" json:"shippedAt,omitempty"`
	DeliveredAt     *time.Time `db:"delivered_at" json:"deliveredAt,omitempty"`
}

type ShopOrder struct {
	UUID            uuid.UUID  `db:"uuid" json:"uuid"`
	TotalCents      int64      `db:"total_cents" json:"totalCents"`
	Currency        string     `db:"currency" json:"currency"`
	Status          string     `db:"status" json:"status"`
	CreatedAt       time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updatedAt"`
	TrackingNumber  *string    `db:"tracking_number" json:"trackingNumber,omitempty"`
	TrackingURL     *string    `db:"tracking_url" json:"trackingUrl,omitempty"`
	ShippingCarrier *string    `db:"shipping_carrier" json:"shippingCarrier,omitempty"`
	ShippedAt       *time.Time `db:"shipped_at" json:"shippedAt,omitempty"`
	DeliveredAt     *time.Time `db:"delivered_at" json:"deliveredAt,omitempty"`
	CustomerUUID    *uuid.UUID `db:"customer_uuid" json:"customerUuid,omitempty"`
	CustomerEmail   string     `db:"customer_email" json:"customerEmail"`
	CustomerName    string     `db:"customer_name" json:"customerName"`
}

type SalesPoint struct {
	Date       string `json:"date"`
	TotalCents int64  `json:"totalCents"`
}

type DashboardMetrics struct {
	TotalSalesCents        int64        `json:"totalSalesCents"`
	OrdersCount            int64        `json:"ordersCount"`
	AverageOrderValueCents int64        `json:"averageOrderValueCents"`
	CustomersCount         int64        `json:"customersCount"`
	SalesSeries            []SalesPoint `json:"salesSeries"`
}

type UserProfile struct {
	UserUUID       uuid.UUID `db:"user_uuid" json:"userUuid"`
	DisplayName    string    `db:"display_name" json:"displayName"`
	Email          string    `db:"email" json:"email"`
	Phone          string    `db:"phone" json:"phone"`
	AvatarURL      *string   `db:"avatar_url" json:"avatarUrl,omitempty"`
	Timezone       string    `db:"timezone" json:"timezone"`
	MarketingOptIn bool      `db:"marketing_opt_in" json:"marketingOptIn"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
}

type UserAddress struct {
	UUID              uuid.UUID `db:"uuid" json:"uuid"`
	UserUUID          uuid.UUID `db:"user_uuid" json:"userUuid"`
	Label             string    `db:"label" json:"label"`
	RecipientName     string    `db:"recipient_name" json:"recipientName"`
	Line1             string    `db:"line1" json:"line1"`
	Line2             string    `db:"line2" json:"line2"`
	City              string    `db:"city" json:"city"`
	Region            string    `db:"region" json:"region"`
	PostalCode        string    `db:"postal_code" json:"postalCode"`
	Country           string    `db:"country" json:"country"`
	Phone             string    `db:"phone" json:"phone"`
	IsDefaultShipping bool      `db:"is_default_shipping" json:"isDefaultShipping"`
	IsDefaultBilling  bool      `db:"is_default_billing" json:"isDefaultBilling"`
	CreatedAt         time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt         time.Time `db:"updated_at" json:"updatedAt"`
}

func New(opts Options) *fiber.App {
	app := fiber.New()
	// Recover from panics with stack traces
	app.Use(fiberrecover.New())

	app.Use(cors.New(cors.Config{
		AllowOrigins:     opts.AllowedOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Authorization,Content-Type,Accept",
		AllowCredentials: true,
	}))

	// Structured request logging with request IDs
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

	// Health
	app.Get("/v1/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"success": true, "message": "ok"}) })
	app.Get("/v1/settings/pricing", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"success": true,
			"data": fiber.Map{
				"taxRatePercent":    opts.TaxRatePercent,
				"shippingFlatCents": opts.ShippingFlatCents,
			},
		})
	})

	// Static uploads
	_ = os.MkdirAll("/data/uploads/products", 0755)
	app.Static("/uploads", "/data/uploads")

	// Catalog: list all published products across shops (marketplace landing)
	app.Get("/v1/products", func(c *fiber.Ctx) error {
		q := strings.TrimSpace(c.Query("q"))
		category := strings.TrimSpace(c.Query("category"))
		minPrice := strings.TrimSpace(c.Query("minPrice"))
		maxPrice := strings.TrimSpace(c.Query("maxPrice"))
		var minCents, maxCents int64
		if minPrice != "" {
			if v, err := parseMoneyToCents(minPrice); err == nil {
				minCents = v
			}
		}
		if maxPrice != "" {
			if v, err := parseMoneyToCents(maxPrice); err == nil {
				maxCents = v
			}
		}
		var items []Product
		base := `SELECT p.uuid, p.shop_uuid, p.title, p.slug, p.summary, p.price_cents, p.currency, p.stock, p.image_url, p.created_at, p.updated_at, p.deleted_at, p.published,
                        p.category, p.category_uuid, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
                        s.name AS shop_name, s.slug AS shop_slug
                 FROM products p
                 JOIN shops s ON s.uuid = p.shop_uuid
                 WHERE p.deleted_at IS NULL AND p.published = true AND s.public = true`
		args := []any{}
		if q != "" {
			base += " AND (LOWER(p.title) LIKE LOWER($1) OR LOWER(s.name) LIKE LOWER($1))"
			args = append(args, "%"+q+"%")
		}
		if category != "" {
			base += " AND LOWER(p.category)=LOWER($" + itoa(len(args)+1) + ")"
			args = append(args, category)
		}
		if minCents > 0 {
			base += " AND p.price_cents >= $" + itoa(len(args)+1)
			args = append(args, minCents)
		}
		if maxCents > 0 {
			base += " AND p.price_cents <= $" + itoa(len(args)+1)
			args = append(args, maxCents)
		}
		base += " ORDER BY p.created_at DESC LIMIT 100"
		if err := opts.DB.Select(&items, base, args...); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": items})
	})
	// Search alias handled by same /v1/products with query params (no separate handler needed)

	// Product detail by UUID
	app.Get("/v1/products/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var item Product
		err := opts.DB.Get(&item, `SELECT p.uuid, p.shop_uuid, p.title, p.slug, p.summary, p.price_cents, p.currency, p.stock, p.image_url,
                                         p.category, p.category_uuid, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
                                         p.created_at, p.updated_at, p.deleted_at, p.published,
                                         s.name AS shop_name, s.slug AS shop_slug
                                  FROM products p JOIN shops s ON s.uuid=p.shop_uuid
                                  WHERE p.uuid=$1 AND p.deleted_at IS NULL`, id)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		return c.JSON(fiber.Map{"success": true, "data": item})
	})

	// Shops list and detail
	app.Get("/v1/shops", func(c *fiber.Ctx) error {
		var shops []Shop
		if err := opts.DB.Select(&shops, `SELECT uuid, name, slug, owner_uuid, public, description, created_at, updated_at FROM shops WHERE public=true ORDER BY created_at DESC LIMIT 100`); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": shops})
	})
	// Categories (distinct across public shops)
	app.Get("/v1/categories", func(c *fiber.Ctx) error {
		rows := []struct {
			Value string `db:"value"`
		}{}
		query := `SELECT value FROM (
                    SELECT DISTINCT LOWER(NULLIF(TRIM(c.slug), '')) AS value
                    FROM categories c
                    JOIN shops s ON s.uuid = c.shop_uuid
                    WHERE c.is_active = true AND s.public = true
                    UNION
                    SELECT DISTINCT LOWER(NULLIF(TRIM(p.category), '')) AS value
                    FROM products p
                    JOIN shops s ON s.uuid = p.shop_uuid
                    WHERE p.published = true AND p.deleted_at IS NULL AND s.public = true
                  ) AS combined
                  WHERE value IS NOT NULL
                  ORDER BY value ASC`
		if err := opts.DB.Select(&rows, query); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		out := make([]string, 0, len(rows))
		for _, r := range rows {
			if r.Value != "" {
				out = append(out, r.Value)
			}
		}
		return c.JSON(fiber.Map{"success": true, "data": out})
	})
	app.Get("/v1/shops/:slug", func(c *fiber.Ctx) error {
		slug := strings.TrimSpace(c.Params("slug"))
		var s Shop
		if err := opts.DB.Get(&s, `SELECT uuid, name, slug, owner_uuid, public, description, created_at, updated_at FROM shops WHERE slug=$1`, slug); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		return c.JSON(fiber.Map{"success": true, "data": s})
	})
	app.Get("/v1/shops/:slug/products", func(c *fiber.Ctx) error {
		slug := strings.TrimSpace(c.Params("slug"))
		var items []Product
		err := opts.DB.Select(&items, `SELECT p.uuid, p.shop_uuid, p.title, p.slug, p.summary, p.price_cents, p.currency, p.stock, p.image_url,
                                              p.category, p.category_uuid, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
                                              p.created_at, p.updated_at, p.deleted_at, p.published,
                                              s.name AS shop_name, s.slug AS shop_slug
                                       FROM products p JOIN shops s ON s.uuid=p.shop_uuid
                                       WHERE s.slug=$1 AND p.published=true AND p.deleted_at IS NULL
                                       ORDER BY p.created_at DESC`, slug)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": items})
	})

	// Auth-required: onboarding and product CRUD (simplified)
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
	app.Post("/v1/shops", requireAuth, func(c *fiber.Ctx) error {
		owner := srvAuth.UserID(c)
		var body struct {
			Name, Slug, Description string
			Meta                    map[string]any `json:"meta"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		name := strings.TrimSpace(body.Name)
		slug := strings.TrimSpace(strings.ToLower(body.Slug))
		if name == "" || slug == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "name and slug required"})
		}
		var exists int
		if err := opts.DB.Get(&exists, `SELECT COUNT(1) FROM shops WHERE slug=$1`, slug); err == nil && exists > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "message": "slug already used"})
		}
		allowed := srvAuth.HasAnyAppRole(c, srvAuth.RoleOwner, srvAuth.RoleManager) ||
			srvAuth.HasPlatformRole(c, srvAuth.PlatformRoleAdmin) ||
			srvAuth.HasPlatformRole(c, srvAuth.PlatformRoleSupport)
		if !allowed {
			var owned int
			_ = opts.DB.Get(&owned, `SELECT COUNT(1) FROM shops WHERE owner_uuid=$1`, owner)
			if owned > 0 {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
			}
		}
		id := uuid.New()
		var meta any
		if body.Meta != nil {
			if b, err := json.Marshal(body.Meta); err == nil {
				meta = string(b) // let Postgres cast text -> jsonb
			} else {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid meta"})
			}
		} else {
			meta = nil
		}
		if _, err := opts.DB.Exec(`INSERT INTO shops(uuid,name,slug,owner_uuid,public,description,meta) VALUES($1,$2,$3,$4,true,$5,$6)`, id, name, slug, owner, strings.TrimSpace(body.Description), meta); err != nil {
			log.Printf("create shop insert error: %v", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id}})
	})

	app.Post("/v1/products", requireAuth, func(c *fiber.Ctx) error {
		var body struct {
			ShopSlug     string   `json:"shopSlug"`
			Title        string   `json:"title"`
			Slug         string   `json:"slug"`
			Summary      string   `json:"summary"`
			PriceCents   int64    `json:"priceCents"`
			Currency     string   `json:"currency"`
			Stock        int64    `json:"stock"`
			ImageURL     *string  `json:"imageUrl"`
			Published    bool     `json:"published"`
			Category     string   `json:"category"`
			CategoryUUID string   `json:"categoryUuid"`
			Images       []string `json:"images"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		if body.Title == "" || body.Slug == "" || body.ShopSlug == "" || body.PriceCents < 0 || body.Currency == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "missing fields"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, body.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		if body.Stock < 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "stock cannot be negative"})
		}
		categoryValue := strings.TrimSpace(body.Category)
		var categoryUUID *uuid.UUID
		if strings.TrimSpace(body.CategoryUUID) != "" {
			categoryID, err := uuid.Parse(strings.TrimSpace(body.CategoryUUID))
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid category uuid"})
			}
			var cat struct {
				UUID uuid.UUID `db:"uuid"`
				Slug string    `db:"slug"`
				Name string    `db:"name"`
			}
			if err := opts.DB.Get(&cat, `SELECT uuid, slug, name FROM categories WHERE uuid=$1 AND shop_uuid=$2 AND is_active=true`, categoryID, shop.UUID); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "category not found for shop"})
			}
			categoryUUID = &cat.UUID
			if categoryValue == "" {
				categoryValue = cat.Slug
			}
		}
		id := uuid.New()
		var categoryUUIDParam any
		if categoryUUID != nil {
			categoryUUIDParam = *categoryUUID
		}
		_, err = opts.DB.Exec(`INSERT INTO products(uuid,shop_uuid,title,slug,summary,price_cents,currency,stock,image_url,published,category,category_uuid,images) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			id, shop.UUID, strings.TrimSpace(body.Title), strings.ToLower(strings.TrimSpace(body.Slug)), strings.TrimSpace(body.Summary), body.PriceCents, strings.ToUpper(body.Currency), body.Stock, body.ImageURL, body.Published, categoryValue, categoryUUIDParam, pqStringArray(body.Images),
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if categoryUUID != nil {
			if _, err := opts.DB.Exec(`INSERT INTO category_products(category_uuid, product_uuid, is_primary)
				VALUES($1,$2,true)
				ON CONFLICT (category_uuid, product_uuid) DO UPDATE SET is_primary=EXCLUDED.is_primary, created_at=category_products.created_at`,
				*categoryUUID, id); err != nil {
				log.Printf("category_products upsert error: %v", err)
			}
		}
		if err := upsertDefaultInventory(opts.DB, shop.UUID, id, body.Stock); err != nil {
			log.Printf("inventory upsert error: %v", err)
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id}})
	})

	app.Patch("/v1/products/:id", requireAuth, func(c *fiber.Ctx) error {
		id := strings.TrimSpace(c.Params("id"))
		var meta struct {
			ShopSlug string `db:"slug"`
		}
		if err := opts.DB.Get(&meta, `SELECT s.slug FROM products p JOIN shops s ON s.uuid=p.shop_uuid WHERE p.uuid=$1 AND p.deleted_at IS NULL`, id); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body map[string]any
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		sets := make([]string, 0, 8)
		args := make([]any, 0, 8)
		add := func(col string, v any) { sets = append(sets, col+"=$"+itoa(len(args)+1)); args = append(args, v) }
		if v, ok := body["title"].(string); ok {
			add("title", strings.TrimSpace(v))
		}
		if v, ok := body["slug"].(string); ok {
			add("slug", strings.ToLower(strings.TrimSpace(v)))
		}
		if v, ok := body["summary"].(string); ok {
			add("summary", strings.TrimSpace(v))
		}
		if v, ok := body["priceCents"].(float64); ok {
			add("price_cents", int64(v))
		}
		if v, ok := body["currency"].(string); ok {
			add("currency", strings.ToUpper(v))
		}
		var stockOverride *int64
		if raw, ok := body["stock"]; ok {
			switch v := raw.(type) {
			case float64:
				qty := int64(v)
				if qty < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "stock cannot be negative"})
				}
				stockOverride = &qty
				add("stock", qty)
			case int:
				qty := int64(v)
				if qty < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "stock cannot be negative"})
				}
				stockOverride = &qty
				add("stock", qty)
			case int64:
				if v < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "stock cannot be negative"})
				}
				val := v
				stockOverride = &val
				add("stock", val)
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid stock value"})
			}
		}
		if v, ok := body["imageUrl"].(string); ok {
			add("image_url", strings.TrimSpace(v))
		}
		if v, ok := body["published"].(bool); ok {
			add("published", v)
		}
		var categoryOverrideValue string
		var categoryOverrideSet bool
		var categoryUUIDOverride *uuid.UUID
		var categoryUUIDOverrideSet bool
		if raw, ok := body["categoryUuid"]; ok {
			categoryUUIDOverrideSet = true
			switch v := raw.(type) {
			case string:
				trimmed := strings.TrimSpace(v)
				if trimmed == "" {
					categoryUUIDOverride = nil
					categoryOverrideValue = ""
					categoryOverrideSet = true
				} else {
					categoryID, err := uuid.Parse(trimmed)
					if err != nil {
						return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid category uuid"})
					}
					var cat struct {
						UUID uuid.UUID `db:"uuid"`
						Slug string    `db:"slug"`
					}
					if err := opts.DB.Get(&cat, `SELECT uuid, slug FROM categories WHERE uuid=$1 AND shop_uuid=$2 AND is_active=true`, categoryID, shop.UUID); err != nil {
						return c.Status(400).JSON(fiber.Map{"success": false, "message": "category not found for shop"})
					}
					categoryUUIDOverride = &cat.UUID
					categoryOverrideValue = cat.Slug
					categoryOverrideSet = true
				}
			case nil:
				categoryUUIDOverride = nil
				categoryOverrideValue = ""
				categoryOverrideSet = true
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid category uuid"})
			}
		}
		if v, ok := body["category"].(string); ok {
			categoryOverrideValue = strings.TrimSpace(v)
			categoryOverrideSet = true
		}
		if categoryUUIDOverrideSet {
			if categoryUUIDOverride != nil {
				add("category_uuid", *categoryUUIDOverride)
			} else {
				add("category_uuid", nil)
			}
		}
		if categoryOverrideSet {
			add("category", categoryOverrideValue)
		}
		if v, ok := body["images"].([]any); ok {
			arr := make([]string, 0, len(v))
			for _, e := range v {
				if s, ok := e.(string); ok {
					arr = append(arr, s)
				}
			}
			add("images", pq.Array(arr))
		}
		if len(sets) == 0 {
			return c.JSON(fiber.Map{"success": true})
		}
		sets = append(sets, "updated_at=now()")
		args = append(args, id)
		query := "UPDATE products SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args)) + " AND deleted_at IS NULL"
		res, err := opts.DB.Exec(query, args...)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		if categoryUUIDOverrideSet {
			if categoryUUIDOverride != nil {
				if _, err := opts.DB.Exec(`INSERT INTO category_products(category_uuid, product_uuid, is_primary)
					VALUES($1,$2,true)
					ON CONFLICT (category_uuid, product_uuid) DO UPDATE SET is_primary=EXCLUDED.is_primary`, *categoryUUIDOverride, id); err != nil {
					log.Printf("category_products upsert error: %v", err)
				}
			} else {
				if _, err := opts.DB.Exec(`DELETE FROM category_products WHERE product_uuid=$1`, id); err != nil {
					log.Printf("category_products delete error: %v", err)
				}
			}
		}
		if stockOverride != nil {
			if productUUID, err := uuid.Parse(id); err == nil {
				if err := upsertDefaultInventory(opts.DB, shop.UUID, productUUID, *stockOverride); err != nil {
					log.Printf("inventory upsert error: %v", err)
				}
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Delete("/v1/products/:id", requireAuth, func(c *fiber.Ctx) error {
		id := strings.TrimSpace(c.Params("id"))
		var meta struct {
			ShopSlug string `db:"slug"`
		}
		if err := opts.DB.Get(&meta, `SELECT s.slug FROM products p JOIN shops s ON s.uuid=p.shop_uuid WHERE p.uuid=$1 AND p.deleted_at IS NULL`, id); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		res, err := tx.Exec(`UPDATE products SET deleted_at=now(), updated_at=now(), published=false WHERE uuid=$1 AND deleted_at IS NULL`, id)
		if err != nil {
			_ = tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			_ = tx.Rollback()
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		if _, err := tx.Exec(`DELETE FROM cart_items WHERE product_uuid=$1`, id); err != nil {
			_ = tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if _, err := tx.Exec(`DELETE FROM wishlist_items WHERE product_uuid=$1`, id); err != nil {
			_ = tx.Rollback()
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// My shops + products
	app.Get("/v1/my/shops", requireAuth, func(c *fiber.Ctx) error {
		if !hasDashboardAccess(c, opts.DB) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		owner := srvAuth.UserID(c)
		var shops []Shop
		const myShopsQuery = `
		SELECT * FROM (
			SELECT uuid,name,slug,owner_uuid,public,description,created_at,updated_at
			FROM shops
			WHERE owner_uuid=$1
			UNION
			SELECT s.uuid,s.name,s.slug,s.owner_uuid,s.public,s.description,s.created_at,s.updated_at
			FROM store_users su
			JOIN shops s ON s.uuid = su.store_uuid
			WHERE su.user_uuid=$1 AND su.status='active'
		) AS combined
		ORDER BY created_at DESC`
		if err := opts.DB.Select(&shops, myShopsQuery, owner); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": shops})
	})
	app.Get("/v1/my/shops/:slug/products", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var items []Product
		if err := opts.DB.Select(&items, `SELECT p.uuid, p.shop_uuid, p.title, p.slug, p.summary, p.price_cents, p.currency, p.stock, p.image_url,
                                              p.category, p.category_uuid, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
                                              p.created_at, p.updated_at, p.deleted_at, p.published,
                                              s.name AS shop_name, s.slug AS shop_slug
                                       FROM products p JOIN shops s ON s.uuid=p.shop_uuid
                                       WHERE s.uuid=$1 AND p.deleted_at IS NULL
                                       ORDER BY p.created_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": items})
	})

	app.Get("/v1/my/shops/:slug/categories", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var categories []Category
		if err := opts.DB.Select(&categories, `SELECT uuid, shop_uuid, parent_uuid, name, slug, description, sort_order, is_active, created_at, updated_at
                                               FROM categories
                                               WHERE shop_uuid=$1
                                               ORDER BY sort_order ASC, name ASC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": categories})
	})

	app.Post("/v1/my/shops/:slug/categories", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Name        string `json:"name"`
			Slug        string `json:"slug"`
			Description string `json:"description"`
			ParentUUID  string `json:"parentUuid"`
			SortOrder   *int   `json:"sortOrder"`
			IsActive    *bool  `json:"isActive"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		name := strings.TrimSpace(body.Name)
		slugValue := strings.TrimSpace(body.Slug)
		if name == "" || slugValue == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "name and slug are required"})
		}
		parentParam := any(nil)
		if strings.TrimSpace(body.ParentUUID) != "" {
			parentID, err := uuid.Parse(strings.TrimSpace(body.ParentUUID))
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid parent uuid"})
			}
			var exists int
			if err := opts.DB.Get(&exists, `SELECT COUNT(1) FROM categories WHERE uuid=$1 AND shop_uuid=$2`, parentID, shop.UUID); err != nil || exists == 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "parent category not found for shop"})
			}
			parentParam = parentID
		}
		sortOrder := 0
		if body.SortOrder != nil {
			sortOrder = *body.SortOrder
		}
		isActive := true
		if body.IsActive != nil {
			isActive = *body.IsActive
		}
		id := uuid.New()
		if _, err := opts.DB.Exec(`INSERT INTO categories(uuid, shop_uuid, parent_uuid, name, slug, description, sort_order, is_active)
                                    VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
			id, shop.UUID, parentParam, name, strings.ToLower(slugValue), strings.TrimSpace(body.Description), sortOrder, isActive); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id}})
	})

	app.Patch("/v1/my/shops/:slug/categories/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		id := c.Params("id")
		categoryID, err := uuid.Parse(id)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid category id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var exists int
		if err := opts.DB.Get(&exists, `SELECT COUNT(1) FROM categories WHERE uuid=$1 AND shop_uuid=$2`, categoryID, shop.UUID); err != nil || exists == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		var body map[string]any
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		sets := make([]string, 0, 8)
		args := make([]any, 0, 8)
		add := func(col string, v any) { sets = append(sets, col+"=$"+itoa(len(args)+1)); args = append(args, v) }
		if v, ok := body["name"].(string); ok {
			name := strings.TrimSpace(v)
			if name != "" {
				add("name", name)
			}
		}
		if v, ok := body["slug"].(string); ok {
			s := strings.TrimSpace(v)
			if s != "" {
				add("slug", strings.ToLower(s))
			}
		}
		if v, ok := body["description"].(string); ok {
			add("description", strings.TrimSpace(v))
		}
		if v, ok := body["sortOrder"]; ok {
			switch val := v.(type) {
			case float64:
				add("sort_order", int(val))
			case int:
				add("sort_order", val)
			}
		}
		if v, ok := body["isActive"].(bool); ok {
			add("is_active", v)
		}
		if raw, ok := body["parentUuid"]; ok {
			if raw == nil {
				add("parent_uuid", nil)
			} else if s, ok := raw.(string); ok {
				trimmed := strings.TrimSpace(s)
				if trimmed == "" {
					add("parent_uuid", nil)
				} else {
					parentID, err := uuid.Parse(trimmed)
					if err != nil || parentID == categoryID {
						return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid parent uuid"})
					}
					var count int
					if err := opts.DB.Get(&count, `SELECT COUNT(1) FROM categories WHERE uuid=$1 AND shop_uuid=$2`, parentID, shop.UUID); err != nil || count == 0 {
						return c.Status(400).JSON(fiber.Map{"success": false, "message": "parent category not found for shop"})
					}
					add("parent_uuid", parentID)
				}
			} else {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid parent uuid"})
			}
		}
		if len(sets) == 0 {
			return c.JSON(fiber.Map{"success": true})
		}
		sets = append(sets, "updated_at=now()")
		args = append(args, categoryID)
		query := "UPDATE categories SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args))
		if _, err := opts.DB.Exec(query, args...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Delete("/v1/my/shops/:slug/categories/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		id := c.Params("id")
		categoryID, err := uuid.Parse(id)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid category id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		res, err := opts.DB.Exec(`UPDATE categories SET is_active=false, updated_at=now() WHERE uuid=$1 AND shop_uuid=$2`, categoryID, shop.UUID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		if _, err := opts.DB.Exec(`DELETE FROM category_products WHERE category_uuid=$1`, categoryID); err != nil {
			log.Printf("category_products cleanup error: %v", err)
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Get("/v1/my/shops/:slug/inventory", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var levels []InventoryEntry
		if err := opts.DB.Select(&levels, `SELECT l.uuid, l.shop_uuid, l.product_uuid, l.location_uuid, l.quantity, l.reserved, l.safety_stock, l.created_at, l.updated_at,
                                                   p.title AS product_title, p.slug AS product_slug,
                                                   loc.name AS location_name, loc.code AS location_code
                                            FROM inventory_levels l
                                            JOIN products p ON p.uuid = l.product_uuid
                                            LEFT JOIN inventory_locations loc ON loc.uuid = l.location_uuid
                                            WHERE l.shop_uuid=$1
                                            ORDER BY p.title ASC, l.updated_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": levels})
	})

	app.Patch("/v1/my/shops/:slug/inventory/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		levelParam := c.Params("id")
		levelID, err := uuid.Parse(levelParam)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid inventory id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var current InventoryLevel
		if err := opts.DB.Get(&current, `SELECT uuid, shop_uuid, product_uuid, location_uuid, quantity, reserved, safety_stock, created_at, updated_at FROM inventory_levels WHERE uuid=$1`, levelID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if current.ShopUUID != shop.UUID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body map[string]any
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		targetQuantity := current.Quantity
		targetReserved := current.Reserved
		targetSafety := current.SafetyStock
		if raw, ok := body["quantity"]; ok {
			switch v := raw.(type) {
			case float64:
				targetQuantity = int64(v)
			case int:
				targetQuantity = int64(v)
			case int64:
				targetQuantity = v
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid quantity"})
			}
			if targetQuantity < 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "quantity cannot be negative"})
			}
		}
		if raw, ok := body["reserved"]; ok {
			switch v := raw.(type) {
			case float64:
				targetReserved = int64(v)
			case int:
				targetReserved = int64(v)
			case int64:
				targetReserved = v
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid reserved value"})
			}
			if targetReserved < 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "reserved cannot be negative"})
			}
		}
		if raw, ok := body["safetyStock"]; ok {
			switch v := raw.(type) {
			case float64:
				targetSafety = int64(v)
			case int:
				targetSafety = int64(v)
			case int64:
				targetSafety = v
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid safety stock value"})
			}
			if targetSafety < 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "safety stock cannot be negative"})
			}
		}
		if targetReserved > targetQuantity {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "reserved cannot exceed quantity"})
		}
		if targetQuantity == current.Quantity && targetReserved == current.Reserved && targetSafety == current.SafetyStock {
			return c.JSON(fiber.Map{"success": true})
		}
		if _, err := opts.DB.Exec(`UPDATE inventory_levels SET quantity=$1, reserved=$2, safety_stock=$3, updated_at=now() WHERE uuid=$4`,
			targetQuantity, targetReserved, targetSafety, levelID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := syncProductStockFromInventory(opts.DB, current.ProductUUID); err != nil {
			log.Printf("inventory sync error: %v", err)
		}
		var updated InventoryEntry
		if err := opts.DB.Get(&updated, `SELECT l.uuid, l.shop_uuid, l.product_uuid, l.location_uuid, l.quantity, l.reserved, l.safety_stock, l.created_at, l.updated_at,
                                                   p.title AS product_title, p.slug AS product_slug,
                                                   loc.name AS location_name, loc.code AS location_code
                                            FROM inventory_levels l
                                            JOIN products p ON p.uuid = l.product_uuid
                                            LEFT JOIN inventory_locations loc ON loc.uuid = l.location_uuid
                                            WHERE l.uuid=$1`, levelID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": updated})
	})

	app.Get("/v1/my/shops/:slug/collections", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var collections []Collection
		if err := opts.DB.Select(&collections, `SELECT uuid, shop_uuid, title, slug, description, is_automatic, rules, sort_order, is_active, created_at, updated_at
                                               FROM collections WHERE shop_uuid=$1 ORDER BY sort_order ASC, title ASC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": collections})
	})

	app.Post("/v1/my/shops/:slug/collections", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Title        string          `json:"title"`
			Slug         string          `json:"slug"`
			Description  string          `json:"description"`
			IsAutomatic  bool            `json:"isAutomatic"`
			Rules        json.RawMessage `json:"rules"`
			SortOrder    *int            `json:"sortOrder"`
			IsActive     *bool           `json:"isActive"`
			ProductUUIDs []string        `json:"productUuids"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		title := strings.TrimSpace(body.Title)
		slugValue := strings.TrimSpace(body.Slug)
		if title == "" || slugValue == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "title and slug are required"})
		}
		sortOrder := 0
		if body.SortOrder != nil {
			sortOrder = *body.SortOrder
		}
		isActive := true
		if body.IsActive != nil {
			isActive = *body.IsActive
		}
		id := uuid.New()
		if _, err := opts.DB.Exec(`INSERT INTO collections(uuid, shop_uuid, title, slug, description, is_automatic, rules, sort_order, is_active)
                                    VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			id, shop.UUID, title, strings.ToLower(slugValue), strings.TrimSpace(body.Description), body.IsAutomatic, nullIfEmptyJSON(body.Rules), sortOrder, isActive); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if len(body.ProductUUIDs) > 0 {
			productIDs, err := parseUUIDList(body.ProductUUIDs)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
			if err := setCollectionProducts(opts.DB, id, shop.UUID, productIDs); err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id}})
	})

	app.Patch("/v1/my/shops/:slug/collections/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		collectionID, err := uuid.Parse(idParam)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid collection id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var exists int
		if err := opts.DB.Get(&exists, `SELECT COUNT(1) FROM collections WHERE uuid=$1 AND shop_uuid=$2`, collectionID, shop.UUID); err != nil || exists == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		var body map[string]any
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		sets := make([]string, 0, 8)
		args := make([]any, 0, 8)
		add := func(col string, v any) { sets = append(sets, col+"=$"+itoa(len(args)+1)); args = append(args, v) }
		if v, ok := body["title"].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				add("title", s)
			}
		}
		if v, ok := body["slug"].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				add("slug", strings.ToLower(s))
			}
		}
		if v, ok := body["description"].(string); ok {
			add("description", strings.TrimSpace(v))
		}
		if v, ok := body["isAutomatic"].(bool); ok {
			add("is_automatic", v)
		}
		if raw, ok := body["rules"]; ok {
			b, err := json.Marshal(raw)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid rules"})
			}
			if len(b) == 0 || string(b) == "null" {
				add("rules", nil)
			} else {
				add("rules", b)
			}
		}
		if raw, ok := body["sortOrder"]; ok {
			switch v := raw.(type) {
			case float64:
				add("sort_order", int(v))
			case int:
				add("sort_order", v)
			case int64:
				add("sort_order", v)
			}
		}
		if v, ok := body["isActive"].(bool); ok {
			add("is_active", v)
		}
		var productUpdateIDs []uuid.UUID
		if raw, ok := body["productUuids"]; ok {
			switch arr := raw.(type) {
			case []any:
				strs := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						strs = append(strs, s)
					}
				}
				ids, err := parseUUIDList(strs)
				if err != nil {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
				}
				productUpdateIDs = ids
			case nil:
				productUpdateIDs = []uuid.UUID{}
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid productUuids"})
			}
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, collectionID)
			query := "UPDATE collections SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args))
			if _, err := opts.DB.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if productUpdateIDs != nil {
			if err := setCollectionProducts(opts.DB, collectionID, shop.UUID, productUpdateIDs); err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Delete("/v1/my/shops/:slug/collections/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		collectionID, err := uuid.Parse(idParam)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid collection id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		res, err := opts.DB.Exec(`UPDATE collections SET is_active=false, updated_at=now() WHERE uuid=$1 AND shop_uuid=$2`, collectionID, shop.UUID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		if _, err := opts.DB.Exec(`DELETE FROM collection_products WHERE collection_uuid=$1`, collectionID); err != nil {
			log.Printf("collection_products cleanup error: %v", err)
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Get("/v1/my/shops/:slug/discounts", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var discounts []Discount
		if err := opts.DB.Select(&discounts, `SELECT uuid, shop_uuid, name, code, description, discount_type, amount_cents, percentage, starts_at, ends_at,
                                                     usage_limit_total, usage_limit_per_customer, auto_apply, status, applies_to, created_at, updated_at
                                              FROM discounts
                                              WHERE shop_uuid=$1
                                              ORDER BY created_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": discounts})
	})

	app.Post("/v1/my/shops/:slug/discounts", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Name                  string          `json:"name"`
			Code                  string          `json:"code"`
			Description           string          `json:"description"`
			DiscountType          string          `json:"discountType"`
			AmountCents           *int64          `json:"amountCents"`
			Percentage            *float64        `json:"percentage"`
			StartsAt              *time.Time      `json:"startsAt"`
			EndsAt                *time.Time      `json:"endsAt"`
			UsageLimitTotal       *int            `json:"usageLimitTotal"`
			UsageLimitPerCustomer *int            `json:"usageLimitPerCustomer"`
			AutoApply             bool            `json:"autoApply"`
			Status                string          `json:"status"`
			AppliesTo             json.RawMessage `json:"appliesTo"`
			ProductUUIDs          []string        `json:"productUuids"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		name := strings.TrimSpace(body.Name)
		code := strings.ToUpper(strings.TrimSpace(body.Code))
		if name == "" || code == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "name and code are required"})
		}
		discountType := strings.ToLower(strings.TrimSpace(body.DiscountType))
		if discountType == "" {
			discountType = "percentage"
		}
		if discountType != "percentage" && discountType != "amount" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount type"})
		}
		amount := int64(0)
		if body.AmountCents != nil {
			amount = *body.AmountCents
		}
		percentage := 0.0
		if body.Percentage != nil {
			percentage = *body.Percentage
		}
		if discountType == "amount" && amount <= 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents must be greater than zero for amount discounts"})
		}
		if discountType == "percentage" && percentage <= 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage must be greater than zero for percentage discounts"})
		}
		status := normalizeDiscountStatus(body.Status)
		id := uuid.New()
		if _, err := opts.DB.Exec(`INSERT INTO discounts(uuid, shop_uuid, name, code, description, discount_type, amount_cents, percentage, starts_at, ends_at,
                                               usage_limit_total, usage_limit_per_customer, auto_apply, status, applies_to)
                                    VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			id, shop.UUID, name, code, strings.TrimSpace(body.Description), discountType, amount, percentage, body.StartsAt, body.EndsAt, body.UsageLimitTotal, body.UsageLimitPerCustomer, body.AutoApply, status, nullIfEmptyJSON(body.AppliesTo)); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if len(body.ProductUUIDs) > 0 {
			productIDs, err := parseUUIDList(body.ProductUUIDs)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
			if err := setDiscountProducts(opts.DB, id, shop.UUID, productIDs); err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id}})
	})

	app.Patch("/v1/my/shops/:slug/discounts/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		discountID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var exists int
		if err := opts.DB.Get(&exists, `SELECT COUNT(1) FROM discounts WHERE uuid=$1 AND shop_uuid=$2`, discountID, shop.UUID); err != nil || exists == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		var body map[string]any
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		sets := make([]string, 0, 12)
		args := make([]any, 0, 12)
		add := func(col string, v any) { sets = append(sets, col+"=$"+itoa(len(args)+1)); args = append(args, v) }
		if v, ok := body["name"].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				add("name", s)
			}
		}
		if v, ok := body["code"].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				add("code", strings.ToUpper(s))
			}
		}
		if v, ok := body["description"].(string); ok {
			add("description", strings.TrimSpace(v))
		}
		if v, ok := body["discountType"].(string); ok {
			t := strings.ToLower(strings.TrimSpace(v))
			if t != "" && t != "percentage" && t != "amount" {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount type"})
			}
			if t != "" {
				add("discount_type", t)
			}
		}
		if raw, ok := body["amountCents"]; ok {
			switch val := raw.(type) {
			case float64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents cannot be negative"})
				}
				add("amount_cents", int64(val))
			case int:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents cannot be negative"})
				}
				add("amount_cents", int64(val))
			case int64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents cannot be negative"})
				}
				add("amount_cents", val)
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid amountCents"})
			}
		}
		if raw, ok := body["percentage"]; ok {
			switch val := raw.(type) {
			case float64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage cannot be negative"})
				}
				add("percentage", val)
			case int:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage cannot be negative"})
				}
				add("percentage", float64(val))
			case int64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage cannot be negative"})
				}
				add("percentage", float64(val))
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid percentage"})
			}
		}
		if v, ok := body["startsAt"].(string); ok {
			if v == "" {
				add("starts_at", nil)
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				add("starts_at", t)
			} else {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid startsAt"})
			}
		}
		if v, ok := body["endsAt"].(string); ok {
			if v == "" {
				add("ends_at", nil)
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				add("ends_at", t)
			} else {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid endsAt"})
			}
		}
		if raw, ok := body["usageLimitTotal"]; ok {
			switch val := raw.(type) {
			case float64:
				add("usage_limit_total", int(val))
			case int:
				add("usage_limit_total", val)
			case int64:
				add("usage_limit_total", val)
			case nil:
				add("usage_limit_total", nil)
			}
		}
		if raw, ok := body["usageLimitPerCustomer"]; ok {
			switch val := raw.(type) {
			case float64:
				add("usage_limit_per_customer", int(val))
			case int:
				add("usage_limit_per_customer", val)
			case int64:
				add("usage_limit_per_customer", val)
			case nil:
				add("usage_limit_per_customer", nil)
			}
		}
		if v, ok := body["autoApply"].(bool); ok {
			add("auto_apply", v)
		}
		if v, ok := body["status"].(string); ok {
			add("status", normalizeDiscountStatus(v))
		}
		if raw, ok := body["appliesTo"]; ok {
			b, err := json.Marshal(raw)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid appliesTo"})
			}
			add("applies_to", nullIfEmptyJSON(b))
		}
		var productUpdateIDs []uuid.UUID
		if raw, ok := body["productUuids"]; ok {
			switch arr := raw.(type) {
			case []any:
				strs := make([]string, 0, len(arr))
				for _, item := range arr {
					if s, ok := item.(string); ok {
						strs = append(strs, s)
					}
				}
				ids, err := parseUUIDList(strs)
				if err != nil {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
				}
				productUpdateIDs = ids
			case nil:
				productUpdateIDs = []uuid.UUID{}
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid productUuids"})
			}
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, discountID)
			query := "UPDATE discounts SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args))
			if _, err := opts.DB.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if productUpdateIDs != nil {
			if err := setDiscountProducts(opts.DB, discountID, shop.UUID, productUpdateIDs); err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Delete("/v1/my/shops/:slug/discounts/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		discountID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		res, err := opts.DB.Exec(`UPDATE discounts SET status='archived', updated_at=now() WHERE uuid=$1 AND shop_uuid=$2`, discountID, shop.UUID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		if _, err := opts.DB.Exec(`DELETE FROM discount_products WHERE discount_uuid=$1`, discountID); err != nil {
			log.Printf("discount_products cleanup error: %v", err)
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Get("/v1/my/shops/:slug/gift-cards", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var cards []GiftCard
		if err := opts.DB.Select(&cards, `SELECT uuid, shop_uuid, code, balance_cents, original_balance_cents, currency, issued_to_email, note, status, expires_at, issued_at, redeemed_at, created_at, updated_at
                                         FROM gift_cards WHERE shop_uuid=$1 ORDER BY created_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": cards})
	})

	app.Post("/v1/my/shops/:slug/gift-cards", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Code         string     `json:"code"`
			BalanceCents int64      `json:"balanceCents"`
			Currency     string     `json:"currency"`
			IssuedTo     *string    `json:"issuedToEmail"`
			Note         string     `json:"note"`
			Status       string     `json:"status"`
			ExpiresAt    *time.Time `json:"expiresAt"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		code := strings.ToUpper(strings.TrimSpace(body.Code))
		if code == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "code is required"})
		}
		if body.BalanceCents <= 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "balance must be greater than zero"})
		}
		currency := strings.ToUpper(strings.TrimSpace(body.Currency))
		if currency == "" {
			currency = "USD"
		}
		status := normalizeGiftCardStatus(body.Status)
		id := uuid.New()
		if _, err := opts.DB.Exec(`INSERT INTO gift_cards(uuid, shop_uuid, code, balance_cents, original_balance_cents, currency, issued_to_email, note, status, expires_at)
                                    VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			id, shop.UUID, code, body.BalanceCents, body.BalanceCents, currency, body.IssuedTo, strings.TrimSpace(body.Note), status, body.ExpiresAt); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := addGiftCardTransaction(opts.DB, id, body.BalanceCents, "issued"); err != nil {
			log.Printf("gift card transaction error: %v", err)
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id}})
	})

	app.Patch("/v1/my/shops/:slug/gift-cards/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		cardID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid gift card id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var current GiftCard
		if err := opts.DB.Get(&current, `SELECT uuid, shop_uuid, code, balance_cents, original_balance_cents, currency, issued_to_email, note, status, expires_at, issued_at, redeemed_at, created_at, updated_at
                                         FROM gift_cards WHERE uuid=$1 AND shop_uuid=$2`, cardID, shop.UUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var body map[string]any
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		sets := make([]string, 0, 8)
		args := make([]any, 0, 8)
		add := func(col string, v any) { sets = append(sets, col+"=$"+itoa(len(args)+1)); args = append(args, v) }
		if v, ok := body["note"].(string); ok {
			add("note", strings.TrimSpace(v))
		}
		if v, ok := body["expiresAt"].(string); ok {
			if v == "" {
				add("expires_at", nil)
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				add("expires_at", t)
			} else {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid expiresAt"})
			}
		}
		if v, ok := body["status"].(string); ok {
			newStatus := normalizeGiftCardStatus(v)
			add("status", newStatus)
			if newStatus == "redeemed" && current.RedeemedAt == nil {
				add("redeemed_at", time.Now())
			}
		}
		if raw, ok := body["balanceAdjustmentCents"]; ok {
			var adjust int64
			switch val := raw.(type) {
			case float64:
				adjust = int64(val)
			case int:
				adjust = int64(val)
			case int64:
				adjust = val
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid balanceAdjustmentCents"})
			}
			newBalance := current.BalanceCents + adjust
			if newBalance < 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "insufficient balance"})
			}
			add("balance_cents", newBalance)
			if adjust != 0 {
				if err := addGiftCardTransaction(opts.DB, cardID, adjust, "manual_adjustment"); err != nil {
					log.Printf("gift card transaction error: %v", err)
				}
				current.BalanceCents = newBalance
			}
		}
		if email, ok := body["issuedToEmail"].(string); ok {
			if strings.TrimSpace(email) == "" {
				add("issued_to_email", nil)
			} else {
				add("issued_to_email", strings.TrimSpace(email))
			}
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, cardID)
			query := "UPDATE gift_cards SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args))
			if _, err := opts.DB.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Get("/v1/my/shops/:slug/gift-cards/:id/transactions", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		cardID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid gift card id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var okCount int
		if err := opts.DB.Get(&okCount, `SELECT COUNT(1) FROM gift_cards WHERE uuid=$1 AND shop_uuid=$2`, cardID, shop.UUID); err != nil || okCount == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		var txs []GiftCardTransaction
		if err := opts.DB.Select(&txs, `SELECT uuid, gift_card_uuid, change_cents, reason, created_at
                                        FROM gift_card_transactions
                                        WHERE gift_card_uuid=$1
                                        ORDER BY created_at DESC`, cardID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": txs})
	})

	app.Get("/v1/my/shops/:slug/customers", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		q := strings.TrimSpace(c.Query("q"))
		tag := strings.TrimSpace(c.Query("tag"))
		query := `SELECT c.uuid, c.shop_uuid, c.user_uuid, c.email, c.first_name, c.last_name, c.phone, c.tags, c.notes, c.marketing_opt_in, c.created_at, c.updated_at,
                         COALESCE(SUM(oi.price_cents * oi.quantity),0) AS total_spent_cents,
                         COUNT(DISTINCT CASE WHEN o.uuid IS NOT NULL AND p.uuid IS NOT NULL THEN o.uuid END) AS orders_count,
                         MAX(o.created_at) AS last_order_at,
                         TRIM(BOTH ' ' FROM COALESCE(c.first_name,'') || ' ' || COALESCE(c.last_name,'')) AS customer_name
                  FROM customers c
                  LEFT JOIN orders o ON c.user_uuid IS NOT NULL AND o.user_uuid=c.user_uuid
                  LEFT JOIN order_items oi ON oi.order_uuid=o.uuid
                  LEFT JOIN products p ON p.uuid=oi.product_uuid AND p.shop_uuid=c.shop_uuid
                  WHERE c.shop_uuid=$1`
		args := []any{shop.UUID}
		if q != "" {
			like := "%" + strings.ToLower(q) + "%"
			args = append(args, like)
			idx := "$" + itoa(len(args))
			query += " AND (LOWER(c.email) LIKE " + idx + " OR LOWER(c.first_name) LIKE " + idx + " OR LOWER(c.last_name) LIKE " + idx + ")"
		}
		if tag != "" {
			args = append(args, strings.ToLower(tag))
			idx := "$" + itoa(len(args))
			query += " AND EXISTS (SELECT 1 FROM unnest(c.tags) AS tag WHERE LOWER(tag)=LOWER(" + idx + "))"
		}
		query += " GROUP BY c.uuid ORDER BY last_order_at DESC NULLS LAST, c.created_at DESC LIMIT 200"
		var customers []CustomerSummary
		if err := opts.DB.Select(&customers, query, args...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": customers})
	})

	app.Post("/v1/my/shops/:slug/customers", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Email          string   `json:"email"`
			FirstName      string   `json:"firstName"`
			LastName       string   `json:"lastName"`
			Phone          string   `json:"phone"`
			Notes          string   `json:"notes"`
			Tags           []string `json:"tags"`
			MarketingOptIn bool     `json:"marketingOptIn"`
			UserUUID       string   `json:"userUuid"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		email, err := normalizeEmail(body.Email)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid email"})
		}
		first := strings.TrimSpace(body.FirstName)
		last := strings.TrimSpace(body.LastName)
		phone := strings.TrimSpace(body.Phone)
		notes := strings.TrimSpace(body.Notes)
		tags := normalizeTags(body.Tags)
		userUUID, err := parseOptionalUUID(body.UserUUID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid user uuid"})
		}
		now := time.Now()
		tempID := uuid.New()
		var userValue any
		if userUUID != nil {
			userValue = *userUUID
		}
		var created CustomerSummary
		if err := opts.DB.Get(&created, `INSERT INTO customers (uuid, shop_uuid, user_uuid, email, first_name, last_name, phone, tags, notes, marketing_opt_in, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)
			ON CONFLICT (shop_uuid, email) DO UPDATE SET
				user_uuid=COALESCE(EXCLUDED.user_uuid, customers.user_uuid),
				first_name=EXCLUDED.first_name,
				last_name=EXCLUDED.last_name,
				phone=EXCLUDED.phone,
				tags=EXCLUDED.tags,
				notes=EXCLUDED.notes,
				marketing_opt_in=EXCLUDED.marketing_opt_in,
				updated_at=now()
			RETURNING uuid, shop_uuid, user_uuid, email, first_name, last_name, phone, tags, notes, marketing_opt_in, created_at, updated_at`,
			tempID, shop.UUID, userValue, email, first, last, phone, pqStringArray(tags), notes, body.MarketingOptIn, now); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		summary, err := fetchCustomerSummary(opts.DB, created.UUID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": summary})
	})

	app.Get("/v1/shops/:slug/team", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var members []StoreUser
		if err := opts.DB.Select(&members, `SELECT uuid,store_uuid,user_uuid,role,status,created_at,updated_at FROM store_users WHERE store_uuid=$1 ORDER BY created_at ASC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var invites []StoreInvitation
		if err := opts.DB.Select(&invites, `SELECT uuid,store_uuid,email,role,token,status,invited_by,expires_at,created_at,updated_at,accepted_at FROM store_invitations WHERE store_uuid=$1 ORDER BY created_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		now := time.Now()
		for i := range invites {
			if !strings.EqualFold(invites[i].Status, "pending") || now.After(invites[i].ExpiresAt) {
				invites[i].Token = ""
			}
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"members":     members,
			"invitations": invites,
		}})
	})

	app.Post("/v1/shops/:slug/team", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsInvites(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			UserUUID string `json:"userUuid"`
			Role     string `json:"role"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		userUUID, err := uuid.Parse(strings.TrimSpace(body.UserUUID))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid userUuid"})
		}
		if shop.OwnerUUID == userUUID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "owner already manages this shop"})
		}
		teamRole, err := normalizeTeamRole(body.Role)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid role"})
		}
		_, err = opts.DB.Exec(`INSERT INTO store_users (store_uuid,user_uuid,role,status) VALUES($1,$2,$3,'active')
			ON CONFLICT (store_uuid,user_uuid) DO UPDATE SET role=excluded.role, status='active', updated_at=now()`,
			shop.UUID, userUUID, teamRole)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var member StoreUser
		if err := opts.DB.Get(&member, `SELECT uuid,store_uuid,user_uuid,role,status,created_at,updated_at FROM store_users WHERE store_uuid=$1 AND user_uuid=$2`, shop.UUID, userUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": member})
	})

	app.Patch("/v1/shops/:slug/team/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		teamID := strings.TrimSpace(c.Params("id"))
		memberUUID, err := uuid.Parse(teamID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid team member id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsInvites(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Role string `json:"role"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		teamRole, err := normalizeTeamRole(body.Role)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid role"})
		}
		res, err := opts.DB.Exec(`UPDATE store_users SET role=$1, updated_at=now() WHERE uuid=$2 AND store_uuid=$3`, teamRole, memberUUID, shop.UUID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "team member not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Delete("/v1/shops/:slug/team/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		memberID := strings.TrimSpace(c.Params("id"))
		memberUUID, err := uuid.Parse(memberID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid team member id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsInvites(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		res, err := opts.DB.Exec(`DELETE FROM store_users WHERE uuid=$1 AND store_uuid=$2`, memberUUID, shop.UUID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "team member not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Post("/v1/shops/:slug/invitations", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsInvites(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		email := strings.ToLower(strings.TrimSpace(body.Email))
		if email == "" || !strings.Contains(email, "@") {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid email"})
		}
		teamRole, err := normalizeTeamRole(body.Role)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid role"})
		}
		invitedBy := srvAuth.UserID(c)
		inviterUUID, err := uuid.Parse(invitedBy)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid inviter"})
		}
		var existing int
		_ = opts.DB.Get(&existing, `SELECT COUNT(1) FROM store_invitations WHERE store_uuid=$1 AND LOWER(email)=LOWER($2) AND status='pending'`, shop.UUID, email)
		if existing > 0 {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "message": "invitation already pending for this email"})
		}
		token := uuid.NewString()
		if _, err := opts.DB.Exec(`INSERT INTO store_invitations (store_uuid,email,role,token,status,invited_by) VALUES($1,$2,$3,$4,'pending',$5)`,
			shop.UUID, email, teamRole, token, inviterUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var invite StoreInvitation
		if err := opts.DB.Get(&invite, `SELECT uuid,store_uuid,email,role,token,status,invited_by,expires_at,created_at,updated_at,accepted_at FROM store_invitations WHERE store_uuid=$1 AND token=$2`, shop.UUID, token); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": invite})
	})

	app.Post("/v1/shops/:slug/transfer", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsTransfer(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			NewOwnerUUID string `json:"newOwnerUuid"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		newOwnerStr := strings.TrimSpace(body.NewOwnerUUID)
		if newOwnerStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "new owner uuid required"})
		}
		newOwnerUUID, err := uuid.Parse(newOwnerStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid owner uuid"})
		}
		if newOwnerUUID == shop.OwnerUUID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "new owner matches current owner"})
		}
		var member StoreUser
		if err := opts.DB.Get(&member, `SELECT uuid, store_uuid, user_uuid, role, status, created_at, updated_at FROM store_users WHERE store_uuid=$1 AND user_uuid=$2`, shop.UUID, newOwnerUUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "target user is not a team member"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()
		if _, err := tx.Exec(`DELETE FROM store_users WHERE store_uuid=$1 AND user_uuid=$2`, shop.UUID, newOwnerUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if _, err := tx.Exec(`INSERT INTO store_users (uuid, store_uuid, user_uuid, role, status)
			VALUES ($1,$2,$3,'manager','active')
			ON CONFLICT (store_uuid, user_uuid) DO UPDATE SET role='manager', status='active', updated_at=now()`,
			uuid.New(), shop.UUID, shop.OwnerUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if _, err := tx.Exec(`UPDATE shops SET owner_uuid=$1, updated_at=now() WHERE uuid=$2`, newOwnerUUID, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"ownerUuid": newOwnerUUID}})
	})

	app.Get("/v1/customers/:id", requireAuth, func(c *fiber.Ctx) error {
		customerParam := strings.TrimSpace(c.Params("id"))
		if customerParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		customerID, err := uuid.Parse(customerParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		meta, err := loadCustomerMeta(opts.DB, customerID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		summary, err := fetchCustomerSummary(opts.DB, customerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": summary})
	})

	app.Patch("/v1/customers/:id", requireAuth, func(c *fiber.Ctx) error {
		customerParam := strings.TrimSpace(c.Params("id"))
		if customerParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		customerID, err := uuid.Parse(customerParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		meta, err := loadCustomerMeta(opts.DB, customerID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Email          *string   `json:"email"`
			FirstName      *string   `json:"firstName"`
			LastName       *string   `json:"lastName"`
			Phone          *string   `json:"phone"`
			Notes          *string   `json:"notes"`
			Tags           *[]string `json:"tags"`
			MarketingOptIn *bool     `json:"marketingOptIn"`
			UserUUID       *string   `json:"userUuid"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		args := []any{}
		sets := []string{}
		add := func(field string, value any) {
			args = append(args, value)
			sets = append(sets, field+"=$"+itoa(len(args)))
		}
		if body.Email != nil {
			email, err := normalizeEmail(*body.Email)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid email"})
			}
			add("email", email)
		}
		if body.FirstName != nil {
			add("first_name", strings.TrimSpace(*body.FirstName))
		}
		if body.LastName != nil {
			add("last_name", strings.TrimSpace(*body.LastName))
		}
		if body.Phone != nil {
			add("phone", strings.TrimSpace(*body.Phone))
		}
		if body.Notes != nil {
			add("notes", strings.TrimSpace(*body.Notes))
		}
		if body.Tags != nil {
			add("tags", pqStringArray(normalizeTags(*body.Tags)))
		}
		if body.MarketingOptIn != nil {
			add("marketing_opt_in", *body.MarketingOptIn)
		}
		if body.UserUUID != nil {
			userUUID, err := parseOptionalUUID(*body.UserUUID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid user uuid"})
			}
			if userUUID != nil {
				args = append(args, *userUUID)
			} else {
				args = append(args, nil)
			}
			sets = append(sets, "user_uuid=$"+itoa(len(args)))
		}
		if len(sets) == 0 {
			summary, err := fetchCustomerSummary(opts.DB, customerID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
				}
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			return c.JSON(fiber.Map{"success": true, "data": summary})
		}
		sets = append(sets, "updated_at=now()")
		args = append(args, customerID, meta.ShopUUID)
		query := "UPDATE customers SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args)-1) + " AND shop_uuid=$" + itoa(len(args))
		if _, err := opts.DB.Exec(query, args...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		summary, err := fetchCustomerSummary(opts.DB, customerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": summary})
	})

	app.Get("/v1/customers/:id/orders", requireAuth, func(c *fiber.Ctx) error {
		customerParam := strings.TrimSpace(c.Params("id"))
		if customerParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		customerID, err := uuid.Parse(customerParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		meta, err := loadCustomerMeta(opts.DB, customerID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		if meta.UserUUID == nil {
			return c.JSON(fiber.Map{"success": true, "data": []CustomerOrder{}})
		}
		var orders []CustomerOrder
		if err := opts.DB.Select(&orders, `SELECT o.uuid,
                                                   SUM(oi.price_cents * oi.quantity) AS total_cents,
                                                   o.currency,
                                                   o.status,
                                                   o.created_at,
                                                   o.updated_at,
                                                   o.tracking_number,
                                                   o.tracking_url,
                                                   o.shipping_carrier,
                                                   o.shipped_at,
                                                   o.delivered_at
                                            FROM orders o
                                            JOIN order_items oi ON oi.order_uuid=o.uuid
                                            JOIN products p ON p.uuid=oi.product_uuid
                                            WHERE p.shop_uuid=$1 AND o.user_uuid=$2
                                            GROUP BY o.uuid, o.currency, o.status, o.created_at, o.updated_at, o.tracking_number, o.tracking_url, o.shipping_carrier, o.shipped_at, o.delivered_at
                                            ORDER BY o.created_at DESC LIMIT 200`, meta.ShopUUID, *meta.UserUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": orders})
	})

	app.Get("/v1/my/shops/:slug/orders", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var orders []ShopOrder
		if err := opts.DB.Select(&orders, `SELECT o.uuid,
                                                  SUM(oi.price_cents * oi.quantity) AS total_cents,
                                                  o.currency,
                                                  o.status,
                                                  o.created_at,
                                                  o.updated_at,
                                                  o.tracking_number,
                                                  o.tracking_url,
                                                  o.shipping_carrier,
                                                  o.shipped_at,
                                                  o.delivered_at,
                                                  c.uuid AS customer_uuid,
                                                  COALESCE(c.email,'') AS customer_email,
                                                  TRIM(BOTH ' ' FROM COALESCE(c.first_name,'') || ' ' || COALESCE(c.last_name,'')) AS customer_name
                                           FROM orders o
                                           JOIN order_items oi ON oi.order_uuid=o.uuid
                                           JOIN products p ON p.uuid=oi.product_uuid
                                           LEFT JOIN customers c ON c.shop_uuid=p.shop_uuid AND c.user_uuid=o.user_uuid
                                           WHERE p.shop_uuid=$1
                                           GROUP BY o.uuid, o.currency, o.status, o.created_at, o.updated_at, o.tracking_number, o.tracking_url, o.shipping_carrier, o.shipped_at, o.delivered_at, c.uuid, c.email, c.first_name, c.last_name
                                           ORDER BY o.created_at DESC
                                           LIMIT 200`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": orders})
	})

	app.Get("/v1/my/shops/:slug/metrics", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var totals struct {
			Sales  sql.NullInt64 `db:"sales"`
			Orders sql.NullInt64 `db:"orders"`
		}
		if err := opts.DB.Get(&totals, `SELECT COALESCE(SUM(oi.price_cents * oi.quantity),0) AS sales,
                                               COUNT(DISTINCT o.uuid) AS orders
                                        FROM orders o
                                        JOIN order_items oi ON oi.order_uuid=o.uuid
                                        JOIN products p ON p.uuid=oi.product_uuid
                                        WHERE p.shop_uuid=$1`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var customersCount int64
		if err := opts.DB.Get(&customersCount, `SELECT COUNT(1) FROM customers WHERE shop_uuid=$1`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		type dayRow struct {
			Day   time.Time `db:"day"`
			Total int64     `db:"total"`
		}
		var rows []dayRow
		if err := opts.DB.Select(&rows, `SELECT date_trunc('day', o.created_at) AS day,
                                                COALESCE(SUM(oi.price_cents * oi.quantity),0) AS total
                                         FROM orders o
                                         JOIN order_items oi ON oi.order_uuid=o.uuid
                                         JOIN products p ON p.uuid=oi.product_uuid
                                         WHERE p.shop_uuid=$1 AND o.created_at >= now() - interval '14 days'
                                         GROUP BY day
                                         ORDER BY day ASC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		now := time.Now().UTC()
		seriesMap := make(map[string]int64, len(rows))
		for _, r := range rows {
			key := r.Day.UTC().Format("2006-01-02")
			seriesMap[key] = r.Total
		}
		points := make([]SalesPoint, 0, 14)
		for i := 13; i >= 0; i-- {
			day := now.AddDate(0, 0, -i)
			key := day.Format("2006-01-02")
			points = append(points, SalesPoint{
				Date:       key,
				TotalCents: seriesMap[key],
			})
		}
		totalSales := totals.Sales.Int64
		ordersCount := totals.Orders.Int64
		average := int64(0)
		if ordersCount > 0 {
			average = totalSales / ordersCount
		}
		metrics := DashboardMetrics{
			TotalSalesCents:        totalSales,
			OrdersCount:            ordersCount,
			AverageOrderValueCents: average,
			CustomersCount:         customersCount,
			SalesSeries:            points,
		}
		return c.JSON(fiber.Map{"success": true, "data": metrics})
	})

	app.Patch("/v1/orders/:id/tracking", requireAuth, func(c *fiber.Ctx) error {
		orderParam := strings.TrimSpace(c.Params("id"))
		if orderParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		orderID, err := uuid.Parse(orderParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		meta, err := loadOrderMeta(opts.DB, orderID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Status          *string `json:"status"`
			TrackingNumber  *string `json:"trackingNumber"`
			TrackingURL     *string `json:"trackingUrl"`
			ShippingCarrier *string `json:"shippingCarrier"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		args := []any{}
		sets := []string{}
		if body.Status != nil {
			status := strings.ToLower(strings.TrimSpace(*body.Status))
			if status == "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid status"})
			}
			if !isValidOrderStatus(status) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid status"})
			}
			args = append(args, status)
			sets = append(sets, "status=$"+itoa(len(args)))
			switch status {
			case "shipped":
				sets = append(sets, "shipped_at=CASE WHEN shipped_at IS NULL THEN now() ELSE shipped_at END")
			case "delivered":
				sets = append(sets, "shipped_at=CASE WHEN shipped_at IS NULL THEN now() ELSE shipped_at END")
				sets = append(sets, "delivered_at=now()")
			}
		}
		if body.TrackingNumber != nil {
			v := strings.TrimSpace(*body.TrackingNumber)
			if v == "" {
				sets = append(sets, "tracking_number=NULL")
			} else {
				args = append(args, v)
				sets = append(sets, "tracking_number=$"+itoa(len(args)))
			}
		}
		if body.TrackingURL != nil {
			v := strings.TrimSpace(*body.TrackingURL)
			if v == "" {
				sets = append(sets, "tracking_url=NULL")
			} else {
				args = append(args, v)
				sets = append(sets, "tracking_url=$"+itoa(len(args)))
			}
		}
		if body.ShippingCarrier != nil {
			v := strings.TrimSpace(*body.ShippingCarrier)
			if v == "" {
				sets = append(sets, "shipping_carrier=NULL")
			} else {
				args = append(args, v)
				sets = append(sets, "shipping_carrier=$"+itoa(len(args)))
			}
		}
		if len(sets) == 0 {
			return c.JSON(fiber.Map{"success": true})
		}
		sets = append(sets, "updated_at=now()")
		args = append(args, orderID, meta.ShopUUID)
		orderIdx := "$" + itoa(len(args)-1)
		shopIdx := "$" + itoa(len(args))
		query := "UPDATE orders SET " + strings.Join(sets, ", ") + " WHERE uuid=" + orderIdx + " AND EXISTS (SELECT 1 FROM order_items oi JOIN products p ON p.uuid=oi.product_uuid WHERE oi.order_uuid=orders.uuid AND p.shop_uuid=" + shopIdx + ")"
		res, err := opts.DB.Exec(query, args...)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Get("/v1/invitations/:token", func(c *fiber.Ctx) error {
		token := strings.TrimSpace(c.Params("token"))
		var payload struct {
			StoreInvitation
			ShopName string `db:"shop_name" json:"shopName"`
			ShopSlug string `db:"shop_slug" json:"shopSlug"`
		}
		if err := opts.DB.Get(&payload, `SELECT i.uuid,i.store_uuid,i.email,i.role,i.token,i.status,i.invited_by,i.expires_at,i.created_at,i.updated_at,i.accepted_at,
			s.name AS shop_name,s.slug AS shop_slug
			FROM store_invitations i JOIN shops s ON s.uuid=i.store_uuid
			WHERE i.token=$1`, token); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "invitation not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if !strings.EqualFold(payload.Status, "pending") {
			return c.Status(fiber.StatusGone).JSON(fiber.Map{"success": false, "message": "invitation no longer valid"})
		}
		if time.Now().After(payload.ExpiresAt) {
			_, _ = opts.DB.Exec(`UPDATE store_invitations SET status='expired', updated_at=now() WHERE uuid=$1`, payload.UUID)
			return c.Status(fiber.StatusGone).JSON(fiber.Map{"success": false, "message": "invitation expired"})
		}
		return c.JSON(fiber.Map{"success": true, "data": payload})
	})

	app.Post("/v1/invitations/:token/accept", requireAuth, func(c *fiber.Ctx) error {
		token := strings.TrimSpace(c.Params("token"))
		var invite StoreInvitation
		if err := opts.DB.Get(&invite, `SELECT uuid,store_uuid,email,role,token,status,invited_by,expires_at,created_at,updated_at,accepted_at FROM store_invitations WHERE token=$1`, token); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "invitation not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if !strings.EqualFold(invite.Status, "pending") {
			return c.Status(fiber.StatusGone).JSON(fiber.Map{"success": false, "message": "invitation no longer valid"})
		}
		if time.Now().After(invite.ExpiresAt) {
			_, _ = opts.DB.Exec(`UPDATE store_invitations SET status='expired', updated_at=now() WHERE uuid=$1`, invite.UUID)
			return c.Status(fiber.StatusGone).JSON(fiber.Map{"success": false, "message": "invitation expired"})
		}
		userID := srvAuth.UserID(c)
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid user"})
		}
		if _, err := opts.DB.Exec(`INSERT INTO store_users (store_uuid,user_uuid,role,status) VALUES($1,$2,$3,'active')
			ON CONFLICT (store_uuid,user_uuid) DO UPDATE SET role=excluded.role, status='active', updated_at=now()`,
			invite.StoreUUID, userUUID, invite.Role); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if _, err := opts.DB.Exec(`UPDATE store_invitations SET status='accepted', accepted_at=now(), updated_at=now(), token=$2 WHERE uuid=$1`, invite.UUID, token); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// Uploads
	app.Post("/v1/uploads", requireAuth, func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "file required"})
		}
		if file.Size == 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "empty file"})
		}
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext == "" {
			ext = ".jpg"
		}
		name := uuid.New().String() + ext
		dest := filepath.Join("/data/uploads/products", name)
		if err := c.SaveFile(file, dest); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "save failed"})
		}
		pub := strings.TrimRight(opts.UploadsPublicBase, "/") + "/products/" + name
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"url": pub}})
	})

	// Order payment stub
	app.Post("/v1/orders/:id/pay", requireAuth, func(c *fiber.Ctx) error {
		id := c.Params("id")
		res, err := opts.DB.Exec(`UPDATE orders SET status='paid', updated_at=now() WHERE uuid=$1 AND status='pending'`, id)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "not payable"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// Dev seed
	app.Post("/v1/dev/seed", requireAuth, func(c *fiber.Ctx) error {
		if !hasDashboardAccess(c, opts.DB) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		owner := srvAuth.UserID(c)
		var count int
		_ = opts.DB.Get(&count, `SELECT COUNT(1) FROM shops WHERE slug='demo-shop' AND owner_uuid=$1`, owner)
		var shopID uuid.UUID
		if count == 0 {
			shopID = uuid.New()
			if _, err := opts.DB.Exec(`INSERT INTO shops(uuid,name,slug,owner_uuid,public,description) VALUES($1,'Demo Shop','demo-shop',$2,true,'Sample demo shop')`, shopID, owner); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false})
			}
		} else {
			_ = opts.DB.Get(&shopID, `SELECT uuid FROM shops WHERE slug='demo-shop' AND owner_uuid=$1 LIMIT 1`, owner)
		}
		_ = opts.DB.Get(&count, `SELECT COUNT(1) FROM products WHERE shop_uuid=$1 AND deleted_at IS NULL`, shopID)
		if count == 0 {
			for i := 1; i <= 3; i++ {
				pid := uuid.New()
				_, _ = opts.DB.Exec(`INSERT INTO products(uuid,shop_uuid,title,slug,summary,price_cents,currency,stock,published,category,category_uuid) VALUES($1,$2,$3,$4,$5,$6,'USD',10,true,'general',NULL)`, pid, shopID, fmt.Sprintf("Product %d", i), fmt.Sprintf("product-%d", i), "Demo product", int64(i*1000))
				if err := upsertDefaultInventory(opts.DB, shopID, pid, 10); err != nil {
					log.Printf("inventory seed error: %v", err)
				}
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// Cart minimal endpoints
	app.Get("/v1/cart", requireAuth, func(c *fiber.Ctx) error { return getCart(c, opts.DB) })
	app.Put("/v1/cart", requireAuth, func(c *fiber.Ctx) error { return replaceCart(c, opts.DB) })
	app.Post("/v1/cart/items", requireAuth, func(c *fiber.Ctx) error { return addCartItem(c, opts.DB) })
	app.Put("/v1/cart/items/:id", requireAuth, func(c *fiber.Ctx) error { return updateCartItem(c, opts.DB) })
	app.Delete("/v1/cart/items/:id", requireAuth, func(c *fiber.Ctx) error { return deleteCartItem(c, opts.DB) })
	app.Delete("/v1/cart", requireAuth, func(c *fiber.Ctx) error { return clearCart(c, opts.DB) })

	// Wishlist endpoints
	app.Get("/v1/wishlist", requireAuth, func(c *fiber.Ctx) error { return getWishlist(c, opts.DB) })
	app.Put("/v1/wishlist", requireAuth, func(c *fiber.Ctx) error { return replaceWishlist(c, opts.DB) })

	// Orders minimal endpoints
	app.Post("/v1/orders", requireAuth, func(c *fiber.Ctx) error { return createOrderFromCart(c, opts.DB) })
	app.Get("/v1/orders", requireAuth, func(c *fiber.Ctx) error { return listOrders(c, opts.DB) })
	app.Get("/v1/orders/:id", requireAuth, func(c *fiber.Ctx) error { return getOrder(c, opts.DB) })

	// Unified search (owner-scoped). Returns sections for products, orders, and placeholders for others.
	app.Get("/v1/search", requireAuth, func(c *fiber.Ctx) error {
		q := strings.TrimSpace(c.Query("q"))
		if len(q) < 2 {
			return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
				"products": []any{}, "orders": []any{}, "customers": []any{}, "marketing": []any{}, "discounts": []any{}, "content": []any{}, "markets": []any{}, "analytics": []any{},
			}})
		}
		owner := srvAuth.UserID(c)
		like := "%" + q + "%"
		// Products owned by this user via their shops
		var prods []Product
		if err := opts.DB.Select(&prods, `SELECT p.uuid, p.shop_uuid, p.title, p.slug, p.summary, p.price_cents, p.currency, p.stock, p.image_url,
                                                 p.category, p.category_uuid, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
                                                 p.created_at, p.updated_at, p.deleted_at, p.published,
                                                 s.name AS shop_name, s.slug AS shop_slug
                                          FROM products p JOIN shops s ON s.uuid=p.shop_uuid
                                          WHERE s.owner_uuid=$1 AND p.deleted_at IS NULL
                                            AND (LOWER(p.title) LIKE LOWER($2) OR LOWER(p.summary) LIKE LOWER($2))
                                          ORDER BY p.updated_at DESC LIMIT 10`, owner, like); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		// Orders for products owned by this user (join via order_items -> products -> shops)
		type OrderRow struct {
			UUID            uuid.UUID `db:"uuid" json:"uuid"`
			TotalCents      int64     `db:"total_cents" json:"totalCents"`
			Currency        string    `db:"currency" json:"currency"`
			Status          string    `db:"status" json:"status"`
			CreatedAt       time.Time `db:"created_at" json:"createdAt"`
			UpdatedAt       time.Time `db:"updated_at" json:"updatedAt"`
			TrackingNumber  *string   `db:"tracking_number" json:"trackingNumber,omitempty"`
			ShippingCarrier *string   `db:"shipping_carrier" json:"shippingCarrier,omitempty"`
		}
		var orders []OrderRow
		if err := opts.DB.Select(&orders, `SELECT DISTINCT o.uuid, o.total_cents, o.currency, o.status, o.created_at, o.updated_at, o.tracking_number, o.shipping_carrier
                                           FROM orders o
                                           JOIN order_items oi ON oi.order_uuid=o.uuid
                                           JOIN products p ON p.uuid=oi.product_uuid AND p.deleted_at IS NULL
                                           JOIN shops s ON s.uuid=p.shop_uuid
                                           WHERE s.owner_uuid=$1 AND (
                                                CAST(o.uuid AS TEXT) LIKE $2 OR LOWER(o.status) LIKE LOWER($2)
                                           )
                                           ORDER BY o.created_at DESC LIMIT 10`, owner, like); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		out := fiber.Map{
			"products":  prods,
			"orders":    orders,
			"customers": []any{},
			"marketing": []any{},
			"discounts": []any{},
			"content":   []any{},
			"markets":   []any{},
			"analytics": []any{},
		}
		return c.JSON(fiber.Map{"success": true, "data": out})
	})

	app.Get("/v1/me/profile", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		profile, err := ensureUserProfile(opts.DB, user)
		if err != nil {
			return respondWithError(c, err)
		}
		addresses, err := listUserAddresses(opts.DB, user)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"profile":   profile,
			"addresses": addresses,
		}})
	})

	app.Put("/v1/me/profile", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		if _, err := ensureUserProfile(opts.DB, user); err != nil {
			return respondWithError(c, err)
		}
		var body struct {
			DisplayName    *string `json:"displayName"`
			Email          *string `json:"email"`
			Phone          *string `json:"phone"`
			AvatarURL      *string `json:"avatarUrl"`
			Timezone       *string `json:"timezone"`
			MarketingOptIn *bool   `json:"marketingOptIn"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		args := []any{}
		sets := []string{}
		add := func(field string, value any) {
			args = append(args, value)
			sets = append(sets, field+"=$"+itoa(len(args)))
		}
		if body.DisplayName != nil {
			add("display_name", strings.TrimSpace(*body.DisplayName))
		}
		if body.Email != nil {
			email := strings.TrimSpace(*body.Email)
			if email == "" {
				add("email", "")
			} else {
				normalized, err := normalizeEmail(email)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid email"})
				}
				add("email", normalized)
			}
		}
		if body.Phone != nil {
			add("phone", strings.TrimSpace(*body.Phone))
		}
		if body.AvatarURL != nil {
			url := strings.TrimSpace(*body.AvatarURL)
			if url == "" {
				sets = append(sets, "avatar_url=NULL")
			} else {
				add("avatar_url", url)
			}
		}
		if body.Timezone != nil {
			add("timezone", strings.TrimSpace(*body.Timezone))
		}
		if body.MarketingOptIn != nil {
			add("marketing_opt_in", *body.MarketingOptIn)
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, user)
			query := "UPDATE user_profiles SET " + strings.Join(sets, ", ") + " WHERE user_uuid=$" + itoa(len(args))
			if _, err := opts.DB.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		profile, err := ensureUserProfile(opts.DB, user)
		if err != nil {
			return respondWithError(c, err)
		}
		addresses, err := listUserAddresses(opts.DB, user)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"profile":   profile,
			"addresses": addresses,
		}})
	})

	app.Get("/v1/me/addresses", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		if _, err := ensureUserProfile(opts.DB, user); err != nil {
			return respondWithError(c, err)
		}
		addresses, err := listUserAddresses(opts.DB, user)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": addresses})
	})

	app.Post("/v1/me/addresses", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		if _, err := ensureUserProfile(opts.DB, user); err != nil {
			return respondWithError(c, err)
		}
		var body struct {
			Label           string `json:"label"`
			RecipientName   string `json:"recipientName"`
			Line1           string `json:"line1"`
			Line2           string `json:"line2"`
			City            string `json:"city"`
			Region          string `json:"region"`
			PostalCode      string `json:"postalCode"`
			Country         string `json:"country"`
			Phone           string `json:"phone"`
			DefaultShipping bool   `json:"defaultShipping"`
			DefaultBilling  bool   `json:"defaultBilling"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		label := strings.TrimSpace(body.Label)
		if label == "" {
			label = "Primary"
		}
		line1 := strings.TrimSpace(body.Line1)
		city := strings.TrimSpace(body.City)
		region := strings.TrimSpace(body.Region)
		postal := strings.TrimSpace(body.PostalCode)
		country := strings.ToUpper(strings.TrimSpace(body.Country))
		if line1 == "" || city == "" || region == "" || postal == "" || country == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "missing required address fields"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()
		if err := clearDefaultFlagsTx(tx, user, body.DefaultShipping, body.DefaultBilling, uuid.Nil); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		addressID := uuid.New()
		if _, err := tx.Exec(`INSERT INTO user_addresses (uuid,user_uuid,label,recipient_name,line1,line2,city,region,postal_code,country,phone,is_default_shipping,is_default_billing)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			addressID, user, label, strings.TrimSpace(body.RecipientName), line1, strings.TrimSpace(body.Line2), city, region, postal, country, strings.TrimSpace(body.Phone),
			body.DefaultShipping, body.DefaultBilling); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		address, err := loadUserAddress(opts.DB, user, addressID)
		if err != nil {
			return respondWithError(c, err)
		}
		return c.JSON(fiber.Map{"success": true, "data": address})
	})

	app.Patch("/v1/me/addresses/:id", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		if _, err := ensureUserProfile(opts.DB, user); err != nil {
			return respondWithError(c, err)
		}
		idParam := strings.TrimSpace(c.Params("id"))
		addressID, err := uuid.Parse(idParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid address id"})
		}
		if _, err := loadUserAddress(opts.DB, user, addressID); err != nil {
			return respondWithError(c, err)
		}
		var body struct {
			Label           *string `json:"label"`
			RecipientName   *string `json:"recipientName"`
			Line1           *string `json:"line1"`
			Line2           *string `json:"line2"`
			City            *string `json:"city"`
			Region          *string `json:"region"`
			PostalCode      *string `json:"postalCode"`
			Country         *string `json:"country"`
			Phone           *string `json:"phone"`
			DefaultShipping *bool   `json:"defaultShipping"`
			DefaultBilling  *bool   `json:"defaultBilling"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()
		args := []any{}
		sets := []string{}
		add := func(field string, value any) {
			args = append(args, value)
			sets = append(sets, field+"=$"+itoa(len(args)))
		}
		if body.Label != nil {
			add("label", strings.TrimSpace(*body.Label))
		}
		if body.RecipientName != nil {
			add("recipient_name", strings.TrimSpace(*body.RecipientName))
		}
		if body.Line1 != nil {
			add("line1", strings.TrimSpace(*body.Line1))
		}
		if body.Line2 != nil {
			add("line2", strings.TrimSpace(*body.Line2))
		}
		if body.City != nil {
			add("city", strings.TrimSpace(*body.City))
		}
		if body.Region != nil {
			add("region", strings.TrimSpace(*body.Region))
		}
		if body.PostalCode != nil {
			add("postal_code", strings.TrimSpace(*body.PostalCode))
		}
		if body.Country != nil {
			add("country", strings.ToUpper(strings.TrimSpace(*body.Country)))
		}
		if body.Phone != nil {
			add("phone", strings.TrimSpace(*body.Phone))
		}
		if body.DefaultShipping != nil {
			if *body.DefaultShipping {
				if err := clearDefaultFlagsTx(tx, user, true, false, addressID); err != nil {
					return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
				}
				sets = append(sets, "is_default_shipping=true")
			} else {
				sets = append(sets, "is_default_shipping=false")
			}
		}
		if body.DefaultBilling != nil {
			if *body.DefaultBilling {
				if err := clearDefaultFlagsTx(tx, user, false, true, addressID); err != nil {
					return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
				}
				sets = append(sets, "is_default_billing=true")
			} else {
				sets = append(sets, "is_default_billing=false")
			}
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, user, addressID)
			query := "UPDATE user_addresses SET " + strings.Join(sets, ", ") + " WHERE user_uuid=$" + itoa(len(args)-1) + " AND uuid=$" + itoa(len(args))
			if _, err := tx.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		address, err := loadUserAddress(opts.DB, user, addressID)
		if err != nil {
			return respondWithError(c, err)
		}
		return c.JSON(fiber.Map{"success": true, "data": address})
	})

	app.Delete("/v1/me/addresses/:id", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		if _, err := ensureUserProfile(opts.DB, user); err != nil {
			return respondWithError(c, err)
		}
		idParam := strings.TrimSpace(c.Params("id"))
		addressID, err := uuid.Parse(idParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid address id"})
		}
		res, err := opts.DB.Exec(`DELETE FROM user_addresses WHERE user_uuid=$1 AND uuid=$2`, user, addressID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "address not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	return app
}

// helpers
func pqStringArray(v []string) interface{} { return pq.Array(v) }

// Wishlist models
type WishlistItem struct {
	UUID        uuid.UUID `db:"uuid" json:"uuid"`
	ProductUUID uuid.UUID `db:"product_uuid" json:"productUuid"`
	Title       string    `db:"title" json:"title"`
	PriceCents  int64     `db:"price_cents" json:"priceCents"`
	Currency    string    `db:"currency" json:"currency"`
	ImageURL    *string   `db:"image_url" json:"imageUrl,omitempty"`
	ShopName    string    `db:"shop_name" json:"shopName"`
	ShopSlug    string    `db:"shop_slug" json:"shopSlug"`
	AddedAt     time.Time `db:"added_at" json:"addedAt"`
}

// Cart models
type CartItem struct {
	UUID        uuid.UUID `db:"uuid" json:"uuid"`
	ProductUUID uuid.UUID `db:"product_uuid" json:"productUuid"`
	Quantity    int       `db:"quantity" json:"quantity"`
	Title       string    `db:"title" json:"title"`
	PriceCents  int64     `db:"price_cents" json:"priceCents"`
	Currency    string    `db:"currency" json:"currency"`
	ImageURL    *string   `db:"image_url" json:"imageUrl,omitempty"`
}

type Order struct {
	UUID            uuid.UUID  `db:"uuid" json:"uuid"`
	TotalCents      int64      `db:"total_cents" json:"totalCents"`
	Currency        string     `db:"currency" json:"currency"`
	Status          string     `db:"status" json:"status"`
	CreatedAt       time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updatedAt"`
	TrackingNumber  *string    `db:"tracking_number" json:"trackingNumber,omitempty"`
	TrackingURL     *string    `db:"tracking_url" json:"trackingUrl,omitempty"`
	ShippingCarrier *string    `db:"shipping_carrier" json:"shippingCarrier,omitempty"`
	ShippedAt       *time.Time `db:"shipped_at" json:"shippedAt,omitempty"`
	DeliveredAt     *time.Time `db:"delivered_at" json:"deliveredAt,omitempty"`
}

var allowedTeamRoles = map[string]bool{
	"manager": true,
	"staff":   true,
}

func hasDashboardAccess(c *fiber.Ctx, db *sqlx.DB) bool {
	if srvAuth.HasAnyAppRole(c, srvAuth.RoleOwner, srvAuth.RoleManager, srvAuth.RoleStaff) ||
		srvAuth.HasPlatformRole(c, srvAuth.PlatformRoleAdmin) ||
		srvAuth.HasPlatformRole(c, srvAuth.PlatformRoleSupport) {
		return true
	}
	user := srvAuth.UserID(c)
	if user == "" {
		return false
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(1) FROM shops WHERE owner_uuid=$1`, user); err == nil && count > 0 {
		return true
	}
	if err := db.Get(&count, `SELECT COUNT(1) FROM store_users WHERE user_uuid=$1 AND status='active'`, user); err == nil && count > 0 {
		return true
	}
	return false
}

func normalizeTeamRole(role string) (string, error) {
	r := strings.ToLower(strings.TrimSpace(role))
	if allowedTeamRoles[r] {
		return r, nil
	}
	return "", fmt.Errorf("invalid role")
}

func membershipRole(db *sqlx.DB, store uuid.UUID, user string) string {
	var row struct {
		Role   string `db:"role"`
		Status string `db:"status"`
	}
	if err := db.Get(&row, `SELECT role, status FROM store_users WHERE store_uuid=$1 AND user_uuid=$2 LIMIT 1`, store, user); err == nil {
		if strings.EqualFold(row.Status, "active") {
			return strings.ToLower(strings.TrimSpace(row.Role))
		}
	}
	return ""
}

func teamRoleAllowsManagement(role string) bool {
	switch strings.ToLower(role) {
	case "owner", "manager", "platform":
		return true
	default:
		return false
	}
}

func teamRoleAllowsInvites(role string) bool {
	switch strings.ToLower(role) {
	case "owner", "platform":
		return true
	default:
		return false
	}
}

func teamRoleAllowsTransfer(role string) bool {
	switch strings.ToLower(role) {
	case "owner", "platform":
		return true
	default:
		return false
	}
}

func teamRoleAllowsView(role string) bool {
	switch strings.ToLower(role) {
	case "owner", "manager", "staff", "platform":
		return true
	default:
		return false
	}
}

func ensureShopAccess(c *fiber.Ctx, db *sqlx.DB, slug string) (Shop, string, error) {
	slug = strings.TrimSpace(slug)
	var shop Shop
	if err := db.Get(&shop, `SELECT uuid, name, slug, owner_uuid, public, description, created_at, updated_at FROM shops WHERE slug=$1`, slug); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Shop{}, "", fiber.NewError(fiber.StatusNotFound, "shop not found")
		}
		return Shop{}, "", fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	user := srvAuth.UserID(c)
	if user == "" {
		return Shop{}, "", fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if shop.OwnerUUID.String() == user {
		return shop, "owner", nil
	}
	if role := membershipRole(db, shop.UUID, user); role != "" {
		return shop, role, nil
	}
	if srvAuth.HasPlatformRole(c, srvAuth.PlatformRoleAdmin) || srvAuth.HasPlatformRole(c, srvAuth.PlatformRoleSupport) {
		return shop, "platform", nil
	}
	return Shop{}, "", fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
}

func ensureCart(db *sqlx.DB, user string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.Get(&id, `SELECT uuid FROM carts WHERE user_uuid=$1`, user)
	if err == sql.ErrNoRows {
		id = uuid.New()
		if _, e := db.Exec(`INSERT INTO carts(uuid,user_uuid) VALUES($1,$2)`, id, user); e != nil {
			return uuid.Nil, e
		}
		return id, nil
	}
	return id, err
}

func respondWithError(c *fiber.Ctx, err error) error {
	if fe, ok := err.(*fiber.Error); ok {
		msg := strings.TrimSpace(fe.Message)
		if msg == "" {
			msg = http.StatusText(fe.Code)
		}
		return c.Status(fe.Code).JSON(fiber.Map{"success": false, "message": msg})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "server error"})
}

func ensureWishlist(db *sqlx.DB, user string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.Get(&id, `SELECT uuid FROM wishlists WHERE user_uuid=$1`, user)
	if err == sql.ErrNoRows {
		id = uuid.New()
		if _, e := db.Exec(`INSERT INTO wishlists(uuid,user_uuid) VALUES($1,$2)`, id, user); e != nil {
			return uuid.Nil, e
		}
		return id, nil
	}
	return id, err
}

func getCart(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	cartID, err := ensureCart(db, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	var items []CartItem
	q := `SELECT ci.uuid, ci.product_uuid, ci.quantity, p.title, p.price_cents, p.currency, p.image_url
          FROM cart_items ci JOIN products p ON p.uuid=ci.product_uuid AND p.deleted_at IS NULL
          WHERE ci.cart_uuid=$1 ORDER BY ci.added_at DESC`
	if err := db.Select(&items, q, cartID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	var total int64
	currency := "USD"
	for _, it := range items {
		total += int64(it.Quantity) * it.PriceCents
		currency = it.Currency
	}
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"items": items, "totalCents": total, "currency": currency, "cartId": cartID}})
}

func replaceCart(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	var body struct {
		Items []struct {
			ProductID string `json:"productId"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
	}
	cartID, err := ensureCart(db, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	tx, err := db.Beginx()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM cart_items WHERE cart_uuid=$1`, cartID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	count := 0
	for _, it := range body.Items {
		pid := strings.TrimSpace(it.ProductID)
		if pid == "" || it.Quantity <= 0 {
			continue
		}
		var exists int
		if err := tx.Get(&exists, `SELECT COUNT(1) FROM products WHERE uuid=$1 AND deleted_at IS NULL`, pid); err != nil || exists == 0 {
			continue
		}
		qty := it.Quantity
		if qty > 1000 {
			qty = 1000
		}
		if _, err := tx.Exec(`INSERT INTO cart_items(uuid,cart_uuid,product_uuid,quantity) VALUES($1,$2,$3,$4)`, uuid.New(), cartID, pid, qty); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		count++
	}
	if _, err := tx.Exec(`UPDATE carts SET updated_at=now() WHERE uuid=$1`, cartID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	if err := tx.Commit(); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return getCart(c, db)
}

func getWishlist(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	wishlistID, err := ensureWishlist(db, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	var items []WishlistItem
	q := `SELECT wi.uuid, wi.product_uuid, wi.added_at,
	             p.title, p.price_cents, p.currency, p.image_url,
	             s.name AS shop_name, s.slug AS shop_slug
	      FROM wishlist_items wi
	      JOIN products p ON p.uuid=wi.product_uuid AND p.deleted_at IS NULL
	      JOIN shops s ON s.uuid=p.shop_uuid
	      WHERE wi.wishlist_uuid=$1
	      ORDER BY wi.added_at DESC`
	if err := db.Select(&items, q, wishlistID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"items": items, "wishlistId": wishlistID}})
}

func replaceWishlist(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	var body struct {
		Items []struct {
			ProductID string `json:"productId"`
		} `json:"items"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
	}
	wishlistID, err := ensureWishlist(db, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	tx, err := db.Beginx()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM wishlist_items WHERE wishlist_uuid=$1`, wishlistID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	seen := make(map[string]struct{})
	for _, it := range body.Items {
		pid := strings.TrimSpace(it.ProductID)
		if pid == "" {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		var exists int
		if err := tx.Get(&exists, `SELECT COUNT(1) FROM products WHERE uuid=$1 AND deleted_at IS NULL`, pid); err != nil || exists == 0 {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO wishlist_items(uuid,wishlist_uuid,product_uuid) VALUES($1,$2,$3)`, uuid.New(), wishlistID, pid); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
	}
	if _, err := tx.Exec(`UPDATE wishlists SET updated_at=now() WHERE uuid=$1`, wishlistID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	if err := tx.Commit(); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return getWishlist(c, db)
}

func addCartItem(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	var body struct {
		ProductID string `json:"productId"`
		Quantity  int    `json:"quantity"`
	}
	if err := c.BodyParser(&body); err != nil || body.ProductID == "" || body.Quantity <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
	}
	pid := strings.TrimSpace(body.ProductID)
	var exists int
	if err := db.Get(&exists, `SELECT COUNT(1) FROM products WHERE uuid=$1 AND deleted_at IS NULL`, pid); err != nil || exists == 0 {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "product unavailable"})
	}
	cartID, err := ensureCart(db, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	// upsert
	_, err = db.Exec(`INSERT INTO cart_items(cart_uuid,product_uuid,quantity) VALUES($1,$2,$3)
                      ON CONFLICT (cart_uuid, product_uuid) DO UPDATE SET quantity=cart_items.quantity+EXCLUDED.quantity`, cartID, pid, body.Quantity)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return getCart(c, db)
}

func updateCartItem(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	_, _ = ensureCart(db, user) // ensure exists
	id := strings.TrimSpace(c.Params("id"))
	var body struct {
		Quantity int `json:"quantity"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
	}
	if body.Quantity <= 0 {
		if _, err := db.Exec(`DELETE FROM cart_items WHERE uuid=$1`, id); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true})
	}
	if _, err := db.Exec(`UPDATE cart_items SET quantity=$1 WHERE uuid=$2`, body.Quantity, id); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return c.JSON(fiber.Map{"success": true})
}

func deleteCartItem(c *fiber.Ctx, db *sqlx.DB) error {
	id := strings.TrimSpace(c.Params("id"))
	if _, err := db.Exec(`DELETE FROM cart_items WHERE uuid=$1`, id); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return c.JSON(fiber.Map{"success": true})
}

func clearCart(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	cartID, err := ensureCart(db, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	if _, err := db.Exec(`DELETE FROM cart_items WHERE cart_uuid=$1`, cartID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return c.JSON(fiber.Map{"success": true})
}

func createOrderFromCart(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	cartID, err := ensureCart(db, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	// load cart
	type row struct {
		ProductUUID uuid.UUID `db:"product_uuid"`
		Quantity    int       `db:"quantity"`
		PriceCents  int64     `db:"price_cents"`
		Currency    string    `db:"currency"`
	}
	var rows []row
	if err := db.Select(&rows, `SELECT ci.product_uuid, ci.quantity, p.price_cents, p.currency
                                FROM cart_items ci
                                JOIN products p ON p.uuid=ci.product_uuid AND p.deleted_at IS NULL
                                WHERE ci.cart_uuid=$1`, cartID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	if len(rows) == 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "cart empty"})
	}
	var total int64
	currency := rows[0].Currency
	for _, r := range rows {
		total += int64(r.Quantity) * r.PriceCents
	}
	orderID := uuid.New()
	tx, err := db.Beginx()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO orders(uuid,user_uuid,total_cents,currency,status) VALUES($1,$2,$3,$4,'pending')`, orderID, user, total, currency); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	for _, r := range rows {
		if _, err := tx.Exec(`INSERT INTO order_items(order_uuid,product_uuid,quantity,price_cents) VALUES($1,$2,$3,$4)`, orderID, r.ProductUUID, r.Quantity, r.PriceCents); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
	}
	if _, err := tx.Exec(`DELETE FROM cart_items WHERE cart_uuid=$1`, cartID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	if err := tx.Commit(); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": orderID, "totalCents": total, "currency": currency}})
}

func listOrders(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	var out []Order
	if err := db.Select(&out, `SELECT uuid,total_cents,currency,status,created_at,updated_at,tracking_number,tracking_url,shipping_carrier,shipped_at,delivered_at
                               FROM orders
                               WHERE user_uuid=$1
                               ORDER BY created_at DESC`, user); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return c.JSON(fiber.Map{"success": true, "data": out})
}

func getOrder(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	id := c.Params("id")
	var o Order
	if err := db.Get(&o, `SELECT uuid,total_cents,currency,status,created_at,updated_at,tracking_number,tracking_url,shipping_carrier,shipped_at,delivered_at
                           FROM orders
                           WHERE uuid=$1 AND user_uuid=$2`, id, user); err != nil {
		return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
	}
	return c.JSON(fiber.Map{"success": true, "data": o})
}

// helpers
func itoa(i int) string { return strconv.Itoa(i) }

// parseMoneyToCents converts strings like "12", "12.3", "12.34", or "1234c" to cents.
func parseMoneyToCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	// explicit cents (e.g. "1234c")
	if strings.HasSuffix(s, "c") || strings.HasSuffix(s, "C") {
		v, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimSuffix(s, "c"), "C"), 10, 64)
		if err != nil {
			return 0, err
		}
		return v, nil
	}
	if strings.Contains(s, ".") {
		parts := strings.SplitN(s, ".", 2)
		intPart := parts[0]
		frac := parts[1]
		for len(frac) < 2 {
			frac += "0"
		}
		if len(frac) > 2 {
			frac = frac[:2]
		}
		iv, err := strconv.ParseInt(intPart, 10, 64)
		if err != nil {
			return 0, err
		}
		fv, err := strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, err
		}
		return iv*100 + fv, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return v * 100, nil
}

func normalizeEmail(value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", errors.New("email required")
	}
	parsed, err := mail.ParseAddress(v)
	if err != nil {
		return "", err
	}
	email := strings.TrimSpace(parsed.Address)
	if email == "" || !strings.Contains(email, "@") {
		return "", errors.New("invalid email")
	}
	return strings.ToLower(email), nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, raw := range tags {
		tag := strings.ToLower(strings.TrimSpace(raw))
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
		if len(normalized) >= 20 {
			break
		}
	}
	sort.Strings(normalized)
	return normalized
}

func parseOptionalUUID(value string) (*uuid.UUID, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(v)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func upsertDefaultInventory(db *sqlx.DB, shop uuid.UUID, product uuid.UUID, quantity int64) error {
	if quantity < 0 {
		quantity = 0
	}
	_, err := db.Exec(`INSERT INTO inventory_levels (shop_uuid, product_uuid, location_uuid, quantity, reserved, safety_stock)
                       VALUES ($1,$2,NULL,$3,0,0)
                       ON CONFLICT ON CONSTRAINT inventory_levels_product_default_idx
                       DO UPDATE SET quantity=EXCLUDED.quantity, updated_at=now()`,
		shop, product, quantity)
	return err
}

func syncProductStockFromInventory(db *sqlx.DB, product uuid.UUID) error {
	var stock int64
	if err := db.Get(&stock, `SELECT COALESCE(SUM(quantity - reserved),0) FROM inventory_levels WHERE product_uuid=$1`, product); err != nil {
		return err
	}
	if stock < 0 {
		stock = 0
	}
	_, err := db.Exec(`UPDATE products SET stock=$1, updated_at=now() WHERE uuid=$2`, stock, product)
	return err
}

func nullIfEmptyJSON(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	return raw
}

func parseUUIDList(values []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(values))
	seen := make(map[uuid.UUID]struct{})
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		id, err := uuid.Parse(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid uuid: %s", trimmed)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func setCollectionProducts(db *sqlx.DB, collection uuid.UUID, shop uuid.UUID, productIDs []uuid.UUID) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM collection_products WHERE collection_uuid=$1`, collection); err != nil {
		return err
	}
	if len(productIDs) > 0 {
		var count int
		if err := tx.Get(&count, `SELECT COUNT(1) FROM products WHERE uuid = ANY($1) AND shop_uuid=$2 AND deleted_at IS NULL`, pq.Array(productIDs), shop); err != nil {
			return err
		}
		if count != len(productIDs) {
			return fmt.Errorf("one or more products do not belong to this shop")
		}
		for index, pid := range productIDs {
			if _, err := tx.Exec(`INSERT INTO collection_products(collection_uuid, product_uuid, position) VALUES($1,$2,$3)`, collection, pid, index); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func setDiscountProducts(db *sqlx.DB, discount uuid.UUID, shop uuid.UUID, productIDs []uuid.UUID) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM discount_products WHERE discount_uuid=$1`, discount); err != nil {
		return err
	}
	if len(productIDs) > 0 {
		var count int
		if err := tx.Get(&count, `SELECT COUNT(1) FROM products WHERE uuid = ANY($1) AND shop_uuid=$2 AND deleted_at IS NULL`, pq.Array(productIDs), shop); err != nil {
			return err
		}
		if count != len(productIDs) {
			return fmt.Errorf("one or more products do not belong to this shop")
		}
		for _, pid := range productIDs {
			if _, err := tx.Exec(`INSERT INTO discount_products(discount_uuid, product_uuid) VALUES($1,$2)`, discount, pid); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func normalizeDiscountStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return "active"
	case "scheduled":
		return "scheduled"
	case "expired":
		return "expired"
	case "archived":
		return "archived"
	default:
		return "draft"
	}
}

func normalizeGiftCardStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		return "active"
	case "disabled":
		return "disabled"
	case "redeemed":
		return "redeemed"
	default:
		return "draft"
	}
}

func addGiftCardTransaction(db *sqlx.DB, card uuid.UUID, change int64, reason string) error {
	if reason == "" {
		reason = "adjustment"
	}
	_, err := db.Exec(`INSERT INTO gift_card_transactions(uuid, gift_card_uuid, change_cents, reason) VALUES($1,$2,$3,$4)`,
		uuid.New(), card, change, reason)
	return err
}

type customerMeta struct {
	CustomerUUID uuid.UUID  `db:"uuid"`
	ShopUUID     uuid.UUID  `db:"shop_uuid"`
	ShopSlug     string     `db:"slug"`
	UserUUID     *uuid.UUID `db:"user_uuid"`
}

func loadCustomerMeta(db *sqlx.DB, id uuid.UUID) (customerMeta, error) {
	var meta customerMeta
	if err := db.Get(&meta, `SELECT c.uuid, c.shop_uuid, s.slug, c.user_uuid
                              FROM customers c
                              JOIN shops s ON s.uuid=c.shop_uuid
                              WHERE c.uuid=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return customerMeta{}, fiber.NewError(fiber.StatusNotFound, "customer not found")
		}
		return customerMeta{}, fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return meta, nil
}

func fetchCustomerSummary(db *sqlx.DB, id uuid.UUID) (CustomerSummary, error) {
	var summary CustomerSummary
	err := db.Get(&summary, `SELECT c.uuid, c.shop_uuid, c.user_uuid, c.email, c.first_name, c.last_name, c.phone, c.tags, c.notes, c.marketing_opt_in, c.created_at, c.updated_at,
                                   COALESCE(SUM(oi.price_cents * oi.quantity),0) AS total_spent_cents,
                                   COUNT(DISTINCT CASE WHEN o.uuid IS NOT NULL AND p.uuid IS NOT NULL THEN o.uuid END) AS orders_count,
                                   MAX(o.created_at) AS last_order_at,
                                   TRIM(BOTH ' ' FROM COALESCE(c.first_name,'') || ' ' || COALESCE(c.last_name,'')) AS customer_name
                            FROM customers c
                            LEFT JOIN orders o ON c.user_uuid IS NOT NULL AND o.user_uuid=c.user_uuid
                            LEFT JOIN order_items oi ON oi.order_uuid=o.uuid
                            LEFT JOIN products p ON p.uuid=oi.product_uuid AND p.shop_uuid=c.shop_uuid
                            WHERE c.uuid=$1
                            GROUP BY c.uuid`, id)
	return summary, err
}

type orderMeta struct {
	OrderUUID uuid.UUID `db:"uuid"`
	ShopUUID  uuid.UUID `db:"shop_uuid"`
	ShopSlug  string    `db:"slug"`
}

func loadOrderMeta(db *sqlx.DB, id uuid.UUID) (orderMeta, error) {
	var meta orderMeta
	if err := db.Get(&meta, `SELECT o.uuid, s.uuid AS shop_uuid, s.slug
                               FROM orders o
                               JOIN order_items oi ON oi.order_uuid=o.uuid
                               JOIN products p ON p.uuid=oi.product_uuid
                               JOIN shops s ON s.uuid=p.shop_uuid
                               WHERE o.uuid=$1
                               LIMIT 1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return orderMeta{}, fiber.NewError(fiber.StatusNotFound, "order not found")
		}
		return orderMeta{}, fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return meta, nil
}

func isValidOrderStatus(status string) bool {
	switch strings.ToLower(status) {
	case "pending", "processing", "shipped", "delivered", "cancelled":
		return true
	default:
		return false
	}
}

func ensureUserProfile(db *sqlx.DB, user string) (UserProfile, error) {
	user = strings.TrimSpace(user)
	if user == "" {
		return UserProfile{}, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	if _, err := uuid.Parse(user); err != nil {
		return UserProfile{}, fiber.NewError(fiber.StatusBadRequest, "invalid user")
	}
	if _, err := db.Exec(`INSERT INTO user_profiles (user_uuid) VALUES ($1) ON CONFLICT (user_uuid) DO NOTHING`, user); err != nil {
		return UserProfile{}, fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	var profile UserProfile
	if err := db.Get(&profile, `SELECT user_uuid, display_name, email, phone, avatar_url, timezone, marketing_opt_in, created_at, updated_at FROM user_profiles WHERE user_uuid=$1`, user); err != nil {
		return UserProfile{}, fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return profile, nil
}

func listUserAddresses(db *sqlx.DB, user string) ([]UserAddress, error) {
	addresses := []UserAddress{}
	if err := db.Select(&addresses, `SELECT uuid,user_uuid,label,recipient_name,line1,line2,city,region,postal_code,country,phone,is_default_shipping,is_default_billing,created_at,updated_at
                                     FROM user_addresses
                                     WHERE user_uuid=$1
                                     ORDER BY is_default_shipping DESC, is_default_billing DESC, created_at ASC`, user); err != nil {
		return nil, err
	}
	return addresses, nil
}

func loadUserAddress(db *sqlx.DB, user string, id uuid.UUID) (UserAddress, error) {
	var address UserAddress
	if err := db.Get(&address, `SELECT uuid,user_uuid,label,recipient_name,line1,line2,city,region,postal_code,country,phone,is_default_shipping,is_default_billing,created_at,updated_at
                                 FROM user_addresses
                                 WHERE user_uuid=$1 AND uuid=$2`, user, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserAddress{}, fiber.NewError(fiber.StatusNotFound, "address not found")
		}
		return UserAddress{}, fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return address, nil
}

func clearDefaultFlagsTx(tx *sqlx.Tx, user string, shipping, billing bool, exclude uuid.UUID) error {
	if shipping {
		query := `UPDATE user_addresses SET is_default_shipping=false, updated_at=now() WHERE user_uuid=$1`
		args := []any{user}
		if exclude != uuid.Nil {
			query += " AND uuid<>$2"
			args = append(args, exclude)
		}
		if _, err := tx.Exec(query, args...); err != nil {
			return err
		}
	}
	if billing {
		query := `UPDATE user_addresses SET is_default_billing=false, updated_at=now() WHERE user_uuid=$1`
		args := []any{user}
		if exclude != uuid.Nil {
			query += " AND uuid<>$2"
			args = append(args, exclude)
		}
		if _, err := tx.Exec(query, args...); err != nil {
			return err
		}
	}
	return nil
}
