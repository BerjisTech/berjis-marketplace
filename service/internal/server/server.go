package server

import (
	"database/sql"
	"encoding/json"
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
	UUID        uuid.UUID `db:"uuid" json:"uuid"`
	ShopUUID    uuid.UUID `db:"shop_uuid" json:"shopUuid"`
	Title       string    `db:"title" json:"title"`
	Slug        string    `db:"slug" json:"slug"`
	Summary     string    `db:"summary" json:"summary"`
	PriceCents  int64     `db:"price_cents" json:"priceCents"`
	Currency    string    `db:"currency" json:"currency"`
	Stock       int64     `db:"stock" json:"stock"`
	ImageURL    *string   `db:"image_url" json:"imageUrl,omitempty"`
	Category    string    `db:"category" json:"category"`
	Images      []string  `db:"images" json:"images"`
	Rating      float32   `db:"rating" json:"rating"`
	ReviewCount int64     `db:"review_count" json:"reviewCount"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
	Published   bool      `db:"published" json:"published"`
	ShopName    string    `db:"shop_name" json:"shopName"`
	ShopSlug    string    `db:"shop_slug" json:"shopSlug"`
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
		base := `SELECT p.uuid, p.shop_uuid, p.title, p.slug, p.summary, p.price_cents, p.currency, p.stock, p.image_url, p.created_at, p.updated_at, p.published,
                        p.category, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
                        s.name AS shop_name, s.slug AS shop_slug
                 FROM products p
                 JOIN shops s ON s.uuid = p.shop_uuid
                 WHERE p.published = true AND s.public = true`
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
                                         p.created_at, p.updated_at, p.published,
                                         s.name AS shop_name, s.slug AS shop_slug
                                  FROM products p JOIN shops s ON s.uuid=p.shop_uuid WHERE p.uuid=$1`, id)
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
		if err := opts.DB.Select(&rows, `SELECT DISTINCT NULLIF(TRIM(category),'') AS category FROM products WHERE published=true ORDER BY 1 ASC`); err != nil {
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
                                              p.created_at, p.updated_at, p.published,
                                              s.name AS shop_name, s.slug AS shop_slug
                                       FROM products p JOIN shops s ON s.uuid=p.shop_uuid WHERE s.slug=$1 AND p.published=true ORDER BY p.created_at DESC`, slug)
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
		owner := srvAuth.UserID(c)
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
		// verify shop ownership
		var shop Shop
		if err := opts.DB.Get(&shop, `SELECT uuid, name, slug, owner_uuid, public, description, created_at, updated_at FROM shops WHERE slug=$1`, body.ShopSlug); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "shop not found"})
		}
		if shop.OwnerUUID.String() != owner {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "not your shop"})
		}
		id := uuid.New()
		_, err := opts.DB.Exec(`INSERT INTO products(uuid,shop_uuid,title,slug,summary,price_cents,currency,stock,image_url,published,category,images) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			id, shop.UUID, strings.TrimSpace(body.Title), strings.ToLower(strings.TrimSpace(body.Slug)), strings.TrimSpace(body.Summary), body.PriceCents, strings.ToUpper(body.Currency), body.Stock, body.ImageURL, body.Published, strings.TrimSpace(body.Category), pqStringArray(body.Images),
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id}})
	})

	// Update/Delete product (owner only)
	app.Patch("/v1/products/:id", requireAuth, func(c *fiber.Ctx) error {
		owner := srvAuth.UserID(c)
		id := strings.TrimSpace(c.Params("id"))
		var shopOwner uuid.UUID
		if err := opts.DB.Get(&shopOwner, `SELECT s.owner_uuid FROM products p JOIN shops s ON s.uuid=p.shop_uuid WHERE p.uuid=$1`, id); err != nil {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		if shopOwner.String() != owner {
			return c.Status(403).JSON(fiber.Map{"success": false, "message": "forbidden"})
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
		if _, err := opts.DB.Exec("UPDATE products SET "+strings.Join(sets, ", ")+" WHERE uuid=$"+itoa(len(args)), args...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Delete("/v1/products/:id", requireAuth, func(c *fiber.Ctx) error {
		owner := srvAuth.UserID(c)
		id := strings.TrimSpace(c.Params("id"))
		var shopOwner uuid.UUID
		if err := opts.DB.Get(&shopOwner, `SELECT s.owner_uuid FROM products p JOIN shops s ON s.uuid=p.shop_uuid WHERE p.uuid=$1`, id); err != nil {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		if shopOwner.String() != owner {
			return c.Status(403).JSON(fiber.Map{"success": false, "message": "forbidden"})
		}
		if _, err := opts.DB.Exec(`DELETE FROM products WHERE uuid=$1`, id); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// My shops + products
	app.Get("/v1/my/shops", requireAuth, func(c *fiber.Ctx) error {
		owner := srvAuth.UserID(c)
		var shops []Shop
		if err := opts.DB.Select(&shops, `SELECT uuid,name,slug,owner_uuid,public,description,created_at,updated_at FROM shops WHERE owner_uuid=$1 ORDER BY created_at DESC`, owner); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": shops})
	})
	app.Get("/v1/my/shops/:slug/products", requireAuth, func(c *fiber.Ctx) error {
		owner := srvAuth.UserID(c)
		slug := c.Params("slug")
		var items []Product
		err := opts.DB.Select(&items, `SELECT p.uuid, p.shop_uuid, p.title, p.slug, p.summary, p.price_cents, p.currency, p.stock, p.image_url,
                                              p.category, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
                                              p.created_at, p.updated_at, p.published,
                                              s.name AS shop_name, s.slug AS shop_slug
                                       FROM products p JOIN shops s ON s.uuid=p.shop_uuid WHERE s.slug=$1 AND s.owner_uuid=$2 ORDER BY p.created_at DESC`, slug, owner)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": items})
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
		_ = opts.DB.Get(&count, `SELECT COUNT(1) FROM products WHERE shop_uuid=$1`, shopID)
		if count == 0 {
			for i := 1; i <= 3; i++ {
				_, _ = opts.DB.Exec(`INSERT INTO products(uuid,shop_uuid,title,slug,summary,price_cents,currency,stock,published,category) VALUES($1,$2,$3,$4,$5,$6,'USD',10,true,'general')`, uuid.New(), shopID, fmt.Sprintf("Product %d", i), fmt.Sprintf("product-%d", i), "Demo product", int64(i*1000))
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	// Cart minimal endpoints
	app.Get("/v1/cart", requireAuth, func(c *fiber.Ctx) error { return getCart(c, opts.DB) })
	app.Post("/v1/cart/items", requireAuth, func(c *fiber.Ctx) error { return addCartItem(c, opts.DB) })
	app.Put("/v1/cart/items/:id", requireAuth, func(c *fiber.Ctx) error { return updateCartItem(c, opts.DB) })
	app.Delete("/v1/cart/items/:id", requireAuth, func(c *fiber.Ctx) error { return deleteCartItem(c, opts.DB) })
	app.Delete("/v1/cart", requireAuth, func(c *fiber.Ctx) error { return clearCart(c, opts.DB) })

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
                                                 p.created_at, p.updated_at, p.published,
                                                 s.name AS shop_name, s.slug AS shop_slug
                                          FROM products p JOIN shops s ON s.uuid=p.shop_uuid
                                          WHERE s.owner_uuid=$1 AND (LOWER(p.title) LIKE LOWER($2) OR LOWER(p.summary) LIKE LOWER($2))
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
                                           JOIN products p ON p.uuid=oi.product_uuid
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

func getCart(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	cartID, err := ensureCart(db, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	var items []CartItem
	q := `SELECT ci.uuid, ci.product_uuid, ci.quantity, p.title, p.price_cents, p.currency, p.image_url
          FROM cart_items ci JOIN products p ON p.uuid=ci.product_uuid WHERE ci.cart_uuid=$1 ORDER BY ci.added_at DESC`
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

func addCartItem(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	var body struct {
		ProductID string `json:"productId"`
		Quantity  int    `json:"quantity"`
	}
	if err := c.BodyParser(&body); err != nil || body.ProductID == "" || body.Quantity <= 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
	}
	cartID, err := ensureCart(db, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	// upsert
	_, err = db.Exec(`INSERT INTO cart_items(cart_uuid,product_uuid,quantity) VALUES($1,$2,$3)
                      ON CONFLICT (cart_uuid, product_uuid) DO UPDATE SET quantity=cart_items.quantity+EXCLUDED.quantity`, cartID, body.ProductID, body.Quantity)
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
	if err := db.Select(&rows, `SELECT ci.product_uuid, ci.quantity, p.price_cents, p.currency FROM cart_items ci JOIN products p ON p.uuid=ci.product_uuid WHERE ci.cart_uuid=$1`, cartID); err != nil {
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
