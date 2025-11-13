package server

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	srvAuth "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/auth"
)

func registerOrderRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
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

	app.Post("/v1/my/shops/:slug/checkouts/:id/recoveries", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		cartParam := strings.TrimSpace(c.Params("id"))
		if cartParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid cart id"})
		}
		cartID, err := uuid.Parse(cartParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid cart id"})
		}
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var body struct {
			Email          string `json:"email"`
			ExpiresInHours int    `json:"expiresInHours"`
		}
		if err := c.BodyParser(&body); err != nil && err != io.EOF {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		if body.ExpiresInHours <= 0 {
			body.ExpiresInHours = 72
		}

		const singleCartQuery = `WITH cart_summary AS (
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
                                   AND cs.cart_uuid=$2
                                   AND cs.subtotal_cents > 0
                                 LIMIT 1`

		var checkout AbandonedCheckout
		if err := opts.DB.Get(&checkout, singleCartQuery, shop.UUID, cartID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "cart not eligible for recovery"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		contactEmail := strings.TrimSpace(body.Email)
		if contactEmail == "" {
			contactEmail = strings.TrimSpace(checkout.CustomerEmail)
		}
		token := uuid.NewString()
		recoveryURL := ""
		if base := strings.TrimSpace(strings.Split(opts.AllowedOrigins, ",")[0]); base != "" && base != "*" {
			recoveryURL = strings.TrimSuffix(base, "/") + "/checkout/recover?token=" + token
		}
		expiresAt := time.Now().Add(time.Duration(body.ExpiresInHours) * time.Hour)
		recoveryID := uuid.New()
		createdBy := uuidPtrFromString(srvAuth.UserID(c))

		var customerUUIDValue any
		if checkout.CustomerUUID != nil {
			customerUUIDValue = *checkout.CustomerUUID
		}
		var recoveryURLValue any
		if recoveryURL != "" {
			recoveryURLValue = recoveryURL
		}
		var createdByValue any
		if createdBy != nil {
			createdByValue = *createdBy
		}

		if _, err := opts.DB.Exec(`INSERT INTO cart_recoveries(uuid, cart_uuid, shop_uuid, user_uuid, customer_uuid, customer_email, token, status, recovery_url, sent_at, clicked_at, converted_order_uuid, expires_at, created_by)
                                   VALUES($1,$2,$3,$4,$5,$6,$7,'pending',$8,NULL,NULL,NULL,$9,$10)`,
			recoveryID, checkout.CartUUID, checkout.ShopUUID, checkout.UserUUID, customerUUIDValue, contactEmail, token, recoveryURLValue, expiresAt, createdByValue); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		response := fiber.Map{
			"token":     token,
			"expiresAt": expiresAt,
		}
		if recoveryURL != "" {
			response["recoveryUrl"] = recoveryURL
		}
		return c.JSON(fiber.Map{"success": true, "data": response})
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

	app.Get("/v1/orders/:id/items", requireAuth, func(c *fiber.Ctx) error {
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
		var items []OrderLineItem
		if err := opts.DB.Select(&items, `SELECT oi.uuid,
                                                 oi.product_uuid,
                                                 p.title,
                                                 oi.quantity,
                                                 oi.price_cents
                                          FROM order_items oi
                                          JOIN products p ON p.uuid=oi.product_uuid
                                          WHERE oi.order_uuid=$1
                                          ORDER BY oi.created_at ASC`, orderID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": items})
	})

	app.Get("/v1/my/shops/:slug/returns", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}
		var returns []OrderReturn
		if err := opts.DB.Select(&returns, `SELECT uuid,
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
                                            WHERE shop_uuid=$1
                                            ORDER BY created_at DESC
                                            LIMIT 200`, shop.UUID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if len(returns) > 0 {
			ids := make([]uuid.UUID, 0, len(returns))
			for _, ret := range returns {
				ids = append(ids, ret.UUID)
			}
			itemsMap, err := loadOrderReturnItems(opts.DB, ids)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			for i := range returns {
				if items, ok := itemsMap[returns[i].UUID]; ok {
					returns[i].Items = items
				} else {
					returns[i].Items = []OrderReturnItem{}
				}
			}
		}
		return c.JSON(fiber.Map{"success": true, "data": returns})
	})

	app.Post("/v1/orders/:id/returns", requireAuth, func(c *fiber.Ctx) error {
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
			Items []struct {
				OrderItemUUID string `json:"orderItemUuid"`
				Quantity      int    `json:"quantity"`
				Reason        string `json:"reason"`
				Condition     string `json:"condition"`
			} `json:"items"`
			Reason            string `json:"reason"`
			Notes             string `json:"notes"`
			RefundAmountCents int64  `json:"refundAmountCents"`
			Restock           bool   `json:"restock"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		if len(body.Items) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "at least one return item is required"})
		}

		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		var orderRow struct {
			ShopUUID uuid.UUID `db:"shop_uuid"`
			UserUUID uuid.UUID `db:"user_uuid"`
		}
		if err := tx.Get(&orderRow, `SELECT shop_uuid, user_uuid FROM orders WHERE uuid=$1 FOR UPDATE`, orderID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "order not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if orderRow.ShopUUID != meta.ShopUUID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var orderItems []struct {
			UUID        uuid.UUID `db:"uuid"`
			ProductUUID uuid.UUID `db:"product_uuid"`
			Quantity    int       `db:"quantity"`
		}
		if err := tx.Select(&orderItems, `SELECT uuid, product_uuid, quantity FROM order_items WHERE order_uuid=$1`, orderID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if len(orderItems) == 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "order has no items"})
		}
		orderItemMap := make(map[uuid.UUID]struct {
			Product  uuid.UUID
			Quantity int
		}, len(orderItems))
		for _, itm := range orderItems {
			orderItemMap[itm.UUID] = struct {
				Product  uuid.UUID
				Quantity int
			}{
				Product:  itm.ProductUUID,
				Quantity: itm.Quantity,
			}
		}
		existingReturns := make(map[uuid.UUID]int)
		var existingRows []struct {
			OrderItemUUID uuid.UUID `db:"order_item_uuid"`
			Quantity      int       `db:"quantity"`
		}
		if err := tx.Select(&existingRows, `SELECT ri.order_item_uuid, SUM(ri.quantity) AS quantity
                                              FROM order_return_items ri
                                              JOIN order_returns r ON r.uuid=ri.return_uuid
                                              WHERE r.order_uuid=$1 AND r.status <> 'rejected'
                                              GROUP BY ri.order_item_uuid`, orderID); err == nil {
			for _, row := range existingRows {
				existingReturns[row.OrderItemUUID] = row.Quantity
			}
		}

		type preparedReturnItem struct {
			OrderItemUUID uuid.UUID
			ProductUUID   uuid.UUID
			Quantity      int
			Reason        string
			Condition     string
		}
		preparedItems := make([]preparedReturnItem, 0, len(body.Items))
		for idx, raw := range body.Items {
			itemID, err := uuid.Parse(strings.TrimSpace(raw.OrderItemUUID))
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("invalid orderItemUuid at position %d", idx)})
			}
			entry, ok := orderItemMap[itemID]
			if !ok {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": fmt.Sprintf("order item not found (%s)", itemID)})
			}
			if raw.Quantity <= 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "return quantities must be greater than zero"})
			}
			already := existingReturns[itemID]
			if already+raw.Quantity > entry.Quantity {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "return quantity exceeds original quantity"})
			}
			reason := strings.TrimSpace(raw.Reason)
			condition := strings.TrimSpace(raw.Condition)
			preparedItems = append(preparedItems, preparedReturnItem{
				OrderItemUUID: itemID,
				ProductUUID:   entry.Product,
				Quantity:      raw.Quantity,
				Reason:        reason,
				Condition:     condition,
			})
		}

		var customerUUID *uuid.UUID
		if err := tx.Get(&customerUUID, `SELECT uuid FROM customers WHERE shop_uuid=$1 AND user_uuid=$2 LIMIT 1`, orderRow.ShopUUID, orderRow.UserUUID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		returnID := uuid.New()
		reason := strings.TrimSpace(body.Reason)
		notes := strings.TrimSpace(body.Notes)
		refundAmount := body.RefundAmountCents
		if refundAmount < 0 {
			refundAmount = 0
		}
		var reasonValue any
		if reason != "" {
			reasonValue = reason
		}
		var notesValue any
		if notes != "" {
			notesValue = notes
		}
		var customerValue any
		if customerUUID != nil {
			customerValue = *customerUUID
		}
		requestedBy := uuidPtrFromString(srvAuth.UserID(c))
		var requestedByValue any
		if requestedBy != nil {
			requestedByValue = *requestedBy
		}

		if _, err := tx.Exec(`INSERT INTO order_returns(uuid, order_uuid, shop_uuid, customer_uuid, status, reason, notes, requested_by, restock, refund_amount_cents)
                               VALUES($1,$2,$3,$4,'requested',$5,$6,$7,$8,$9)`,
			returnID, orderID, orderRow.ShopUUID, customerValue, reasonValue, notesValue, requestedByValue, body.Restock, refundAmount); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		for _, item := range preparedItems {
			var reasonValue any
			if item.Reason != "" {
				reasonValue = item.Reason
			}
			var conditionValue any
			if item.Condition != "" {
				conditionValue = item.Condition
			}
			if _, err := tx.Exec(`INSERT INTO order_return_items(uuid, return_uuid, order_item_uuid, product_uuid, quantity, reason, condition)
                                   VALUES($1,$2,$3,$4,$5,$6,$7)`,
				uuid.New(), returnID, item.OrderItemUUID, item.ProductUUID, item.Quantity, reasonValue, conditionValue); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}

		eventMeta := map[string]any{
			"returnUuid": returnID.String(),
			"items":      len(preparedItems),
			"restock":    body.Restock,
		}
		if refundAmount > 0 {
			eventMeta["refundAmountCents"] = refundAmount
		}
		if reason != "" {
			eventMeta["reason"] = reason
		}
		if _, err := recordOrderEvent(tx, orderID, "order.return_requested", fmt.Sprintf("Return requested (%d items)", len(preparedItems)), requestedBy, eventMeta); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "failed to record order event"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		ret, err := loadOrderReturn(opts.DB, returnID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": ret})
	})

	app.Patch("/v1/returns/:id", requireAuth, func(c *fiber.Ctx) error {
		returnParam := strings.TrimSpace(c.Params("id"))
		if returnParam == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid return id"})
		}
		returnID, err := uuid.Parse(returnParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid return id"})
		}
		var body struct {
			Status            string  `json:"status"`
			Notes             *string `json:"notes"`
			RefundAmountCents *int64  `json:"refundAmountCents"`
			Restock           *bool   `json:"restock"`
		}
		if err := c.BodyParser(&body); err != nil && err != io.EOF {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}

		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		var retRow struct {
			OrderReturn
			ShopSlug string `db:"slug"`
		}
		if err := tx.Get(&retRow, `SELECT r.uuid,
                                           r.order_uuid,
                                           r.shop_uuid,
                                           r.customer_uuid,
                                           r.status,
                                           r.reason,
                                           r.notes,
                                           r.requested_by,
                                           r.processed_by,
                                           r.restock,
                                           r.restocked_at,
                                           r.refund_amount_cents,
                                           r.created_at,
                                           r.updated_at,
                                           s.slug
                                    FROM order_returns r
                                    JOIN shops s ON s.uuid=r.shop_uuid
                                    WHERE r.uuid=$1
                                    FOR UPDATE`, returnID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"success": false, "message": "return not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		_, role, err := ensureShopAccess(c, opts.DB, retRow.ShopSlug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		sets := []string{"updated_at=now()"}
		args := []any{returnID}
		argPos := 2
		statusChanged := false
		if strings.TrimSpace(body.Status) != "" {
			normalized := normalizeReturnStatus(body.Status)
			if normalized != retRow.Status {
				sets = append(sets, fmt.Sprintf("status=$%d", argPos))
				args = append(args, normalized)
				argPos++
				statusChanged = true
				if normalized == "restocked" {
					sets = append(sets, "restocked_at=now()")
				}
				if retRow.ProcessedBy == nil {
					if processed := uuidPtrFromString(srvAuth.UserID(c)); processed != nil {
						sets = append(sets, fmt.Sprintf("processed_by=$%d", argPos))
						args = append(args, *processed)
						argPos++
					}
				}
			}
		}
		if body.Notes != nil {
			note := strings.TrimSpace(*body.Notes)
			if note == "" {
				sets = append(sets, "notes=NULL")
			} else {
				sets = append(sets, fmt.Sprintf("notes=$%d", argPos))
				args = append(args, note)
				argPos++
			}
		}
		if body.RefundAmountCents != nil {
			value := *body.RefundAmountCents
			if value < 0 {
				value = 0
			}
			sets = append(sets, fmt.Sprintf("refund_amount_cents=$%d", argPos))
			args = append(args, value)
			argPos++
		}
		if body.Restock != nil {
			sets = append(sets, fmt.Sprintf("restock=$%d", argPos))
			args = append(args, *body.Restock)
			argPos++
		}

		if len(sets) == 1 {
			return c.JSON(fiber.Map{"success": true, "data": retRow.OrderReturn})
		}

		query := fmt.Sprintf("UPDATE order_returns SET %s WHERE uuid=$1", strings.Join(sets, ", "))
		if _, err := tx.Exec(query, args...); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		updatedReturn, err := loadOrderReturn(tx, returnID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		var restockedProducts []uuid.UUID
		if updatedReturn.Status == "restocked" && updatedReturn.Restock {
			var touched []uuid.UUID
			var restocked OrderReturn
			restocked, touched, err = restockReturnTx(tx, updatedReturn, srvAuth.UserID(c))
			if err != nil {
				return respondWithError(c, err)
			}
			updatedReturn = restocked
			restockedProducts = touched
		}

		eventMeta := map[string]any{
			"returnUuid": updatedReturn.UUID.String(),
			"status":     updatedReturn.Status,
		}
		if body.RefundAmountCents != nil {
			eventMeta["refundAmountCents"] = *body.RefundAmountCents
		}
		if body.Restock != nil {
			eventMeta["restock"] = *body.Restock
		}
		if body.Notes != nil {
			eventMeta["notes"] = strings.TrimSpace(*body.Notes)
		}
		if statusChanged {
			eventMeta["previousStatus"] = retRow.Status
		}
		eventType := "order.return_updated"
		eventMessage := fmt.Sprintf("Return updated (%s)", strings.ReplaceAll(updatedReturn.Status, "_", " "))
		if _, err := recordOrderEvent(tx, updatedReturn.OrderUUID, eventType, eventMessage, uuidPtrFromString(srvAuth.UserID(c)), eventMeta); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "failed to record order event"})
		}

		if err := tx.Commit(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		if len(restockedProducts) > 0 {
			for _, productID := range restockedProducts {
				if syncErr := syncProductStockFromInventory(opts.DB, productID); syncErr != nil {
					log.Printf("sync product stock failed for %s: %v", productID, syncErr)
				}
			}
		}

		finalReturn, err := loadOrderReturn(opts.DB, updatedReturn.UUID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		return c.JSON(fiber.Map{"success": true, "data": finalReturn})
	})

}
