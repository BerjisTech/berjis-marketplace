package server

import (
	srvAuth "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"strings"
	"time"
)

func registerSearchRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
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
		if err := attachVariantsList(opts.DB, prods); err != nil {
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

}
