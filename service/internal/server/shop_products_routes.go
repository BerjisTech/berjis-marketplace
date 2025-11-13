package server

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	srvAuth "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	pq "github.com/lib/pq"
)

func registerShopProductRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
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
			ShopSlug     string           `json:"shopSlug"`
			Title        string           `json:"title"`
			Slug         string           `json:"slug"`
			Summary      string           `json:"summary"`
			PriceCents   int64            `json:"priceCents"`
			Currency     string           `json:"currency"`
			Stock        int64            `json:"stock"`
			ImageURL     *string          `json:"imageUrl"`
			Published    bool             `json:"published"`
			Category     string           `json:"category"`
			CategoryUUID string           `json:"categoryUuid"`
			Images       []string         `json:"images"`
			Variants     []VariantPayload `json:"variants"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, body.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		productID, _, err := createProductWithVariants(opts.DB, shop, ProductCreateInput{
			Title:        body.Title,
			Slug:         body.Slug,
			Summary:      body.Summary,
			PriceCents:   body.PriceCents,
			Currency:     body.Currency,
			Stock:        body.Stock,
			ImageURL:     body.ImageURL,
			Published:    body.Published,
			Category:     body.Category,
			CategoryUUID: body.CategoryUUID,
			Images:       body.Images,
			Variants:     body.Variants,
		})
		if err != nil {
			return writeErrorResponse(c, err)
		}
		if err := refreshAutomaticCollectionsForShop(opts.DB, shop.UUID); err != nil {
			log.Printf("automatic collection refresh error: %v", err)
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": productID}})
	})

	app.Patch("/v1/products/:id", requireAuth, func(c *fiber.Ctx) error {
		id := strings.TrimSpace(c.Params("id"))
		productUUID, err := uuid.Parse(id)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid product id"})
		}
		var meta struct {
			ShopSlug    string `db:"shop_slug"`
			ProductSlug string `db:"product_slug"`
			Title       string `db:"title"`
			PriceCents  int64  `db:"price_cents"`
			Stock       int64  `db:"stock"`
		}
		if err := opts.DB.Get(&meta, `SELECT s.slug AS shop_slug, p.slug AS product_slug, p.title, p.price_cents, p.stock
                                       FROM products p
                                       JOIN shops s ON s.uuid=p.shop_uuid
                                       WHERE p.uuid=$1 AND p.deleted_at IS NULL`, productUUID); err != nil {
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
		var variantPayloads []VariantPayload
		if raw, ok := body["variants"]; ok {
			b, err := json.Marshal(raw)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid variants"})
			}
			if err := json.Unmarshal(b, &variantPayloads); err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid variants"})
			}
			delete(body, "variants")
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()
		sets := make([]string, 0, 12)
		args := make([]any, 0, 12)
		add := func(col string, v any) { sets = append(sets, col+"=$"+itoa(len(args)+1)); args = append(args, v) }
		currentTitle := meta.Title
		currentPrice := meta.PriceCents
		currentStock := meta.Stock
		if v, ok := body["title"].(string); ok {
			title := strings.TrimSpace(v)
			if title != "" {
				add("title", title)
				currentTitle = title
			}
		}
		if v, ok := body["slug"].(string); ok {
			newSlug := strings.ToLower(strings.TrimSpace(v))
			if newSlug != "" {
				add("slug", newSlug)
				meta.ProductSlug = newSlug
			}
		}
		if v, ok := body["summary"].(string); ok {
			add("summary", strings.TrimSpace(v))
		}
		if v, ok := body["priceCents"].(float64); ok {
			val := int64(v)
			add("price_cents", val)
			currentPrice = val
		}
		if v, ok := body["currency"].(string); ok {
			add("currency", strings.ToUpper(v))
		}
		if raw, ok := body["stock"]; ok {
			switch val := raw.(type) {
			case float64:
				q := int64(val)
				if q < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "stock cannot be negative"})
				}
				add("stock", q)
				currentStock = q
			case int:
				q := int64(val)
				if q < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "stock cannot be negative"})
				}
				add("stock", q)
				currentStock = q
			case int64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "stock cannot be negative"})
				}
				add("stock", val)
				currentStock = val
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
		var categoryUUIDOverride *uuid.UUID
		categoryUpdated := false
		if raw, ok := body["categoryUuid"]; ok {
			switch val := raw.(type) {
			case string:
				trimmed := strings.TrimSpace(val)
				if trimmed != "" {
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
					add("category_uuid", cat.UUID)
					add("category", cat.Slug)
					categoryUpdated = true
				} else {
					categoryUUIDOverride = nil
					add("category_uuid", nil)
					add("category", "")
					categoryUpdated = true
				}
			case nil:
				categoryUUIDOverride = nil
				add("category_uuid", nil)
				add("category", "")
				categoryUpdated = true
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid category uuid"})
			}
		} else if v, ok := body["category"].(string); ok {
			add("category", strings.TrimSpace(v))
			categoryUpdated = true
		}
		if v, ok := body["images"].([]any); ok {
			arr := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok {
					arr = append(arr, s)
				}
			}
			add("images", pq.Array(arr))
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, productUUID)
			query := "UPDATE products SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args)) + " AND deleted_at IS NULL"
			res, err := tx.Exec(query, args...)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if affected, _ := res.RowsAffected(); affected == 0 {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
		}
		if variantPayloads != nil {
			totalStock, err := replaceProductVariantsTx(tx, productUUID, meta.ProductSlug, currentTitle, currentPrice, currentStock, variantPayloads)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
			currentStock = totalStock
			if _, err := tx.Exec(`UPDATE products SET stock=$1, updated_at=now() WHERE uuid=$2`, totalStock, productUUID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if categoryUpdated {
			if categoryUUIDOverride != nil {
				if _, err := tx.Exec(`INSERT INTO category_products(category_uuid, product_uuid, is_primary)
					VALUES($1,$2,true)
					ON CONFLICT (category_uuid, product_uuid) DO UPDATE SET is_primary=EXCLUDED.is_primary`, *categoryUUIDOverride, productUUID); err != nil {
					return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
				}
			} else {
				if _, err := tx.Exec(`DELETE FROM category_products WHERE product_uuid=$1`, productUUID); err != nil {
					return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
				}
			}
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := upsertDefaultInventory(opts.DB, shop.UUID, productUUID, currentStock); err != nil {
			log.Printf("inventory upsert error: %v", err)
		}
		if err := refreshAutomaticCollectionsForShop(opts.DB, shop.UUID); err != nil {
			log.Printf("automatic collection refresh error: %v", err)
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
		shop, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
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
		if err := refreshAutomaticCollectionsForShop(opts.DB, shop.UUID); err != nil {
			log.Printf("automatic collection refresh error: %v", err)
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
		if err := attachVariantsList(opts.DB, items); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": items})
	})

	app.Get("/v1/my/shops/:slug/products/export", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var products []Product
		if err := opts.DB.Select(&products, `SELECT p.uuid, p.shop_uuid, p.title, p.slug, p.summary, p.price_cents, p.currency, p.stock, p.image_url,
		                                      p.category, p.category_uuid, p.images, COALESCE(p.rating,0) AS rating, COALESCE(p.review_count,0) AS review_count,
		                                      p.created_at, p.updated_at, p.deleted_at, p.published,
		                                      s.name AS shop_name, s.slug AS shop_slug
		                               FROM products p
		                               JOIN shops s ON s.uuid=p.shop_uuid
		                               WHERE s.uuid=$1 AND p.deleted_at IS NULL
		                               ORDER BY p.created_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := attachVariantsList(opts.DB, products); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)
		header := []string{"title", "slug", "summary", "price_cents", "currency", "stock", "category", "category_uuid", "image_url", "published", "images", "variants"}
		if err := writer.Write(header); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "csv error"})
		}
		for _, product := range products {
			var variantsJSON string
			if len(product.Variants) > 0 {
				if encoded, err := json.Marshal(product.Variants); err == nil {
					variantsJSON = string(encoded)
				}
			}
			imageURL := ""
			if product.ImageURL != nil {
				imageURL = *product.ImageURL
			}
			row := []string{
				product.Title,
				product.Slug,
				product.Summary,
				strconv.FormatInt(product.PriceCents, 10),
				product.Currency,
				strconv.FormatInt(product.Stock, 10),
				product.Category,
				func() string {
					if product.CategoryUUID != nil {
						return product.CategoryUUID.String()
					}
					return ""
				}(),
				imageURL,
				strconv.FormatBool(product.Published),
				joinCSVList(product.Images),
				variantsJSON,
			}
			if err := writer.Write(row); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "csv error"})
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "csv error"})
		}
		filename := fmt.Sprintf("%s-products.csv", slug)
		c.Set(fiber.HeaderContentType, "text/csv")
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		return c.Send(buf.Bytes())
	})

	app.Post("/v1/my/shops/:slug/products/import", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		reader, cleanup, err := csvReaderFromRequest(c)
		if err != nil {
			return writeErrorResponse(c, err)
		}
		defer cleanup()
		header, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "empty csv"})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid csv"})
		}
		index := make(map[string]int)
		for i, col := range header {
			index[strings.ToLower(strings.TrimSpace(col))] = i
		}
		get := func(row []string, key string) string {
			if idx, ok := index[key]; ok && idx >= 0 && idx < len(row) {
				return row[idx]
			}
			return ""
		}
		created := make([]string, 0)
		rowNumber := 1
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			rowNumber++
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("row %d: %v", rowNumber, err)})
			}
			title := get(record, "title")
			if strings.TrimSpace(title) == "" {
				continue
			}
			slugValue := get(record, "slug")
			price, err := parseCSVInt(get(record, "price_cents"), 0)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("row %d: invalid price_cents", rowNumber)})
			}
			stock, err := parseCSVInt(get(record, "stock"), 0)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("row %d: invalid stock", rowNumber)})
			}
			currency := get(record, "currency")
			imageURL := strings.TrimSpace(get(record, "image_url"))
			var imagePtr *string
			if imageURL != "" {
				imagePtr = &imageURL
			}
			images := splitCSVList(get(record, "images"))
			var variants []VariantPayload
			if variantsRaw := get(record, "variants"); strings.TrimSpace(variantsRaw) != "" {
				if err := json.Unmarshal([]byte(variantsRaw), &variants); err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("row %d: invalid variants json", rowNumber)})
				}
			}
			productID, _, err := createProductWithVariants(opts.DB, shop, ProductCreateInput{
				Title:        title,
				Slug:         slugValue,
				Summary:      get(record, "summary"),
				PriceCents:   price,
				Currency:     currency,
				Stock:        stock,
				ImageURL:     imagePtr,
				Published:    parseCSVBool(get(record, "published")),
				Category:     get(record, "category"),
				CategoryUUID: get(record, "category_uuid"),
				Images:       images,
				Variants:     variants,
			})
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("row %d: %s", rowNumber, errorMessage(err))})
			}
			created = append(created, productID.String())
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"created":     len(created),
			"productIds":  created,
			"shopUuid":    shop.UUID.String(),
			"shopSlug":    shop.Slug,
			"importedCsv": true,
		}})
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

}
