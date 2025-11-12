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
	"log"
	"math"
	"math/big"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"sort"
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
	UUID         uuid.UUID        `db:"uuid" json:"uuid"`
	ShopUUID     uuid.UUID        `db:"shop_uuid" json:"shopUuid"`
	Title        string           `db:"title" json:"title"`
	Slug         string           `db:"slug" json:"slug"`
	Summary      string           `db:"summary" json:"summary"`
	PriceCents   int64            `db:"price_cents" json:"priceCents"`
	Currency     string           `db:"currency" json:"currency"`
	Stock        int64            `db:"stock" json:"stock"`
	ImageURL     *string          `db:"image_url" json:"imageUrl,omitempty"`
	Category     string           `db:"category" json:"category"`
	CategoryUUID *uuid.UUID       `db:"category_uuid" json:"categoryUuid,omitempty"`
	Images       []string         `db:"images" json:"images"`
	Rating       float32          `db:"rating" json:"rating"`
	ReviewCount  int64            `db:"review_count" json:"reviewCount"`
	CreatedAt    time.Time        `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time        `db:"updated_at" json:"updatedAt"`
	DeletedAt    *time.Time       `db:"deleted_at" json:"-"`
	Published    bool             `db:"published" json:"published"`
	ShopName     string           `db:"shop_name" json:"shopName"`
	ShopSlug     string           `db:"shop_slug" json:"shopSlug"`
	Variants     []ProductVariant `db:"-" json:"variants"`
}

type ProductVariant struct {
	UUID           uuid.UUID       `db:"uuid" json:"uuid"`
	ProductUUID    uuid.UUID       `db:"product_uuid" json:"productUuid"`
	SKU            string          `db:"sku" json:"sku"`
	Title          string          `db:"title" json:"title"`
	OptionValues   json.RawMessage `db:"option_values" json:"optionValues,omitempty"`
	PriceCents     int64           `db:"price_cents" json:"priceCents"`
	CompareAtCents *int64          `db:"compare_at_cents" json:"compareAtCents,omitempty"`
	Stock          int64           `db:"stock" json:"stock"`
	Barcode        *string         `db:"barcode" json:"barcode,omitempty"`
	CreatedAt      time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time       `db:"updated_at" json:"updatedAt"`
}

type Category struct {
	UUID        uuid.UUID  `db:"uuid" json:"uuid"`
	ShopUUID    uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	ParentUUID  *uuid.UUID `db:"parent_uuid" json:"parentUuid,omitempty"`
	Name        string     `db:"name" json:"name"`
	Slug        string     `db:"slug" json:"slug"`
	Description string     `db:"description" json:"description"`
	SortOrder   int        `db:"sort_order" json:"sortOrder"`
	IsActive    bool       `db:"is_active" json:"isActive"`
	CreatedAt   time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updatedAt"`
}

type InventoryLocation struct {
	UUID        uuid.UUID `db:"uuid" json:"uuid"`
	ShopUUID    uuid.UUID `db:"shop_uuid" json:"shopUuid"`
	Name        string    `db:"name" json:"name"`
	Code        string    `db:"code" json:"code"`
	Description string    `db:"description" json:"description"`
	IsPrimary   bool      `db:"is_primary" json:"isPrimary"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

type InventoryLevel struct {
	UUID         uuid.UUID  `db:"uuid" json:"uuid"`
	ShopUUID     uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	ProductUUID  uuid.UUID  `db:"product_uuid" json:"productUuid"`
	LocationUUID *uuid.UUID `db:"location_uuid" json:"locationUuid,omitempty"`
	Quantity     int64      `db:"quantity" json:"quantity"`
	Reserved     int64      `db:"reserved" json:"reserved"`
	SafetyStock  int64      `db:"safety_stock" json:"safetyStock"`
	CreatedAt    time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updatedAt"`
}

type InventoryEntry struct {
	InventoryLevel
	ProductTitle string  `db:"product_title" json:"productTitle"`
	ProductSlug  string  `db:"product_slug" json:"productSlug"`
	LocationName *string `db:"location_name" json:"locationName,omitempty"`
	LocationCode *string `db:"location_code" json:"locationCode,omitempty"`
}

type InventoryAdjustment struct {
	UUID               uuid.UUID  `db:"uuid" json:"uuid"`
	InventoryLevelUUID uuid.UUID  `db:"inventory_level_uuid" json:"inventoryLevelUuid"`
	ShopUUID           uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	ProductUUID        uuid.UUID  `db:"product_uuid" json:"productUuid"`
	LocationUUID       *uuid.UUID `db:"location_uuid" json:"locationUuid,omitempty"`
	UserUUID           *uuid.UUID `db:"user_uuid" json:"userUuid,omitempty"`
	DeltaQuantity      int64      `db:"delta_quantity" json:"deltaQuantity"`
	DeltaReserved      int64      `db:"delta_reserved" json:"deltaReserved"`
	ResultingQuantity  int64      `db:"resulting_quantity" json:"resultingQuantity"`
	ResultingReserved  int64      `db:"resulting_reserved" json:"resultingReserved"`
	Reason             string     `db:"reason" json:"reason"`
	Note               string     `db:"note" json:"note"`
	AdjustmentSource   string     `db:"adjustment_source" json:"adjustmentSource"`
	CreatedAt          time.Time  `db:"created_at" json:"createdAt"`
}

type InventoryAlert struct {
	UUID               uuid.UUID  `db:"uuid" json:"uuid"`
	InventoryLevelUUID uuid.UUID  `db:"inventory_level_uuid" json:"inventoryLevelUuid"`
	ShopUUID           uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	ProductUUID        uuid.UUID  `db:"product_uuid" json:"productUuid"`
	LocationUUID       *uuid.UUID `db:"location_uuid" json:"locationUuid,omitempty"`
	Quantity           int64      `db:"quantity" json:"quantity"`
	SafetyStock        int64      `db:"safety_stock" json:"safetyStock"`
	Status             string     `db:"status" json:"status"`
	TriggeredAt        time.Time  `db:"triggered_at" json:"triggeredAt"`
	ResolvedAt         *time.Time `db:"resolved_at" json:"resolvedAt,omitempty"`
	Note               string     `db:"note" json:"note"`
}

type InventoryAlertEntry struct {
	InventoryAlert
	ProductTitle string  `db:"product_title" json:"productTitle"`
	ProductSlug  string  `db:"product_slug" json:"productSlug"`
	LocationName *string `db:"location_name" json:"locationName,omitempty"`
	LocationCode *string `db:"location_code" json:"locationCode,omitempty"`
}

type Collection struct {
	UUID        uuid.UUID       `db:"uuid" json:"uuid"`
	ShopUUID    uuid.UUID       `db:"shop_uuid" json:"shopUuid"`
	Title       string          `db:"title" json:"title"`
	Slug        string          `db:"slug" json:"slug"`
	Description string          `db:"description" json:"description"`
	IsAutomatic bool            `db:"is_automatic" json:"isAutomatic"`
	Rules       json.RawMessage `db:"rules" json:"rules,omitempty"`
	SortOrder   int             `db:"sort_order" json:"sortOrder"`
	IsActive    bool            `db:"is_active" json:"isActive"`
	CreatedAt   time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updatedAt"`
}

type collectionRule struct {
	Field    string          `json:"field"`
	Operator string          `json:"operator"`
	Value    json.RawMessage `json:"value"`
	Min      *float64        `json:"min"`
	Max      *float64        `json:"max"`
}

type Discount struct {
	UUID                  uuid.UUID       `db:"uuid" json:"uuid"`
	ShopUUID              uuid.UUID       `db:"shop_uuid" json:"shopUuid"`
	Name                  string          `db:"name" json:"name"`
	Code                  string          `db:"code" json:"code"`
	Description           string          `db:"description" json:"description"`
	DiscountType          string          `db:"discount_type" json:"discountType"`
	AmountCents           int64           `db:"amount_cents" json:"amountCents"`
	Percentage            float64         `db:"percentage" json:"percentage"`
	StartsAt              *time.Time      `db:"starts_at" json:"startsAt,omitempty"`
	EndsAt                *time.Time      `db:"ends_at" json:"endsAt,omitempty"`
	UsageLimitTotal       *int            `db:"usage_limit_total" json:"usageLimitTotal,omitempty"`
	UsageLimitPerCustomer *int            `db:"usage_limit_per_customer" json:"usageLimitPerCustomer,omitempty"`
	AutoApply             bool            `db:"auto_apply" json:"autoApply"`
	Status                string          `db:"status" json:"status"`
	AppliesTo             json.RawMessage `db:"applies_to" json:"appliesTo,omitempty"`
	CreatedAt             time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt             time.Time       `db:"updated_at" json:"updatedAt"`
}

type GiftCard struct {
	UUID                 uuid.UUID  `db:"uuid" json:"uuid"`
	ShopUUID             uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	Code                 string     `db:"code" json:"code"`
	BalanceCents         int64      `db:"balance_cents" json:"balanceCents"`
	OriginalBalanceCents int64      `db:"original_balance_cents" json:"originalBalanceCents"`
	Currency             string     `db:"currency" json:"currency"`
	IssuedToEmail        *string    `db:"issued_to_email" json:"issuedToEmail,omitempty"`
	Note                 string     `db:"note" json:"note"`
	Status               string     `db:"status" json:"status"`
	ExpiresAt            *time.Time `db:"expires_at" json:"expiresAt,omitempty"`
	IssuedAt             time.Time  `db:"issued_at" json:"issuedAt"`
	RedeemedAt           *time.Time `db:"redeemed_at" json:"redeemedAt,omitempty"`
	CreatedAt            time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt            time.Time  `db:"updated_at" json:"updatedAt"`
}

type GiftCardTransaction struct {
	UUID         uuid.UUID `db:"uuid" json:"uuid"`
	GiftCardUUID uuid.UUID `db:"gift_card_uuid" json:"giftCardUuid"`
	ChangeCents  int64     `db:"change_cents" json:"changeCents"`
	Reason       string    `db:"reason" json:"reason"`
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
}

type DiscountReportRow struct {
	UUID                  uuid.UUID  `db:"uuid" json:"uuid"`
	ShopUUID              uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	Name                  string     `db:"name" json:"name"`
	Code                  string     `db:"code" json:"code"`
	Description           string     `db:"description" json:"description"`
	DiscountType          string     `db:"discount_type" json:"discountType"`
	AmountCents           int64      `db:"amount_cents" json:"amountCents"`
	Percentage            float64    `db:"percentage" json:"percentage"`
	StartsAt              *time.Time `db:"starts_at" json:"startsAt,omitempty"`
	EndsAt                *time.Time `db:"ends_at" json:"endsAt,omitempty"`
	UsageLimitTotal       *int       `db:"usage_limit_total" json:"usageLimitTotal,omitempty"`
	UsageLimitPerCustomer *int       `db:"usage_limit_per_customer" json:"usageLimitPerCustomer,omitempty"`
	AutoApply             bool       `db:"auto_apply" json:"autoApply"`
	Status                string     `db:"status" json:"status"`
	CreatedAt             time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt             time.Time  `db:"updated_at" json:"updatedAt"`
	RedemptionCount       int        `db:"redemption_count" json:"redemptionCount"`
	UniqueCustomers       int        `db:"unique_customers" json:"uniqueCustomers"`
	TotalAmountCents      int64      `db:"total_amount_cents" json:"totalAmountCents"`
}

type GiftCardReportSummary struct {
	TotalCards       int   `db:"total_cards" json:"totalCards"`
	ActiveCards      int   `db:"active_cards" json:"activeCards"`
	RedeemedCards    int   `db:"redeemed_cards" json:"redeemedCards"`
	IssuedCents      int64 `db:"issued_cents" json:"issuedCents"`
	OutstandingCents int64 `db:"outstanding_cents" json:"outstandingCents"`
	RedeemedCents    int64 `db:"redeemed_cents" json:"redeemedCents"`
}

