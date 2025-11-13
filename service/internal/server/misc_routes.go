package server

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	srvAuth "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func registerMiscRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
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
}
