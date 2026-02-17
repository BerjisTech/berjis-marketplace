package server

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func registerAnalyticsRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {

	// ── Sales analytics (revenue, orders, AOV time series) ──────────────
	app.Get("/v1/my/shops/:slug/analytics/sales", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		from, to, err := parseDateRange(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		granularity := strings.ToLower(strings.TrimSpace(c.Query("granularity", "day")))
		if granularity != "day" && granularity != "week" && granularity != "month" {
			granularity = "day"
		}
		truncExpr := "date_trunc('" + granularity + "', o.created_at)"

		// Aggregated totals
		type totals struct {
			TotalRevenueCents      int64 `db:"total_revenue_cents" json:"totalRevenueCents"`
			OrdersCount            int64 `db:"orders_count" json:"ordersCount"`
			AverageOrderValueCents int64 `db:"average_order_value_cents" json:"averageOrderValueCents"`
		}
		var t totals
		if err := opts.DB.Get(&t, `SELECT COALESCE(SUM(o.total_cents),0) AS total_revenue_cents,
		                                  COUNT(o.uuid) AS orders_count,
		                                  CASE WHEN COUNT(o.uuid)>0 THEN COALESCE(SUM(o.total_cents),0)/COUNT(o.uuid) ELSE 0 END AS average_order_value_cents
		                           FROM orders o
		                           WHERE o.shop_uuid=$1 AND o.created_at>=$2 AND o.created_at<$3
		                             AND o.status NOT IN ('cancelled','refunded')`, shop.UUID, from, to); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		// Time series
		type seriesPoint struct {
			Date      time.Time `db:"date" json:"date"`
			Revenue   int64     `db:"revenue" json:"revenueCents"`
			OrdersCnt int64     `db:"orders_cnt" json:"ordersCount"`
		}
		var series []seriesPoint
		if err := opts.DB.Select(&series, `SELECT `+truncExpr+` AS date,
		                                          COALESCE(SUM(o.total_cents),0) AS revenue,
		                                          COUNT(o.uuid) AS orders_cnt
		                                   FROM orders o
		                                   WHERE o.shop_uuid=$1 AND o.created_at>=$2 AND o.created_at<$3
		                                     AND o.status NOT IN ('cancelled','refunded')
		                                   GROUP BY date ORDER BY date ASC`, shop.UUID, from, to); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if series == nil {
			series = []seriesPoint{}
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"totalRevenueCents":      t.TotalRevenueCents,
			"ordersCount":            t.OrdersCount,
			"averageOrderValueCents": t.AverageOrderValueCents,
			"series":                 series,
		}})
	})

	// ── Top products by revenue ─────────────────────────────────────────
	app.Get("/v1/my/shops/:slug/analytics/top-products", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		from, to, err := parseDateRange(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
		}
		limit := 10
		if l := strings.TrimSpace(c.Query("limit")); l != "" {
			if v, parseErr := strconv.Atoi(l); parseErr == nil && v > 0 && v <= 100 {
				limit = v
			}
		}

		type topProduct struct {
			ProductUUID uuid.UUID `db:"product_uuid" json:"productUuid"`
			Title       string    `db:"title" json:"title"`
			UnitsSold   int64     `db:"units_sold" json:"unitsSold"`
			RevenueCents int64    `db:"revenue_cents" json:"revenueCents"`
		}
		var products []topProduct
		if err := opts.DB.Select(&products, `SELECT oi.product_uuid, p.title,
		                                            SUM(oi.quantity) AS units_sold,
		                                            SUM(oi.price_cents * oi.quantity) AS revenue_cents
		                                     FROM order_items oi
		                                     JOIN orders o ON o.uuid=oi.order_uuid
		                                     JOIN products p ON p.uuid=oi.product_uuid
		                                     WHERE o.shop_uuid=$1 AND o.created_at>=$2 AND o.created_at<$3
		                                       AND o.status NOT IN ('cancelled','refunded')
		                                     GROUP BY oi.product_uuid, p.title
		                                     ORDER BY revenue_cents DESC
		                                     LIMIT `+strconv.Itoa(limit), shop.UUID, from, to); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if products == nil {
			products = []topProduct{}
		}

		return c.JSON(fiber.Map{"success": true, "data": products})
	})

	// ── Customer analytics (new vs returning) ───────────────────────────
	app.Get("/v1/my/shops/:slug/analytics/customers", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		from, to, err := parseDateRange(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
		}

		// New customers: first order in this shop falls within the date range
		var newCustomers int64
		if err := opts.DB.Get(&newCustomers, `SELECT COUNT(DISTINCT o.user_uuid)
		                                       FROM orders o
		                                       WHERE o.shop_uuid=$1 AND o.created_at>=$2 AND o.created_at<$3
		                                         AND o.status NOT IN ('cancelled','refunded')
		                                         AND o.user_uuid NOT IN (
		                                           SELECT DISTINCT o2.user_uuid FROM orders o2
		                                           WHERE o2.shop_uuid=$1 AND o2.created_at<$2
		                                             AND o2.status NOT IN ('cancelled','refunded')
		                                         )`, shop.UUID, from, to); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		// Total unique customers in range
		var totalCustomers int64
		if err := opts.DB.Get(&totalCustomers, `SELECT COUNT(DISTINCT o.user_uuid)
		                                         FROM orders o
		                                         WHERE o.shop_uuid=$1 AND o.created_at>=$2 AND o.created_at<$3
		                                           AND o.status NOT IN ('cancelled','refunded')`, shop.UUID, from, to); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		returning := totalCustomers - newCustomers
		if returning < 0 {
			returning = 0
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"newCustomers":       newCustomers,
			"returningCustomers": returning,
			"totalCustomers":     totalCustomers,
		}})
	})

	// ── Conversion funnel from analytics_events ─────────────────────────
	app.Get("/v1/my/shops/:slug/analytics/conversions", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		from, to, err := parseDateRange(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": err.Error()})
		}

		type eventCount struct {
			EventName string `db:"event_name"`
			Count     int64  `db:"count"`
		}
		var counts []eventCount
		if err := opts.DB.Select(&counts, `SELECT event_name, COUNT(1) AS count
		                                    FROM analytics_events
		                                    WHERE shop_uuid=$1 AND occurred_at>=$2 AND occurred_at<$3
		                                      AND event_name IN ('product_view','add_to_cart','checkout_started','checkout_completed')
		                                    GROUP BY event_name`, shop.UUID, from, to); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		m := map[string]int64{}
		for _, ec := range counts {
			m[ec.EventName] = ec.Count
		}

		productViews := m["product_view"]
		addToCarts := m["add_to_cart"]
		checkoutsStarted := m["checkout_started"]
		checkoutsCompleted := m["checkout_completed"]

		var conversionRate float64
		if productViews > 0 {
			conversionRate = float64(checkoutsCompleted) / float64(productViews) * 100.0
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"productViews":       productViews,
			"addToCarts":         addToCarts,
			"checkoutsStarted":   checkoutsStarted,
			"checkoutsCompleted": checkoutsCompleted,
			"conversionRate":     conversionRate,
		}})
	})

	// ── Recent analytics events (for live view) ─────────────────────────
	app.Get("/v1/my/shops/:slug/analytics/events", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsView(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		limit := 50
		if l := strings.TrimSpace(c.Query("limit")); l != "" {
			if v, parseErr := strconv.Atoi(l); parseErr == nil && v > 0 && v <= 200 {
				limit = v
			}
		}

		type analyticsEvent struct {
			UUID       uuid.UUID        `db:"uuid" json:"uuid"`
			EventName  string           `db:"event_name" json:"eventName"`
			SessionID  *uuid.UUID       `db:"session_id" json:"sessionId,omitempty"`
			Payload    *json.RawMessage `db:"payload" json:"payload,omitempty"`
			OccurredAt time.Time        `db:"occurred_at" json:"occurredAt"`
		}
		var events []analyticsEvent
		if err := opts.DB.Select(&events, `SELECT uuid, event_name, session_id, payload, occurred_at
		                                    FROM analytics_events
		                                    WHERE shop_uuid=$1
		                                    ORDER BY occurred_at DESC
		                                    LIMIT `+strconv.Itoa(limit), shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if events == nil {
			events = []analyticsEvent{}
		}

		return c.JSON(fiber.Map{"success": true, "data": events})
	})

	// ── Public: track storefront events ─────────────────────────────────
	app.Post("/v1/analytics/event", func(c *fiber.Ctx) error {
		var body struct {
			ShopUUID  string           `json:"shopUuid"`
			EventType string           `json:"eventType"`
			Metadata  *json.RawMessage `json:"metadata"`
			SessionID string           `json:"sessionId"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		shopUUID, err := uuid.Parse(strings.TrimSpace(body.ShopUUID))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "invalid shopUuid"})
		}
		eventType := strings.TrimSpace(body.EventType)
		if eventType == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "message": "eventType required"})
		}

		id := uuid.New()
		var sessionValue interface{}
		if sid := strings.TrimSpace(body.SessionID); sid != "" {
			if parsed, err := uuid.Parse(sid); err == nil {
				sessionValue = parsed
			}
		}

		var payloadValue interface{}
		if body.Metadata != nil {
			payloadValue = string(*body.Metadata)
		}

		if _, err := opts.DB.Exec(`INSERT INTO analytics_events(uuid, shop_uuid, session_id, event_name, payload, occurred_at)
		                            VALUES($1,$2,$3,$4,$5,now())`,
			id, shopUUID, sessionValue, eventType, payloadValue); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"uuid": id}})
	})
}

// parseDateRange extracts "from" and "to" query parameters as time.Time values.
// Defaults to last 30 days if not supplied.
func parseDateRange(c *fiber.Ctx) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	toDate := now.AddDate(0, 0, 1).Truncate(24 * time.Hour) // end of today
	fromDate := now.AddDate(0, 0, -30).Truncate(24 * time.Hour)

	if f := strings.TrimSpace(c.Query("from")); f != "" {
		parsed, err := time.Parse("2006-01-02", f)
		if err != nil {
			return time.Time{}, time.Time{}, fiber.NewError(fiber.StatusBadRequest, "invalid from date, expected YYYY-MM-DD")
		}
		fromDate = parsed
	}
	if t := strings.TrimSpace(c.Query("to")); t != "" {
		parsed, err := time.Parse("2006-01-02", t)
		if err != nil {
			return time.Time{}, time.Time{}, fiber.NewError(fiber.StatusBadRequest, "invalid to date, expected YYYY-MM-DD")
		}
		toDate = parsed.AddDate(0, 0, 1) // inclusive end
	}
	return fromDate, toDate, nil
}
