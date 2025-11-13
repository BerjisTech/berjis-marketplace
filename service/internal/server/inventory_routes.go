package server

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	srvAuth "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	pq "github.com/lib/pq"
)

func registerInventoryRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
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
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		var current InventoryLevel
		if err := tx.Get(&current, `SELECT uuid, shop_uuid, product_uuid, location_uuid, quantity, reserved, safety_stock, created_at, updated_at
                                    FROM inventory_levels WHERE uuid=$1 FOR UPDATE`, levelID); err != nil {
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
		deltaQuantity := targetQuantity - current.Quantity
		deltaReserved := targetReserved - current.Reserved

		if deltaQuantity == 0 && deltaReserved == 0 && targetSafety == current.SafetyStock {
			return c.JSON(fiber.Map{"success": true})
		}

		if _, err := tx.Exec(`UPDATE inventory_levels SET quantity=$1, reserved=$2, safety_stock=$3, updated_at=now() WHERE uuid=$4`,
			targetQuantity, targetReserved, targetSafety, levelID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if deltaQuantity != 0 || deltaReserved != 0 {
			if _, err := recordInventoryAdjustmentTx(tx, current, targetQuantity, targetReserved, deltaQuantity, deltaReserved, srvAuth.UserID(c), "manual_update", "direct_edit", ""); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}

		current.Quantity = targetQuantity
		current.Reserved = targetReserved
		current.SafetyStock = targetSafety
		if err := syncLowStockAlertTx(tx, current, ""); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if err := tx.Commit(); err != nil {
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

	app.Get("/v1/my/shops/:slug/inventory/:id/history", requireAuth, func(c *fiber.Ctx) error {
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
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var levelShop uuid.UUID
		if err := opts.DB.Get(&levelShop, `SELECT shop_uuid FROM inventory_levels WHERE uuid=$1`, levelID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if levelShop != shop.UUID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		limit := 50
		if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
			if v, err := strconv.Atoi(raw); err == nil && v > 0 {
				if v > 200 {
					v = 200
				}
				limit = v
			}
		}
		var history []InventoryAdjustment
		if err := opts.DB.Select(&history, `SELECT uuid, inventory_level_uuid, shop_uuid, product_uuid, location_uuid, user_uuid, delta_quantity, delta_reserved,
                                                     resulting_quantity, resulting_reserved, reason, note, adjustment_source, created_at
                                              FROM inventory_adjustments
                                              WHERE inventory_level_uuid=$1
                                              ORDER BY created_at DESC
                                              LIMIT $2`, levelID, limit); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": history})
	})

	app.Post("/v1/my/shops/:slug/inventory/:id/adjustments", requireAuth, func(c *fiber.Ctx) error {
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
		var body struct {
			DeltaQuantity *int64 `json:"deltaQuantity"`
			DeltaReserved *int64 `json:"deltaReserved"`
			Reason        string `json:"reason"`
			Note          string `json:"note"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		deltaQuantity := int64(0)
		if body.DeltaQuantity != nil {
			deltaQuantity = *body.DeltaQuantity
		}
		deltaReserved := int64(0)
		if body.DeltaReserved != nil {
			deltaReserved = *body.DeltaReserved
		}
		if deltaQuantity == 0 && deltaReserved == 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "deltaQuantity or deltaReserved required"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		var level InventoryLevel
		if err := tx.Get(&level, `SELECT uuid, shop_uuid, product_uuid, location_uuid, quantity, reserved, safety_stock, created_at, updated_at
                                   FROM inventory_levels
                                   WHERE uuid=$1 FOR UPDATE`, levelID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if level.ShopUUID != shop.UUID {
			return c.Status(403).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		targetQuantity := level.Quantity + deltaQuantity
		targetReserved := level.Reserved + deltaReserved
		if targetQuantity < 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "resulting quantity cannot be negative"})
		}
		if targetReserved < 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "resulting reserved cannot be negative"})
		}
		if targetReserved > targetQuantity {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "reserved cannot exceed quantity"})
		}

		if _, err := tx.Exec(`UPDATE inventory_levels SET quantity=$1, reserved=$2, updated_at=now() WHERE uuid=$3`,
			targetQuantity, targetReserved, levelID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		reason := strings.TrimSpace(body.Reason)
		if reason == "" {
			reason = "manual_adjustment"
		}
		note := strings.TrimSpace(body.Note)
		adjustment, err := recordInventoryAdjustmentTx(tx, level, targetQuantity, targetReserved, deltaQuantity, deltaReserved, srvAuth.UserID(c), reason, "adjustment_endpoint", note)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		level.Quantity = targetQuantity
		level.Reserved = targetReserved
		if err := syncLowStockAlertTx(tx, level, note); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if err := syncProductStockFromInventory(opts.DB, level.ProductUUID); err != nil {
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
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"inventory": updated, "adjustment": adjustment}})
	})

	app.Get("/v1/my/shops/:slug/inventory/alerts", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		status := strings.ToLower(strings.TrimSpace(c.Query("status", "open")))
		params := []any{shop.UUID}
		query := `SELECT a.uuid, a.inventory_level_uuid, a.shop_uuid, a.product_uuid, a.location_uuid, a.quantity, a.safety_stock, a.status, a.triggered_at, a.resolved_at, a.note,
                          p.title AS product_title, p.slug AS product_slug,
                          loc.name AS location_name, loc.code AS location_code
                   FROM inventory_alerts a
                   JOIN products p ON p.uuid = a.product_uuid
                   LEFT JOIN inventory_locations loc ON loc.uuid = a.location_uuid
                   WHERE a.shop_uuid=$1`
		switch status {
		case "resolved":
			query += " AND a.status='resolved'"
		case "all":
			// no additional filter
		default:
			query += " AND a.status='open'"
		}
		query += " ORDER BY a.triggered_at DESC"
		var alerts []InventoryAlertEntry
		if err := opts.DB.Select(&alerts, query, params...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": alerts})
	})

	app.Post("/v1/my/shops/:slug/inventory/alerts/:id/resolve", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		alertParam := c.Params("id")
		alertID, err := uuid.Parse(alertParam)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid alert id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Note string `json:"note"`
		}
		if err := c.BodyParser(&body); err != nil && err != io.EOF {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		note := strings.TrimSpace(body.Note)
		query := `UPDATE inventory_alerts
                  SET status='resolved', resolved_at=now(), note=CASE WHEN $3 <> '' THEN $3 ELSE note END
                  WHERE uuid=$1 AND shop_uuid=$2 AND status='open'`
		res, err := opts.DB.Exec(query, alertID, shop.UUID, note)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found or already resolved"})
		}
		var alert InventoryAlertEntry
		if err := opts.DB.Get(&alert, `SELECT a.uuid, a.inventory_level_uuid, a.shop_uuid, a.product_uuid, a.location_uuid, a.quantity, a.safety_stock, a.status, a.triggered_at, a.resolved_at, a.note,
                                                p.title AS product_title, p.slug AS product_slug,
                                                loc.name AS location_name, loc.code AS location_code
                                         FROM inventory_alerts a
                                         JOIN products p ON p.uuid = a.product_uuid
                                         LEFT JOIN inventory_locations loc ON loc.uuid = a.location_uuid
                                         WHERE a.uuid=$1 AND a.shop_uuid=$2`, alertID, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": alert})
	})

	app.Get("/v1/my/shops/:slug/inventory/locations", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var locations []InventoryLocation
		if err := opts.DB.Select(&locations, `SELECT uuid, shop_uuid, name, code, description, is_primary, created_at, updated_at
                                              FROM inventory_locations
                                              WHERE shop_uuid=$1
                                              ORDER BY is_primary DESC, name ASC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": locations})
	})

	app.Post("/v1/my/shops/:slug/inventory/locations", requireAuth, func(c *fiber.Ctx) error {
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
			Code        string `json:"code"`
			Description string `json:"description"`
			IsPrimary   bool   `json:"isPrimary"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		name := strings.TrimSpace(body.Name)
		code := strings.ToUpper(strings.TrimSpace(body.Code))
		if name == "" || code == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "name and code are required"})
		}
		description := strings.TrimSpace(body.Description)
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()
		var locationID uuid.UUID
		if err := tx.QueryRow(`INSERT INTO inventory_locations (shop_uuid, name, code, description, is_primary)
                               VALUES ($1,$2,$3,$4,$5)
                               RETURNING uuid`, shop.UUID, name, code, description, body.IsPrimary).Scan(&locationID); err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				return c.Status(409).JSON(fiber.Map{"success": false, "message": "code already in use for this shop"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if body.IsPrimary {
			if _, err := tx.Exec(`UPDATE inventory_locations
                                   SET is_primary=false, updated_at=now()
                                   WHERE shop_uuid=$1 AND uuid<>$2 AND is_primary`, shop.UUID, locationID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var location InventoryLocation
		if err := opts.DB.Get(&location, `SELECT uuid, shop_uuid, name, code, description, is_primary, created_at, updated_at
                                         FROM inventory_locations WHERE uuid=$1`, locationID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": location})
	})

	app.Patch("/v1/my/shops/:slug/inventory/locations/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		locParam := c.Params("id")
		locationID, err := uuid.Parse(strings.TrimSpace(locParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid location id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Name        *string `json:"name"`
			Code        *string `json:"code"`
			Description *string `json:"description"`
			IsPrimary   *bool   `json:"isPrimary"`
		}
		if err := c.BodyParser(&body); err != nil && err != io.EOF {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()
		var existing InventoryLocation
		if err := tx.Get(&existing, `SELECT uuid, shop_uuid, name, code, description, is_primary, created_at, updated_at
                                      FROM inventory_locations
                                      WHERE uuid=$1 FOR UPDATE`, locationID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if existing.ShopUUID != shop.UUID {
			return c.Status(403).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		sets := make([]string, 0, 4)
		args := make([]any, 0, 4)
		if body.Name != nil {
			name := strings.TrimSpace(*body.Name)
			if name == "" {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "name cannot be empty"})
			}
			sets = append(sets, "name=$"+itoa(len(args)+1))
			args = append(args, name)
		}
		if body.Code != nil {
			code := strings.ToUpper(strings.TrimSpace(*body.Code))
			if code == "" {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "code cannot be empty"})
			}
			sets = append(sets, "code=$"+itoa(len(args)+1))
			args = append(args, code)
		}
		if body.Description != nil {
			sets = append(sets, "description=$"+itoa(len(args)+1))
			args = append(args, strings.TrimSpace(*body.Description))
		}
		if body.IsPrimary != nil {
			sets = append(sets, "is_primary=$"+itoa(len(args)+1))
			args = append(args, *body.IsPrimary)
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, locationID)
			query := "UPDATE inventory_locations SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args))
			if _, err := tx.Exec(query, args...); err != nil {
				var pqErr *pq.Error
				if errors.As(err, &pqErr) && pqErr.Code == "23505" {
					return c.Status(409).JSON(fiber.Map{"success": false, "message": "code already in use for this shop"})
				}
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if body.IsPrimary != nil && *body.IsPrimary {
			if _, err := tx.Exec(`UPDATE inventory_locations
                                   SET is_primary=false, updated_at=now()
                                   WHERE shop_uuid=$1 AND uuid<>$2 AND is_primary`, shop.UUID, locationID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var location InventoryLocation
		if err := opts.DB.Get(&location, `SELECT uuid, shop_uuid, name, code, description, is_primary, created_at, updated_at
                                         FROM inventory_locations WHERE uuid=$1`, locationID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": location})
	})

	app.Get("/v1/my/shops/:slug/suppliers", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var suppliers []Supplier
		if err := opts.DB.Select(&suppliers, `SELECT uuid, shop_uuid, name, contact_email, phone, notes, created_at, updated_at
                                              FROM suppliers WHERE shop_uuid=$1 ORDER BY name ASC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": suppliers})
	})

	app.Post("/v1/my/shops/:slug/suppliers", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Name         string `json:"name"`
			ContactEmail string `json:"contactEmail"`
			Phone        string `json:"phone"`
			Notes        string `json:"notes"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		name := strings.TrimSpace(body.Name)
		if name == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "name is required"})
		}
		contactEmail, err := normalizeOptionalEmail(body.ContactEmail)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid contact email"})
		}
		phone := strings.TrimSpace(body.Phone)
		notes := strings.TrimSpace(body.Notes)
		var supplier Supplier
		err = opts.DB.Get(&supplier, `INSERT INTO suppliers (shop_uuid, name, contact_email, phone, notes)
                                      VALUES ($1,$2,$3,NULLIF($4,''),$5)
                                      RETURNING uuid, shop_uuid, name, contact_email, phone, notes, created_at, updated_at`,
			shop.UUID, name, contactEmail, phone, notes)
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				return c.Status(409).JSON(fiber.Map{"success": false, "message": "supplier already exists"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": supplier})
	})

	app.Get("/v1/my/shops/:slug/purchase-orders", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		status := strings.TrimSpace(strings.ToLower(c.Query("status")))
		query := `SELECT po.uuid, po.shop_uuid, po.supplier_uuid, po.status, po.expected_at, po.notes, po.created_by, po.created_at, po.updated_at,
                         s.name AS supplier_name, s.contact_email AS supplier_email, s.phone AS supplier_phone
                  FROM purchase_orders po
                  LEFT JOIN suppliers s ON s.uuid = po.supplier_uuid
                  WHERE po.shop_uuid=$1`
		args := []any{shop.UUID}
		switch status {
		case "draft", "pending", "submitted", "partial", "received", "cancelled":
			query += " AND po.status=$2"
			args = append(args, status)
		}
		query += " ORDER BY po.created_at DESC LIMIT 200"
		var orders []PurchaseOrder
		if err := opts.DB.Select(&orders, query, args...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if len(orders) == 0 {
			return c.JSON(fiber.Map{"success": true, "data": orders})
		}
		orderIDs := make([]uuid.UUID, 0, len(orders))
		index := make(map[uuid.UUID]*PurchaseOrder, len(orders))
		for i := range orders {
			orderIDs = append(orderIDs, orders[i].UUID)
			index[orders[i].UUID] = &orders[i]
		}
		query, params, err := sqlx.In(`SELECT poi.purchase_order_uuid, poi.product_uuid, poi.quantity, poi.cost_cents, poi.received_quantity,
                                              p.title AS product_title
                                       FROM purchase_order_items poi
                                       JOIN products p ON p.uuid = poi.product_uuid
                                       WHERE poi.purchase_order_uuid IN (?)
                                       ORDER BY p.title ASC`, orderIDs)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		query = opts.DB.Rebind(query)
		var items []PurchaseOrderItem
		if err := opts.DB.Select(&items, query, params...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		for _, item := range items {
			if order, ok := index[item.PurchaseOrderUUID]; ok {
				order.Items = append(order.Items, item)
			}
		}
		return c.JSON(fiber.Map{"success": true, "data": orders})
	})

	app.Post("/v1/my/shops/:slug/purchase-orders", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			SupplierUUID string `json:"supplierUuid"`
			SupplierName string `json:"supplierName"`
			ContactEmail string `json:"contactEmail"`
			Phone        string `json:"phone"`
			Notes        string `json:"notes"`
			ExpectedAt   string `json:"expectedAt"`
			Status       string `json:"status"`
			Items        []struct {
				ProductUUID string `json:"productUuid"`
				Quantity    int64  `json:"quantity"`
				CostCents   int64  `json:"costCents"`
			} `json:"items"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		if len(body.Items) == 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "items are required"})
		}
		status := strings.ToLower(strings.TrimSpace(body.Status))
		if status == "" {
			status = "pending"
		}
		switch status {
		case "draft", "pending", "submitted":
		default:
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid status"})
		}
		var expectedAt *time.Time
		if strings.TrimSpace(body.ExpectedAt) != "" {
			if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(body.ExpectedAt)); err == nil {
				expectedAt = &parsed
			} else {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid expectedAt"})
			}
		}
		poNotes := strings.TrimSpace(body.Notes)

		type orderItem struct {
			Product   uuid.UUID
			Quantity  int64
			CostCents int64
		}

		items := make([]orderItem, 0, len(body.Items))
		seenProducts := make(map[uuid.UUID]struct{}, len(body.Items))

		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		var supplierID *uuid.UUID
		if trimmed := strings.TrimSpace(body.SupplierUUID); trimmed != "" {
			parsed, err := uuid.Parse(trimmed)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid supplierUuid"})
			}
			var exists int
			if err := tx.Get(&exists, `SELECT COUNT(1) FROM suppliers WHERE uuid=$1 AND shop_uuid=$2`, parsed, shop.UUID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if exists == 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "supplier not found for this shop"})
			}
			supplierID = &parsed
		} else if name := strings.TrimSpace(body.SupplierName); name != "" {
			contactEmail, err := normalizeOptionalEmail(body.ContactEmail)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid contact email"})
			}
			phone := strings.TrimSpace(body.Phone)
			var inserted uuid.UUID
			if err := tx.QueryRow(`INSERT INTO suppliers (shop_uuid, name, contact_email, phone, notes)
                                    VALUES ($1,$2,$3,NULLIF($4,''),'')
                                    ON CONFLICT (shop_uuid, name)
                                    DO UPDATE SET contact_email=COALESCE(EXCLUDED.contact_email, suppliers.contact_email),
                                                  phone=COALESCE(NULLIF(EXCLUDED.phone,''), suppliers.phone),
                                                  updated_at=now()
                                    RETURNING uuid`,
				shop.UUID, name, contactEmail, phone).Scan(&inserted); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			supplierID = &inserted
		}

		for _, raw := range body.Items {
			productID, err := uuid.Parse(strings.TrimSpace(raw.ProductUUID))
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid productUuid"})
			}
			if _, exists := seenProducts[productID]; exists {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "duplicate product in items"})
			}
			if raw.Quantity <= 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "quantity must be greater than zero"})
			}
			if raw.CostCents < 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "costCents cannot be negative"})
			}
			var productShop uuid.UUID
			if err := tx.Get(&productShop, `SELECT shop_uuid FROM products WHERE uuid=$1`, productID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "product not found"})
				}
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if productShop != shop.UUID {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "product does not belong to this shop"})
			}
			seenProducts[productID] = struct{}{}
			items = append(items, orderItem{Product: productID, Quantity: raw.Quantity, CostCents: raw.CostCents})
		}

		var createdBy any
		if userID := strings.TrimSpace(srvAuth.UserID(c)); userID != "" {
			if parsed, err := uuid.Parse(userID); err == nil {
				createdBy = parsed
			}
		}

		poID := uuid.New()
		if _, err := tx.Exec(`INSERT INTO purchase_orders (uuid, shop_uuid, supplier_uuid, status, expected_at, notes, created_by)
                              VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			poID, shop.UUID, supplierID, status, expectedAt, poNotes, createdBy); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		for _, item := range items {
			if _, err := tx.Exec(`INSERT INTO purchase_order_items (purchase_order_uuid, product_uuid, quantity, cost_cents)
                                  VALUES ($1,$2,$3,$4)`,
				poID, item.Product, item.Quantity, item.CostCents); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		order, err := fetchPurchaseOrderWithItems(opts.DB, shop.UUID, poID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": order})
	})

	app.Post("/v1/my/shops/:slug/purchase-orders/:id/receive", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		poID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid purchase order id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			LocationUUID string `json:"locationUuid"`
			Note         string `json:"note"`
			Items        []struct {
				ProductUUID string `json:"productUuid"`
				Quantity    int64  `json:"quantity"`
			} `json:"items"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		if len(body.Items) == 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "items are required"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		var locationID *uuid.UUID
		if strings.TrimSpace(body.LocationUUID) != "" {
			locPtr, err := parseOptionalUUID(body.LocationUUID)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid locationUuid"})
			}
			var exists int
			if err := tx.Get(&exists, `SELECT COUNT(1) FROM inventory_locations WHERE uuid=$1 AND shop_uuid=$2`, *locPtr, shop.UUID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if exists == 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "location not found for this shop"})
			}
			locationID = locPtr
		}

		var orderRow struct {
			ShopUUID uuid.UUID `db:"shop_uuid"`
			Status   string    `db:"status"`
		}
		if err := tx.Get(&orderRow, `SELECT shop_uuid, status FROM purchase_orders WHERE uuid=$1 FOR UPDATE`, poID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if orderRow.ShopUUID != shop.UUID {
			return c.Status(403).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		if strings.EqualFold(orderRow.Status, "cancelled") {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "purchase order is cancelled"})
		}

		note := strings.TrimSpace(body.Note)
		productsToSync := make(map[uuid.UUID]struct{}, len(body.Items))

		for _, raw := range body.Items {
			productID, err := uuid.Parse(strings.TrimSpace(raw.ProductUUID))
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid productUuid"})
			}
			if raw.Quantity <= 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "quantity must be greater than zero"})
			}
			var itemRow struct {
				Quantity         int64 `db:"quantity"`
				ReceivedQuantity int64 `db:"received_quantity"`
			}
			if err := tx.Get(&itemRow, `SELECT quantity, received_quantity
                                        FROM purchase_order_items
                                        WHERE purchase_order_uuid=$1 AND product_uuid=$2
                                        FOR UPDATE`, poID, productID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "product not present on purchase order"})
				}
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			remaining := itemRow.Quantity - itemRow.ReceivedQuantity
			if remaining <= 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "item already fully received"})
			}
			if raw.Quantity > remaining {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "received quantity exceeds ordered amount"})
			}
			if _, err := tx.Exec(`UPDATE purchase_order_items
                                  SET received_quantity = received_quantity + $3
                                  WHERE purchase_order_uuid=$1 AND product_uuid=$2`, poID, productID, raw.Quantity); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			level, err := ensureInventoryLevelTx(tx, shop.UUID, productID, locationID)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			newQuantity := level.Quantity + raw.Quantity
			if _, err := tx.Exec(`UPDATE inventory_levels SET quantity=$1, updated_at=now() WHERE uuid=$2`, newQuantity, level.UUID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if _, err := recordInventoryAdjustmentTx(tx, level, newQuantity, level.Reserved, raw.Quantity, 0, srvAuth.UserID(c), "purchase_order_receive", "purchase_order_receive", note); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			level.Quantity = newQuantity
			if err := syncLowStockAlertTx(tx, level, note); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			productsToSync[productID] = struct{}{}
		}

		var remaining int
		if err := tx.Get(&remaining, `SELECT COUNT(1) FROM purchase_order_items WHERE purchase_order_uuid=$1 AND quantity > received_quantity`, poID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		newStatus := "partial"
		if remaining == 0 {
			newStatus = "received"
		}
		if _, err := tx.Exec(`UPDATE purchase_orders SET status=$1, updated_at=now() WHERE uuid=$2`, newStatus, poID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		for productID := range productsToSync {
			if err := syncProductStockFromInventory(opts.DB, productID); err != nil {
				log.Printf("inventory sync error: %v", err)
			}
		}

		order, err := fetchPurchaseOrderWithItems(opts.DB, shop.UUID, poID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": order})
	})

	app.Get("/v1/my/shops/:slug/transfers", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var transfers []Transfer
		if err := opts.DB.Select(&transfers, `SELECT t.uuid, t.shop_uuid, t.source_location_uuid, t.destination_location_uuid, t.status, t.notes, t.created_by, t.created_at, t.updated_at,
                                                       src.name AS source_location_name, src.code AS source_location_code,
                                                       dst.name AS destination_location_name, dst.code AS destination_location_code
                                                FROM transfers t
                                                LEFT JOIN inventory_locations src ON src.uuid = t.source_location_uuid
                                                LEFT JOIN inventory_locations dst ON dst.uuid = t.destination_location_uuid
                                                WHERE t.shop_uuid=$1
                                                ORDER BY t.created_at DESC
                                                LIMIT 200`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if len(transfers) == 0 {
			return c.JSON(fiber.Map{"success": true, "data": transfers})
		}
		ids := make([]uuid.UUID, 0, len(transfers))
		index := make(map[uuid.UUID]*Transfer, len(transfers))
		for i := range transfers {
			ids = append(ids, transfers[i].UUID)
			index[transfers[i].UUID] = &transfers[i]
		}
		query, params, err := sqlx.In(`SELECT ti.transfer_uuid, ti.product_uuid, ti.quantity, p.title AS product_title
                                      FROM transfer_items ti
                                      JOIN products p ON p.uuid = ti.product_uuid
                                      WHERE ti.transfer_uuid IN (?)
                                      ORDER BY p.title ASC`, ids)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		query = opts.DB.Rebind(query)
		var items []TransferItem
		if err := opts.DB.Select(&items, query, params...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		for _, item := range items {
			if transfer, ok := index[item.TransferUUID]; ok {
				transfer.Items = append(transfer.Items, item)
			}
		}
		return c.JSON(fiber.Map{"success": true, "data": transfers})
	})

	app.Post("/v1/my/shops/:slug/transfers", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			SourceLocationUUID      string `json:"sourceLocationUuid"`
			DestinationLocationUUID string `json:"destinationLocationUuid"`
			Notes                   string `json:"notes"`
			Items                   []struct {
				ProductUUID string `json:"productUuid"`
				Quantity    int64  `json:"quantity"`
			} `json:"items"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		if len(body.Items) == 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "items are required"})
		}
		sourceID, err := parseOptionalUUID(body.SourceLocationUUID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid sourceLocationUuid"})
		}
		destID, err := parseOptionalUUID(body.DestinationLocationUUID)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid destinationLocationUuid"})
		}
		if (sourceID == nil && destID == nil) || (sourceID != nil && destID != nil && *sourceID == *destID) {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "source and destination must differ"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		if sourceID != nil {
			var exists int
			if err := tx.Get(&exists, `SELECT COUNT(1) FROM inventory_locations WHERE uuid=$1 AND shop_uuid=$2`, *sourceID, shop.UUID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if exists == 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "source location not found"})
			}
		}
		if destID != nil {
			var exists int
			if err := tx.Get(&exists, `SELECT COUNT(1) FROM inventory_locations WHERE uuid=$1 AND shop_uuid=$2`, *destID, shop.UUID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if exists == 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "destination location not found"})
			}
		}

		type transferItemInput struct {
			Product  uuid.UUID
			Quantity int64
		}
		items := make([]transferItemInput, 0, len(body.Items))
		seenProducts := make(map[uuid.UUID]struct{}, len(body.Items))

		for _, raw := range body.Items {
			productID, err := uuid.Parse(strings.TrimSpace(raw.ProductUUID))
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid productUuid"})
			}
			if raw.Quantity <= 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "quantity must be greater than zero"})
			}
			if _, exists := seenProducts[productID]; exists {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "duplicate product in items"})
			}
			var productShop uuid.UUID
			if err := tx.Get(&productShop, `SELECT shop_uuid FROM products WHERE uuid=$1`, productID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "product not found"})
				}
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if productShop != shop.UUID {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "product does not belong to this shop"})
			}
			seenProducts[productID] = struct{}{}
			items = append(items, transferItemInput{Product: productID, Quantity: raw.Quantity})
		}

		var createdBy any
		if userID := strings.TrimSpace(srvAuth.UserID(c)); userID != "" {
			if parsed, err := uuid.Parse(userID); err == nil {
				createdBy = parsed
			}
		}

		transferID := uuid.New()
		if _, err := tx.Exec(`INSERT INTO transfers (uuid, shop_uuid, source_location_uuid, destination_location_uuid, status, notes, created_by)
                              VALUES ($1,$2,$3,$4,'draft',$5,$6)`,
			transferID, shop.UUID, sourceID, destID, strings.TrimSpace(body.Notes), createdBy); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		for _, item := range items {
			if _, err := tx.Exec(`INSERT INTO transfer_items (transfer_uuid, product_uuid, quantity)
                                  VALUES ($1,$2,$3)`,
				transferID, item.Product, item.Quantity); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		transfer, err := fetchTransferWithItems(opts.DB, shop.UUID, transferID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": transfer})
	})

	app.Post("/v1/my/shops/:slug/transfers/:id/commit", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		transferID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid transfer id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Note string `json:"note"`
		}
		if err := c.BodyParser(&body); err != nil && err != io.EOF {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		note := strings.TrimSpace(body.Note)

		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		var transferRow struct {
			ShopUUID                uuid.UUID  `db:"shop_uuid"`
			Status                  string     `db:"status"`
			SourceLocationUUID      *uuid.UUID `db:"source_location_uuid"`
			DestinationLocationUUID *uuid.UUID `db:"destination_location_uuid"`
		}
		if err := tx.Get(&transferRow, `SELECT shop_uuid, status, source_location_uuid, destination_location_uuid
                                        FROM transfers
                                        WHERE uuid=$1 FOR UPDATE`, transferID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if transferRow.ShopUUID != shop.UUID {
			return c.Status(403).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		if strings.EqualFold(transferRow.Status, "completed") {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "transfer already completed"})
		}
		if strings.EqualFold(transferRow.Status, "cancelled") {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "transfer cancelled"})
		}
		if (transferRow.SourceLocationUUID == nil && transferRow.DestinationLocationUUID == nil) || (transferRow.SourceLocationUUID != nil && transferRow.DestinationLocationUUID != nil && *transferRow.SourceLocationUUID == *transferRow.DestinationLocationUUID) {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid transfer locations"})
		}

		var items []TransferItem
		if err := tx.Select(&items, `SELECT ti.transfer_uuid, ti.product_uuid, ti.quantity, p.title AS product_title
                                     FROM transfer_items ti
                                     JOIN products p ON p.uuid = ti.product_uuid
                                     WHERE ti.transfer_uuid=$1
                                     FOR UPDATE`, transferID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if len(items) == 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "transfer has no items"})
		}

		productsToSync := make(map[uuid.UUID]struct{}, len(items))

		for _, item := range items {
			if item.Quantity <= 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid item quantity"})
			}
			sourceLevel, err := ensureInventoryLevelTx(tx, shop.UUID, item.ProductUUID, transferRow.SourceLocationUUID)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if sourceLevel.Quantity < item.Quantity {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "insufficient quantity at source"})
			}
			newSourceQuantity := sourceLevel.Quantity - item.Quantity
			if _, err := tx.Exec(`UPDATE inventory_levels SET quantity=$1, updated_at=now() WHERE uuid=$2`, newSourceQuantity, sourceLevel.UUID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if _, err := recordInventoryAdjustmentTx(tx, sourceLevel, newSourceQuantity, sourceLevel.Reserved, -item.Quantity, 0, srvAuth.UserID(c), "transfer_out", "inventory_transfer", note); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			sourceLevel.Quantity = newSourceQuantity
			if err := syncLowStockAlertTx(tx, sourceLevel, note); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}

			destLevel, err := ensureInventoryLevelTx(tx, shop.UUID, item.ProductUUID, transferRow.DestinationLocationUUID)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			newDestQuantity := destLevel.Quantity + item.Quantity
			if _, err := tx.Exec(`UPDATE inventory_levels SET quantity=$1, updated_at=now() WHERE uuid=$2`, newDestQuantity, destLevel.UUID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if _, err := recordInventoryAdjustmentTx(tx, destLevel, newDestQuantity, destLevel.Reserved, item.Quantity, 0, srvAuth.UserID(c), "transfer_in", "inventory_transfer", note); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			destLevel.Quantity = newDestQuantity
			if err := syncLowStockAlertTx(tx, destLevel, note); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}

			productsToSync[item.ProductUUID] = struct{}{}
		}

		if _, err := tx.Exec(`UPDATE transfers SET status='completed', updated_at=now() WHERE uuid=$1`, transferID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		for productID := range productsToSync {
			if err := syncProductStockFromInventory(opts.DB, productID); err != nil {
				log.Printf("inventory sync error: %v", err)
			}
		}

		transfer, err := fetchTransferWithItems(opts.DB, shop.UUID, transferID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": transfer})
	})

}
