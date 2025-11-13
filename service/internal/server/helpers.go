package server

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/mail"
	"sort"
	"strconv"
	"strings"
	"time"

	srvAuth "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	pq "github.com/lib/pq"
)

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

type cartPricingRow struct {
	ProductUUID uuid.UUID `db:"product_uuid"`
	Quantity    int       `db:"quantity"`
	PriceCents  int64     `db:"price_cents"`
	Currency    string    `db:"currency"`
	ShopUUID    uuid.UUID `db:"shop_uuid"`
}

type Order struct {
	UUID                uuid.UUID  `db:"uuid" json:"uuid"`
	ShopUUID            *uuid.UUID `db:"shop_uuid" json:"shopUuid,omitempty"`
	SubtotalCents       int64      `db:"subtotal_cents" json:"subtotalCents"`
	TotalCents          int64      `db:"total_cents" json:"totalCents"`
	Currency            string     `db:"currency" json:"currency"`
	Status              string     `db:"status" json:"status"`
	DiscountCode        *string    `db:"discount_code" json:"discountCode,omitempty"`
	DiscountAmountCents int64      `db:"discount_amount_cents" json:"discountAmountCents"`
	GiftCardCode        *string    `db:"gift_card_code" json:"giftCardCode,omitempty"`
	GiftCardAmountCents int64      `db:"gift_card_amount_cents" json:"giftCardAmountCents"`
	CreatedAt           time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt           time.Time  `db:"updated_at" json:"updatedAt"`
	TrackingNumber      *string    `db:"tracking_number" json:"trackingNumber,omitempty"`
	TrackingURL         *string    `db:"tracking_url" json:"trackingUrl,omitempty"`
	ShippingCarrier     *string    `db:"shipping_carrier" json:"shippingCarrier,omitempty"`
	ShippedAt           *time.Time `db:"shipped_at" json:"shippedAt,omitempty"`
	DeliveredAt         *time.Time `db:"delivered_at" json:"deliveredAt,omitempty"`
	RefundTotalCents    int64      `db:"refund_total_cents" json:"refundTotalCents"`
	RefundedAt          *time.Time `db:"refunded_at" json:"refundedAt,omitempty"`
	CancelledAt         *time.Time `db:"cancelled_at" json:"cancelledAt,omitempty"`
	DraftSourceUUID     *uuid.UUID `db:"draft_source_uuid" json:"draftSourceUuid,omitempty"`
}

type OrderEvent struct {
	UUID      uuid.UUID       `db:"uuid" json:"uuid"`
	OrderUUID uuid.UUID       `db:"order_uuid" json:"orderUuid"`
	EventType string          `db:"event_type" json:"eventType"`
	Message   string          `db:"message" json:"message"`
	Metadata  json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedBy *uuid.UUID      `db:"created_by" json:"createdBy,omitempty"`
	CreatedAt time.Time       `db:"created_at" json:"createdAt"`
}

type OrderDraft struct {
	UUID                uuid.UUID   `db:"uuid" json:"uuid"`
	ShopUUID            uuid.UUID   `db:"shop_uuid" json:"shopUuid"`
	CustomerUUID        *uuid.UUID  `db:"customer_uuid" json:"customerUuid,omitempty"`
	CustomerEmail       *string     `db:"customer_email" json:"customerEmail,omitempty"`
	CustomerName        *string     `db:"customer_name" json:"customerName,omitempty"`
	Currency            string      `db:"currency" json:"currency"`
	SubtotalCents       int64       `db:"subtotal_cents" json:"subtotalCents"`
	DiscountCode        *string     `db:"discount_code" json:"discountCode,omitempty"`
	DiscountAmountCents int64       `db:"discount_amount_cents" json:"discountAmountCents"`
	GiftCardCode        *string     `db:"gift_card_code" json:"giftCardCode,omitempty"`
	GiftCardAmountCents int64       `db:"gift_card_amount_cents" json:"giftCardAmountCents"`
	ShippingAddress     *string     `db:"shipping_address" json:"shippingAddress,omitempty"`
	PaymentMethod       *string     `db:"payment_method" json:"paymentMethod,omitempty"`
	Notes               *string     `db:"notes" json:"notes,omitempty"`
	Status              string      `db:"status" json:"status"`
	ExpiresAt           *time.Time  `db:"expires_at" json:"expiresAt,omitempty"`
	CreatedBy           *uuid.UUID  `db:"created_by" json:"createdBy,omitempty"`
	UpdatedBy           *uuid.UUID  `db:"updated_by" json:"updatedBy,omitempty"`
	CreatedAt           time.Time   `db:"created_at" json:"createdAt"`
	UpdatedAt           time.Time   `db:"updated_at" json:"updatedAt"`
	Items               []DraftItem `db:"-" json:"items"`
}

type DraftItem struct {
	UUID           uuid.UUID  `db:"uuid" json:"uuid"`
	DraftUUID      uuid.UUID  `db:"draft_uuid" json:"draftUuid"`
	ProductUUID    uuid.UUID  `db:"product_uuid" json:"productUuid"`
	VariantUUID    *uuid.UUID `db:"variant_uuid" json:"variantUuid,omitempty"`
	SKU            *string    `db:"sku" json:"sku,omitempty"`
	Quantity       int        `db:"quantity" json:"quantity"`
	PriceCents     int64      `db:"price_cents" json:"priceCents"`
	LineTotalCents int64      `db:"line_total_cents" json:"lineTotalCents"`
	Currency       string     `db:"currency" json:"currency"`
	Title          string     `db:"title" json:"title"`
	VariantTitle   *string    `db:"variant_title" json:"variantTitle,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time  `db:"updated_at" json:"updatedAt"`
}

type manualOrderItemInput struct {
	ProductUUID uuid.UUID
	Quantity    int
}

type manualOrderPreparedItem struct {
	ProductUUID uuid.UUID
	Quantity    int
	PriceCents  int64
	Currency    string
	Title       string
}

type manualOrderOptions struct {
	DiscountCode    string
	GiftCardCode    string
	Status          string
	ShippingAddress string
	PaymentMethod   string
	DraftSourceUUID *uuid.UUID
	EventType       string
	EventMessage    string
	EventMetadata   map[string]any
	CreatedBy       *uuid.UUID
	CustomerUUID    uuid.UUID
	CustomerEmail   string
}

type manualOrderResult struct {
	OrderID             uuid.UUID
	SubtotalCents       int64
	DiscountAmountCents int64
	GiftCardAmountCents int64
	TotalCents          int64
	Currency            string
	AppliedDiscountUUID *uuid.UUID
	AppliedGiftCardUUID *uuid.UUID
	Status              string
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
                      ON CONFLICT (cart_uuid, product_uuid) DO UPDATE SET quantity=cart_items.quantity+EXCLUDED.quantity,
                                                                        added_at=now()`, cartID, pid, body.Quantity)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	_, _ = db.Exec(`UPDATE carts SET updated_at=now() WHERE uuid=$1`, cartID)
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
		_, _ = db.Exec(`UPDATE carts SET updated_at=now() WHERE user_uuid=$1`, user)
		return c.JSON(fiber.Map{"success": true})
	}
	if _, err := db.Exec(`UPDATE cart_items SET quantity=$1, added_at=now() WHERE uuid=$2`, body.Quantity, id); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	_, _ = db.Exec(`UPDATE carts SET updated_at=now() WHERE user_uuid=$1`, user)
	return c.JSON(fiber.Map{"success": true})
}

func deleteCartItem(c *fiber.Ctx, db *sqlx.DB) error {
	id := strings.TrimSpace(c.Params("id"))
	if _, err := db.Exec(`DELETE FROM cart_items WHERE uuid=$1`, id); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	_, _ = db.Exec(`UPDATE carts SET updated_at=now() WHERE user_uuid=$1`, srvAuth.UserID(c))
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
	_, _ = db.Exec(`UPDATE carts SET updated_at=now() WHERE uuid=$1`, cartID)
	return c.JSON(fiber.Map{"success": true})
}

