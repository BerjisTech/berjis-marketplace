package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	UUID        uuid.UUID  `db:"uuid" json:"uuid"`
	ShopUUID    uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	Title       string     `db:"title" json:"title"`
	Slug        string     `db:"slug" json:"slug"`
	Summary     string     `db:"summary" json:"summary"`
	PriceCents  int64      `db:"price_cents" json:"priceCents"`
	Currency    string     `db:"currency" json:"currency"`
	Stock       int64      `db:"stock" json:"stock"`
	ImageURL    *string    `db:"image_url" json:"imageUrl,omitempty"`
	Category    string     `db:"category" json:"category"`
	Images      []string   `db:"images" json:"images"`
	Rating      float32    `db:"rating" json:"rating"`
	ReviewCount int64      `db:"review_count" json:"reviewCount"`
	CreatedAt   time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updatedAt"`
	DeletedAt   *time.Time `db:"deleted_at" json:"-"`
	Published   bool       `db:"published" json:"published"`
	ShopName    string     `db:"shop_name" json:"shopName"`
	ShopSlug    string     `db:"shop_slug" json:"shopSlug"`
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
                        p.category, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
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
                                         p.category, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
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
	// Categories (distinct)
	app.Get("/v1/categories", func(c *fiber.Ctx) error {
		rows := []struct {
			Category *string `db:"category"`
		}{}
		if err := opts.DB.Select(&rows, `SELECT DISTINCT NULLIF(TRIM(category),'') AS category FROM products WHERE published=true AND deleted_at IS NULL ORDER BY 1 ASC`); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		out := make([]string, 0, len(rows))
		for _, r := range rows {
			if r.Category != nil {
				out = append(out, *r.Category)
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
                                              p.category, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
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
			ShopSlug   string   `json:"shopSlug"`
			Title      string   `json:"title"`
			Slug       string   `json:"slug"`
			Summary    string   `json:"summary"`
			PriceCents int64    `json:"priceCents"`
			Currency   string   `json:"currency"`
			Stock      int64    `json:"stock"`
			ImageURL   *string  `json:"imageUrl"`
			Published  bool     `json:"published"`
			Category   string   `json:"category"`
			Images     []string `json:"images"`
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
		id := uuid.New()
		_, err = opts.DB.Exec(`INSERT INTO products(uuid,shop_uuid,title,slug,summary,price_cents,currency,stock,image_url,published,category,images) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			id, shop.UUID, strings.TrimSpace(body.Title), strings.ToLower(strings.TrimSpace(body.Slug)), strings.TrimSpace(body.Summary), body.PriceCents, strings.ToUpper(body.Currency), body.Stock, body.ImageURL, body.Published, strings.TrimSpace(body.Category), pqStringArray(body.Images),
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
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
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
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
		if v, ok := body["stock"].(float64); ok {
			add("stock", int64(v))
		}
		if v, ok := body["imageUrl"].(string); ok {
			add("image_url", strings.TrimSpace(v))
		}
		if v, ok := body["published"].(bool); ok {
			add("published", v)
		}
		if v, ok := body["category"].(string); ok {
			add("category", strings.TrimSpace(v))
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
                                              p.category, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
                                              p.created_at, p.updated_at, p.deleted_at, p.published,
                                              s.name AS shop_name, s.slug AS shop_slug
                                       FROM products p JOIN shops s ON s.uuid=p.shop_uuid
                                       WHERE s.uuid=$1 AND p.deleted_at IS NULL
                                       ORDER BY p.created_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": items})
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
				_, _ = opts.DB.Exec(`INSERT INTO products(uuid,shop_uuid,title,slug,summary,price_cents,currency,stock,published,category) VALUES($1,$2,$3,$4,$5,$6,'USD',10,true,'general')`, uuid.New(), shopID, fmt.Sprintf("Product %d", i), fmt.Sprintf("product-%d", i), "Demo product", int64(i*1000))
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
                                                 p.category, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
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
			UUID       uuid.UUID `db:"uuid" json:"uuid"`
			TotalCents int64     `db:"total_cents" json:"totalCents"`
			Currency   string    `db:"currency" json:"currency"`
			Status     string    `db:"status" json:"status"`
			CreatedAt  time.Time `db:"created_at" json:"createdAt"`
		}
		var orders []OrderRow
		if err := opts.DB.Select(&orders, `SELECT DISTINCT o.uuid, o.total_cents, o.currency, o.status, o.created_at
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
	UUID       uuid.UUID `db:"uuid" json:"uuid"`
	TotalCents int64     `db:"total_cents" json:"totalCents"`
	Currency   string    `db:"currency" json:"currency"`
	Status     string    `db:"status" json:"status"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`
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
	if err := db.Select(&out, `SELECT uuid,total_cents,currency,status,created_at FROM orders WHERE user_uuid=$1 ORDER BY created_at DESC`, user); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return c.JSON(fiber.Map{"success": true, "data": out})
}

func getOrder(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	id := c.Params("id")
	var o Order
	if err := db.Get(&o, `SELECT uuid,total_cents,currency,status,created_at FROM orders WHERE uuid=$1 AND user_uuid=$2`, id, user); err != nil {
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
