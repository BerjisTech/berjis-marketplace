package server

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	pq "github.com/lib/pq"
)

type ShippingZone struct {
	UUID           uuid.UUID      `db:"uuid" json:"uuid"`
	ShopUUID       uuid.UUID      `db:"shop_uuid" json:"shopUuid"`
	Name           string         `db:"name" json:"name"`
	Countries      pq.StringArray `db:"countries" json:"countries"`
	IsRestOfWorld  bool           `db:"is_rest_of_world" json:"isRestOfWorld"`
	CreatedAt      string         `db:"created_at" json:"createdAt"`
	UpdatedAt      string         `db:"updated_at" json:"updatedAt"`
	Rates          []ShippingRate `db:"-" json:"rates"`
}

type ShippingRate struct {
	UUID            uuid.UUID `db:"uuid" json:"uuid"`
	ZoneUUID        uuid.UUID `db:"zone_uuid" json:"zoneUuid"`
	ShopUUID        uuid.UUID `db:"shop_uuid" json:"shopUuid"`
	Name            string    `db:"name" json:"name"`
	PriceCents      int64     `db:"price_cents" json:"priceCents"`
	MinWeightGrams  *int      `db:"min_weight_grams" json:"minWeightGrams,omitempty"`
	MaxWeightGrams  *int      `db:"max_weight_grams" json:"maxWeightGrams,omitempty"`
	MinOrderCents   *int64    `db:"min_order_cents" json:"minOrderCents,omitempty"`
	MaxOrderCents   *int64    `db:"max_order_cents" json:"maxOrderCents,omitempty"`
	FreeAboveCents  *int64    `db:"free_above_cents" json:"freeAboveCents,omitempty"`
	EstimatedDaysMin *int     `db:"estimated_days_min" json:"estimatedDaysMin,omitempty"`
	EstimatedDaysMax *int     `db:"estimated_days_max" json:"estimatedDaysMax,omitempty"`
	CreatedAt       string    `db:"created_at" json:"createdAt"`
	UpdatedAt       string    `db:"updated_at" json:"updatedAt"`
}

func registerShippingRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
	// Public: calculate shipping rates for an address + cart
	app.Post("/v1/shipping/rates", func(c *fiber.Ctx) error {
		var body struct {
			ShopUUID     string `json:"shopUuid"`
			Country      string `json:"country"`
			SubtotalCents int64 `json:"subtotalCents"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		shopUUID, err := uuid.Parse(strings.TrimSpace(body.ShopUUID))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid shop id"})
		}
		country := strings.TrimSpace(strings.ToUpper(body.Country))
		if country == "" {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "country required"})
		}

		rates, err := calculateShippingRates(opts, shopUUID, country, body.SubtotalCents)
		if err != nil {
			return respondWithError(c, err)
		}

		return c.JSON(fiber.Map{"success": true, "data": rates})
	})

	// Shop owner: manage shipping zones
	app.Get("/v1/my/shops/:slug/shipping-zones", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var zones []ShippingZone
		if err := opts.DB.Select(&zones, `SELECT uuid, shop_uuid, name, countries, is_rest_of_world, created_at, updated_at
		                                  FROM shipping_zones WHERE shop_uuid=$1 ORDER BY is_rest_of_world, name`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		for i := range zones {
			var rates []ShippingRate
			if err := opts.DB.Select(&rates, `SELECT uuid, zone_uuid, shop_uuid, name, price_cents, min_weight_grams, max_weight_grams, min_order_cents, max_order_cents, free_above_cents, estimated_days_min, estimated_days_max, created_at, updated_at
			                                   FROM shipping_rates WHERE zone_uuid=$1 ORDER BY price_cents`, zones[i].UUID); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
			zones[i].Rates = rates
		}

		return c.JSON(fiber.Map{"success": true, "data": zones})
	})

	app.Put("/v1/my/shops/:slug/shipping-zones", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var body struct {
			Zones []struct {
				UUID          string   `json:"uuid"`
				Name          string   `json:"name"`
				Countries     []string `json:"countries"`
				IsRestOfWorld bool     `json:"isRestOfWorld"`
				Rates         []struct {
					UUID            string `json:"uuid"`
					Name            string `json:"name"`
					PriceCents      int64  `json:"priceCents"`
					MinWeightGrams  *int   `json:"minWeightGrams"`
					MaxWeightGrams  *int   `json:"maxWeightGrams"`
					MinOrderCents   *int64 `json:"minOrderCents"`
					MaxOrderCents   *int64 `json:"maxOrderCents"`
					FreeAboveCents  *int64 `json:"freeAboveCents"`
					EstimatedDaysMin *int  `json:"estimatedDaysMin"`
					EstimatedDaysMax *int  `json:"estimatedDaysMax"`
				} `json:"rates"`
			} `json:"zones"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}

		tx, err := opts.DB.Beginx()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		defer tx.Rollback()

		// Delete existing zones and rates for this shop (full replacement)
		if _, err := tx.Exec(`DELETE FROM shipping_rates WHERE shop_uuid=$1`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if _, err := tx.Exec(`DELETE FROM shipping_zones WHERE shop_uuid=$1`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		for _, zone := range body.Zones {
			zoneID := uuid.New()
			countries := make([]string, 0, len(zone.Countries))
			for _, c := range zone.Countries {
				c = strings.TrimSpace(strings.ToUpper(c))
				if c != "" {
					countries = append(countries, c)
				}
			}

			if _, err := tx.Exec(`INSERT INTO shipping_zones(uuid, shop_uuid, name, countries, is_rest_of_world)
			                      VALUES($1,$2,$3,$4,$5)`,
				zoneID, shop.UUID, strings.TrimSpace(zone.Name), pq.Array(countries), zone.IsRestOfWorld); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}

			for _, rate := range zone.Rates {
				if _, err := tx.Exec(`INSERT INTO shipping_rates(uuid, zone_uuid, shop_uuid, name, price_cents, min_weight_grams, max_weight_grams, min_order_cents, max_order_cents, free_above_cents, estimated_days_min, estimated_days_max)
				                      VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
					uuid.New(), zoneID, shop.UUID, strings.TrimSpace(rate.Name), rate.PriceCents,
					rate.MinWeightGrams, rate.MaxWeightGrams, rate.MinOrderCents, rate.MaxOrderCents,
					rate.FreeAboveCents, rate.EstimatedDaysMin, rate.EstimatedDaysMax); err != nil {
					return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true, "message": "shipping zones saved"})
	})
}

type calculatedShippingRate struct {
	UUID           uuid.UUID `json:"uuid"`
	ZoneName       string    `json:"zoneName"`
	RateName       string    `json:"rateName"`
	PriceCents     int64     `json:"priceCents"`
	FreeAboveCents *int64    `json:"freeAboveCents,omitempty"`
	EstimatedDays  string    `json:"estimatedDays,omitempty"`
	IsFree         bool      `json:"isFree"`
}

func calculateShippingRates(opts Options, shopUUID uuid.UUID, country string, subtotalCents int64) ([]calculatedShippingRate, error) {
	// Find zones matching the country
	var zones []ShippingZone
	if err := opts.DB.Select(&zones, `SELECT uuid, shop_uuid, name, countries, is_rest_of_world, created_at, updated_at
	                                  FROM shipping_zones
	                                  WHERE shop_uuid=$1 AND ($2=ANY(countries) OR is_rest_of_world=true)
	                                  ORDER BY is_rest_of_world`, shopUUID, country); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "db error")
	}

	if len(zones) == 0 {
		// No shipping zones configured — fall back to flat rate
		return []calculatedShippingRate{
			{
				UUID:       uuid.Nil,
				ZoneName:   "Standard",
				RateName:   "Flat rate",
				PriceCents: opts.ShippingFlatCents,
				IsFree:     false,
			},
		}, nil
	}

	var result []calculatedShippingRate
	for _, zone := range zones {
		var rates []ShippingRate
		if err := opts.DB.Select(&rates, `SELECT uuid, zone_uuid, shop_uuid, name, price_cents, min_weight_grams, max_weight_grams, min_order_cents, max_order_cents, free_above_cents, estimated_days_min, estimated_days_max, created_at, updated_at
		                                   FROM shipping_rates WHERE zone_uuid=$1 ORDER BY price_cents`, zone.UUID); err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "db error")
		}

		for _, rate := range rates {
			if rate.MinOrderCents != nil && subtotalCents < *rate.MinOrderCents {
				continue
			}
			if rate.MaxOrderCents != nil && subtotalCents > *rate.MaxOrderCents {
				continue
			}

			priceCents := rate.PriceCents
			isFree := false
			if rate.FreeAboveCents != nil && subtotalCents >= *rate.FreeAboveCents {
				priceCents = 0
				isFree = true
			}

			estimatedDays := ""
			if rate.EstimatedDaysMin != nil && rate.EstimatedDaysMax != nil {
				estimatedDays = formatDaysRange(*rate.EstimatedDaysMin, *rate.EstimatedDaysMax)
			} else if rate.EstimatedDaysMin != nil {
				estimatedDays = formatDaysRange(*rate.EstimatedDaysMin, *rate.EstimatedDaysMin)
			}

			result = append(result, calculatedShippingRate{
				UUID:           rate.UUID,
				ZoneName:       zone.Name,
				RateName:       rate.Name,
				PriceCents:     priceCents,
				FreeAboveCents: rate.FreeAboveCents,
				EstimatedDays:  estimatedDays,
				IsFree:         isFree,
			})
		}
	}

	if len(result) == 0 {
		return []calculatedShippingRate{
			{
				UUID:       uuid.Nil,
				ZoneName:   "Standard",
				RateName:   "Flat rate",
				PriceCents: opts.ShippingFlatCents,
				IsFree:     false,
			},
		}, nil
	}

	return result, nil
}

func lookupShippingRate(db interface {
	Get(dest interface{}, query string, args ...interface{}) error
}, rateUUID uuid.UUID) (*ShippingRate, error) {
	var rate ShippingRate
	if err := db.Get(&rate, `SELECT uuid, zone_uuid, shop_uuid, name, price_cents, min_weight_grams, max_weight_grams, min_order_cents, max_order_cents, free_above_cents, estimated_days_min, estimated_days_max, created_at, updated_at
	                         FROM shipping_rates WHERE uuid=$1`, rateUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fiber.NewError(fiber.StatusBadRequest, "invalid shipping rate")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "db error")
	}
	return &rate, nil
}

func formatDaysRange(min, max int) string {
	if min == max {
		if min == 1 {
			return "1 business day"
		}
		return fmt.Sprintf("%d business days", min)
	}
	return fmt.Sprintf("%d-%d business days", min, max)
}