func createOrderFromCart(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	userUUID := uuidPtrFromString(user)
	cartID, err := ensureCart(db, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}

	var body struct {
		DiscountCode string `json:"discountCode"`
		GiftCardCode string `json:"giftCardCode"`
	}
	if err := c.BodyParser(&body); err != nil && err != io.EOF {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid checkout payload"})
	}
	discountCode := strings.TrimSpace(body.DiscountCode)
	giftCardCode := strings.TrimSpace(body.GiftCardCode)

	tx, err := db.Beginx()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	defer tx.Rollback()

	items, err := fetchCartPricingRowsForUpdate(tx, cartID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	if len(items) == 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "cart empty"})
	}

	subtotal, currency, shopUUID, multiShop, err := computeCartPricingTotals(items)
	if err != nil {
		return respondWithError(c, err)
	}

	var discount *Discount
	var discountAmount int64
	var discountCodeStored *string
	if discountCode != "" {
		if shopUUID == uuid.Nil || multiShop {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "discounts require items from a single shop"})
		}
		disc, amount, err := applyDiscountInTx(tx, discountCode, shopUUID, user, subtotal)
		if err != nil {
			return respondWithError(c, err)
		}
		if disc != nil && amount > 0 {
			discount = disc
			discountAmount = amount
			code := strings.ToUpper(strings.TrimSpace(disc.Code))
			discountCodeStored = &code
		}
	}

	remaining := subtotal - discountAmount
	if remaining < 0 {
		remaining = 0
	}

	var giftCard *GiftCard
	var giftCardAmount int64
	var giftCardCodeStored *string
	if giftCardCode != "" {
		if shopUUID == uuid.Nil || multiShop {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "gift cards require items from a single shop"})
		}
		card, amount, err := applyGiftCardInTx(tx, giftCardCode, shopUUID, remaining)
		if err != nil {
			return respondWithError(c, err)
		}
		if card != nil && amount > 0 {
			giftCard = card
			giftCardAmount = amount
			if giftCardAmount > remaining {
				giftCardAmount = remaining
			}
			remaining -= giftCardAmount
			if remaining < 0 {
				remaining = 0
			}
			code := strings.ToUpper(strings.TrimSpace(card.Code))
			giftCardCodeStored = &code
		}
	}

	total := remaining

	var shopValue any
	if shopUUID != uuid.Nil && !multiShop {
		shopValue = shopUUID
	} else {
		shopValue = nil
	}

	orderID := uuid.New()
	var discountUUIDValue any
	if discount != nil {
		discountUUIDValue = discount.UUID
	}
	var giftCardUUIDValue any
	if giftCard != nil {
		giftCardUUIDValue = giftCard.UUID
	}

	if _, err := tx.Exec(`INSERT INTO orders(uuid,user_uuid,shop_uuid,subtotal_cents,total_cents,currency,status,discount_uuid,discount_code,discount_amount_cents,gift_card_uuid,gift_card_code,gift_card_amount_cents)
                          VALUES($1,$2,$3,$4,$5,$6,'pending',$7,$8,$9,$10,$11,$12)`,
		orderID,
		user,
		shopValue,
		subtotal,
		total,
		currency,
		discountUUIDValue,
		discountCodeStored,
		discountAmount,
		giftCardUUIDValue,
		giftCardCodeStored,
		giftCardAmount); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}

	for _, item := range items {
		if _, err := tx.Exec(`INSERT INTO order_items(order_uuid,product_uuid,quantity,price_cents) VALUES($1,$2,$3,$4)`,
			orderID, item.ProductUUID, item.Quantity, item.PriceCents); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
	}

	if discount != nil && discountAmount > 0 {
		if _, err := tx.Exec(`INSERT INTO discount_redemptions(uuid, discount_uuid, order_uuid, user_uuid, shop_uuid, amount_cents)
                               VALUES($1,$2,$3,$4,$5,$6)`,
			uuid.New(), discount.UUID, orderID, user, shopUUID, discountAmount); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
	}

	if giftCard != nil && giftCardAmount > 0 {
		if err := redeemGiftCard(tx, giftCard, giftCardAmount); err != nil {
			return respondWithError(c, err)
		}
	}

	creationMeta := map[string]any{
		"subtotalCents": subtotal,
		"totalCents":    total,
		"currency":      currency,
	}
	if discountAmount > 0 {
		creationMeta["discountAmountCents"] = discountAmount
	}
	if giftCardAmount > 0 {
		creationMeta["giftCardAmountCents"] = giftCardAmount
	}
	if _, err := recordOrderEvent(tx, orderID, "order.created", "Order created from checkout", userUUID, creationMeta); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "failed to record order event"})
	}

	if _, err := tx.Exec(`DELETE FROM cart_items WHERE cart_uuid=$1`, cartID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	if _, err := tx.Exec(`UPDATE carts SET updated_at=now() WHERE uuid=$1`, cartID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	if _, err := tx.Exec(`UPDATE cart_recoveries
                          SET status='converted', converted_order_uuid=$2, updated_at=now()
                          WHERE cart_uuid=$1 AND status<>'converted'`, cartID, orderID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	if err := tx.Commit(); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}

	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"uuid":                orderID,
		"subtotalCents":       subtotal,
		"discountAmountCents": discountAmount,
		"giftCardAmountCents": giftCardAmount,
		"totalCents":          total,
		"currency":            currency,
	}})
}

func fetchCartPricingRowsForUpdate(tx *sqlx.Tx, cart uuid.UUID) ([]cartPricingRow, error) {
	var rows []cartPricingRow
	if err := tx.Select(&rows, `SELECT ci.product_uuid, ci.quantity, p.price_cents, p.currency, p.shop_uuid
                                FROM cart_items ci
                                JOIN products p ON p.uuid=ci.product_uuid AND p.deleted_at IS NULL
                                WHERE ci.cart_uuid=$1
                                FOR UPDATE`, cart); err != nil {
		return nil, err
	}
	return rows, nil
}

func fetchCartPricingRows(db *sqlx.DB, cart uuid.UUID) ([]cartPricingRow, error) {
	var rows []cartPricingRow
	if err := db.Select(&rows, `SELECT ci.product_uuid, ci.quantity, p.price_cents, p.currency, p.shop_uuid
                                FROM cart_items ci
                                JOIN products p ON p.uuid=ci.product_uuid AND p.deleted_at IS NULL
                                WHERE ci.cart_uuid=$1`, cart); err != nil {
		return nil, err
	}
	return rows, nil
}

func fetchPurchaseOrderWithItems(db *sqlx.DB, shopUUID, orderUUID uuid.UUID) (PurchaseOrder, error) {
	var order PurchaseOrder
	if err := db.Get(&order, `SELECT po.uuid, po.shop_uuid, po.supplier_uuid, po.status, po.expected_at, po.notes, po.created_by, po.created_at, po.updated_at,
                                     s.name AS supplier_name, s.contact_email AS supplier_email, s.phone AS supplier_phone
                              FROM purchase_orders po
                              LEFT JOIN suppliers s ON s.uuid = po.supplier_uuid
                              WHERE po.uuid=$1 AND po.shop_uuid=$2`, orderUUID, shopUUID); err != nil {
		return PurchaseOrder{}, err
	}
	var items []PurchaseOrderItem
	if err := db.Select(&items, `SELECT poi.purchase_order_uuid, poi.product_uuid, poi.quantity, poi.cost_cents, poi.received_quantity,
                                        p.title AS product_title
                                 FROM purchase_order_items poi
                                 JOIN products p ON p.uuid = poi.product_uuid
                                 WHERE poi.purchase_order_uuid=$1
                                 ORDER BY p.title ASC`, orderUUID); err != nil {
		return PurchaseOrder{}, err
	}
	order.Items = items
	return order, nil
}

func fetchTransferWithItems(db *sqlx.DB, shopUUID, transferUUID uuid.UUID) (Transfer, error) {
	var transfer Transfer
	if err := db.Get(&transfer, `SELECT t.uuid, t.shop_uuid, t.source_location_uuid, t.destination_location_uuid, t.status, t.notes, t.created_by, t.created_at, t.updated_at,
                                       src.name AS source_location_name, src.code AS source_location_code,
                                       dst.name AS destination_location_name, dst.code AS destination_location_code
                                FROM transfers t
                                LEFT JOIN inventory_locations src ON src.uuid = t.source_location_uuid
                                LEFT JOIN inventory_locations dst ON dst.uuid = t.destination_location_uuid
                                WHERE t.uuid=$1 AND t.shop_uuid=$2`, transferUUID, shopUUID); err != nil {
		return Transfer{}, err
	}
	var items []TransferItem
	if err := db.Select(&items, `SELECT ti.transfer_uuid, ti.product_uuid, ti.quantity, p.title AS product_title
                                 FROM transfer_items ti
                                 JOIN products p ON p.uuid = ti.product_uuid
                                 WHERE ti.transfer_uuid=$1
                                 ORDER BY p.title ASC`, transferUUID); err != nil {
		return Transfer{}, err
	}
	transfer.Items = items
	return transfer, nil
}

func computeCartPricingTotals(rows []cartPricingRow) (int64, string, uuid.UUID, bool, error) {
	if len(rows) == 0 {
		return 0, "", uuid.Nil, false, fiber.NewError(fiber.StatusBadRequest, "cart empty")
	}
	currency := strings.TrimSpace(rows[0].Currency)
	var subtotal int64
	shopUUID := uuid.Nil
	multiShop := false
	for _, r := range rows {
		if strings.TrimSpace(r.Currency) != currency {
			return 0, "", uuid.Nil, false, fiber.NewError(fiber.StatusBadRequest, "mixed currency cart not supported")
		}
		subtotal += int64(r.Quantity) * r.PriceCents
		if shopUUID == uuid.Nil {
			shopUUID = r.ShopUUID
		} else if shopUUID != r.ShopUUID {
			multiShop = true
		}
	}
	return subtotal, currency, shopUUID, multiShop, nil
}

func applyDiscountInTx(tx *sqlx.Tx, code string, shop uuid.UUID, user string, subtotal int64) (*Discount, int64, error) {
	normalized := strings.TrimSpace(code)
	if normalized == "" {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest, "invalid discount code")
	}
	var discount Discount
	if err := tx.Get(&discount, `SELECT uuid, shop_uuid, name, code, description, discount_type, amount_cents, percentage, starts_at, ends_at,
                                       usage_limit_total, usage_limit_per_customer, auto_apply, status
                                 FROM discounts
                                 WHERE LOWER(code)=LOWER($1) AND shop_uuid=$2 AND status='active'
                                 FOR UPDATE`, normalized, shop); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, fiber.NewError(fiber.StatusNotFound, "discount not found")
		}
		return nil, 0, err
	}
	now := time.Now()
	if discount.StartsAt != nil && now.Before(*discount.StartsAt) {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest, "discount not yet active")
	}
	if discount.EndsAt != nil && now.After(*discount.EndsAt) {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest, "discount expired")
	}
	if discount.UsageLimitTotal != nil {
		var totalCount int
		if err := tx.Get(&totalCount, `SELECT COUNT(1) FROM discount_redemptions WHERE discount_uuid=$1`, discount.UUID); err != nil {
			return nil, 0, err
		}
		if totalCount >= *discount.UsageLimitTotal {
			return nil, 0, fiber.NewError(fiber.StatusBadRequest, "discount usage limit reached")
		}
	}
	if discount.UsageLimitPerCustomer != nil {
		var userCount int
		if err := tx.Get(&userCount, `SELECT COUNT(1) FROM discount_redemptions WHERE discount_uuid=$1 AND user_uuid=$2`, discount.UUID, user); err != nil {
			return nil, 0, err
		}
		if userCount >= *discount.UsageLimitPerCustomer {
			return nil, 0, fiber.NewError(fiber.StatusBadRequest, "discount already used by customer")
		}
	}
	var amount int64
	switch strings.ToLower(strings.TrimSpace(discount.DiscountType)) {
	case "amount":
		amount = discount.AmountCents
	case "percentage":
		amount = int64(math.Round(float64(subtotal) * discount.Percentage / 100.0))
	default:
		amount = 0
	}
	if amount < 0 {
		amount = 0
	}
	if amount > subtotal {
		amount = subtotal
	}
	return &discount, amount, nil
}