type Supplier struct {
	UUID         uuid.UUID `db:"uuid" json:"uuid"`
	ShopUUID     uuid.UUID `db:"shop_uuid" json:"shopUuid"`
	Name         string    `db:"name" json:"name"`
	ContactEmail *string   `db:"contact_email" json:"contactEmail,omitempty"`
	Phone        *string   `db:"phone" json:"phone,omitempty"`
	Notes        string    `db:"notes" json:"notes"`
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time `db:"updated_at" json:"updatedAt"`
}

type PurchaseOrder struct {
	UUID          uuid.UUID           `db:"uuid" json:"uuid"`
	ShopUUID      uuid.UUID           `db:"shop_uuid" json:"shopUuid"`
	SupplierUUID  *uuid.UUID          `db:"supplier_uuid" json:"supplierUuid,omitempty"`
	Status        string              `db:"status" json:"status"`
	ExpectedAt    *time.Time          `db:"expected_at" json:"expectedAt,omitempty"`
	Notes         string              `db:"notes" json:"notes"`
	CreatedBy     *uuid.UUID          `db:"created_by" json:"createdBy,omitempty"`
	CreatedAt     time.Time           `db:"created_at" json:"createdAt"`
	UpdatedAt     time.Time           `db:"updated_at" json:"updatedAt"`
	SupplierName  *string             `db:"supplier_name" json:"supplierName,omitempty"`
	SupplierEmail *string             `db:"supplier_email" json:"supplierEmail,omitempty"`
	SupplierPhone *string             `db:"supplier_phone" json:"supplierPhone,omitempty"`
	Items         []PurchaseOrderItem `db:"-" json:"items"`
}

type PurchaseOrderItem struct {
	PurchaseOrderUUID uuid.UUID `db:"purchase_order_uuid" json:"purchaseOrderUuid"`
	ProductUUID       uuid.UUID `db:"product_uuid" json:"productUuid"`
	ProductTitle      string    `db:"product_title" json:"productTitle"`
	Quantity          int64     `db:"quantity" json:"quantity"`
	CostCents         int64     `db:"cost_cents" json:"costCents"`
	ReceivedQuantity  int64     `db:"received_quantity" json:"receivedQuantity"`
}

type Transfer struct {
	UUID                    uuid.UUID      `db:"uuid" json:"uuid"`
	ShopUUID                uuid.UUID      `db:"shop_uuid" json:"shopUuid"`
	SourceLocationUUID      *uuid.UUID     `db:"source_location_uuid" json:"sourceLocationUuid,omitempty"`
	DestinationLocationUUID *uuid.UUID     `db:"destination_location_uuid" json:"destinationLocationUuid,omitempty"`
	Status                  string         `db:"status" json:"status"`
	Notes                   string         `db:"notes" json:"notes"`
	CreatedBy               *uuid.UUID     `db:"created_by" json:"createdBy,omitempty"`
	CreatedAt               time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt               time.Time      `db:"updated_at" json:"updatedAt"`
	SourceLocationName      *string        `db:"source_location_name" json:"sourceLocationName,omitempty"`
	SourceLocationCode      *string        `db:"source_location_code" json:"sourceLocationCode,omitempty"`
	DestinationLocationName *string        `db:"destination_location_name" json:"destinationLocationName,omitempty"`
	DestinationLocationCode *string        `db:"destination_location_code" json:"destinationLocationCode,omitempty"`
	Items                   []TransferItem `db:"-" json:"items"`
}

type TransferItem struct {
	TransferUUID uuid.UUID `db:"transfer_uuid" json:"transferUuid"`
	ProductUUID  uuid.UUID `db:"product_uuid" json:"productUuid"`
	ProductTitle string    `db:"product_title" json:"productTitle"`
	Quantity     int64     `db:"quantity" json:"quantity"`
}

type VariantPayload struct {
	UUID           string         `json:"uuid"`
	SKU            string         `json:"sku"`
	Title          string         `json:"title"`
	OptionValues   map[string]any `json:"optionValues"`
	PriceCents     *int64         `json:"priceCents"`
	CompareAtCents *int64         `json:"compareAtCents"`
	Stock          *int64         `json:"stock"`
	Barcode        string         `json:"barcode"`
	Deleted        bool           `json:"deleted"`
}

