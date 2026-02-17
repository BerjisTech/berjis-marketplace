package server

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	srvAuth "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/auth"
)

type ProductReview struct {
	UUID               uuid.UUID `db:"uuid" json:"uuid"`
	ProductUUID        uuid.UUID `db:"product_uuid" json:"productUuid"`
	UserUUID           uuid.UUID `db:"user_uuid" json:"userUuid"`
	ShopUUID           uuid.UUID `db:"shop_uuid" json:"shopUuid"`
	Rating             int       `db:"rating" json:"rating"`
	Title              string    `db:"title" json:"title"`
	Body               string    `db:"body" json:"body"`
	Status             string    `db:"status" json:"status"`
	IsVerifiedPurchase bool      `db:"is_verified_purchase" json:"isVerifiedPurchase"`
	CreatedAt          time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt          time.Time `db:"updated_at" json:"updatedAt"`
}

func registerReviewRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
	// Public: get reviews for a product
	app.Get("/v1/products/:id/reviews", func(c *fiber.Ctx) error {
		productID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid product id"})
		}

		page, _ := strconv.Atoi(c.Query("page", "1"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(c.Query("limit", "10"))
		if limit < 1 || limit > 50 {
			limit = 10
		}
		offset := (page - 1) * limit

		var reviews []ProductReview
		if err := opts.DB.Select(&reviews, `SELECT uuid, product_uuid, user_uuid, shop_uuid, rating, title, body, status, is_verified_purchase, created_at, updated_at
		                                    FROM product_reviews
		                                    WHERE product_uuid=$1 AND status='approved'
		                                    ORDER BY created_at DESC
		                                    LIMIT $2 OFFSET $3`, productID, limit, offset); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		var total int
		if err := opts.DB.Get(&total, `SELECT COUNT(*) FROM product_reviews WHERE product_uuid=$1 AND status='approved'`, productID); err != nil {
			total = 0
		}

		var avgRating float64
		if err := opts.DB.Get(&avgRating, `SELECT COALESCE(AVG(rating), 0) FROM product_reviews WHERE product_uuid=$1 AND status='approved'`, productID); err != nil {
			avgRating = 0
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"reviews":   reviews,
			"total":     total,
			"avgRating": avgRating,
			"page":      page,
			"limit":     limit,
		}})
	})

	// Authenticated: create review
	app.Post("/v1/products/:id/reviews", requireAuth, func(c *fiber.Ctx) error {
		productID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid product id"})
		}
		userID := srvAuth.UserID(c)
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"success": false, "message": "unauthorized"})
		}

		var body struct {
			Rating int    `json:"rating"`
			Title  string `json:"title"`
			Body   string `json:"body"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		if body.Rating < 1 || body.Rating > 5 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "rating must be 1-5"})
		}

		// Get product shop
		var shopUUID uuid.UUID
		if err := opts.DB.Get(&shopUUID, `SELECT shop_uuid FROM products WHERE uuid=$1 AND deleted_at IS NULL`, productID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "product not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		// Check for existing review
		var existing int
		if err := opts.DB.Get(&existing, `SELECT COUNT(*) FROM product_reviews WHERE product_uuid=$1 AND user_uuid=$2`, productID, userUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if existing > 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "you already reviewed this product"})
		}

		// Check if verified purchase
		var purchaseCount int
		if err := opts.DB.Get(&purchaseCount, `SELECT COUNT(*)
		                                       FROM orders o
		                                       JOIN order_items oi ON oi.order_uuid=o.uuid
		                                       WHERE o.user_uuid=$1 AND oi.product_uuid=$2 AND o.status NOT IN ('cancelled')`,
			userID, productID); err != nil {
			purchaseCount = 0
		}

		reviewID := uuid.New()
		if _, err := opts.DB.Exec(`INSERT INTO product_reviews(uuid, product_uuid, user_uuid, shop_uuid, rating, title, body, status, is_verified_purchase)
		                           VALUES($1,$2,$3,$4,$5,$6,$7,'pending',$8)`,
			reviewID, productID, userUUID, shopUUID, body.Rating, strings.TrimSpace(body.Title), strings.TrimSpace(body.Body), purchaseCount > 0); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": reviewID, "status": "pending"}})
	})

	// Shop owner: moderation
	app.Get("/v1/my/shops/:slug/reviews", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		status := c.Query("status", "")
		query := `SELECT uuid, product_uuid, user_uuid, shop_uuid, rating, title, body, status, is_verified_purchase, created_at, updated_at
		          FROM product_reviews WHERE shop_uuid=$1`
		args := []any{shop.UUID}
		if status != "" {
			query += ` AND status=$2`
			args = append(args, status)
		}
		query += ` ORDER BY created_at DESC LIMIT 100`

		var reviews []ProductReview
		if err := opts.DB.Select(&reviews, query, args...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true, "data": reviews})
	})

	// Approve/reject review
	app.Patch("/v1/reviews/:id", requireAuth, func(c *fiber.Ctx) error {
		reviewID, err := uuid.Parse(strings.TrimSpace(c.Params("id")))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid review id"})
		}

		var review ProductReview
		if err := opts.DB.Get(&review, `SELECT uuid, product_uuid, user_uuid, shop_uuid, rating, title, body, status, is_verified_purchase, created_at, updated_at
		                                FROM product_reviews WHERE uuid=$1`, reviewID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "review not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		// Verify shop access
		var shopSlug string
		if err := opts.DB.Get(&shopSlug, `SELECT slug FROM shops WHERE uuid=$1`, review.ShopUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		_, role, err := ensureShopAccess(c, opts.DB, shopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var body struct {
			Status string `json:"status"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		newStatus := strings.TrimSpace(body.Status)
		if newStatus != "approved" && newStatus != "rejected" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "status must be approved or rejected"})
		}

		if _, err := opts.DB.Exec(`UPDATE product_reviews SET status=$2, updated_at=now() WHERE uuid=$1`, reviewID, newStatus); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		// Update product rating cache
		if newStatus == "approved" {
			var stats struct {
				Avg   float64 `db:"avg_rating"`
				Count int64   `db:"review_count"`
			}
			if err := opts.DB.Get(&stats, `SELECT COALESCE(AVG(rating),0) AS avg_rating, COUNT(*) AS review_count
			                               FROM product_reviews WHERE product_uuid=$1 AND status='approved'`, review.ProductUUID); err == nil {
				opts.DB.Exec(`UPDATE products SET rating=$2, review_count=$3, updated_at=now() WHERE uuid=$1`, review.ProductUUID, stats.Avg, stats.Count)
			}
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": reviewID, "status": newStatus}})
	})

	// Search autocomplete
	app.Get("/v1/search/autocomplete", func(c *fiber.Ctx) error {
		q := strings.TrimSpace(c.Query("q"))
		if len(q) < 2 {
			return c.JSON(fiber.Map{"success": true, "data": []string{}})
		}

		var suggestions []struct {
			Title string `db:"title" json:"title"`
			UUID  string `db:"uuid" json:"uuid"`
			Type  string `json:"type"`
		}
		if err := opts.DB.Select(&suggestions, `SELECT uuid::text, title FROM products
		                                        WHERE deleted_at IS NULL AND published=true
		                                        AND title ILIKE '%' || $1 || '%'
		                                        ORDER BY review_count DESC, title
		                                        LIMIT 5`, q); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		for i := range suggestions {
			suggestions[i].Type = "product"
		}

		return c.JSON(fiber.Map{"success": true, "data": suggestions})
	})
}