func applyGiftCardInTx(tx *sqlx.Tx, code string, shop uuid.UUID, remaining int64) (*GiftCard, int64, error) {
	normalized := strings.TrimSpace(code)
	if normalized == "" {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest, "invalid gift card code")
	}
	var card GiftCard
	if err := tx.Get(&card, `SELECT uuid, shop_uuid, code, balance_cents, original_balance_cents, currency, issued_to_email, note, status, expires_at, issued_at, redeemed_at, created_at, updated_at
                             FROM gift_cards
                             WHERE LOWER(code)=LOWER($1) AND shop_uuid=$2
                             FOR UPDATE`, normalized, shop); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, fiber.NewError(fiber.StatusNotFound, "gift card not found")
		}
		return nil, 0, err
	}
	if !strings.EqualFold(card.Status, "active") {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest, "gift card is not active")
	}
	if card.ExpiresAt != nil && time.Now().After(*card.ExpiresAt) {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest, "gift card expired")
	}
	if card.BalanceCents <= 0 {
		return nil, 0, fiber.NewError(fiber.StatusBadRequest, "gift card has no remaining balance")
	}
	amount := card.BalanceCents
	if amount > remaining {
		amount = remaining
	}
	if amount < 0 {
		amount = 0
	}
	return &card, amount, nil
}

func redeemGiftCard(tx *sqlx.Tx, card *GiftCard, amount int64) error {
	if amount <= 0 {
		return nil
	}
	if amount > card.BalanceCents {
		return fiber.NewError(fiber.StatusBadRequest, "gift card balance insufficient")
	}
	newBalance := card.BalanceCents - amount
	status := card.Status
	if newBalance == 0 {
		status = "redeemed"
	}
	if _, err := tx.Exec(`UPDATE gift_cards
                          SET balance_cents=$1, status=$2, redeemed_at=CASE WHEN $1=0 THEN now() ELSE redeemed_at END, updated_at=now()
                          WHERE uuid=$3`, newBalance, status, card.UUID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO gift_card_transactions(uuid, gift_card_uuid, change_cents, reason)
                          VALUES($1,$2,$3,$4)`, uuid.New(), card.UUID, -amount, "order redemption"); err != nil {
		return err
	}
	card.BalanceCents = newBalance
	card.Status = status
	if newBalance == 0 {
		now := time.Now()
		card.RedeemedAt = &now
	}
	return nil
}

func refundGiftCard(tx *sqlx.Tx, cardUUID uuid.UUID, amount int64) error {
	if amount <= 0 {
		return nil
	}
	var card GiftCard
	if err := tx.Get(&card, `SELECT uuid, balance_cents, original_balance_cents, status, redeemed_at
                              FROM gift_cards
                              WHERE uuid=$1
                              FOR UPDATE`, cardUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "gift card not found")
		}
		return err
	}
	maxCredit := card.OriginalBalanceCents - card.BalanceCents
	if maxCredit <= 0 {
		return nil
	}
	credit := amount
	if credit > maxCredit {
		credit = maxCredit
	}
	if credit <= 0 {
		return nil
	}
	newBalance := card.BalanceCents + credit
	status := card.Status
	if newBalance > 0 && !strings.EqualFold(status, "active") {
		status = "active"
	}
	var redeemedAt any
	if newBalance == 0 {
		redeemedAt = card.RedeemedAt
	} else {
		redeemedAt = nil
	}
	if _, err := tx.Exec(`UPDATE gift_cards
                           SET balance_cents=$1,
                               status=$2,
                               redeemed_at=$3,
                               updated_at=now()
                           WHERE uuid=$4`,
		newBalance, status, redeemedAt, cardUUID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO gift_card_transactions(uuid, gift_card_uuid, change_cents, reason)
                           VALUES($1,$2,$3,$4)`,
		uuid.New(), cardUUID, credit, "order cancellation refund"); err != nil {
		return err
	}
	return nil
}

func previewCartPricing(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	cartID, err := ensureCart(db, user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	var body struct {
		DiscountCode string `json:"discountCode"`
		GiftCardCode string `json:"giftCardCode"`
	}
	if err := c.BodyParser(&body); err != nil && err != io.EOF {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid checkout payload"})
	}
	discountCode := strings.TrimSpace(body.DiscountCode)
	giftCardCode := strings.TrimSpace(body.GiftCardCode)

	tx, err := db.Beginx()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	defer tx.Rollback()

	items, err := fetchCartPricingRowsForUpdate(tx, cartID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	if len(items) == 0 {
		return c.Status(400).JSON(fiber.Map{"success": false, "message": "cart empty"})
	}

	subtotal, currency, shopUUID, multiShop, err := computeCartPricingTotals(items)
	if err != nil {
		return respondWithError(c, err)
	}

	var discountAmount int64
	if discountCode != "" {
		if shopUUID == uuid.Nil || multiShop {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "discounts require items from a single shop"})
		}
		_, amount, err := applyDiscountInTx(tx, discountCode, shopUUID, user, subtotal)
		if err != nil {
			return respondWithError(c, err)
		}
		discountAmount = amount
	}

	remaining := subtotal - discountAmount
	if remaining < 0 {
		remaining = 0
	}

	var giftCardAmount int64
	if giftCardCode != "" {
		if shopUUID == uuid.Nil || multiShop {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "gift cards require items from a single shop"})
		}
		_, amount, err := applyGiftCardInTx(tx, giftCardCode, shopUUID, remaining)
		if err != nil {
			return respondWithError(c, err)
		}
		if amount > remaining {
			amount = remaining
		}
		giftCardAmount = amount
		remaining -= giftCardAmount
		if remaining < 0 {
			remaining = 0
		}
	}

	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"subtotalCents":       subtotal,
		"discountAmountCents": discountAmount,
		"giftCardAmountCents": giftCardAmount,
		"totalCents":          remaining,
		"currency":            currency,
	}})
}

func getDiscountReport(c *fiber.Ctx, db *sqlx.DB) error {
	slug := c.Params("slug")
	shop, role, err := ensureShopAccess(c, db, slug)
	if err != nil {
		return respondWithError(c, err)
	}
	if !teamRoleAllowsManagement(role) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
	}
	var rows []DiscountReportRow
	query := `SELECT d.uuid, d.shop_uuid, d.name, d.code, d.description, d.discount_type, d.amount_cents, d.percentage,
                     d.starts_at, d.ends_at, d.usage_limit_total, d.usage_limit_per_customer, d.auto_apply, d.status,
                     d.created_at, d.updated_at,
                     COALESCE(SUM(r.amount_cents),0) AS total_amount_cents,
                     COUNT(r.uuid) AS redemption_count,
                     COUNT(DISTINCT r.user_uuid) AS unique_customers
              FROM discounts d
              LEFT JOIN discount_redemptions r ON r.discount_uuid = d.uuid
              WHERE d.shop_uuid=$1
              GROUP BY d.uuid
              ORDER BY d.created_at DESC`
	if err := db.Select(&rows, query, shop.UUID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return c.JSON(fiber.Map{"success": true, "data": rows})
}

func getGiftCardReport(c *fiber.Ctx, db *sqlx.DB) error {
	slug := c.Params("slug")
	shop, role, err := ensureShopAccess(c, db, slug)
	if err != nil {
		return respondWithError(c, err)
	}
	if !teamRoleAllowsManagement(role) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
	}
	var summary GiftCardReportSummary
	if err := db.Get(&summary, `SELECT
                                 COUNT(*) AS total_cards,
                                 COALESCE(SUM(CASE WHEN status='active' THEN 1 ELSE 0 END),0) AS active_cards,
                                 COALESCE(SUM(CASE WHEN status='redeemed' THEN 1 ELSE 0 END),0) AS redeemed_cards,
                                 COALESCE(SUM(original_balance_cents),0) AS issued_cents,
                                 COALESCE(SUM(balance_cents),0) AS outstanding_cents,
                                 COALESCE(SUM(original_balance_cents - balance_cents),0) AS redeemed_cents
                               FROM gift_cards
                               WHERE shop_uuid=$1`, shop.UUID); err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
	}
	return c.JSON(fiber.Map{"success": true, "data": summary})
}

