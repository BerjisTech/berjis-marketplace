package server

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	pq "github.com/lib/pq"
)

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
	Images       pq.StringArray   `db:"images" json:"images"`
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
	MinimumSubtotalCents  int64           `db:"minimum_subtotal_cents" json:"minimumSubtotalCents"`
	FreeShipping          bool            `db:"free_shipping" json:"freeShipping"`
	BuyQuantity           *int            `db:"buy_quantity" json:"buyQuantity,omitempty"`
	GetQuantity           *int            `db:"get_quantity" json:"getQuantity,omitempty"`
	GetPercentage         float64         `db:"get_percentage" json:"getPercentage"`
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
	MinimumSubtotalCents  int64      `db:"minimum_subtotal_cents" json:"minimumSubtotalCents"`
	FreeShipping          bool       `db:"free_shipping" json:"freeShipping"`
	BuyQuantity           *int       `db:"buy_quantity" json:"buyQuantity,omitempty"`
	GetQuantity           *int       `db:"get_quantity" json:"getQuantity,omitempty"`
	GetPercentage         float64    `db:"get_percentage" json:"getPercentage"`
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

type MarketingCampaign struct {
	UUID        uuid.UUID       `db:"uuid" json:"uuid"`
	ShopUUID    uuid.UUID       `db:"shop_uuid" json:"shopUuid"`
	Name        string          `db:"name" json:"name"`
	Channel     string          `db:"channel" json:"channel"`
	Status      string          `db:"status" json:"status"`
	BudgetCents int64           `db:"budget_cents" json:"budgetCents"`
	SpendCents  int64           `db:"spend_cents" json:"spendCents"`
	StartsAt    *time.Time      `db:"starts_at" json:"startsAt,omitempty"`
	EndsAt      *time.Time      `db:"ends_at" json:"endsAt,omitempty"`
	Metadata    json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt   time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updatedAt"`
}

type CampaignMessage struct {
	UUID         uuid.UUID       `db:"uuid" json:"uuid"`
	CampaignUUID uuid.UUID       `db:"campaign_uuid" json:"campaignUuid"`
	ShopUUID     uuid.UUID       `db:"shop_uuid" json:"shopUuid"`
	Subject      string          `db:"subject" json:"subject"`
	Body         string          `db:"body" json:"body"`
	Status       string          `db:"status" json:"status"`
	ScheduledAt  time.Time       `db:"scheduled_at" json:"scheduledAt"`
	SendAfter    *time.Time      `db:"send_after" json:"sendAfter,omitempty"`
	SentAt       *time.Time      `db:"sent_at" json:"sentAt,omitempty"`
	Error        *string         `db:"error" json:"error,omitempty"`
	Metadata     json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt    time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updatedAt"`
}

