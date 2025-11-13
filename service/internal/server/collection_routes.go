package server

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func registerCollectionRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
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
		if body.IsAutomatic {
			if len(body.ProductUUIDs) > 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "automatic collections cannot specify manual product assignments"})
			}
			if err := validateCollectionRulesForShop(shop.UUID, body.Rules); err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid rules"})
			}
		}
		var manualProducts []uuid.UUID
		if !body.IsAutomatic && len(body.ProductUUIDs) > 0 {
			productIDs, err := parseUUIDList(body.ProductUUIDs)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
			manualProducts = productIDs
		}
		id := uuid.New()
		rulesValue := nullIfEmptyJSON(body.Rules)
		if _, err := opts.DB.Exec(`INSERT INTO collections(uuid, shop_uuid, title, slug, description, is_automatic, rules, sort_order, is_active)
                                    VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			id, shop.UUID, title, strings.ToLower(slugValue), strings.TrimSpace(body.Description), body.IsAutomatic, rulesValue, sortOrder, isActive); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if body.IsAutomatic {
			if err := refreshAutomaticCollection(opts.DB, id); err != nil {
				_, _ = opts.DB.Exec(`DELETE FROM collections WHERE uuid=$1`, id)
				log.Printf("automatic collection refresh error: %v", err)
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "could not refresh automatic collection"})
			}
		} else if manualProducts != nil {
			if err := setCollectionProducts(opts.DB, id, shop.UUID, manualProducts); err != nil {
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
		var current struct {
			IsAutomatic bool            `db:"is_automatic"`
			Rules       json.RawMessage `db:"rules"`
		}
		if err := opts.DB.Get(&current, `SELECT is_automatic, rules FROM collections WHERE uuid=$1 AND shop_uuid=$2`, collectionID, shop.UUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		var body map[string]any
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		sets := make([]string, 0, 8)
		args := make([]any, 0, 8)
		add := func(col string, v any) { sets = append(sets, col+"=$"+itoa(len(args)+1)); args = append(args, v) }
		newIsAutomatic := current.IsAutomatic
		newRulesRaw := current.Rules
		updatedRules := false
		isAutomaticProvided := false
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
			newIsAutomatic = v
			isAutomaticProvided = true
		}
		if raw, ok := body["rules"]; ok {
			updatedRules = true
			b, err := json.Marshal(raw)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid rules"})
			}
			if len(bytes.TrimSpace(b)) == 0 || string(bytes.TrimSpace(b)) == "null" {
				add("rules", nil)
				newRulesRaw = nil
			} else {
				add("rules", b)
				newRulesRaw = append(json.RawMessage(nil), b...)
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
		if newIsAutomatic {
			if err := validateCollectionRulesForShop(shop.UUID, newRulesRaw); err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid rules"})
			}
			if productUpdateIDs != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "cannot set manual products for an automatic collection"})
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
		if newIsAutomatic && (updatedRules || isAutomaticProvided) {
			if err := refreshAutomaticCollection(opts.DB, collectionID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "could not refresh automatic collection"})
			}
		} else if productUpdateIDs != nil {
			if err := setCollectionProducts(opts.DB, collectionID, shop.UUID, productUpdateIDs); err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Post("/v1/my/shops/:slug/collections/:id/rebuild", requireAuth, func(c *fiber.Ctx) error {
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
		var meta struct {
			ShopUUID    uuid.UUID `db:"shop_uuid"`
			IsAutomatic bool      `db:"is_automatic"`
		}
		if err := opts.DB.Get(&meta, `SELECT shop_uuid, is_automatic FROM collections WHERE uuid=$1`, collectionID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if meta.ShopUUID != shop.UUID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		if !meta.IsAutomatic {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "collection is not automatic"})
		}
		if err := refreshAutomaticCollection(opts.DB, collectionID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "could not refresh collection"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Get("/v1/my/shops/:slug/collections/:id/products", requireAuth, func(c *fiber.Ctx) error {
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
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var exists int
		if err := opts.DB.Get(&exists, `SELECT COUNT(1) FROM collections WHERE uuid=$1 AND shop_uuid=$2`, collectionID, shop.UUID); err != nil || exists == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		type collectionProduct struct {
			UUID       uuid.UUID `db:"uuid" json:"uuid"`
			Title      string    `db:"title" json:"title"`
			PriceCents int64     `db:"price_cents" json:"priceCents"`
			Currency   string    `db:"currency" json:"currency"`
			Stock      int64     `db:"stock" json:"stock"`
			Published  bool      `db:"published" json:"published"`
		}
		var products []collectionProduct
		if err := opts.DB.Select(&products, `SELECT p.uuid, p.title, p.price_cents, p.currency, p.stock, p.published
                                             FROM collection_products cp
                                             JOIN products p ON p.uuid=cp.product_uuid
                                             WHERE cp.collection_uuid=$1
                                             ORDER BY cp.position ASC`, collectionID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": products})
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

}