func listOrders(c *fiber.Ctx, db *sqlx.DB) error {
	user := srvAuth.UserID(c)
	var out []Order
	if err := db.Select(&out, `SELECT uuid,shop_uuid,subtotal_cents,total_cents,currency,status,discount_code,discount_amount_cents,gift_card_code,gift_card_amount_cents,
                                      created_at,updated_at,tracking_number,tracking_url,shipping_carrier,shipped_at,delivered_at,cancelled_at,refunded_at,refund_total_cents
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
	if err := db.Get(&o, `SELECT uuid,shop_uuid,subtotal_cents,total_cents,currency,status,discount_code,discount_amount_cents,gift_card_code,gift_card_amount_cents,
                                  created_at,updated_at,tracking_number,tracking_url,shipping_carrier,shipped_at,delivered_at,cancelled_at,refunded_at,refund_total_cents
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

func normalizeOptionalEmail(value string) (*string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil, nil
	}
	email, err := normalizeEmail(v)
	if err != nil {
		return nil, err
	}
	return &email, nil
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

func ensureInventoryLevelTx(tx *sqlx.Tx, shopUUID, productUUID uuid.UUID, locationUUID *uuid.UUID) (InventoryLevel, error) {
	var level InventoryLevel
	var err error
	if locationUUID == nil {
		err = tx.Get(&level, `SELECT uuid, shop_uuid, product_uuid, location_uuid, quantity, reserved, safety_stock, created_at, updated_at
                               FROM inventory_levels
                               WHERE product_uuid=$1 AND location_uuid IS NULL
                               FOR UPDATE`, productUUID)
	} else {
		err = tx.Get(&level, `SELECT uuid, shop_uuid, product_uuid, location_uuid, quantity, reserved, safety_stock, created_at, updated_at
                               FROM inventory_levels
                               WHERE product_uuid=$1 AND location_uuid=$2
                               FOR UPDATE`, productUUID, *locationUUID)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			levelID := uuid.New()
			var locationArg any
			if locationUUID != nil {
				locationArg = *locationUUID
			}
			if _, insertErr := tx.Exec(`INSERT INTO inventory_levels (uuid, shop_uuid, product_uuid, location_uuid, quantity, reserved, safety_stock)
                                        VALUES ($1,$2,$3,$4,0,0,0)`,
				levelID, shopUUID, productUUID, locationArg); insertErr != nil {
				return InventoryLevel{}, insertErr
			}
			if err := tx.Get(&level, `SELECT uuid, shop_uuid, product_uuid, location_uuid, quantity, reserved, safety_stock, created_at, updated_at
                                     FROM inventory_levels
                                     WHERE uuid=$1 FOR UPDATE`, levelID); err != nil {
				return InventoryLevel{}, err
			}
		} else {
			return InventoryLevel{}, err
		}
	}
	if level.ShopUUID != shopUUID {
		return InventoryLevel{}, fmt.Errorf("inventory level shop mismatch")
	}
	return level, nil
}

func recordInventoryAdjustmentTx(tx *sqlx.Tx, level InventoryLevel, resultingQuantity, resultingReserved, deltaQuantity, deltaReserved int64, userID, reason, source, note string) (InventoryAdjustment, error) {
	adjID := uuid.New()
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		trimmedReason = "manual"
	}
	trimmedSource := strings.TrimSpace(source)
	if trimmedSource == "" {
		trimmedSource = "manual"
	}
	trimmedNote := strings.TrimSpace(note)

	var createdAt time.Time
	var userPtr *uuid.UUID
	var userVal any
	if strings.TrimSpace(userID) != "" {
		if parsed, err := uuid.Parse(strings.TrimSpace(userID)); err == nil {
			userPtr = &parsed
			userVal = parsed
		}
	}
	if err := tx.QueryRow(`INSERT INTO inventory_adjustments(uuid, inventory_level_uuid, shop_uuid, product_uuid, location_uuid, user_uuid,
                                                             delta_quantity, delta_reserved, resulting_quantity, resulting_reserved, reason, note, adjustment_source)
                              VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
                              RETURNING created_at`,
		adjID, level.UUID, level.ShopUUID, level.ProductUUID, level.LocationUUID, userVal,
		deltaQuantity, deltaReserved, resultingQuantity, resultingReserved, trimmedReason, trimmedNote, trimmedSource).Scan(&createdAt); err != nil {
		return InventoryAdjustment{}, err
	}

	return InventoryAdjustment{
		UUID:               adjID,
		InventoryLevelUUID: level.UUID,
		ShopUUID:           level.ShopUUID,
		ProductUUID:        level.ProductUUID,
		LocationUUID:       level.LocationUUID,
		UserUUID:           userPtr,
		DeltaQuantity:      deltaQuantity,
		DeltaReserved:      deltaReserved,
		ResultingQuantity:  resultingQuantity,
		ResultingReserved:  resultingReserved,
		Reason:             trimmedReason,
		Note:               trimmedNote,
		AdjustmentSource:   trimmedSource,
		CreatedAt:          createdAt,
	}, nil
}

func syncLowStockAlertTx(tx *sqlx.Tx, level InventoryLevel, note string) error {
	threshold := level.SafetyStock
	if threshold <= 0 {
		_, err := tx.Exec(`UPDATE inventory_alerts
                           SET status='resolved', resolved_at=COALESCE(resolved_at, now()), quantity=$2, safety_stock=$3
                           WHERE inventory_level_uuid=$1 AND status='open'`,
			level.UUID, level.Quantity, level.SafetyStock)
		return err
	}
	if level.Quantity <= threshold {
		trimmedNote := strings.TrimSpace(note)
		_, err := tx.Exec(`INSERT INTO inventory_alerts(uuid, inventory_level_uuid, shop_uuid, product_uuid, location_uuid, quantity, safety_stock, note)
                           VALUES($1,$2,$3,$4,$5,$6,$7,$8)
                           ON CONFLICT ON CONSTRAINT inventory_alerts_open_unique_idx
                           DO UPDATE SET quantity=EXCLUDED.quantity,
                                         safety_stock=EXCLUDED.safety_stock,
                                         note=CASE WHEN EXCLUDED.note <> '' THEN EXCLUDED.note ELSE inventory_alerts.note END`,
			uuid.New(), level.UUID, level.ShopUUID, level.ProductUUID, level.LocationUUID, level.Quantity, level.SafetyStock, trimmedNote)
		return err
	}
	_, err := tx.Exec(`UPDATE inventory_alerts
                       SET status='resolved', resolved_at=COALESCE(resolved_at, now()), quantity=$2, safety_stock=$3
                       WHERE inventory_level_uuid=$1 AND status='open'`,
		level.UUID, level.Quantity, level.SafetyStock)
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

func parseCollectionRules(raw json.RawMessage) ([]collectionRule, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}
	var rules []collectionRule
	if err := json.Unmarshal(trimmed, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func ruleStringValue(raw json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return "", fmt.Errorf("string value required")
	}
	var str string
	if err := json.Unmarshal(trimmed, &str); err == nil {
		return strings.TrimSpace(str), nil
	}
	return "", fmt.Errorf("string value required")
}

func ruleNumberValue(raw json.RawMessage) (int64, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return 0, fmt.Errorf("numeric value required")
	}
	var num float64
	if err := json.Unmarshal(trimmed, &num); err == nil {
		return int64(math.Round(num)), nil
	}
	var str string
	if err := json.Unmarshal(trimmed, &str); err == nil {
		if strings.TrimSpace(str) == "" {
			return 0, fmt.Errorf("numeric value required")
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(str), 64)
		if err != nil {
			return 0, err
		}
		return int64(math.Round(parsed)), nil
	}
	return 0, fmt.Errorf("numeric value required")
}

func ruleBoolValue(raw json.RawMessage) (bool, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return false, fmt.Errorf("boolean value required")
	}
	var b bool
	if err := json.Unmarshal(trimmed, &b); err == nil {
		return b, nil
	}
	var str string
	if err := json.Unmarshal(trimmed, &str); err == nil {
		switch strings.ToLower(strings.TrimSpace(str)) {
		case "true", "1", "yes":
			return true, nil
		case "false", "0", "no":
			return false, nil
		}
	}
	return false, fmt.Errorf("boolean value required")
}