type ProductCreateInput struct {
	Title        string
	Slug         string
	Summary      string
	PriceCents   int64
	Currency     string
	Stock        int64
	ImageURL     *string
	Published    bool
	Category     string
	CategoryUUID string
	Images       []string
	Variants     []VariantPayload
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

type CustomerSummary struct {
	UUID             uuid.UUID      `db:"uuid" json:"uuid"`
	ShopUUID         uuid.UUID      `db:"shop_uuid" json:"shopUuid"`
	UserUUID         *uuid.UUID     `db:"user_uuid" json:"userUuid,omitempty"`
	Email            string         `db:"email" json:"email"`
	FirstName        string         `db:"first_name" json:"firstName"`
	LastName         string         `db:"last_name" json:"lastName"`
	Phone            string         `db:"phone" json:"phone"`
	Tags             pq.StringArray `db:"tags" json:"tags"`
	Notes            string         `db:"notes" json:"notes"`
	MarketingOptIn   bool           `db:"marketing_opt_in" json:"marketingOptIn"`
	CreatedAt        time.Time      `db:"created_at" json:"createdAt"`
	UpdatedAt        time.Time      `db:"updated_at" json:"updatedAt"`
	TotalSpentCents  int64          `db:"total_spent_cents" json:"totalSpentCents"`
	OrdersCount      int64          `db:"orders_count" json:"ordersCount"`
	LastOrderAt      *time.Time     `db:"last_order_at" json:"lastOrderAt,omitempty"`
	CustomerFullName string         `db:"customer_name" json:"customerName"`
}

type CustomerOrder struct {
	UUID            uuid.UUID  `db:"uuid" json:"uuid"`
	TotalCents      int64      `db:"total_cents" json:"totalCents"`
	Currency        string     `db:"currency" json:"currency"`
	Status          string     `db:"status" json:"status"`
	CreatedAt       time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updatedAt"`
	TrackingNumber  *string    `db:"tracking_number" json:"trackingNumber,omitempty"`
	TrackingURL     *string    `db:"tracking_url" json:"trackingUrl,omitempty"`
	ShippingCarrier *string    `db:"shipping_carrier" json:"shippingCarrier,omitempty"`
	ShippedAt       *time.Time `db:"shipped_at" json:"shippedAt,omitempty"`
	DeliveredAt     *time.Time `db:"delivered_at" json:"deliveredAt,omitempty"`
}

type ShopOrder struct {
	UUID             uuid.UUID  `db:"uuid" json:"uuid"`
	TotalCents       int64      `db:"total_cents" json:"totalCents"`
	Currency         string     `db:"currency" json:"currency"`
	Status           string     `db:"status" json:"status"`
	CreatedAt        time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt        time.Time  `db:"updated_at" json:"updatedAt"`
	TrackingNumber   *string    `db:"tracking_number" json:"trackingNumber,omitempty"`
	TrackingURL      *string    `db:"tracking_url" json:"trackingUrl,omitempty"`
	ShippingCarrier  *string    `db:"shipping_carrier" json:"shippingCarrier,omitempty"`
	ShippedAt        *time.Time `db:"shipped_at" json:"shippedAt,omitempty"`
	DeliveredAt      *time.Time `db:"delivered_at" json:"deliveredAt,omitempty"`
	RefundedAt       *time.Time `db:"refunded_at" json:"refundedAt,omitempty"`
	RefundTotalCents int64      `db:"refund_total_cents" json:"refundTotalCents"`
	CancelledAt      *time.Time `db:"cancelled_at" json:"cancelledAt,omitempty"`
	CustomerUUID     *uuid.UUID `db:"customer_uuid" json:"customerUuid,omitempty"`
	CustomerEmail    string     `db:"customer_email" json:"customerEmail"`
	CustomerName     string     `db:"customer_name" json:"customerName"`
	DraftSourceUUID  *uuid.UUID `db:"draft_source_uuid" json:"draftSourceUuid,omitempty"`
}

type AbandonedCheckout struct {
	CartUUID       uuid.UUID  `db:"cart_uuid" json:"cartUuid"`
	UserUUID       uuid.UUID  `db:"user_uuid" json:"userUuid"`
	ShopUUID       uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	CustomerUUID   *uuid.UUID `db:"customer_uuid" json:"customerUuid,omitempty"`
	CustomerEmail  string     `db:"customer_email" json:"customerEmail"`
	CustomerName   string     `db:"customer_name" json:"customerName"`
	ItemCount      int        `db:"item_count" json:"itemCount"`
	SubtotalCents  int64      `db:"subtotal_cents" json:"subtotalCents"`
	Currency       string     `db:"currency" json:"currency"`
	FirstAddedAt   *time.Time `db:"first_added_at" json:"firstAddedAt,omitempty"`
	LastAddedAt    *time.Time `db:"last_added_at" json:"lastAddedAt,omitempty"`
	LastActivityAt time.Time  `db:"last_activity_at" json:"lastActivityAt"`
}

type SalesPoint struct {
	Date       string `json:"date"`
	TotalCents int64  `json:"totalCents"`
}

type DashboardMetrics struct {
	TotalSalesCents        int64        `json:"totalSalesCents"`
	OrdersCount            int64        `json:"ordersCount"`
	AverageOrderValueCents int64        `json:"averageOrderValueCents"`
	CustomersCount         int64        `json:"customersCount"`
	SalesSeries            []SalesPoint `json:"salesSeries"`
}

type UserProfile struct {
	UserUUID       uuid.UUID `db:"user_uuid" json:"userUuid"`
	DisplayName    string    `db:"display_name" json:"displayName"`
	Email          string    `db:"email" json:"email"`
	Phone          string    `db:"phone" json:"phone"`
	AvatarURL      *string   `db:"avatar_url" json:"avatarUrl,omitempty"`
	Timezone       string    `db:"timezone" json:"timezone"`
	MarketingOptIn bool      `db:"marketing_opt_in" json:"marketingOptIn"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
}

type UserAddress struct {
	UUID              uuid.UUID `db:"uuid" json:"uuid"`
	UserUUID          uuid.UUID `db:"user_uuid" json:"userUuid"`
	Label             string    `db:"label" json:"label"`
	RecipientName     string    `db:"recipient_name" json:"recipientName"`
	Line1             string    `db:"line1" json:"line1"`
	Line2             string    `db:"line2" json:"line2"`
	City              string    `db:"city" json:"city"`
	Region            string    `db:"region" json:"region"`
	PostalCode        string    `db:"postal_code" json:"postalCode"`
	Country           string    `db:"country" json:"country"`
	Phone             string    `db:"phone" json:"phone"`
	IsDefaultShipping bool      `db:"is_default_shipping" json:"isDefaultShipping"`
	IsDefaultBilling  bool      `db:"is_default_billing" json:"isDefaultBilling"`
	CreatedAt         time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt         time.Time `db:"updated_at" json:"updatedAt"`
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

	app.Get("/v1/my/shops/:slug/discounts", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var discounts []Discount
		if err := opts.DB.Select(&discounts, `SELECT uuid, shop_uuid, name, code, description, discount_type, amount_cents, percentage, starts_at, ends_at,
                                                     usage_limit_total, usage_limit_per_customer, auto_apply, status, applies_to, created_at, updated_at
                                              FROM discounts
                                              WHERE shop_uuid=$1
                                              ORDER BY created_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": discounts})
	})
	app.Get("/v1/my/shops/:slug/discounts/report", requireAuth, func(c *fiber.Ctx) error {
		return getDiscountReport(c, opts.DB)
	})

	app.Post("/v1/my/shops/:slug/discounts", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Name                  string          `json:"name"`
			Code                  string          `json:"code"`
			Description           string          `json:"description"`
			DiscountType          string          `json:"discountType"`
			AmountCents           *int64          `json:"amountCents"`
			Percentage            *float64        `json:"percentage"`
			StartsAt              *time.Time      `json:"startsAt"`
			EndsAt                *time.Time      `json:"endsAt"`
			UsageLimitTotal       *int            `json:"usageLimitTotal"`
			UsageLimitPerCustomer *int            `json:"usageLimitPerCustomer"`
			AutoApply             bool            `json:"autoApply"`
			Status                string          `json:"status"`
			AppliesTo             json.RawMessage `json:"appliesTo"`
			ProductUUIDs          []string        `json:"productUuids"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		name := strings.TrimSpace(body.Name)
		code := strings.ToUpper(strings.TrimSpace(body.Code))
		if name == "" || code == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "name and code are required"})
		}
		discountType := strings.ToLower(strings.TrimSpace(body.DiscountType))
		if discountType == "" {
			discountType = "percentage"
		}
		if discountType != "percentage" && discountType != "amount" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount type"})
		}
		amount := int64(0)
		if body.AmountCents != nil {
			amount = *body.AmountCents
		}
		percentage := 0.0
		if body.Percentage != nil {
			percentage = *body.Percentage
		}
		if discountType == "amount" && amount <= 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents must be greater than zero for amount discounts"})
		}
		if discountType == "percentage" && percentage <= 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage must be greater than zero for percentage discounts"})
		}
		status := normalizeDiscountStatus(body.Status)
		id := uuid.New()
		if _, err := opts.DB.Exec(`INSERT INTO discounts(uuid, shop_uuid, name, code, description, discount_type, amount_cents, percentage, starts_at, ends_at,
                                               usage_limit_total, usage_limit_per_customer, auto_apply, status, applies_to)
                                    VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			id, shop.UUID, name, code, strings.TrimSpace(body.Description), discountType, amount, percentage, body.StartsAt, body.EndsAt, body.UsageLimitTotal, body.UsageLimitPerCustomer, body.AutoApply, status, nullIfEmptyJSON(body.AppliesTo)); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if len(body.ProductUUIDs) > 0 {
			productIDs, err := parseUUIDList(body.ProductUUIDs)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
			if err := setDiscountProducts(opts.DB, id, shop.UUID, productIDs); err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id}})
	})

	app.Patch("/v1/my/shops/:slug/discounts/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		discountID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var exists int
		if err := opts.DB.Get(&exists, `SELECT COUNT(1) FROM discounts WHERE uuid=$1 AND shop_uuid=$2`, discountID, shop.UUID); err != nil || exists == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		var body map[string]any
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		sets := make([]string, 0, 12)
		args := make([]any, 0, 12)
		add := func(col string, v any) { sets = append(sets, col+"=$"+itoa(len(args)+1)); args = append(args, v) }
		if v, ok := body["name"].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				add("name", s)
			}
		}
		if v, ok := body["code"].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				add("code", strings.ToUpper(s))
			}
		}
		if v, ok := body["description"].(string); ok {
			add("description", strings.TrimSpace(v))
		}
		if v, ok := body["discountType"].(string); ok {
			t := strings.ToLower(strings.TrimSpace(v))
			if t != "" && t != "percentage" && t != "amount" {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount type"})
			}
			if t != "" {
				add("discount_type", t)
			}
		}
		if raw, ok := body["amountCents"]; ok {
			switch val := raw.(type) {
			case float64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents cannot be negative"})
				}
				add("amount_cents", int64(val))
			case int:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents cannot be negative"})
				}
				add("amount_cents", int64(val))
			case int64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "amountCents cannot be negative"})
				}
				add("amount_cents", val)
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid amountCents"})
			}
		}
		if raw, ok := body["percentage"]; ok {
			switch val := raw.(type) {
			case float64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage cannot be negative"})
				}
				add("percentage", val)
			case int:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage cannot be negative"})
				}
				add("percentage", float64(val))
			case int64:
				if val < 0 {
					return c.Status(400).JSON(fiber.Map{"success": false, "message": "percentage cannot be negative"})
				}
				add("percentage", float64(val))
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid percentage"})
			}
		}
		if v, ok := body["startsAt"].(string); ok {
			if v == "" {
				add("starts_at", nil)
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				add("starts_at", t)
			} else {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid startsAt"})
			}
		}
		if v, ok := body["endsAt"].(string); ok {
			if v == "" {
				add("ends_at", nil)
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				add("ends_at", t)
			} else {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid endsAt"})
			}
		}
		if raw, ok := body["usageLimitTotal"]; ok {
			switch val := raw.(type) {
			case float64:
				add("usage_limit_total", int(val))
			case int:
				add("usage_limit_total", val)
			case int64:
				add("usage_limit_total", val)
			case nil:
				add("usage_limit_total", nil)
			}
		}
		if raw, ok := body["usageLimitPerCustomer"]; ok {
			switch val := raw.(type) {
			case float64:
				add("usage_limit_per_customer", int(val))
			case int:
				add("usage_limit_per_customer", val)
			case int64:
				add("usage_limit_per_customer", val)
			case nil:
				add("usage_limit_per_customer", nil)
			}
		}
		if v, ok := body["autoApply"].(bool); ok {
			add("auto_apply", v)
		}
		if v, ok := body["status"].(string); ok {
			add("status", normalizeDiscountStatus(v))
		}
		if raw, ok := body["appliesTo"]; ok {
			b, err := json.Marshal(raw)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid appliesTo"})
			}
			add("applies_to", nullIfEmptyJSON(b))
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
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, discountID)
			query := "UPDATE discounts SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args))
			if _, err := opts.DB.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if productUpdateIDs != nil {
			if err := setDiscountProducts(opts.DB, discountID, shop.UUID, productUpdateIDs); err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": err.Error()})
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Delete("/v1/my/shops/:slug/discounts/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		discountID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid discount id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		res, err := opts.DB.Exec(`UPDATE discounts SET status='archived', updated_at=now() WHERE uuid=$1 AND shop_uuid=$2`, discountID, shop.UUID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		if _, err := opts.DB.Exec(`DELETE FROM discount_products WHERE discount_uuid=$1`, discountID); err != nil {
			log.Printf("discount_products cleanup error: %v", err)
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Get("/v1/my/shops/:slug/gift-cards", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var cards []GiftCard
		if err := opts.DB.Select(&cards, `SELECT uuid, shop_uuid, code, balance_cents, original_balance_cents, currency, issued_to_email, note, status, expires_at, issued_at, redeemed_at, created_at, updated_at
                                         FROM gift_cards WHERE shop_uuid=$1 ORDER BY created_at DESC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": cards})
	})

	app.Post("/v1/my/shops/:slug/gift-cards", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Code         string     `json:"code"`
			BalanceCents int64      `json:"balanceCents"`
			Currency     string     `json:"currency"`
			IssuedTo     *string    `json:"issuedToEmail"`
			Note         string     `json:"note"`
			Status       string     `json:"status"`
			ExpiresAt    *time.Time `json:"expiresAt"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		code := normalizeGiftCardCode(body.Code)
		if code == "" {
			generated, err := generateUniqueGiftCardCode(opts.DB, shop.UUID)
			if err != nil {
				log.Printf("gift card code generation failed: %v", err)
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "could not generate gift card code"})
			}
			code = generated
		} else {
			exists, err := giftCardCodeExists(opts.DB, shop.UUID, code)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if exists {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{"success": false, "message": "a gift card with that code already exists"})
			}
		}
		if body.BalanceCents <= 0 {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "balance must be greater than zero"})
		}
		currency := strings.ToUpper(strings.TrimSpace(body.Currency))
		if currency == "" {
			currency = "USD"
		}
		status := normalizeGiftCardStatus(body.Status)
		id := uuid.New()
		if _, err := opts.DB.Exec(`INSERT INTO gift_cards(uuid, shop_uuid, code, balance_cents, original_balance_cents, currency, issued_to_email, note, status, expires_at)
                                    VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			id, shop.UUID, code, body.BalanceCents, body.BalanceCents, currency, body.IssuedTo, strings.TrimSpace(body.Note), status, body.ExpiresAt); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := addGiftCardTransaction(opts.DB, id, body.BalanceCents, "issued"); err != nil {
			log.Printf("gift card transaction error: %v", err)
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id, "code": code}})
	})

	app.Get("/v1/my/shops/:slug/gift-cards/report", requireAuth, func(c *fiber.Ctx) error {
		return getGiftCardReport(c, opts.DB)
	})

	app.Patch("/v1/my/shops/:slug/gift-cards/:id", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		cardID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid gift card id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var current GiftCard
		if err := opts.DB.Get(&current, `SELECT uuid, shop_uuid, code, balance_cents, original_balance_cents, currency, issued_to_email, note, status, expires_at, issued_at, redeemed_at, created_at, updated_at
                                         FROM gift_cards WHERE uuid=$1 AND shop_uuid=$2`, cardID, shop.UUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var body map[string]any
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		sets := make([]string, 0, 8)
		args := make([]any, 0, 8)
		add := func(col string, v any) { sets = append(sets, col+"=$"+itoa(len(args)+1)); args = append(args, v) }
		if v, ok := body["note"].(string); ok {
			add("note", strings.TrimSpace(v))
		}
		if v, ok := body["expiresAt"].(string); ok {
			if v == "" {
				add("expires_at", nil)
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				add("expires_at", t)
			} else {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid expiresAt"})
			}
		}
		if v, ok := body["status"].(string); ok {
			newStatus := normalizeGiftCardStatus(v)
			add("status", newStatus)
			if newStatus == "redeemed" && current.RedeemedAt == nil {
				add("redeemed_at", time.Now())
			}
		}
		if raw, ok := body["balanceAdjustmentCents"]; ok {
			var adjust int64
			switch val := raw.(type) {
			case float64:
				adjust = int64(val)
			case int:
				adjust = int64(val)
			case int64:
				adjust = val
			default:
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid balanceAdjustmentCents"})
			}
			newBalance := current.BalanceCents + adjust
			if newBalance < 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "insufficient balance"})
			}
			add("balance_cents", newBalance)
			if adjust != 0 {
				if err := addGiftCardTransaction(opts.DB, cardID, adjust, "manual_adjustment"); err != nil {
					log.Printf("gift card transaction error: %v", err)
				}
				current.BalanceCents = newBalance
			}
		}
		if email, ok := body["issuedToEmail"].(string); ok {
			if strings.TrimSpace(email) == "" {
				add("issued_to_email", nil)
			} else {
				add("issued_to_email", strings.TrimSpace(email))
			}
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, cardID)
			query := "UPDATE gift_cards SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args))
			if _, err := opts.DB.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Get("/v1/my/shops/:slug/gift-cards/:id/transactions", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		idParam := c.Params("id")
		cardID, err := uuid.Parse(strings.TrimSpace(idParam))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid gift card id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var okCount int
		if err := opts.DB.Get(&okCount, `SELECT COUNT(1) FROM gift_cards WHERE uuid=$1 AND shop_uuid=$2`, cardID, shop.UUID); err != nil || okCount == 0 {
			return c.Status(404).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		var txs []GiftCardTransaction
		if err := opts.DB.Select(&txs, `SELECT uuid, gift_card_uuid, change_cents, reason, created_at
                                        FROM gift_card_transactions
                                        WHERE gift_card_uuid=$1
                                        ORDER BY created_at DESC`, cardID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": txs})
	})

	app.Get("/v1/my/shops/:slug/customers", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		q := strings.TrimSpace(c.Query("q"))
		tag := strings.TrimSpace(c.Query("tag"))
		query := `SELECT c.uuid, c.shop_uuid, c.user_uuid, c.email, c.first_name, c.last_name, c.phone, c.tags, c.notes, c.marketing_opt_in, c.created_at, c.updated_at,
                         COALESCE(SUM(oi.price_cents * oi.quantity),0) AS total_spent_cents,
                         COUNT(DISTINCT CASE WHEN o.uuid IS NOT NULL AND p.uuid IS NOT NULL THEN o.uuid END) AS orders_count,
                         MAX(o.created_at) AS last_order_at,
                         TRIM(BOTH ' ' FROM COALESCE(c.first_name,'') || ' ' || COALESCE(c.last_name,'')) AS customer_name
                  FROM customers c
                  LEFT JOIN orders o ON c.user_uuid IS NOT NULL AND o.user_uuid=c.user_uuid
                  LEFT JOIN order_items oi ON oi.order_uuid=o.uuid
                  LEFT JOIN products p ON p.uuid=oi.product_uuid AND p.shop_uuid=c.shop_uuid
                  WHERE c.shop_uuid=$1`
		args := []any{shop.UUID}
		if q != "" {
			like := "%" + strings.ToLower(q) + "%"
			args = append(args, like)
			idx := "$" + itoa(len(args))
			query += " AND (LOWER(c.email) LIKE " + idx + " OR LOWER(c.first_name) LIKE " + idx + " OR LOWER(c.last_name) LIKE " + idx + ")"
		}
		if tag != "" {
			args = append(args, strings.ToLower(tag))
			idx := "$" + itoa(len(args))
			query += " AND EXISTS (SELECT 1 FROM unnest(c.tags) AS tag WHERE LOWER(tag)=LOWER(" + idx + "))"
		}
		query += " GROUP BY c.uuid ORDER BY last_order_at DESC NULLS LAST, c.created_at DESC LIMIT 200"
		var customers []CustomerSummary
		if err := opts.DB.Select(&customers, query, args...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": customers})
	})

	app.Post("/v1/my/shops/:slug/customers", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Email          string   `json:"email"`
			FirstName      string   `json:"firstName"`
			LastName       string   `json:"lastName"`
			Phone          string   `json:"phone"`
			Notes          string   `json:"notes"`
			Tags           []string `json:"tags"`
			MarketingOptIn bool     `json:"marketingOptIn"`
			UserUUID       string   `json:"userUuid"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		email, err := normalizeEmail(body.Email)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid email"})
		}
		first := strings.TrimSpace(body.FirstName)
		last := strings.TrimSpace(body.LastName)
		phone := strings.TrimSpace(body.Phone)
		notes := strings.TrimSpace(body.Notes)
		tags := normalizeTags(body.Tags)
		userUUID, err := parseOptionalUUID(body.UserUUID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid user uuid"})
		}
		now := time.Now()
		tempID := uuid.New()
		var userValue any
		if userUUID != nil {
			userValue = *userUUID
		}
		var created CustomerSummary
		if err := opts.DB.Get(&created, `INSERT INTO customers (uuid, shop_uuid, user_uuid, email, first_name, last_name, phone, tags, notes, marketing_opt_in, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)
			ON CONFLICT (shop_uuid, email) DO UPDATE SET
				user_uuid=COALESCE(EXCLUDED.user_uuid, customers.user_uuid),
				first_name=EXCLUDED.first_name,
				last_name=EXCLUDED.last_name,
				phone=EXCLUDED.phone,
				tags=EXCLUDED.tags,
				notes=EXCLUDED.notes,
				marketing_opt_in=EXCLUDED.marketing_opt_in,
				updated_at=now()
			RETURNING uuid, shop_uuid, user_uuid, email, first_name, last_name, phone, tags, notes, marketing_opt_in, created_at, updated_at`,
			tempID, shop.UUID, userValue, email, first, last, phone, pqStringArray(tags), notes, body.MarketingOptIn, now); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		summary, err := fetchCustomerSummary(opts.DB, created.UUID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": summary})
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

	app.Post("/v1/shops/:slug/transfer", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsTransfer(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			NewOwnerUUID string `json:"newOwnerUuid"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		newOwnerStr := strings.TrimSpace(body.NewOwnerUUID)
		if newOwnerStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "new owner uuid required"})
		}
		newOwnerUUID, err := uuid.Parse(newOwnerStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid owner uuid"})
		}
		if newOwnerUUID == shop.OwnerUUID {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "new owner matches current owner"})
		}
		var member StoreUser
		if err := opts.DB.Get(&member, `SELECT uuid, store_uuid, user_uuid, role, status, created_at, updated_at FROM store_users WHERE store_uuid=$1 AND user_uuid=$2`, shop.UUID, newOwnerUUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "target user is not a team member"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()
		if _, err := tx.Exec(`DELETE FROM store_users WHERE store_uuid=$1 AND user_uuid=$2`, shop.UUID, newOwnerUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if _, err := tx.Exec(`INSERT INTO store_users (uuid, store_uuid, user_uuid, role, status)
			VALUES ($1,$2,$3,'manager','active')
			ON CONFLICT (store_uuid, user_uuid) DO UPDATE SET role='manager', status='active', updated_at=now()`,
			uuid.New(), shop.UUID, shop.OwnerUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if _, err := tx.Exec(`UPDATE shops SET owner_uuid=$1, updated_at=now() WHERE uuid=$2`, newOwnerUUID, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"ownerUuid": newOwnerUUID}})
	})

	app.Get("/v1/customers/:id", requireAuth, func(c *fiber.Ctx) error {
		customerParam := strings.TrimSpace(c.Params("id"))
		if customerParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		customerID, err := uuid.Parse(customerParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		meta, err := loadCustomerMeta(opts.DB, customerID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		summary, err := fetchCustomerSummary(opts.DB, customerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": summary})
	})

	app.Patch("/v1/customers/:id", requireAuth, func(c *fiber.Ctx) error {
		customerParam := strings.TrimSpace(c.Params("id"))
		if customerParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		customerID, err := uuid.Parse(customerParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		meta, err := loadCustomerMeta(opts.DB, customerID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Email          *string   `json:"email"`
			FirstName      *string   `json:"firstName"`
			LastName       *string   `json:"lastName"`
			Phone          *string   `json:"phone"`
			Notes          *string   `json:"notes"`
			Tags           *[]string `json:"tags"`
			MarketingOptIn *bool     `json:"marketingOptIn"`
			UserUUID       *string   `json:"userUuid"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		args := []any{}
		sets := []string{}
		add := func(field string, value any) {
			args = append(args, value)
			sets = append(sets, field+"=$"+itoa(len(args)))
		}
		if body.Email != nil {
			email, err := normalizeEmail(*body.Email)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid email"})
			}
			add("email", email)
		}
		if body.FirstName != nil {
			add("first_name", strings.TrimSpace(*body.FirstName))
		}
		if body.LastName != nil {
			add("last_name", strings.TrimSpace(*body.LastName))
		}
		if body.Phone != nil {
			add("phone", strings.TrimSpace(*body.Phone))
		}
		if body.Notes != nil {
			add("notes", strings.TrimSpace(*body.Notes))
		}
		if body.Tags != nil {
			add("tags", pqStringArray(normalizeTags(*body.Tags)))
		}
		if body.MarketingOptIn != nil {
			add("marketing_opt_in", *body.MarketingOptIn)
		}
		if body.UserUUID != nil {
			userUUID, err := parseOptionalUUID(*body.UserUUID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid user uuid"})
			}
			if userUUID != nil {
				args = append(args, *userUUID)
			} else {
				args = append(args, nil)
			}
			sets = append(sets, "user_uuid=$"+itoa(len(args)))
		}
		if len(sets) == 0 {
			summary, err := fetchCustomerSummary(opts.DB, customerID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
				}
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			return c.JSON(fiber.Map{"success": true, "data": summary})
		}
		sets = append(sets, "updated_at=now()")
		args = append(args, customerID, meta.ShopUUID)
		query := "UPDATE customers SET " + strings.Join(sets, ", ") + " WHERE uuid=$" + itoa(len(args)-1) + " AND shop_uuid=$" + itoa(len(args))
		if _, err := opts.DB.Exec(query, args...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		summary, err := fetchCustomerSummary(opts.DB, customerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": summary})
	})

	app.Get("/v1/customers/:id/orders", requireAuth, func(c *fiber.Ctx) error {
		customerParam := strings.TrimSpace(c.Params("id"))
		if customerParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		customerID, err := uuid.Parse(customerParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		meta, err := loadCustomerMeta(opts.DB, customerID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		if meta.UserUUID == nil {
			return c.JSON(fiber.Map{"success": true, "data": []CustomerOrder{}})
		}
		var orders []CustomerOrder
		if err := opts.DB.Select(&orders, `SELECT o.uuid,
                                                   SUM(oi.price_cents * oi.quantity) AS total_cents,
                                                   o.currency,
                                                   o.status,
                                                   o.created_at,
                                                   o.updated_at,
                                                   o.tracking_number,
                                                   o.tracking_url,
                                                   o.shipping_carrier,
                                                   o.shipped_at,
                                                   o.delivered_at
                                            FROM orders o
                                            JOIN order_items oi ON oi.order_uuid=o.uuid
                                            JOIN products p ON p.uuid=oi.product_uuid
                                            WHERE p.shop_uuid=$1 AND o.user_uuid=$2
                                            GROUP BY o.uuid, o.currency, o.status, o.created_at, o.updated_at, o.tracking_number, o.tracking_url, o.shipping_carrier, o.shipped_at, o.delivered_at
                                            ORDER BY o.created_at DESC LIMIT 200`, meta.ShopUUID, *meta.UserUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": orders})
	})

	app.Post("/v1/my/shops/:slug/orders", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			CustomerUUID string `json:"customerUuid"`
			Items        []struct {
				ProductUUID string `json:"productUuid"`
				Quantity    int    `json:"quantity"`
			} `json:"items"`
			DiscountCode    string `json:"discountCode"`
			GiftCardCode    string `json:"giftCardCode"`
			Status          string `json:"status"`
			ShippingAddress string `json:"shippingAddress"`
			PaymentMethod   string `json:"paymentMethod"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		customerID, err := uuid.Parse(strings.TrimSpace(body.CustomerUUID))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
		}
		requestItems := make([]manualOrderItemInput, 0, len(body.Items))
		for idx, item := range body.Items {
			productID, err := uuid.Parse(strings.TrimSpace(item.ProductUUID))
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("invalid product id at position %d", idx)})
			}
			if item.Quantity <= 0 {
				return c.Status(400).JSON(fiber.Map{"success": false, "message": "item quantities must be greater than zero"})
			}
			requestItems = append(requestItems, manualOrderItemInput{
				ProductUUID: productID,
				Quantity:    item.Quantity,
			})
		}
		var customerRow struct {
			UserUUID *uuid.UUID `db:"user_uuid"`
			Email    string     `db:"email"`
		}
		if err := opts.DB.Get(&customerRow, `SELECT user_uuid, email FROM customers WHERE uuid=$1 AND shop_uuid=$2`, customerID, shop.UUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(404).JSON(fiber.Map{"success": false, "message": "customer not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if customerRow.UserUUID == nil || *customerRow.UserUUID == uuid.Nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "customer must be linked to an active user account"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		preparedItems, err := prepareManualOrderItems(tx, shop.UUID, requestItems)
		if err != nil {
			return respondWithError(c, err)
		}

		result, err := createManualOrderTx(tx, shop, *customerRow.UserUUID, preparedItems, manualOrderOptions{
			DiscountCode:    body.DiscountCode,
			GiftCardCode:    body.GiftCardCode,
			Status:          body.Status,
			ShippingAddress: body.ShippingAddress,
			PaymentMethod:   body.PaymentMethod,
			CreatedBy:       uuidPtrFromString(srvAuth.UserID(c)),
			CustomerUUID:    customerID,
			CustomerEmail:   customerRow.Email,
		})
		if err != nil {
			return respondWithError(c, err)
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"uuid":                result.OrderID,
			"subtotalCents":       result.SubtotalCents,
			"discountAmountCents": result.DiscountAmountCents,
			"giftCardAmountCents": result.GiftCardAmountCents,
			"totalCents":          result.TotalCents,
			"currency":            result.Currency,
			"status":              result.Status,
		}})
	})

	app.Get("/v1/my/shops/:slug/orders/drafts", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var drafts []OrderDraft
		if err := opts.DB.Select(&drafts, `SELECT uuid,
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
                                           WHERE shop_uuid=$1
                                           ORDER BY updated_at DESC
                                           LIMIT 200`, shop.UUID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if len(drafts) > 0 {
			ids := make([]uuid.UUID, 0, len(drafts))
			for _, draft := range drafts {
				ids = append(ids, draft.UUID)
			}
			itemsMap, err := loadDraftItems(opts.DB, ids)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			for i := range drafts {
				if items, ok := itemsMap[drafts[i].UUID]; ok {
					drafts[i].Items = items
				} else {
					drafts[i].Items = []DraftItem{}
				}
			}
		}
		return c.JSON(fiber.Map{"success": true, "data": drafts})
	})

	app.Post("/v1/my/shops/:slug/orders/drafts", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			CustomerUUID    string     `json:"customerUuid"`
			CustomerEmail   string     `json:"customerEmail"`
			CustomerName    string     `json:"customerName"`
			Notes           string     `json:"notes"`
			DiscountCode    string     `json:"discountCode"`
			GiftCardCode    string     `json:"giftCardCode"`
			ShippingAddress string     `json:"shippingAddress"`
			PaymentMethod   string     `json:"paymentMethod"`
			Status          string     `json:"status"`
			ExpiresAt       *time.Time `json:"expiresAt"`
			Items           []struct {
				ProductUUID string `json:"productUuid"`
				Quantity    int    `json:"quantity"`
			} `json:"items"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		if len(body.Items) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "at least one draft item is required"})
		}
		requestItems := make([]manualOrderItemInput, 0, len(body.Items))
		for idx, item := range body.Items {
			productID, err := uuid.Parse(strings.TrimSpace(item.ProductUUID))
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("invalid product id at position %d", idx)})
			}
			if item.Quantity <= 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "item quantities must be greater than zero"})
			}
			requestItems = append(requestItems, manualOrderItemInput{
				ProductUUID: productID,
				Quantity:    item.Quantity,
			})
		}
		var customerUUIDPtr *uuid.UUID
		customerEmail := strings.TrimSpace(body.CustomerEmail)
		customerName := strings.TrimSpace(body.CustomerName)
		if strings.TrimSpace(body.CustomerUUID) != "" {
			customerID, err := uuid.Parse(strings.TrimSpace(body.CustomerUUID))
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
			}
			var customerRow struct {
				UUID      uuid.UUID `db:"uuid"`
				Email     *string   `db:"email"`
				FirstName *string   `db:"first_name"`
				LastName  *string   `db:"last_name"`
			}
			if err := opts.DB.Get(&customerRow, `SELECT uuid, email, first_name, last_name FROM customers WHERE uuid=$1 AND shop_uuid=$2`, customerID, shop.UUID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "customer not found"})
				}
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			customerUUIDPtr = &customerRow.UUID
			if customerEmail == "" && customerRow.Email != nil {
				customerEmail = strings.TrimSpace(*customerRow.Email)
			}
			if customerName == "" {
				first := ""
				last := ""
				if customerRow.FirstName != nil {
					first = strings.TrimSpace(*customerRow.FirstName)
				}
				if customerRow.LastName != nil {
					last = strings.TrimSpace(*customerRow.LastName)
				}
				customerName = strings.TrimSpace(strings.TrimSpace(first) + " " + strings.TrimSpace(last))
			}
		}

		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		preparedItems, err := prepareManualOrderItems(tx, shop.UUID, requestItems)
		if err != nil {
			return respondWithError(c, err)
		}
		subtotal := int64(0)
		currency := ""
		for _, item := range preparedItems {
			if currency == "" {
				currency = item.Currency
			}
			subtotal += int64(item.Quantity) * item.PriceCents
		}
		draftID := uuid.New()
		status := strings.TrimSpace(strings.ToLower(body.Status))
		if status == "" {
			status = "open"
		}
		discountCode := strings.ToUpper(strings.TrimSpace(body.DiscountCode))
		if discountCode == "" {
			discountCode = ""
		}
		giftCardCode := strings.ToUpper(strings.TrimSpace(body.GiftCardCode))
		if giftCardCode == "" {
			giftCardCode = ""
		}
		shippingAddress := strings.TrimSpace(body.ShippingAddress)
		paymentMethod := strings.TrimSpace(body.PaymentMethod)
		notes := strings.TrimSpace(body.Notes)
		createdBy := uuidPtrFromString(srvAuth.UserID(c))

		var expiresAtValue any
		if body.ExpiresAt != nil && !body.ExpiresAt.IsZero() {
			expiresAtValue = body.ExpiresAt.UTC()
		}
		var customerUUIDValue any
		if customerUUIDPtr != nil {
			customerUUIDValue = *customerUUIDPtr
		}
		var customerEmailValue any
		if customerEmail != "" {
			customerEmailValue = customerEmail
		}
		var customerNameValue any
		if customerName != "" {
			customerNameValue = customerName
		}
		var discountCodeValue any
		if discountCode != "" {
			discountCodeValue = discountCode
		}
		var giftCardCodeValue any
		if giftCardCode != "" {
			giftCardCodeValue = giftCardCode
		}
		var shippingAddressValue any
		if shippingAddress != "" {
			shippingAddressValue = shippingAddress
		}
		var paymentMethodValue any
		if paymentMethod != "" {
			paymentMethodValue = paymentMethod
		}
		var notesValue any
		if notes != "" {
			notesValue = notes
		}
		var createdByValue any
		if createdBy != nil {
			createdByValue = *createdBy
		}

		var createdAt, updatedAt time.Time
		if err := tx.QueryRowx(`INSERT INTO order_drafts(uuid,
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
                                                        updated_by)
                                 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
                                 RETURNING created_at, updated_at`,
			draftID,
			shop.UUID,
			customerUUIDValue,
			customerEmailValue,
			customerNameValue,
			currency,
			subtotal,
			discountCodeValue,
			int64(0),
			giftCardCodeValue,
			int64(0),
			shippingAddressValue,
			paymentMethodValue,
			notesValue,
			status,
			expiresAtValue,
			createdByValue,
			createdByValue).Scan(&createdAt, &updatedAt); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		for _, item := range preparedItems {
			lineTotal := int64(item.Quantity) * item.PriceCents
			if _, err := tx.Exec(`INSERT INTO order_draft_items(uuid, draft_uuid, product_uuid, quantity, price_cents, line_total_cents, currency, title)
                                   VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
				uuid.New(),
				draftID,
				item.ProductUUID,
				item.Quantity,
				item.PriceCents,
				lineTotal,
				item.Currency,
				item.Title); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}

		draft, err := loadDraftByUUID(tx, draftID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true, "data": draft})
	})

	app.Patch("/v1/orders/drafts/:id", requireAuth, func(c *fiber.Ctx) error {
		draftParam := strings.TrimSpace(c.Params("id"))
		if draftParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid draft id"})
		}
		draftID, err := uuid.Parse(draftParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid draft id"})
		}
		var draftMeta struct {
			ShopSlug string    `db:"slug"`
			ShopUUID uuid.UUID `db:"shop_uuid"`
		}
		if err := opts.DB.Get(&draftMeta, `SELECT s.slug, s.uuid AS shop_uuid
                                           FROM order_drafts d
                                           JOIN shops s ON s.uuid=d.shop_uuid
                                           WHERE d.uuid=$1`, draftID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "draft not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		_, role, err := ensureShopAccess(c, opts.DB, draftMeta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			CustomerUUID    *string    `json:"customerUuid"`
			CustomerEmail   *string    `json:"customerEmail"`
			CustomerName    *string    `json:"customerName"`
			Notes           *string    `json:"notes"`
			DiscountCode    *string    `json:"discountCode"`
			GiftCardCode    *string    `json:"giftCardCode"`
			ShippingAddress *string    `json:"shippingAddress"`
			PaymentMethod   *string    `json:"paymentMethod"`
			Status          *string    `json:"status"`
			ExpiresAt       *time.Time `json:"expiresAt"`
			Items           *[]struct {
				ProductUUID string `json:"productUuid"`
				Quantity    int    `json:"quantity"`
			} `json:"items"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}

		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		var customerUUIDValue any
		if body.CustomerUUID != nil {
			trimmed := strings.TrimSpace(*body.CustomerUUID)
			if trimmed == "" {
				customerUUIDValue = nil
			} else {
				customerID, parseErr := uuid.Parse(trimmed)
				if parseErr != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid customer id"})
				}
				var exists struct {
					UUID uuid.UUID `db:"uuid"`
				}
				if err := tx.Get(&exists, `SELECT uuid FROM customers WHERE uuid=$1 AND shop_uuid=$2`, customerID, draftMeta.ShopUUID); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "customer not found"})
					}
					return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
				}
				customerUUIDValue = customerID
			}
		}

		setParts := []string{"updated_at=now()"}
		args := []any{draftID}
		argPos := 2
		if customerUUIDValue != nil || (body.CustomerUUID != nil && strings.TrimSpace(*body.CustomerUUID) == "") {
			if customerUUIDValue != nil {
				setParts = append(setParts, fmt.Sprintf("customer_uuid=$%d", argPos))
				args = append(args, customerUUIDValue)
				argPos++
			} else {
				setParts = append(setParts, "customer_uuid=NULL")
			}
		}
		if body.CustomerEmail != nil {
			email := strings.TrimSpace(*body.CustomerEmail)
			if email == "" {
				setParts = append(setParts, "customer_email=NULL")
			} else {
				setParts = append(setParts, fmt.Sprintf("customer_email=$%d", argPos))
				args = append(args, email)
				argPos++
			}
		}
		if body.CustomerName != nil {
			name := strings.TrimSpace(*body.CustomerName)
			if name == "" {
				setParts = append(setParts, "customer_name=NULL")
			} else {
				setParts = append(setParts, fmt.Sprintf("customer_name=$%d", argPos))
				args = append(args, name)
				argPos++
			}
		}
		if body.Notes != nil {
			note := strings.TrimSpace(*body.Notes)
			if note == "" {
				setParts = append(setParts, "notes=NULL")
			} else {
				setParts = append(setParts, fmt.Sprintf("notes=$%d", argPos))
				args = append(args, note)
				argPos++
			}
		}
		if body.DiscountCode != nil {
			code := strings.ToUpper(strings.TrimSpace(*body.DiscountCode))
			if code == "" {
				setParts = append(setParts, "discount_code=NULL", "discount_amount_cents=0")
			} else {
				setParts = append(setParts, fmt.Sprintf("discount_code=$%d", argPos), "discount_amount_cents=0")
				args = append(args, code)
				argPos++
			}
		}
		if body.GiftCardCode != nil {
			code := strings.ToUpper(strings.TrimSpace(*body.GiftCardCode))
			if code == "" {
				setParts = append(setParts, "gift_card_code=NULL", "gift_card_amount_cents=0")
			} else {
				setParts = append(setParts, fmt.Sprintf("gift_card_code=$%d", argPos), "gift_card_amount_cents=0")
				args = append(args, code)
				argPos++
			}
		}
		if body.ShippingAddress != nil {
			address := strings.TrimSpace(*body.ShippingAddress)
			if address == "" {
				setParts = append(setParts, "shipping_address=NULL")
			} else {
				setParts = append(setParts, fmt.Sprintf("shipping_address=$%d", argPos))
				args = append(args, address)
				argPos++
			}
		}
		if body.PaymentMethod != nil {
			method := strings.TrimSpace(*body.PaymentMethod)
			if method == "" {
				setParts = append(setParts, "payment_method=NULL")
			} else {
				setParts = append(setParts, fmt.Sprintf("payment_method=$%d", argPos))
				args = append(args, method)
				argPos++
			}
		}
		if body.Status != nil {
			status := strings.TrimSpace(strings.ToLower(*body.Status))
			if status == "" {
				status = "open"
			}
			setParts = append(setParts, fmt.Sprintf("status=$%d", argPos))
			args = append(args, status)
			argPos++
		}
		if body.ExpiresAt != nil {
			exp := body.ExpiresAt.UTC()
			setParts = append(setParts, fmt.Sprintf("expires_at=$%d", argPos))
			args = append(args, exp)
			argPos++
		}

		if body.Items != nil {
			if len(*body.Items) == 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "draft must contain at least one item"})
			}
			itemInputs := make([]manualOrderItemInput, 0, len(*body.Items))
			for idx, raw := range *body.Items {
				productID, parseErr := uuid.Parse(strings.TrimSpace(raw.ProductUUID))
				if parseErr != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("invalid product id at position %d", idx)})
				}
				if raw.Quantity <= 0 {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "item quantities must be greater than zero"})
				}
				itemInputs = append(itemInputs, manualOrderItemInput{
					ProductUUID: productID,
					Quantity:    raw.Quantity,
				})
			}
			prepared, prepErr := prepareManualOrderItems(tx, draftMeta.ShopUUID, itemInputs)
			if prepErr != nil {
				return respondWithError(c, prepErr)
			}
			if _, err := tx.Exec(`DELETE FROM order_draft_items WHERE draft_uuid=$1`, draftID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			subtotal := int64(0)
			currency := ""
			for _, item := range prepared {
				if currency == "" {
					currency = item.Currency
				}
				lineTotal := int64(item.Quantity) * item.PriceCents
				subtotal += lineTotal
				if _, err := tx.Exec(`INSERT INTO order_draft_items(uuid, draft_uuid, product_uuid, quantity, price_cents, line_total_cents, currency, title)
                                       VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
					uuid.New(),
					draftID,
					item.ProductUUID,
					item.Quantity,
					item.PriceCents,
					lineTotal,
					item.Currency,
					item.Title); err != nil {
					return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
				}
			}
			setParts = append(setParts, fmt.Sprintf("subtotal_cents=$%d", argPos))
			args = append(args, subtotal)
			argPos++
			setParts = append(setParts, fmt.Sprintf("currency=$%d", argPos))
			args = append(args, currency)
			argPos++
		}

		if updatedBy := uuidPtrFromString(srvAuth.UserID(c)); updatedBy != nil {
			setParts = append(setParts, fmt.Sprintf("updated_by=$%d", argPos))
			args = append(args, *updatedBy)
			argPos++
		}

		query := fmt.Sprintf("UPDATE order_drafts SET %s WHERE uuid=$1", strings.Join(setParts, ", "))
		if _, err := tx.Exec(query, args...); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		draft, err := loadDraftByUUID(tx, draftID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true, "data": draft})
	})

	app.Delete("/v1/orders/drafts/:id", requireAuth, func(c *fiber.Ctx) error {
		draftParam := strings.TrimSpace(c.Params("id"))
		if draftParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid draft id"})
		}
		draftID, err := uuid.Parse(draftParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid draft id"})
		}
		var draftMeta struct {
			ShopSlug string `db:"slug"`
		}
		if err := opts.DB.Get(&draftMeta, `SELECT s.slug FROM order_drafts d JOIN shops s ON s.uuid=d.shop_uuid WHERE d.uuid=$1`, draftID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "draft not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		_, role, err := ensureShopAccess(c, opts.DB, draftMeta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		if _, err := opts.DB.Exec(`DELETE FROM order_drafts WHERE uuid=$1`, draftID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Post("/v1/orders/drafts/:id/commit", requireAuth, func(c *fiber.Ctx) error {
		draftParam := strings.TrimSpace(c.Params("id"))
		if draftParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid draft id"})
		}
		draftID, err := uuid.Parse(draftParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid draft id"})
		}
		var draftMeta struct {
			ShopSlug string    `db:"slug"`
			ShopUUID uuid.UUID `db:"shop_uuid"`
		}
		if err := opts.DB.Get(&draftMeta, `SELECT s.slug, s.uuid AS shop_uuid
                                           FROM order_drafts d
                                           JOIN shops s ON s.uuid=d.shop_uuid
                                           WHERE d.uuid=$1`, draftID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "draft not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, draftMeta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var body struct {
			Status          string `json:"status"`
			DiscountCode    string `json:"discountCode"`
			GiftCardCode    string `json:"giftCardCode"`
			ShippingAddress string `json:"shippingAddress"`
			PaymentMethod   string `json:"paymentMethod"`
		}
		if err := c.BodyParser(&body); err != nil && err != io.EOF {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}

		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		draft, err := loadDraftByUUID(tx, draftID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "draft not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if len(draft.Items) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "draft has no items"})
		}
		if draft.CustomerUUID == nil || *draft.CustomerUUID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "draft must be linked to a customer before committing"})
		}
		var customerRow struct {
			UserUUID *uuid.UUID `db:"user_uuid"`
			Email    string     `db:"email"`
		}
		if err := tx.Get(&customerRow, `SELECT user_uuid, email FROM customers WHERE uuid=$1 AND shop_uuid=$2`, *draft.CustomerUUID, draftMeta.ShopUUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "customer not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if customerRow.UserUUID == nil || *customerRow.UserUUID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "customer must have an active platform account"})
		}

		for _, item := range draft.Items {
			var productRow struct {
				ShopUUID uuid.UUID `db:"shop_uuid"`
			}
			if err := tx.Get(&productRow, `SELECT shop_uuid FROM products WHERE uuid=$1 AND deleted_at IS NULL`, item.ProductUUID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "one or more draft products are no longer available"})
				}
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			if productRow.ShopUUID != draftMeta.ShopUUID {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "draft contains products from another shop"})
			}
		}

		preparedItems := make([]manualOrderPreparedItem, 0, len(draft.Items))
		for _, item := range draft.Items {
			preparedItems = append(preparedItems, manualOrderPreparedItem{
				ProductUUID: item.ProductUUID,
				Quantity:    item.Quantity,
				PriceCents:  item.PriceCents,
				Currency:    item.Currency,
				Title:       item.Title,
			})
		}

		discountCode := strings.TrimSpace(body.DiscountCode)
		if discountCode == "" && draft.DiscountCode != nil {
			discountCode = strings.TrimSpace(*draft.DiscountCode)
		}
		giftCardCode := strings.TrimSpace(body.GiftCardCode)
		if giftCardCode == "" && draft.GiftCardCode != nil {
			giftCardCode = strings.TrimSpace(*draft.GiftCardCode)
		}
		shippingAddress := strings.TrimSpace(body.ShippingAddress)
		if shippingAddress == "" && draft.ShippingAddress != nil {
			shippingAddress = strings.TrimSpace(*draft.ShippingAddress)
		}
		paymentMethod := strings.TrimSpace(body.PaymentMethod)
		if paymentMethod == "" && draft.PaymentMethod != nil {
			paymentMethod = strings.TrimSpace(*draft.PaymentMethod)
		}
		targetStatus := strings.TrimSpace(body.Status)
		if targetStatus == "" {
			targetStatus = "pending"
		}
		eventMeta := map[string]any{
			"draftUuid": draft.UUID.String(),
		}
		if draft.Notes != nil && strings.TrimSpace(*draft.Notes) != "" {
			eventMeta["draftNotes"] = strings.TrimSpace(*draft.Notes)
		}
		eventMeta["draftStatus"] = draft.Status

		result, err := createManualOrderTx(tx, shop, *customerRow.UserUUID, preparedItems, manualOrderOptions{
			DiscountCode:    discountCode,
			GiftCardCode:    giftCardCode,
			Status:          targetStatus,
			ShippingAddress: shippingAddress,
			PaymentMethod:   paymentMethod,
			DraftSourceUUID: &draft.UUID,
			EventType:       "order.created",
			EventMessage:    "Order created from draft",
			EventMetadata:   eventMeta,
			CreatedBy:       uuidPtrFromString(srvAuth.UserID(c)),
			CustomerUUID:    *draft.CustomerUUID,
			CustomerEmail:   customerRow.Email,
		})
		if err != nil {
			return respondWithError(c, err)
		}

		if _, err := tx.Exec(`DELETE FROM order_drafts WHERE uuid=$1`, draft.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"orderUuid":           result.OrderID,
			"draftUuid":           draft.UUID,
			"status":              result.Status,
			"subtotalCents":       result.SubtotalCents,
			"discountAmountCents": result.DiscountAmountCents,
			"giftCardAmountCents": result.GiftCardAmountCents,
			"totalCents":          result.TotalCents,
			"currency":            result.Currency,
		}})
	})

	app.Get("/v1/my/shops/:slug/checkouts/abandoned", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		const query = `WITH cart_summary AS (
                         SELECT
                           c.uuid AS cart_uuid,
                           c.user_uuid,
                           p.shop_uuid,
                           SUM(ci.quantity) AS item_count,
                           SUM(ci.quantity * p.price_cents) AS subtotal_cents,
                           MIN(ci.added_at) AS first_added_at,
                           MAX(ci.added_at) AS last_added_at,
                           GREATEST(c.updated_at, COALESCE(MAX(ci.added_at), c.updated_at)) AS last_activity_at,
                           MIN(p.currency) AS currency
                         FROM carts c
                         JOIN cart_items ci ON ci.cart_uuid = c.uuid
                         JOIN products p ON p.uuid = ci.product_uuid AND p.deleted_at IS NULL
                         GROUP BY c.uuid, c.user_uuid, p.shop_uuid, c.updated_at
                         HAVING SUM(ci.quantity) > 0
                            AND COUNT(DISTINCT p.currency) = 1
                       )
                       SELECT
                         cs.cart_uuid,
                         cs.user_uuid,
                         cs.shop_uuid,
                         cs.item_count,
                         cs.subtotal_cents,
                         cs.currency,
                         cs.first_added_at,
                         cs.last_added_at,
                         cs.last_activity_at,
                         cust.uuid AS customer_uuid,
                         COALESCE(cust.email, up.email, '') AS customer_email,
                         COALESCE(NULLIF(TRIM(COALESCE(cust.first_name,'') || ' ' || COALESCE(cust.last_name,'')), ''), up.display_name, '') AS customer_name
                       FROM cart_summary cs
                       LEFT JOIN customers cust ON cust.shop_uuid = cs.shop_uuid AND cust.user_uuid = cs.user_uuid
                       LEFT JOIN user_profiles up ON up.user_uuid = cs.user_uuid
                       WHERE cs.shop_uuid=$1
                         AND cs.subtotal_cents > 0
                         AND cs.last_activity_at <= now() - interval '24 hours'
                         AND NOT EXISTS (
                           SELECT 1 FROM orders o
                           WHERE o.shop_uuid = cs.shop_uuid
                             AND o.user_uuid = cs.user_uuid
                             AND o.created_at >= cs.last_activity_at
                         )
                       ORDER BY cs.last_activity_at ASC
                       LIMIT 200`
		var rows []AbandonedCheckout
		if err := opts.DB.Select(&rows, query, shop.UUID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		for i := range rows {
			if rows[i].CustomerName == "" && rows[i].CustomerEmail != "" {
				rows[i].CustomerName = rows[i].CustomerEmail
			}
		}
		return c.JSON(fiber.Map{"success": true, "data": rows})
	})

	app.Get("/v1/my/shops/:slug/orders", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var orders []ShopOrder
		if err := opts.DB.Select(&orders, `SELECT o.uuid,
                                                  SUM(oi.price_cents * oi.quantity) AS total_cents,
                                                  o.currency,
                                                  o.status,
                                                  o.created_at,
                                                  o.updated_at,
                                                  o.tracking_number,
                                                  o.tracking_url,
                                                  o.shipping_carrier,
                                                  o.shipped_at,
                                                  o.delivered_at,
                                                  o.cancelled_at,
                                                  o.refunded_at,
                                                  o.refund_total_cents,
                                                  o.draft_source_uuid,
                                                  c.uuid AS customer_uuid,
                                                  COALESCE(c.email,'') AS customer_email,
                                                  TRIM(BOTH ' ' FROM COALESCE(c.first_name,'') || ' ' || COALESCE(c.last_name,'')) AS customer_name
                                           FROM orders o
                                           JOIN order_items oi ON oi.order_uuid=o.uuid
                                           JOIN products p ON p.uuid=oi.product_uuid
                                           LEFT JOIN customers c ON c.shop_uuid=p.shop_uuid AND c.user_uuid=o.user_uuid
                                           WHERE p.shop_uuid=$1
                                           GROUP BY o.uuid, o.currency, o.status, o.created_at, o.updated_at, o.tracking_number, o.tracking_url, o.shipping_carrier, o.shipped_at, o.delivered_at, o.cancelled_at, o.refunded_at, o.refund_total_cents, o.draft_source_uuid, c.uuid, c.email, c.first_name, c.last_name
                                           ORDER BY o.created_at DESC
                                           LIMIT 200`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": orders})
	})

	app.Get("/v1/my/shops/:slug/metrics", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var totals struct {
			Sales  sql.NullInt64 `db:"sales"`
			Orders sql.NullInt64 `db:"orders"`
		}
		if err := opts.DB.Get(&totals, `SELECT COALESCE(SUM(oi.price_cents * oi.quantity),0) AS sales,
                                               COUNT(DISTINCT o.uuid) AS orders
                                        FROM orders o
                                        JOIN order_items oi ON oi.order_uuid=o.uuid
                                        JOIN products p ON p.uuid=oi.product_uuid
                                        WHERE p.shop_uuid=$1`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		var customersCount int64
		if err := opts.DB.Get(&customersCount, `SELECT COUNT(1) FROM customers WHERE shop_uuid=$1`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		type dayRow struct {
			Day   time.Time `db:"day"`
			Total int64     `db:"total"`
		}
		var rows []dayRow
		if err := opts.DB.Select(&rows, `SELECT date_trunc('day', o.created_at) AS day,
                                                COALESCE(SUM(oi.price_cents * oi.quantity),0) AS total
                                         FROM orders o
                                         JOIN order_items oi ON oi.order_uuid=o.uuid
                                         JOIN products p ON p.uuid=oi.product_uuid
                                         WHERE p.shop_uuid=$1 AND o.created_at >= now() - interval '14 days'
                                         GROUP BY day
                                         ORDER BY day ASC`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		now := time.Now().UTC()
		seriesMap := make(map[string]int64, len(rows))
		for _, r := range rows {
			key := r.Day.UTC().Format("2006-01-02")
			seriesMap[key] = r.Total
		}
		points := make([]SalesPoint, 0, 14)
		for i := 13; i >= 0; i-- {
			day := now.AddDate(0, 0, -i)
			key := day.Format("2006-01-02")
			points = append(points, SalesPoint{
				Date:       key,
				TotalCents: seriesMap[key],
			})
		}
		totalSales := totals.Sales.Int64
		ordersCount := totals.Orders.Int64
		average := int64(0)
		if ordersCount > 0 {
			average = totalSales / ordersCount
		}
		metrics := DashboardMetrics{
			TotalSalesCents:        totalSales,
			OrdersCount:            ordersCount,
			AverageOrderValueCents: average,
			CustomersCount:         customersCount,
			SalesSeries:            points,
		}
		return c.JSON(fiber.Map{"success": true, "data": metrics})
	})

	app.Patch("/v1/orders/:id/tracking", requireAuth, func(c *fiber.Ctx) error {
		orderParam := strings.TrimSpace(c.Params("id"))
		if orderParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		orderID, err := uuid.Parse(orderParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		meta, err := loadOrderMeta(opts.DB, orderID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Status          *string `json:"status"`
			TrackingNumber  *string `json:"trackingNumber"`
			TrackingURL     *string `json:"trackingUrl"`
			ShippingCarrier *string `json:"shippingCarrier"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		args := []any{}
		sets := []string{}
		if body.Status != nil {
			status := strings.ToLower(strings.TrimSpace(*body.Status))
			if status == "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid status"})
			}
			if !isValidOrderStatus(status) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid status"})
			}
			args = append(args, status)
			sets = append(sets, "status=$"+itoa(len(args)))
			switch status {
			case "shipped":
				sets = append(sets, "shipped_at=CASE WHEN shipped_at IS NULL THEN now() ELSE shipped_at END")
			case "delivered":
				sets = append(sets, "shipped_at=CASE WHEN shipped_at IS NULL THEN now() ELSE shipped_at END")
				sets = append(sets, "delivered_at=now()")
			}
		}
		if body.TrackingNumber != nil {
			v := strings.TrimSpace(*body.TrackingNumber)
			if v == "" {
				sets = append(sets, "tracking_number=NULL")
			} else {
				args = append(args, v)
				sets = append(sets, "tracking_number=$"+itoa(len(args)))
			}
		}
		if body.TrackingURL != nil {
			v := strings.TrimSpace(*body.TrackingURL)
			if v == "" {
				sets = append(sets, "tracking_url=NULL")
			} else {
				args = append(args, v)
				sets = append(sets, "tracking_url=$"+itoa(len(args)))
			}
		}
		if body.ShippingCarrier != nil {
			v := strings.TrimSpace(*body.ShippingCarrier)
			if v == "" {
				sets = append(sets, "shipping_carrier=NULL")
			} else {
				args = append(args, v)
				sets = append(sets, "shipping_carrier=$"+itoa(len(args)))
			}
		}
		if len(sets) == 0 {
			return c.JSON(fiber.Map{"success": true})
		}
		sets = append(sets, "updated_at=now()")
		args = append(args, orderID, meta.ShopUUID)
		orderIdx := "$" + itoa(len(args)-1)
		shopIdx := "$" + itoa(len(args))
		query := "UPDATE orders SET " + strings.Join(sets, ", ") + " WHERE uuid=" + orderIdx + " AND EXISTS (SELECT 1 FROM order_items oi JOIN products p ON p.uuid=oi.product_uuid WHERE oi.order_uuid=orders.uuid AND p.shop_uuid=" + shopIdx + ")"
		res, err := opts.DB.Exec(query, args...)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "not found"})
		}
		return c.JSON(fiber.Map{"success": true})
	})

	app.Post("/v1/orders/:id/cancel", requireAuth, func(c *fiber.Ctx) error {
		orderParam := strings.TrimSpace(c.Params("id"))
		if orderParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		orderID, err := uuid.Parse(orderParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		meta, err := loadOrderMeta(opts.DB, orderID)
		if err != nil {
			return respondWithError(c, err)
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
		defer tx.Rollback()

		var row struct {
			Status           string     `db:"status"`
			DiscountUUID     *uuid.UUID `db:"discount_uuid"`
			DiscountAmount   int64      `db:"discount_amount_cents"`
			GiftCardUUID     *uuid.UUID `db:"gift_card_uuid"`
			GiftCardAmount   int64      `db:"gift_card_amount_cents"`
			GiftCardCode     *string    `db:"gift_card_code"`
			RefundTotalCents int64      `db:"refund_total_cents"`
		}
		if err := tx.Get(&row, `SELECT status, discount_uuid, discount_amount_cents, gift_card_uuid, gift_card_amount_cents, gift_card_code, refund_total_cents
                                FROM orders
                                WHERE uuid=$1
                                FOR UPDATE`, orderID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "order not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		status := strings.ToLower(strings.TrimSpace(row.Status))
		switch status {
		case "cancelled":
			return c.JSON(fiber.Map{"success": true})
		case "delivered", "shipped":
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "order can no longer be cancelled"})
		}
		if row.RefundTotalCents > 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "order has refunds applied"})
		}

		if row.DiscountUUID != nil && row.DiscountAmount > 0 {
			if _, err := tx.Exec(`DELETE FROM discount_redemptions WHERE order_uuid=$1`, orderID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}

		if row.GiftCardUUID != nil && row.GiftCardAmount > 0 {
			if err := refundGiftCard(tx, *row.GiftCardUUID, row.GiftCardAmount); err != nil {
				return respondWithError(c, err)
			}
		}

		if _, err := tx.Exec(`UPDATE orders SET status='cancelled', cancelled_at=now(), updated_at=now() WHERE uuid=$1`, orderID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		cancelMeta := map[string]any{
			"status":         "cancelled",
			"previousStatus": status,
		}
		if row.DiscountUUID != nil && row.DiscountAmount > 0 {
			cancelMeta["discountAmountCents"] = row.DiscountAmount
		}
		if row.GiftCardUUID != nil && row.GiftCardAmount > 0 {
			cancelMeta["giftCardAmountCents"] = row.GiftCardAmount
			if row.GiftCardCode != nil && strings.TrimSpace(*row.GiftCardCode) != "" {
				cancelMeta["giftCardCode"] = strings.TrimSpace(*row.GiftCardCode)
			}
		}

		userUUID := uuidPtrFromString(srvAuth.UserID(c))
		if _, err := recordOrderEvent(tx, orderID, "order.cancelled", "Order cancelled", userUUID, cancelMeta); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "failed to record order event"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true})
	})

	app.Post("/v1/orders/:id/refunds", requireAuth, func(c *fiber.Ctx) error {
		orderParam := strings.TrimSpace(c.Params("id"))
		if orderParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		orderID, err := uuid.Parse(orderParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		meta, err := loadOrderMeta(opts.DB, orderID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var body struct {
			AmountCents int64  `json:"amountCents"`
			Reason      string `json:"reason"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		reason := strings.TrimSpace(body.Reason)
		if reason == "" {
			reason = "manual refund"
		}

		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		var row struct {
			Status           string     `db:"status"`
			TotalCents       int64      `db:"total_cents"`
			RefundTotalCents int64      `db:"refund_total_cents"`
			DiscountUUID     *uuid.UUID `db:"discount_uuid"`
			DiscountAmount   int64      `db:"discount_amount_cents"`
			GiftCardUUID     *uuid.UUID `db:"gift_card_uuid"`
			GiftCardAmount   int64      `db:"gift_card_amount_cents"`
		}
		if err := tx.Get(&row, `SELECT status, total_cents, refund_total_cents, discount_uuid, discount_amount_cents, gift_card_uuid, gift_card_amount_cents
                                FROM orders
                                WHERE uuid=$1
                                FOR UPDATE`, orderID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "order not found"})
			}
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		status := strings.ToLower(strings.TrimSpace(row.Status))
		if status == "cancelled" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "cancelled orders cannot be refunded"})
		}

		refundable := row.TotalCents - row.RefundTotalCents
		if refundable <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "order already refunded"})
		}

		amount := body.AmountCents
		if amount <= 0 {
			amount = refundable
		}
		if amount <= 0 || amount > refundable {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid refund amount"})
		}

		if row.DiscountUUID != nil && row.DiscountAmount > 0 {
			if _, err := tx.Exec(`DELETE FROM discount_redemptions WHERE order_uuid=$1`, orderID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}

		giftCardRefund := int64(0)
		if row.GiftCardUUID != nil {
			var giftCardSums struct {
				Amount int64 `db:"gift_card_refunded"`
			}
			if err := tx.Get(&giftCardSums, `SELECT COALESCE(SUM(gift_card_amount_cents),0) AS gift_card_refunded
			                                 FROM order_refunds WHERE order_uuid=$1`, orderID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			remainingGiftCard := row.GiftCardAmount - giftCardSums.Amount
			if remainingGiftCard < 0 {
				remainingGiftCard = 0
			}
			if remainingGiftCard > 0 {
				if amount < remainingGiftCard {
					giftCardRefund = amount
				} else {
					giftCardRefund = remainingGiftCard
				}
			}
			if giftCardRefund > 0 {
				if err := refundGiftCard(tx, *row.GiftCardUUID, giftCardRefund); err != nil {
					return respondWithError(c, err)
				}
			}
		}

		processedUUID := uuidPtrFromString(srvAuth.UserID(c))

		if _, err := tx.Exec(`INSERT INTO order_refunds(uuid, order_uuid, amount_cents, gift_card_amount_cents, discount_amount_cents, reason, processed_by)
                               VALUES($1,$2,$3,$4,$5,$6,$7)`,
			uuid.New(), orderID, amount, giftCardRefund, int64(0), reason, processedUUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		newRefundTotal := row.RefundTotalCents + amount
		remainingRefundable := row.TotalCents - newRefundTotal
		if remainingRefundable < 0 {
			remainingRefundable = 0
		}
		newStatus := status
		if newRefundTotal >= row.TotalCents {
			newStatus = "refunded"
		} else if newRefundTotal > 0 {
			newStatus = "partially_refunded"
		}
		if _, err := tx.Exec(`UPDATE orders
                              SET status=$2,
                                  refund_total_cents=$3,
                                  refunded_at=CASE WHEN $2='refunded' THEN now() ELSE refunded_at END,
                                  updated_at=now()
                              WHERE uuid=$1`, orderID, newStatus, newRefundTotal); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		eventMeta := map[string]any{
			"amountCents":      amount,
			"refundTotalCents": newRefundTotal,
			"reason":           reason,
			"status":           newStatus,
		}
		if giftCardRefund > 0 {
			eventMeta["giftCardAmountCents"] = giftCardRefund
		}
		if remainingRefundable > 0 {
			eventMeta["remainingRefundableCents"] = remainingRefundable
		}
		if _, err := recordOrderEvent(tx, orderID, "order.refunded", fmt.Sprintf("Refunded %s", formatCents(amount)), processedUUID, eventMeta); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "failed to record order event"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"amountCents":      amount,
			"refundTotalCents": newRefundTotal,
			"status":           newStatus,
		}})
	})

	app.Get("/v1/orders/:id/events", requireAuth, func(c *fiber.Ctx) error {
		orderParam := strings.TrimSpace(c.Params("id"))
		if orderParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		orderID, err := uuid.Parse(orderParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		meta, err := loadOrderMeta(opts.DB, orderID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		events, err := fetchOrderEvents(opts.DB, orderID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": events})
	})

	app.Post("/v1/orders/:id/notes", requireAuth, func(c *fiber.Ctx) error {
		orderParam := strings.TrimSpace(c.Params("id"))
		if orderParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		orderID, err := uuid.Parse(orderParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid order id"})
		}
		meta, err := loadOrderMeta(opts.DB, orderID)
		if err != nil {
			return respondWithError(c, err)
		}
		_, role, err := ensureShopAccess(c, opts.DB, meta.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Message string `json:"message"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		message := strings.TrimSpace(body.Message)
		if message == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "message is required"})
		}

		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		eventID, err := recordOrderEvent(tx, orderID, "order.note", message, uuidPtrFromString(srvAuth.UserID(c)), nil)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "failed to record order event"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		event, err := fetchOrderEventByID(opts.DB, eventID)
		if err != nil {
			// Event recorded but fetch failed; return success without payload.
			return c.JSON(fiber.Map{"success": true})
		}

		return c.JSON(fiber.Map{"success": true, "data": event})
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
	app.Get("/v1/cart", requireAuth, func(c *fiber.Ctx) error { return getCart(c, opts.DB) })
	app.Put("/v1/cart", requireAuth, func(c *fiber.Ctx) error { return replaceCart(c, opts.DB) })
	app.Post("/v1/cart/items", requireAuth, func(c *fiber.Ctx) error { return addCartItem(c, opts.DB) })
	app.Put("/v1/cart/items/:id", requireAuth, func(c *fiber.Ctx) error { return updateCartItem(c, opts.DB) })
	app.Delete("/v1/cart/items/:id", requireAuth, func(c *fiber.Ctx) error { return deleteCartItem(c, opts.DB) })
	app.Delete("/v1/cart", requireAuth, func(c *fiber.Ctx) error { return clearCart(c, opts.DB) })
	app.Post("/v1/cart/preview", requireAuth, func(c *fiber.Ctx) error { return previewCartPricing(c, opts.DB) })

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

	app.Get("/v1/me/profile", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		profile, err := ensureUserProfile(opts.DB, user)
		if err != nil {
			return respondWithError(c, err)
		}
		addresses, err := listUserAddresses(opts.DB, user)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"profile":   profile,
			"addresses": addresses,
		}})
	})

	app.Put("/v1/me/profile", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		if _, err := ensureUserProfile(opts.DB, user); err != nil {
			return respondWithError(c, err)
		}
		var body struct {
			DisplayName    *string `json:"displayName"`
			Email          *string `json:"email"`
			Phone          *string `json:"phone"`
			AvatarURL      *string `json:"avatarUrl"`
			Timezone       *string `json:"timezone"`
			MarketingOptIn *bool   `json:"marketingOptIn"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		args := []any{}
		sets := []string{}
		add := func(field string, value any) {
			args = append(args, value)
			sets = append(sets, field+"=$"+itoa(len(args)))
		}
		if body.DisplayName != nil {
			add("display_name", strings.TrimSpace(*body.DisplayName))
		}
		if body.Email != nil {
			email := strings.TrimSpace(*body.Email)
			if email == "" {
				add("email", "")
			} else {
				normalized, err := normalizeEmail(email)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid email"})
				}
				add("email", normalized)
			}
		}
		if body.Phone != nil {
			add("phone", strings.TrimSpace(*body.Phone))
		}
		if body.AvatarURL != nil {
			url := strings.TrimSpace(*body.AvatarURL)
			if url == "" {
				sets = append(sets, "avatar_url=NULL")
			} else {
				add("avatar_url", url)
			}
		}
		if body.Timezone != nil {
			add("timezone", strings.TrimSpace(*body.Timezone))
		}
		if body.MarketingOptIn != nil {
			add("marketing_opt_in", *body.MarketingOptIn)
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, user)
			query := "UPDATE user_profiles SET " + strings.Join(sets, ", ") + " WHERE user_uuid=$" + itoa(len(args))
			if _, err := opts.DB.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		profile, err := ensureUserProfile(opts.DB, user)
		if err != nil {
			return respondWithError(c, err)
		}
		addresses, err := listUserAddresses(opts.DB, user)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"profile":   profile,
			"addresses": addresses,
		}})
	})

	app.Get("/v1/me/addresses", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		if _, err := ensureUserProfile(opts.DB, user); err != nil {
			return respondWithError(c, err)
		}
		addresses, err := listUserAddresses(opts.DB, user)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": addresses})
	})

	app.Post("/v1/me/addresses", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		if _, err := ensureUserProfile(opts.DB, user); err != nil {
			return respondWithError(c, err)
		}
		var body struct {
			Label           string `json:"label"`
			RecipientName   string `json:"recipientName"`
			Line1           string `json:"line1"`
			Line2           string `json:"line2"`
			City            string `json:"city"`
			Region          string `json:"region"`
			PostalCode      string `json:"postalCode"`
			Country         string `json:"country"`
			Phone           string `json:"phone"`
			DefaultShipping bool   `json:"defaultShipping"`
			DefaultBilling  bool   `json:"defaultBilling"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		label := strings.TrimSpace(body.Label)
		if label == "" {
			label = "Primary"
		}
		line1 := strings.TrimSpace(body.Line1)
		city := strings.TrimSpace(body.City)
		region := strings.TrimSpace(body.Region)
		postal := strings.TrimSpace(body.PostalCode)
		country := strings.ToUpper(strings.TrimSpace(body.Country))
		if line1 == "" || city == "" || region == "" || postal == "" || country == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "missing required address fields"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()
		if err := clearDefaultFlagsTx(tx, user, body.DefaultShipping, body.DefaultBilling, uuid.Nil); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		addressID := uuid.New()
		if _, err := tx.Exec(`INSERT INTO user_addresses (uuid,user_uuid,label,recipient_name,line1,line2,city,region,postal_code,country,phone,is_default_shipping,is_default_billing)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			addressID, user, label, strings.TrimSpace(body.RecipientName), line1, strings.TrimSpace(body.Line2), city, region, postal, country, strings.TrimSpace(body.Phone),
			body.DefaultShipping, body.DefaultBilling); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		address, err := loadUserAddress(opts.DB, user, addressID)
		if err != nil {
			return respondWithError(c, err)
		}
		return c.JSON(fiber.Map{"success": true, "data": address})
	})

	app.Patch("/v1/me/addresses/:id", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		if _, err := ensureUserProfile(opts.DB, user); err != nil {
			return respondWithError(c, err)
		}
		idParam := strings.TrimSpace(c.Params("id"))
		addressID, err := uuid.Parse(idParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid address id"})
		}
		if _, err := loadUserAddress(opts.DB, user, addressID); err != nil {
			return respondWithError(c, err)
		}
		var body struct {
			Label           *string `json:"label"`
			RecipientName   *string `json:"recipientName"`
			Line1           *string `json:"line1"`
			Line2           *string `json:"line2"`
			City            *string `json:"city"`
			Region          *string `json:"region"`
			PostalCode      *string `json:"postalCode"`
			Country         *string `json:"country"`
			Phone           *string `json:"phone"`
			DefaultShipping *bool   `json:"defaultShipping"`
			DefaultBilling  *bool   `json:"defaultBilling"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()
		args := []any{}
		sets := []string{}
		add := func(field string, value any) {
			args = append(args, value)
			sets = append(sets, field+"=$"+itoa(len(args)))
		}
		if body.Label != nil {
			add("label", strings.TrimSpace(*body.Label))
		}
		if body.RecipientName != nil {
			add("recipient_name", strings.TrimSpace(*body.RecipientName))
		}
		if body.Line1 != nil {
			add("line1", strings.TrimSpace(*body.Line1))
		}
		if body.Line2 != nil {
			add("line2", strings.TrimSpace(*body.Line2))
		}
		if body.City != nil {
			add("city", strings.TrimSpace(*body.City))
		}
		if body.Region != nil {
			add("region", strings.TrimSpace(*body.Region))
		}
		if body.PostalCode != nil {
			add("postal_code", strings.TrimSpace(*body.PostalCode))
		}
		if body.Country != nil {
			add("country", strings.ToUpper(strings.TrimSpace(*body.Country)))
		}
		if body.Phone != nil {
			add("phone", strings.TrimSpace(*body.Phone))
		}
		if body.DefaultShipping != nil {
			if *body.DefaultShipping {
				if err := clearDefaultFlagsTx(tx, user, true, false, addressID); err != nil {
					return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
				}
				sets = append(sets, "is_default_shipping=true")
			} else {
				sets = append(sets, "is_default_shipping=false")
			}
		}
		if body.DefaultBilling != nil {
			if *body.DefaultBilling {
				if err := clearDefaultFlagsTx(tx, user, false, true, addressID); err != nil {
					return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
				}
				sets = append(sets, "is_default_billing=true")
			} else {
				sets = append(sets, "is_default_billing=false")
			}
		}
		if len(sets) > 0 {
			sets = append(sets, "updated_at=now()")
			args = append(args, user, addressID)
			query := "UPDATE user_addresses SET " + strings.Join(sets, ", ") + " WHERE user_uuid=$" + itoa(len(args)-1) + " AND uuid=$" + itoa(len(args))
			if _, err := tx.Exec(query, args...); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}
		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		address, err := loadUserAddress(opts.DB, user, addressID)
		if err != nil {
			return respondWithError(c, err)
		}
		return c.JSON(fiber.Map{"success": true, "data": address})
	})

	app.Delete("/v1/me/addresses/:id", requireAuth, func(c *fiber.Ctx) error {
		user := srvAuth.UserID(c)
		if _, err := ensureUserProfile(opts.DB, user); err != nil {
			return respondWithError(c, err)
		}
		idParam := strings.TrimSpace(c.Params("id"))
		addressID, err := uuid.Parse(idParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid address id"})
		}
		res, err := opts.DB.Exec(`DELETE FROM user_addresses WHERE user_uuid=$1 AND uuid=$2`, user, addressID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "address not found"})
		}
		return c.JSON(fiber.Map{"success": true})
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