type MarketingAttribution struct {
	UUID        uuid.UUID       `db:"uuid" json:"uuid"`
	ShopUUID    uuid.UUID       `db:"shop_uuid" json:"shopUuid"`
	UserUUID    uuid.UUID       `db:"user_uuid" json:"userUuid"`
	OrderUUID   *uuid.UUID      `db:"order_uuid" json:"orderUuid,omitempty"`
	Source      string          `db:"source" json:"source"`
	Medium      string          `db:"medium" json:"medium"`
	Campaign    string          `db:"campaign" json:"campaign"`
	Term        string          `db:"term" json:"term"`
	Content     string          `db:"content" json:"content"`
	Referrer    string          `db:"referrer" json:"referrer"`
	LandingPage string          `db:"landing_page" json:"landingPage"`
	Metadata    json.RawMessage `db:"metadata" json:"metadata,omitempty"`
	CreatedAt   time.Time       `db:"created_at" json:"createdAt"`
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

type CartRecovery struct {
	UUID               uuid.UUID  `db:"uuid" json:"uuid"`
	CartUUID           uuid.UUID  `db:"cart_uuid" json:"cartUuid"`
	ShopUUID           uuid.UUID  `db:"shop_uuid" json:"shopUuid"`
	UserUUID           uuid.UUID  `db:"user_uuid" json:"userUuid"`
	CustomerUUID       *uuid.UUID `db:"customer_uuid" json:"customerUuid,omitempty"`
	CustomerEmail      string     `db:"customer_email" json:"customerEmail"`
	Token              string     `db:"token" json:"token"`
	Status             string     `db:"status" json:"status"`
	RecoveryURL        *string    `db:"recovery_url" json:"recoveryUrl,omitempty"`
	SentAt             *time.Time `db:"sent_at" json:"sentAt,omitempty"`
	ClickedAt          *time.Time `db:"clicked_at" json:"clickedAt,omitempty"`
	ConvertedOrderUUID *uuid.UUID `db:"converted_order_uuid" json:"convertedOrderUuid,omitempty"`
	CreatedBy          *uuid.UUID `db:"created_by" json:"createdBy,omitempty"`
	CreatedAt          time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt          time.Time  `db:"updated_at" json:"updatedAt"`
}

type OrderReturn struct {
	UUID              uuid.UUID         `db:"uuid" json:"uuid"`
	OrderUUID         uuid.UUID         `db:"order_uuid" json:"orderUuid"`
	ShopUUID          uuid.UUID         `db:"shop_uuid" json:"shopUuid"`
	CustomerUUID      *uuid.UUID        `db:"customer_uuid" json:"customerUuid,omitempty"`
	Status            string            `db:"status" json:"status"`
	Reason            *string           `db:"reason" json:"reason,omitempty"`
	Notes             *string           `db:"notes" json:"notes,omitempty"`
	RequestedBy       *uuid.UUID        `db:"requested_by" json:"requestedBy,omitempty"`
	ProcessedBy       *uuid.UUID        `db:"processed_by" json:"processedBy,omitempty"`
	Restock           bool              `db:"restock" json:"restock"`
	RestockedAt       *time.Time        `db:"restocked_at" json:"restockedAt,omitempty"`
	RefundAmountCents int64             `db:"refund_amount_cents" json:"refundAmountCents"`
	CreatedAt         time.Time         `db:"created_at" json:"createdAt"`
	UpdatedAt         time.Time         `db:"updated_at" json:"updatedAt"`
	Items             []OrderReturnItem `db:"-" json:"items"`
}

type OrderReturnItem struct {
	UUID              uuid.UUID `db:"uuid" json:"uuid"`
	ReturnUUID        uuid.UUID `db:"return_uuid" json:"returnUuid"`
	OrderItemUUID     uuid.UUID `db:"order_item_uuid" json:"orderItemUuid"`
	ProductUUID       uuid.UUID `db:"product_uuid" json:"productUuid"`
	Quantity          int       `db:"quantity" json:"quantity"`
	Reason            *string   `db:"reason" json:"reason,omitempty"`
	Condition         *string   `db:"condition" json:"condition,omitempty"`
	RestockedQuantity int       `db:"restocked_quantity" json:"restockedQuantity"`
	CreatedAt         time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt         time.Time `db:"updated_at" json:"updatedAt"`
}

type OrderLineItem struct {
	UUID        uuid.UUID `db:"uuid" json:"uuid"`
	ProductUUID uuid.UUID `db:"product_uuid" json:"productUuid"`
	Title       string    `db:"title" json:"title"`
	Quantity    int       `db:"quantity" json:"quantity"`
	PriceCents  int64     `db:"price_cents" json:"priceCents"`
}

type OrderFulfillment struct {
	UUID             uuid.UUID              `db:"uuid" json:"uuid"`
	OrderUUID        uuid.UUID              `db:"order_uuid" json:"orderUuid"`
	ShopUUID         uuid.UUID              `db:"shop_uuid" json:"shopUuid"`
	LocationUUID     *uuid.UUID             `db:"location_uuid" json:"locationUuid,omitempty"`
	LocationName     *string                `db:"location_name" json:"locationName,omitempty"`
	LocationCode     *string                `db:"location_code" json:"locationCode,omitempty"`
	Status           string                 `db:"status" json:"status"`
	TrackingNumber   *string                `db:"tracking_number" json:"trackingNumber,omitempty"`
	TrackingURL      *string                `db:"tracking_url" json:"trackingUrl,omitempty"`
	ShippingCarrier  *string                `db:"shipping_carrier" json:"shippingCarrier,omitempty"`
	LabelURL         *string                `db:"label_url" json:"labelUrl,omitempty"`
	LabelData        json.RawMessage        `db:"label_data" json:"labelData,omitempty"`
	LabelGeneratedAt *time.Time             `db:"label_generated_at" json:"labelGeneratedAt,omitempty"`
	Notes            *string                `db:"notes" json:"notes,omitempty"`
	ShippedAt        *time.Time             `db:"shipped_at" json:"shippedAt,omitempty"`
	DeliveredAt      *time.Time             `db:"delivered_at" json:"deliveredAt,omitempty"`
	CancelledAt      *time.Time             `db:"cancelled_at" json:"cancelledAt,omitempty"`
	CreatedBy        *uuid.UUID             `db:"created_by" json:"createdBy,omitempty"`
	UpdatedBy        *uuid.UUID             `db:"updated_by" json:"updatedBy,omitempty"`
	CreatedAt        time.Time              `db:"created_at" json:"createdAt"`
	UpdatedAt        time.Time              `db:"updated_at" json:"updatedAt"`
	Items            []OrderFulfillmentItem `db:"-" json:"items"`
}

type OrderFulfillmentItem struct {
	UUID            uuid.UUID `db:"uuid" json:"uuid"`
	FulfillmentUUID uuid.UUID `db:"fulfillment_uuid" json:"fulfillmentUuid"`
	OrderItemUUID   uuid.UUID `db:"order_item_uuid" json:"orderItemUuid"`
	ProductUUID     uuid.UUID `db:"product_uuid" json:"productUuid"`
	Quantity        int       `db:"quantity" json:"quantity"`
	OrderQuantity   int       `db:"order_quantity" json:"orderQuantity"`
	ProductTitle    string    `db:"product_title" json:"productTitle"`
	PriceCents      int64     `db:"price_cents" json:"priceCents"`
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