func buildCollectionRuleFilter(rules []collectionRule, baseArgs []any) (string, []any, error) {
	if len(rules) == 0 {
		return "", append([]any{}, baseArgs...), nil
	}
	args := append([]any{}, baseArgs...)
	conditions := make([]string, 0, len(rules))
	for _, rule := range rules {
		field := strings.ToLower(strings.TrimSpace(rule.Field))
		if field == "" {
			return "", nil, fmt.Errorf("rule field required")
		}
		operator := strings.ToLower(strings.TrimSpace(rule.Operator))
		if operator == "" {
			return "", nil, fmt.Errorf("rule operator required")
		}
		switch field {
		case "title":
			val, err := ruleStringValue(rule.Value)
			if err != nil {
				return "", nil, err
			}
			column := "p.title"
			switch operator {
			case "contains":
				args = append(args, "%"+strings.ToLower(val)+"%")
				conditions = append(conditions, fmt.Sprintf(" AND LOWER(%s) LIKE $%d", column, len(args)))
			case "starts_with":
				args = append(args, strings.ToLower(val)+"%")
				conditions = append(conditions, fmt.Sprintf(" AND LOWER(%s) LIKE $%d", column, len(args)))
			case "ends_with":
				args = append(args, "%"+strings.ToLower(val))
				conditions = append(conditions, fmt.Sprintf(" AND LOWER(%s) LIKE $%d", column, len(args)))
			case "equals":
				args = append(args, strings.ToLower(val))
				conditions = append(conditions, fmt.Sprintf(" AND LOWER(%s) = $%d", column, len(args)))
			default:
				return "", nil, fmt.Errorf("unsupported operator for title: %s", operator)
			}
		case "category":
			val, err := ruleStringValue(rule.Value)
			if err != nil {
				return "", nil, err
			}
			column := "p.category"
			switch operator {
			case "contains":
				args = append(args, "%"+strings.ToLower(val)+"%")
				conditions = append(conditions, fmt.Sprintf(" AND LOWER(%s) LIKE $%d", column, len(args)))
			case "equals":
				args = append(args, strings.ToLower(val))
				conditions = append(conditions, fmt.Sprintf(" AND LOWER(%s) = $%d", column, len(args)))
			default:
				return "", nil, fmt.Errorf("unsupported operator for category: %s", operator)
			}
		case "price", "price_cents":
			column := "p.price_cents"
			switch operator {
			case "gte":
				val, err := ruleNumberValue(rule.Value)
				if err != nil {
					return "", nil, err
				}
				args = append(args, val)
				conditions = append(conditions, fmt.Sprintf(" AND %s >= $%d", column, len(args)))
			case "lte":
				val, err := ruleNumberValue(rule.Value)
				if err != nil {
					return "", nil, err
				}
				args = append(args, val)
				conditions = append(conditions, fmt.Sprintf(" AND %s <= $%d", column, len(args)))
			case "gt":
				val, err := ruleNumberValue(rule.Value)
				if err != nil {
					return "", nil, err
				}
				args = append(args, val)
				conditions = append(conditions, fmt.Sprintf(" AND %s > $%d", column, len(args)))
			case "lt":
				val, err := ruleNumberValue(rule.Value)
				if err != nil {
					return "", nil, err
				}
				args = append(args, val)
				conditions = append(conditions, fmt.Sprintf(" AND %s < $%d", column, len(args)))
			case "between":
				if rule.Min == nil || rule.Max == nil {
					return "", nil, fmt.Errorf("between operator requires min and max")
				}
				minVal := int64(math.Round(*rule.Min))
				maxVal := int64(math.Round(*rule.Max))
				if minVal > maxVal {
					return "", nil, fmt.Errorf("min cannot be greater than max")
				}
				args = append(args, minVal)
				first := len(args)
				args = append(args, maxVal)
				conditions = append(conditions, fmt.Sprintf(" AND %s BETWEEN $%d AND $%d", column, first, len(args)))
			default:
				return "", nil, fmt.Errorf("unsupported operator for price: %s", operator)
			}
		case "stock":
			column := "p.stock"
			switch operator {
			case "gte":
				val, err := ruleNumberValue(rule.Value)
				if err != nil {
					return "", nil, err
				}
				args = append(args, val)
				conditions = append(conditions, fmt.Sprintf(" AND %s >= $%d", column, len(args)))
			case "lte":
				val, err := ruleNumberValue(rule.Value)
				if err != nil {
					return "", nil, err
				}
				args = append(args, val)
				conditions = append(conditions, fmt.Sprintf(" AND %s <= $%d", column, len(args)))
			case "gt":
				val, err := ruleNumberValue(rule.Value)
				if err != nil {
					return "", nil, err
				}
				args = append(args, val)
				conditions = append(conditions, fmt.Sprintf(" AND %s > $%d", column, len(args)))
			case "lt":
				val, err := ruleNumberValue(rule.Value)
				if err != nil {
					return "", nil, err
				}
				args = append(args, val)
				conditions = append(conditions, fmt.Sprintf(" AND %s < $%d", column, len(args)))
			default:
				return "", nil, fmt.Errorf("unsupported operator for stock: %s", operator)
			}
		case "published":
			val, err := ruleBoolValue(rule.Value)
			if err != nil {
				return "", nil, err
			}
			args = append(args, val)
			conditions = append(conditions, fmt.Sprintf(" AND p.published = $%d", len(args)))
		default:
			return "", nil, fmt.Errorf("unsupported rule field: %s", field)
		}
	}
	return strings.Join(conditions, ""), args, nil
}

func validateCollectionRulesForShop(shop uuid.UUID, raw json.RawMessage) error {
	rules, err := parseCollectionRules(raw)
	if err != nil {
		return err
	}
	_, _, err = buildCollectionRuleFilter(rules, []any{shop})
	return err
}

func refreshAutomaticCollection(db *sqlx.DB, collection uuid.UUID) error {
	var meta struct {
		ShopUUID    uuid.UUID       `db:"shop_uuid"`
		Rules       json.RawMessage `db:"rules"`
		IsAutomatic bool            `db:"is_automatic"`
	}
	if err := db.Get(&meta, `SELECT shop_uuid, rules, is_automatic FROM collections WHERE uuid=$1`, collection); err != nil {
		return err
	}
	if !meta.IsAutomatic {
		return nil
	}
	rules, err := parseCollectionRules(meta.Rules)
	if err != nil {
		return err
	}
	condition, args, err := buildCollectionRuleFilter(rules, []any{meta.ShopUUID})
	if err != nil {
		return err
	}
	query := `SELECT uuid FROM products WHERE shop_uuid=$1 AND deleted_at IS NULL`
	query += condition
	var productIDs []uuid.UUID
	if err := db.Select(&productIDs, query, args...); err != nil {
		return err
	}
	return setCollectionProducts(db, collection, meta.ShopUUID, productIDs)
}

func refreshAutomaticCollectionsForShop(db *sqlx.DB, shop uuid.UUID) error {
	var ids []uuid.UUID
	if err := db.Select(&ids, `SELECT uuid FROM collections WHERE shop_uuid=$1 AND is_automatic=true`, shop); err != nil {
		return err
	}
	for _, id := range ids {
		if err := refreshAutomaticCollection(db, id); err != nil {
			return err
		}
	}
	return nil
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

const (
	giftCardCodeSegments      = 4
	giftCardCodeSegmentLength = 4
)

var giftCardCodeAlphabet = []byte("ABCDEFGHJKLMNPQRSTUVWXYZ23456789")

func normalizeGiftCardCode(code string) string {
	if code == "" {
		return ""
	}
	normalized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r - ('a' - 'A')
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-':
			return '-'
		default:
			return -1
		}
	}, code)
	normalized = strings.Trim(normalized, "-")
	for strings.Contains(normalized, "--") {
		normalized = strings.ReplaceAll(normalized, "--", "-")
	}
	return normalized
}

