package server

import (
	"database/sql"
	"errors"
	"math"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func registerTaxRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
	// GET tax settings + zones for a shop
	app.Get("/v1/my/shops/:slug/tax-settings", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var settings TaxSettings
		err = opts.DB.Get(&settings, `SELECT uuid, shop_uuid, auto_calculate, default_rate_percent, prices_include_tax, created_at, updated_at
		                               FROM tax_settings WHERE shop_uuid=$1`, shop.UUID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if errors.Is(err, sql.ErrNoRows) {
			settings = TaxSettings{ShopUUID: shop.UUID}
		}

		var zones []TaxZone
		if err := opts.DB.Select(&zones, `SELECT uuid, shop_uuid, country_code, region_code, rate_percent, name, created_at, updated_at
		                                   FROM tax_zones WHERE shop_uuid=$1 ORDER BY country_code, region_code`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}
		if zones == nil {
			zones = []TaxZone{}
		}

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"settings": settings,
			"zones":    zones,
		}})
	})

	// PUT tax settings for a shop
	app.Put("/v1/my/shops/:slug/tax-settings", requireAuth, func(c *fiber.Ctx) error {
		slug := c.Params("slug")
		shop, role, err := ensureShopAccess(c, opts.DB, slug)
		if err != nil {
			return respondWithError(c, err)
		}
		if !teamRoleAllowsManagement(role) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"success": false, "message": "insufficient permissions"})
		}

		var body struct {
			AutoCalculate      bool    `json:"autoCalculate"`
			DefaultRatePercent float64 `json:"defaultRatePercent"`
			PricesIncludeTax   bool    `json:"pricesIncludeTax"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}

		_, err = opts.DB.Exec(`INSERT INTO tax_settings(uuid, shop_uuid, auto_calculate, default_rate_percent, prices_include_tax)
		                       VALUES($1,$2,$3,$4,$5)
		                       ON CONFLICT (shop_uuid) DO UPDATE SET
		                         auto_calculate=EXCLUDED.auto_calculate,
		                         default_rate_percent=EXCLUDED.default_rate_percent,
		                         prices_include_tax=EXCLUDED.prices_include_tax,
		                         updated_at=now()`,
			uuid.New(), shop.UUID, body.AutoCalculate, body.DefaultRatePercent, body.PricesIncludeTax)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true, "message": "tax settings saved"})
	})

	// PUT tax zones (full replacement) for a shop
	app.Put("/v1/my/shops/:slug/tax-zones", requireAuth, func(c *fiber.Ctx) error {
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
				CountryCode string  `json:"countryCode"`
				RegionCode  string  `json:"regionCode"`
				RatePercent float64 `json:"ratePercent"`
				Name        string  `json:"name"`
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

		if _, err := tx.Exec(`DELETE FROM tax_zones WHERE shop_uuid=$1`, shop.UUID); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		for _, z := range body.Zones {
			countryCode := strings.TrimSpace(strings.ToUpper(z.CountryCode))
			if countryCode == "" {
				continue
			}
			regionCode := strings.TrimSpace(strings.ToUpper(z.RegionCode))
			name := strings.TrimSpace(z.Name)
			if _, err := tx.Exec(`INSERT INTO tax_zones(uuid, shop_uuid, country_code, region_code, rate_percent, name)
			                      VALUES($1,$2,$3,$4,$5,$6)`,
				uuid.New(), shop.UUID, countryCode, regionCode, z.RatePercent, name); err != nil {
				return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
			}
		}

		if err := tx.Commit(); err != nil {
			return c.Status(500).JSON(fiber.Map{"success": false, "message": "db error"})
		}

		return c.JSON(fiber.Map{"success": true, "message": "tax zones saved"})
	})

	// POST calculate tax (public endpoint)
	app.Post("/v1/tax/calculate", func(c *fiber.Ctx) error {
		var body struct {
			ShopUUID string `json:"shopUuid"`
			Country  string `json:"country"`
			Region   string `json:"region"`
			Items    []struct {
				ProductUUID string `json:"productUuid"`
				PriceCents  int64  `json:"priceCents"`
				Quantity    int    `json:"quantity"`
			} `json:"items"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid body"})
		}
		shopUUID, err := uuid.Parse(strings.TrimSpace(body.ShopUUID))
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"success": false, "message": "invalid shop id"})
		}
		country := strings.TrimSpace(strings.ToUpper(body.Country))
		region := strings.TrimSpace(strings.ToUpper(body.Region))

		ratePercent := resolveTaxRate(opts, shopUUID, country, region)

		// Compute taxable amount, excluding tax-exempt products
		taxableAmount := int64(0)
		for _, item := range body.Items {
			productUUID, err := uuid.Parse(strings.TrimSpace(item.ProductUUID))
			if err != nil {
				continue
			}
			var exempt bool
			if err := opts.DB.Get(&exempt, `SELECT is_tax_exempt FROM products WHERE uuid=$1`, productUUID); err != nil {
				exempt = false
			}
			if !exempt {
				qty := item.Quantity
				if qty < 1 {
					qty = 1
				}
				taxableAmount += int64(qty) * item.PriceCents
			}
		}

		taxCents := int64(math.Round(float64(taxableAmount) * ratePercent / 100.0))

		return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
			"taxCents":       taxCents,
			"ratePercent":    ratePercent,
			"taxableAmount":  taxableAmount,
		}})
	})
}

// resolveTaxRate looks up the tax rate for a shop given country/region.
// It checks tax_zones first (region-specific, then country-level), then falls
// back to the shop's default_rate_percent in tax_settings, then to the global
// TaxRatePercent from Options.
func resolveTaxRate(opts Options, shopUUID uuid.UUID, country, region string) float64 {
	// Try region-specific zone first
	if region != "" {
		var rate float64
		err := opts.DB.Get(&rate, `SELECT rate_percent FROM tax_zones WHERE shop_uuid=$1 AND country_code=$2 AND region_code=$3`,
			shopUUID, country, region)
		if err == nil {
			return rate
		}
	}

	// Try country-level zone (empty region)
	if country != "" {
		var rate float64
		err := opts.DB.Get(&rate, `SELECT rate_percent FROM tax_zones WHERE shop_uuid=$1 AND country_code=$2 AND region_code=''`,
			shopUUID, country)
		if err == nil {
			return rate
		}
	}

	// Fall back to shop default rate
	var defaultRate float64
	err := opts.DB.Get(&defaultRate, `SELECT default_rate_percent FROM tax_settings WHERE shop_uuid=$1`, shopUUID)
	if err == nil && defaultRate > 0 {
		return defaultRate
	}

	// Fall back to global rate
	return opts.TaxRatePercent
}
