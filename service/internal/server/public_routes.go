package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"strings"
)

func registerPublicRoutes(app *fiber.App, opts Options) {
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

	// Catalog: list all published products across shops (marketplace landing)
	app.Get("/v1/products", func(c *fiber.Ctx) error {
		q := strings.TrimSpace(c.Query("q"))
		category := strings.TrimSpace(c.Query("category"))
		minPrice := strings.TrimSpace(c.Query("minPrice"))
		maxPrice := strings.TrimSpace(c.Query("maxPrice"))
		collectionFilter := strings.TrimSpace(c.Query("collection"))
		shopFilter := strings.TrimSpace(c.Query("shop"))

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
                 JOIN shops s ON s.uuid = p.shop_uuid`
		if collectionFilter != "" {
			base += `
                 JOIN collection_products cp ON cp.product_uuid = p.uuid
                 JOIN collections col ON col.uuid = cp.collection_uuid`
		}
		base += `
                 WHERE p.deleted_at IS NULL AND p.published = true AND s.public = true`
		args := make([]any, 0, 10)

		if collectionFilter != "" {
			if id, err := uuid.Parse(collectionFilter); err == nil {
				placeholder := "$" + itoa(len(args)+1)
				args = append(args, id)
				base += " AND col.uuid=" + placeholder
			} else {
				placeholder := "$" + itoa(len(args)+1)
				args = append(args, strings.ToLower(collectionFilter))
				base += " AND LOWER(col.slug)=LOWER(" + placeholder + ")"
			}
			base += " AND col.is_active=true AND col.shop_uuid = p.shop_uuid"
		}
		if shopFilter != "" {
			placeholder := "$" + itoa(len(args)+1)
			args = append(args, strings.ToLower(shopFilter))
			base += " AND LOWER(s.slug)=LOWER(" + placeholder + ")"
		}
		if q != "" {
			placeholder := "$" + itoa(len(args)+1)
			args = append(args, "%"+strings.ToLower(q)+"%")
			base += " AND (LOWER(p.title) LIKE " + placeholder + " OR LOWER(s.name) LIKE " + placeholder + ")"
		}
		if category != "" {
			placeholder := "$" + itoa(len(args)+1)
			args = append(args, strings.ToLower(category))
			base += " AND LOWER(p.category)=LOWER(" + placeholder + ")"
		}
		if minCents > 0 {
			placeholder := "$" + itoa(len(args)+1)
			args = append(args, minCents)
			base += " AND p.price_cents >= " + placeholder
		}
		if maxCents > 0 {
			placeholder := "$" + itoa(len(args)+1)
			args = append(args, maxCents)
			base += " AND p.price_cents <= " + placeholder
		}
		if collectionFilter != "" {
			base += " ORDER BY cp.position ASC, p.created_at DESC LIMIT 100"
		} else {
			base += " ORDER BY p.created_at DESC LIMIT 100"
		}

		if err := opts.DB.Select(&items, base, args...); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := attachVariantsList(opts.DB, items); err != nil {
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
		if err := attachVariantsSingle(opts.DB, &item); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
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
	app.Get("/v1/shops/:slug/collections", func(c *fiber.Ctx) error {
		slug := strings.TrimSpace(c.Params("slug"))
		var shop Shop
		if err := opts.DB.Get(&shop, `SELECT uuid, name, slug, owner_uuid, public, description, created_at, updated_at FROM shops WHERE slug=$1`, slug); err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		if !shop.Public {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		type collectionSummary struct {
			UUID         uuid.UUID `db:"uuid" json:"uuid"`
			Title        string    `db:"title" json:"title"`
			Slug         string    `db:"slug" json:"slug"`
			Description  string    `db:"description" json:"description"`
			IsAutomatic  bool      `db:"is_automatic" json:"isAutomatic"`
			SortOrder    int       `db:"sort_order" json:"sortOrder"`
			ProductCount int       `db:"product_count" json:"productCount"`
		}
		var collections []collectionSummary
		if err := opts.DB.Select(&collections, `SELECT c.uuid, c.title, c.slug, c.description, c.is_automatic, c.sort_order,
                                                        COUNT(cp.product_uuid) AS product_count
                                                 FROM collections c
                                                 LEFT JOIN collection_products cp ON cp.collection_uuid = c.uuid
                                                 WHERE c.shop_uuid=$1 AND c.is_active=true
                                                 GROUP BY c.uuid
                                                 ORDER BY c.sort_order ASC, c.title ASC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": collections})
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
		if err := attachVariantsList(opts.DB, items); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": items})
	})

}