func generateUniqueGiftCardCode(db *sqlx.DB, shopUUID uuid.UUID) (string, error) {
	for attempts := 0; attempts < 10; attempts++ {
		code, err := generateRandomGiftCardCode()
		if err != nil {
			return "", err
		}
		exists, err := giftCardCodeExists(db, shopUUID, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", errors.New("unable to generate a unique gift card code")
}

func giftCardCodeExists(db *sqlx.DB, shopUUID uuid.UUID, code string) (bool, error) {
	var exists bool
	if err := db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM gift_cards WHERE shop_uuid=$1 AND code=$2)`,
		shopUUID, code); err != nil {
		return false, err
	}
	return exists, nil
}

func generateRandomGiftCardCode() (string, error) {
	segments := make([]string, giftCardCodeSegments)
	for i := 0; i < giftCardCodeSegments; i++ {
		segment, err := generateGiftCardCodeSegment(giftCardCodeSegmentLength)
		if err != nil {
			return "", err
		}
		segments[i] = segment
	}
	return strings.Join(segments, "-"), nil
}

func generateGiftCardCodeSegment(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("gift card code segment length must be positive")
	}
	var builder strings.Builder
	builder.Grow(length)
	alphabetLength := big.NewInt(int64(len(giftCardCodeAlphabet)))
	for i := 0; i < length; i++ {
		index, err := rand.Int(rand.Reader, alphabetLength)
		if err != nil {
			return "", err
		}
		builder.WriteByte(giftCardCodeAlphabet[index.Int64()])
	}
	return builder.String(), nil
}

var allowedReturnStatuses = map[string]string{
	"requested": "requested",
	"approved":  "approved",
	"received":  "received",
	"restocked": "restocked",
	"rejected":  "rejected",
	"refunded":  "refunded",
}

var allowedOrderStatuses = map[string]string{
	"draft":              "draft",
	"pending":            "pending",
	"processing":         "processing",
	"shipped":            "shipped",
	"delivered":          "delivered",
	"cancelled":          "cancelled",
	"refunded":           "refunded",
	"partially_refunded": "partially_refunded",
}

func normalizeOrderStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if value, ok := allowedOrderStatuses[normalized]; ok {
		return value
	}
	return "pending"
}

func normalizeReturnStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	if value, ok := allowedReturnStatuses[normalized]; ok {
		return value
	}
	return "requested"
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

func attachVariants(db *sqlx.DB, products []*Product) error {
	if len(products) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(products))
	index := make(map[uuid.UUID]*Product, len(products))
	for _, product := range products {
		if product == nil {
			continue
		}
		ids = append(ids, product.UUID)
		index[product.UUID] = product
		product.Variants = product.Variants[:0]
	}
	if len(ids) == 0 {
		return nil
	}
	var variants []ProductVariant
	if err := db.Select(&variants, `SELECT uuid, product_uuid, sku, title, option_values, price_cents, compare_at_cents, stock, barcode, created_at, updated_at
                                     FROM product_variants
                                     WHERE product_uuid = ANY($1)
                                     ORDER BY created_at ASC`, pq.Array(ids)); err != nil {
		return err
	}
	for _, variant := range variants {
		if product, ok := index[variant.ProductUUID]; ok {
			product.Variants = append(product.Variants, variant)
		}
	}
	return nil
}

func attachVariantsList(db *sqlx.DB, products []Product) error {
	ptrs := make([]*Product, 0, len(products))
	for i := range products {
		ptrs = append(ptrs, &products[i])
	}
	return attachVariants(db, ptrs)
}

func attachVariantsSingle(db *sqlx.DB, product *Product) error {
	if product == nil {
		return nil
	}
	return attachVariants(db, []*Product{product})
}

func writeErrorResponse(c *fiber.Ctx, err error) error {
	if err == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "unknown error"})
	}
	if ferr, ok := err.(*fiber.Error); ok {
		return c.Status(ferr.Code).JSON(fiber.Map{"success": false, "message": ferr.Message})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "server error"})
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	if ferr, ok := err.(*fiber.Error); ok {
		return ferr.Message
	}
	return err.Error()
}

func createProductWithVariants(db *sqlx.DB, shop Shop, input ProductCreateInput) (uuid.UUID, int64, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return uuid.Nil, 0, fiber.NewError(fiber.StatusBadRequest, "title is required")
	}
	if input.PriceCents < 0 {
		return uuid.Nil, 0, fiber.NewError(fiber.StatusBadRequest, "priceCents cannot be negative")
	}
	if input.Stock < 0 {
		return uuid.Nil, 0, fiber.NewError(fiber.StatusBadRequest, "stock cannot be negative")
	}
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = "USD"
	}
	desiredSlug := strings.TrimSpace(input.Slug)
	if desiredSlug == "" {
		desiredSlug = generateProductSlug(title)
	} else {
		desiredSlug = generateProductSlug(desiredSlug)
	}
	categoryValue := strings.TrimSpace(input.Category)

	tx, err := db.Beginx()
	if err != nil {
		return uuid.Nil, 0, err
	}
	defer tx.Rollback()

	slug, err := ensureUniqueProductSlugTx(tx, shop.UUID, desiredSlug)
	if err != nil {
		return uuid.Nil, 0, err
	}

	var categoryUUID *uuid.UUID
	if trimmed := strings.TrimSpace(input.CategoryUUID); trimmed != "" {
		categoryID, err := uuid.Parse(trimmed)
		if err != nil {
			return uuid.Nil, 0, fiber.NewError(fiber.StatusBadRequest, "invalid category uuid")
		}
		var cat struct {
			UUID uuid.UUID `db:"uuid"`
			Slug string    `db:"slug"`
		}
		if err := tx.Get(&cat, `SELECT uuid, slug FROM categories WHERE uuid=$1 AND shop_uuid=$2 AND is_active=true`, categoryID, shop.UUID); err != nil {
			return uuid.Nil, 0, fiber.NewError(fiber.StatusBadRequest, "category not found for shop")
		}
		categoryUUID = &cat.UUID
		if categoryValue == "" {
			categoryValue = cat.Slug
		}
	}

	productID := uuid.New()
	images := input.Images
	if len(images) == 0 {
		images = []string{}
	}
	if _, err := tx.Exec(`INSERT INTO products(uuid,shop_uuid,title,slug,summary,price_cents,currency,stock,image_url,published,category,category_uuid,images)
                               VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		productID,
		shop.UUID,
		title,
		slug,
		strings.TrimSpace(input.Summary),
		input.PriceCents,
		currency,
		input.Stock,
		input.ImageURL,
		input.Published,
		categoryValue,
		categoryUUID,
		pqStringArray(images),
	); err != nil {
		return uuid.Nil, 0, err
	}
	if categoryUUID != nil {
		if _, err := tx.Exec(`INSERT INTO category_products(category_uuid, product_uuid, is_primary)
				VALUES($1,$2,true)
				ON CONFLICT (category_uuid, product_uuid) DO UPDATE SET is_primary=EXCLUDED.is_primary, created_at=category_products.created_at`,
			*categoryUUID, productID); err != nil {
			return uuid.Nil, 0, err
		}
	}
	totalStock, err := replaceProductVariantsTx(tx, productID, slug, title, input.PriceCents, input.Stock, input.Variants)
	if err != nil {
		return uuid.Nil, 0, err
	}
	if _, err := tx.Exec(`UPDATE products SET stock=$1, updated_at=now() WHERE uuid=$2`, totalStock, productID); err != nil {
		return uuid.Nil, 0, err
	}
	if err := tx.Commit(); err != nil {
		return uuid.Nil, 0, err
	}
	if err := upsertDefaultInventory(db, shop.UUID, productID, totalStock); err != nil {
		return uuid.Nil, 0, err
	}
	return productID, totalStock, nil
}

func ensureUniqueProductSlugTx(tx *sqlx.Tx, shop uuid.UUID, base string) (string, error) {
	slug := base
	if slug == "" {
		slug = fmt.Sprintf("product-%s", uuid.New().String())
	}
	for attempt := 0; attempt < 100; attempt++ {
		var exists int
		if err := tx.Get(&exists, `SELECT COUNT(1) FROM products WHERE shop_uuid=$1 AND slug=$2`, shop, slug); err != nil {
			return "", err
		}
		if exists == 0 {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, attempt+1)
	}
	return "", fiber.NewError(fiber.StatusBadRequest, "could not generate unique slug")
}

func generateProductSlug(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.ToLower(raw)
	var builder strings.Builder
	lastHyphen := false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastHyphen = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastHyphen = false
		case r == ' ' || r == '-' || r == '_':
			if !lastHyphen {
				builder.WriteRune('-')
				lastHyphen = true
			}
		}
	}
	slug := strings.Trim(builder.String(), "-")
	if slug == "" {
		return "product"
	}
	return slug
}

func csvReaderFromRequest(c *fiber.Ctx) (*csv.Reader, func(), error) {
	if fileHeader, err := c.FormFile("file"); err == nil && fileHeader != nil {
		f, err := fileHeader.Open()
		if err != nil {
			return nil, nil, fiber.NewError(fiber.StatusBadRequest, "could not open uploaded file")
		}
		reader := csv.NewReader(f)
		reader.FieldsPerRecord = -1
		return reader, func() { _ = f.Close() }, nil
	}
	body := c.Body()
	if len(body) == 0 {
		return nil, nil, fiber.NewError(fiber.StatusBadRequest, "empty body")
	}
	buf := bytes.NewReader(body)
	reader := csv.NewReader(buf)
	reader.FieldsPerRecord = -1
	return reader, func() {}, nil
}

func parseCSVInt(value string, fallback int64) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	num, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return num, nil
}

func parseCSVBool(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return false
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return value == "1" || value == "yes" || value == "y" || value == "true"
	}
	return b
}

func splitCSVList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func joinCSVList(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, "|")
}

func replaceProductVariantsTx(tx *sqlx.Tx, product uuid.UUID, slug string, title string, basePrice int64, fallbackStock int64, variants []VariantPayload) (int64, error) {
	if _, err := tx.Exec(`DELETE FROM product_variants WHERE product_uuid=$1`, product); err != nil {
		return 0, err
	}
	usableSeed := slug
	if usableSeed == "" {
		usableSeed = title
	}
	counter := 0
	used := make(map[string]struct{})
	totalStock := int64(0)
	defaultStock := int64(0)
	activeVariants := make([]VariantPayload, 0, len(variants))
	for _, variant := range variants {
		if variant.Deleted {
			continue
		}
		activeVariants = append(activeVariants, variant)
	}
	if fallbackStock > 0 && len(activeVariants) > 0 {
		defaultStock = fallbackStock / int64(len(activeVariants))
	}
	for i, variant := range activeVariants {
		sku := strings.ToUpper(strings.TrimSpace(variant.SKU))
		if sku == "" {
			sku = makeSKU(usableSeed, &counter, used)
		} else {
			sku = sanitizeSKUSeed(sku)
			if sku == "" {
				sku = makeSKU(usableSeed, &counter, used)
			} else {
				if _, exists := used[sku]; exists {
					sku = makeSKU(sku, &counter, used)
				} else {
					used[sku] = struct{}{}
				}
			}
		}
		name := strings.TrimSpace(variant.Title)
		if name == "" {
			name = title
		}
		optionValues, err := marshalOptionValues(variant.OptionValues)
		if err != nil {
			return 0, err
		}
		price := basePrice
		if variant.PriceCents != nil {
			price = *variant.PriceCents
		}
		var compare interface{} = nil
		if variant.CompareAtCents != nil {
			compare = *variant.CompareAtCents
		}
		stock := defaultStock
		if variant.Stock != nil {
			stock = *variant.Stock
		} else if defaultStock == 0 && fallbackStock > 0 && len(activeVariants) == 1 {
			stock = fallbackStock
		}
		// distribute remainder to last variant
		if defaultStock > 0 && variant.Stock == nil && i == len(activeVariants)-1 {
			calculated := int64(len(activeVariants)) * defaultStock
			if calculated < fallbackStock {
				stock += fallbackStock - calculated
			}
		}
		if stock < 0 {
			stock = 0
		}
		totalStock += stock
		var barcode interface{} = nil
		if strings.TrimSpace(variant.Barcode) != "" {
			barcode = strings.TrimSpace(variant.Barcode)
		}
		if _, err := tx.Exec(`INSERT INTO product_variants(uuid, product_uuid, sku, title, option_values, price_cents, compare_at_cents, stock, barcode)
                               VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			uuid.New(), product, sku, name, optionValues, price, compare, stock, barcode); err != nil {
			return 0, err
		}
	}
	if len(activeVariants) == 0 {
		sku := makeSKU(usableSeed, &counter, used)
		stock := fallbackStock
		totalStock = stock
		if _, err := tx.Exec(`INSERT INTO product_variants(uuid, product_uuid, sku, title, option_values, price_cents, compare_at_cents, stock, barcode)
                               VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			uuid.New(), product, sku, title, json.RawMessage(`{}`), basePrice, nil, stock, nil); err != nil {
			return 0, err
		}
	}
	return totalStock, nil
}

func marshalOptionValues(values map[string]any) (json.RawMessage, error) {
	if len(values) == 0 {
		return json.RawMessage(`{}`), nil
	}
	b, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func sanitizeSKUSeed(seed string) string {
	if seed == "" {
		seed = "SKU"
	}
	upper := strings.ToUpper(seed)
	var builder strings.Builder
	for _, r := range upper {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	result := builder.String()
	if result == "" {
		result = "SKU"
	}
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}

func makeSKU(seed string, counter *int, used map[string]struct{}) string {
	base := sanitizeSKUSeed(seed)
	for {
		sku := fmt.Sprintf("%s-%03d", base, *counter+1)
		*counter++
		if _, exists := used[sku]; !exists {
			used[sku] = struct{}{}
			return sku
		}
	}
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
	OrderUUID        uuid.UUID `db:"uuid"`
	ShopUUID         uuid.UUID `db:"shop_uuid"`
	ShopSlug         string    `db:"slug"`
	Status           string    `db:"status"`
	TotalCents       int64     `db:"total_cents"`
	RefundTotalCents int64     `db:"refund_total_cents"`
}

func loadOrderMeta(db *sqlx.DB, id uuid.UUID) (orderMeta, error) {
	var meta orderMeta
	if err := db.Get(&meta, `SELECT o.uuid, s.uuid AS shop_uuid, s.slug, o.status, o.total_cents, o.refund_total_cents
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

func uuidPtrFromString(value string) *uuid.UUID {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil
	}
	return &id
}

func recordOrderEvent(ext sqlx.Ext, order uuid.UUID, eventType, message string, createdBy *uuid.UUID, metadata map[string]any) (uuid.UUID, error) {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return uuid.Nil, errors.New("event type required")
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return uuid.Nil, errors.New("event message required")
	}
	var metaJSON any
	if metadata != nil {
		buf, err := json.Marshal(metadata)
		if err != nil {
			return uuid.Nil, err
		}
		metaJSON = buf
	}
	eventID := uuid.New()
	if _, err := ext.Exec(`INSERT INTO order_events(uuid, order_uuid, event_type, message, metadata, created_by)
                           VALUES($1,$2,$3,$4,$5,$6)`,
		eventID, order, eventType, message, metaJSON, createdBy); err != nil {
		return uuid.Nil, err
	}
	return eventID, nil
}

func fetchOrderEvents(db *sqlx.DB, order uuid.UUID) ([]OrderEvent, error) {
	var events []OrderEvent
	if err := db.Select(&events, `SELECT uuid, order_uuid, event_type, message, metadata, created_by, created_at
                                   FROM order_events
                                   WHERE order_uuid=$1
                                   ORDER BY created_at ASC`, order); err != nil {
		return nil, err
	}
	return events, nil
}

func fetchOrderEventByID(db *sqlx.DB, id uuid.UUID) (OrderEvent, error) {
	var event OrderEvent
	if err := db.Get(&event, `SELECT uuid, order_uuid, event_type, message, metadata, created_by, created_at
                               FROM order_events
                               WHERE uuid=$1`, id); err != nil {
		return OrderEvent{}, err
	}
	return event, nil
}

func prepareManualOrderItems(tx *sqlx.Tx, shopUUID uuid.UUID, requested []manualOrderItemInput) ([]manualOrderPreparedItem, error) {
	if len(requested) == 0 {
		return nil, fiber.NewError(fiber.StatusBadRequest, "at least one order item is required")
	}
	prepared := make([]manualOrderPreparedItem, 0, len(requested))
	currency := ""
	for idx, item := range requested {
		if item.ProductUUID == uuid.Nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("invalid product id at position %d", idx))
		}
		if item.Quantity <= 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, "item quantities must be greater than zero")
		}
		var productRow struct {
			ShopUUID uuid.UUID `db:"shop_uuid"`
			Price    int64     `db:"price_cents"`
			Currency string    `db:"currency"`
			Title    string    `db:"title"`
		}
		if err := tx.Get(&productRow, `SELECT shop_uuid, price_cents, currency, title FROM products WHERE uuid=$1 AND deleted_at IS NULL`, item.ProductUUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("product not found for item %d", idx))
			}
			return nil, err
		}
		if productRow.ShopUUID != shopUUID {
			return nil, fiber.NewError(fiber.StatusForbidden, "product does not belong to this shop")
		}
		if currency == "" {
			currency = productRow.Currency
		} else if !strings.EqualFold(currency, productRow.Currency) {
			return nil, fiber.NewError(fiber.StatusBadRequest, "all items must use the same currency")
		}
		prepared = append(prepared, manualOrderPreparedItem{
			ProductUUID: item.ProductUUID,
			Quantity:    item.Quantity,
			PriceCents:  productRow.Price,
			Currency:    productRow.Currency,
			Title:       productRow.Title,
		})
	}
	return prepared, nil
}

func createManualOrderTx(tx *sqlx.Tx, shop Shop, customerUserUUID uuid.UUID, preparedItems []manualOrderPreparedItem, opts manualOrderOptions) (manualOrderResult, error) {
	if len(preparedItems) == 0 {
		return manualOrderResult{}, fiber.NewError(fiber.StatusBadRequest, "order must include at least one item")
	}
	currency := ""
	subtotal := int64(0)
	for _, item := range preparedItems {
		if item.Quantity <= 0 {
			return manualOrderResult{}, fiber.NewError(fiber.StatusBadRequest, "item quantities must be greater than zero")
		}
		if item.PriceCents < 0 {
			return manualOrderResult{}, fiber.NewError(fiber.StatusBadRequest, "item price must be non-negative")
		}
		if currency == "" {
			currency = item.Currency
		} else if !strings.EqualFold(currency, item.Currency) {
			return manualOrderResult{}, fiber.NewError(fiber.StatusBadRequest, "all items must use the same currency")
		}
		subtotal += int64(item.Quantity) * item.PriceCents
	}
	if subtotal <= 0 {
		return manualOrderResult{}, fiber.NewError(fiber.StatusBadRequest, "order subtotal must be greater than zero")
	}

	userString := customerUserUUID.String()
	discountCode := strings.TrimSpace(opts.DiscountCode)
	var discount *Discount
	var discountAmount int64
	var discountCodeStored *string
	if discountCode != "" {
		disc, amount, err := applyDiscountInTx(tx, discountCode, shop.UUID, userString, subtotal)
		if err != nil {
			return manualOrderResult{}, err
		}
		if disc != nil && amount > 0 {
			discount = disc
			discountAmount = amount
			code := strings.ToUpper(strings.TrimSpace(disc.Code))
			discountCodeStored = &code
		}
	}

	remaining := subtotal - discountAmount
	if remaining < 0 {
		remaining = 0
	}
	giftCardCode := strings.TrimSpace(opts.GiftCardCode)
	var giftCard *GiftCard
	var giftCardAmount int64
	var giftCardCodeStored *string
	if giftCardCode != "" {
		card, amount, err := applyGiftCardInTx(tx, giftCardCode, shop.UUID, remaining)
		if err != nil {
			return manualOrderResult{}, err
		}
		if card != nil && amount > 0 {
			giftCard = card
			giftCardAmount = amount
			if giftCardAmount > remaining {
				giftCardAmount = remaining
			}
			remaining -= giftCardAmount
			if remaining < 0 {
				remaining = 0
			}
			code := strings.ToUpper(strings.TrimSpace(card.Code))
			giftCardCodeStored = &code
		}
	}

	total := remaining
	status := normalizeOrderStatus(opts.Status)

	orderID := uuid.New()
	var discountUUIDValue any
	if discount != nil {
		discountUUIDValue = discount.UUID
	}
	var giftCardUUIDValue any
	if giftCard != nil {
		giftCardUUIDValue = giftCard.UUID
	}
	shippingAddress := strings.TrimSpace(opts.ShippingAddress)
	var shippingAddressValue any
	if shippingAddress != "" {
		shippingAddressValue = shippingAddress
	}
	paymentMethod := strings.TrimSpace(opts.PaymentMethod)
	var paymentMethodValue any
	if paymentMethod != "" {
		paymentMethodValue = paymentMethod
	}
	var draftSourceValue any
	if opts.DraftSourceUUID != nil && *opts.DraftSourceUUID != uuid.Nil {
		draftSourceValue = *opts.DraftSourceUUID
	}

	if _, err := tx.Exec(`INSERT INTO orders(uuid,
                                            user_uuid,
                                            shop_uuid,
                                            subtotal_cents,
                                            total_cents,
                                            currency,
                                            status,
                                            discount_uuid,
                                            discount_code,
                                            discount_amount_cents,
                                            gift_card_uuid,
                                            gift_card_code,
                                            gift_card_amount_cents,
                                            shipping_address,
                                            payment_method,
                                            draft_source_uuid)
                          VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		orderID,
		customerUserUUID,
		shop.UUID,
		subtotal,
		total,
		currency,
		status,
		discountUUIDValue,
		discountCodeStored,
		discountAmount,
		giftCardUUIDValue,
		giftCardCodeStored,
		giftCardAmount,
		shippingAddressValue,
		paymentMethodValue,
		draftSourceValue); err != nil {
		return manualOrderResult{}, err
	}

	for _, item := range preparedItems {
		if _, err := tx.Exec(`INSERT INTO order_items(order_uuid, product_uuid, quantity, price_cents) VALUES($1,$2,$3,$4)`,
			orderID, item.ProductUUID, item.Quantity, item.PriceCents); err != nil {
			return manualOrderResult{}, err
		}
	}

	if discount != nil && discountAmount > 0 {
		if _, err := tx.Exec(`INSERT INTO discount_redemptions(uuid, discount_uuid, order_uuid, user_uuid, shop_uuid, amount_cents)
                               VALUES($1,$2,$3,$4,$5,$6)`,
			uuid.New(), discount.UUID, orderID, customerUserUUID, shop.UUID, discountAmount); err != nil {
			return manualOrderResult{}, err
		}
	}

	if giftCard != nil && giftCardAmount > 0 {
		if err := redeemGiftCard(tx, giftCard, giftCardAmount); err != nil {
			return manualOrderResult{}, err
		}
	}

	eventType := opts.EventType
	if eventType == "" {
		eventType = "order.created"
	}
	eventMessage := opts.EventMessage
	if strings.TrimSpace(eventMessage) == "" {
		eventMessage = "Order created manually"
	}
	eventMeta := map[string]any{
		"subtotalCents": subtotal,
		"totalCents":    total,
		"currency":      currency,
	}
	if discountAmount > 0 {
		eventMeta["discountAmountCents"] = discountAmount
	}
	if giftCardAmount > 0 {
		eventMeta["giftCardAmountCents"] = giftCardAmount
	}
	if opts.DraftSourceUUID != nil && *opts.DraftSourceUUID != uuid.Nil {
		eventMeta["draftSourceUuid"] = opts.DraftSourceUUID.String()
	}
	if opts.CustomerUUID != uuid.Nil {
		eventMeta["customerUuid"] = opts.CustomerUUID.String()
	}
	if strings.TrimSpace(opts.CustomerEmail) != "" {
		eventMeta["customerEmail"] = strings.TrimSpace(opts.CustomerEmail)
	}
	for key, value := range opts.EventMetadata {
		eventMeta[key] = value
	}
	if _, err := recordOrderEvent(tx, orderID, eventType, eventMessage, opts.CreatedBy, eventMeta); err != nil {
		return manualOrderResult{}, err
	}

	return manualOrderResult{
		OrderID:             orderID,
		SubtotalCents:       subtotal,
		DiscountAmountCents: discountAmount,
		GiftCardAmountCents: giftCardAmount,
		TotalCents:          total,
		Currency:            currency,
		AppliedDiscountUUID: func() *uuid.UUID {
			if discount != nil {
				return &discount.UUID
			}
			return nil
		}(),
		AppliedGiftCardUUID: func() *uuid.UUID {
			if giftCard != nil {
				return &giftCard.UUID
			}
			return nil
		}(),
		Status: status,
	}, nil
}

func loadDraftByUUID(q sqlx.Queryer, draftID uuid.UUID) (OrderDraft, error) {
	var draft OrderDraft
	if err := sqlx.Get(q, &draft, `SELECT uuid,
                                           shop_uuid,
                                           customer_uuid,
                                           customer_email,
                                           customer_name,
                                           currency,
                                           subtotal_cents,
                                           discount_code,
                                           discount_amount_cents,
                                           gift_card_code,
                                           gift_card_amount_cents,
                                           shipping_address,
                                           payment_method,
                                           notes,
                                           status,
                                           expires_at,
                                           created_by,
                                           updated_by,
                                           created_at,
                                           updated_at
                                    FROM order_drafts
                                    WHERE uuid=$1`, draftID); err != nil {
		return OrderDraft{}, err
	}
	itemsMap, err := loadDraftItems(q, []uuid.UUID{draftID})
	if err != nil {
		return OrderDraft{}, err
	}
	if items, ok := itemsMap[draftID]; ok {
		draft.Items = items
	} else {
		draft.Items = []DraftItem{}
	}
	return draft, nil
}

func loadDraftItems(q sqlx.Queryer, draftIDs []uuid.UUID) (map[uuid.UUID][]DraftItem, error) {
	result := make(map[uuid.UUID][]DraftItem, len(draftIDs))
	if len(draftIDs) == 0 {
		return result, nil
	}
	rows := []DraftItem{}
	if err := sqlx.Select(q, &rows, `SELECT uuid,
                                            draft_uuid,
                                            product_uuid,
                                            variant_uuid,
                                            sku,
                                            quantity,
                                            price_cents,
                                            line_total_cents,
                                            currency,
                                            title,
                                            variant_title,
                                            created_at,
                                            updated_at
                                     FROM order_draft_items
                                     WHERE draft_uuid = ANY($1)
                                     ORDER BY created_at ASC`, pq.Array(draftIDs)); err != nil {
		return nil, err
	}
	for _, draftID := range draftIDs {
		result[draftID] = []DraftItem{}
	}
	for _, item := range rows {
		result[item.DraftUUID] = append(result[item.DraftUUID], item)
	}
	return result, nil
}

func loadOrderReturn(db sqlx.Queryer, returnUUID uuid.UUID) (OrderReturn, error) {
	var ret OrderReturn
	if err := sqlx.Get(db, &ret, `SELECT uuid,
                                         order_uuid,
                                         shop_uuid,
                                         customer_uuid,
                                         status,
                                         reason,
                                         notes,
                                         requested_by,
                                         processed_by,
                                         restock,
                                         restocked_at,
                                         refund_amount_cents,
                                         created_at,
                                         updated_at
                                  FROM order_returns
                                  WHERE uuid=$1`, returnUUID); err != nil {
		return OrderReturn{}, err
	}
	itemsMap, err := loadOrderReturnItems(db, []uuid.UUID{returnUUID})
	if err != nil {
		return OrderReturn{}, err
	}
	if items, ok := itemsMap[returnUUID]; ok {
		ret.Items = items
	} else {
		ret.Items = []OrderReturnItem{}
	}
	return ret, nil
}

func loadOrderReturnItems(db sqlx.Queryer, returnUUIDs []uuid.UUID) (map[uuid.UUID][]OrderReturnItem, error) {
	result := make(map[uuid.UUID][]OrderReturnItem, len(returnUUIDs))
	if len(returnUUIDs) == 0 {
		return result, nil
	}
	var rows []OrderReturnItem
	if err := sqlx.Select(db, &rows, `SELECT uuid,
                                            return_uuid,
                                            order_item_uuid,
                                            product_uuid,
                                            quantity,
                                            reason,
                                            condition,
                                            restocked_quantity,
                                            created_at,
                                            updated_at
                                     FROM order_return_items
                                     WHERE return_uuid = ANY($1)
                                     ORDER BY created_at ASC`, pq.Array(returnUUIDs)); err != nil {
		return nil, err
	}
	for _, id := range returnUUIDs {
		result[id] = []OrderReturnItem{}
	}
	for _, item := range rows {
		result[item.ReturnUUID] = append(result[item.ReturnUUID], item)
	}
	return result, nil
}

func restockReturnTx(tx *sqlx.Tx, ret OrderReturn, userID string) (OrderReturn, []uuid.UUID, error) {
	itemsMap, err := loadOrderReturnItems(tx, []uuid.UUID{ret.UUID})
	if err != nil {
		return ret, nil, err
	}
	items := itemsMap[ret.UUID]
	if len(items) == 0 {
		return ret, nil, nil
	}
	touched := make(map[uuid.UUID]struct{})
	restockedAny := false
	now := time.Now()
	for _, item := range items {
		remaining := item.Quantity - item.RestockedQuantity
		if remaining <= 0 {
			continue
		}
		level, err := ensureInventoryLevelTx(tx, ret.ShopUUID, item.ProductUUID, nil)
		if err != nil {
			return ret, nil, err
		}
		newQuantity := level.Quantity + int64(remaining)
		if _, err := tx.Exec(`UPDATE inventory_levels SET quantity=$1, updated_at=now() WHERE uuid=$2`, newQuantity, level.UUID); err != nil {
			return ret, nil, err
		}
		note := fmt.Sprintf("Return %s restock", ret.UUID.String())
		if _, err := recordInventoryAdjustmentTx(tx, level, newQuantity, level.Reserved, int64(remaining), 0, userID, "return_restock", "order_return", note); err != nil {
			return ret, nil, err
		}
		if _, err := tx.Exec(`UPDATE order_return_items SET restocked_quantity=restocked_quantity+$1, updated_at=now() WHERE uuid=$2`, remaining, item.UUID); err != nil {
			return ret, nil, err
		}
		touched[item.ProductUUID] = struct{}{}
		restockedAny = true
	}
	if restockedAny {
		if _, err := tx.Exec(`UPDATE order_returns SET restocked_at=COALESCE(restocked_at, now()), updated_at=now() WHERE uuid=$1`, ret.UUID); err != nil {
			return ret, nil, err
		}
		ret.RestockedAt = &now
	}
	updatedItemsMap, err := loadOrderReturnItems(tx, []uuid.UUID{ret.UUID})
	if err != nil {
		return ret, nil, err
	}
	ret.Items = updatedItemsMap[ret.UUID]
	touchedList := make([]uuid.UUID, 0, len(touched))
	for productID := range touched {
		touchedList = append(touchedList, productID)
	}
	return ret, touchedList, nil
}

func formatCents(amount int64) string {
	return fmt.Sprintf("$%.2f", float64(amount)/100)
}

func isValidOrderStatus(status string) bool {
	switch strings.ToLower(status) {
	case "pending", "processing", "shipped", "delivered", "cancelled", "refunded", "partially_refunded":
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
