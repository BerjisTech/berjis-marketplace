package server

import (
	srvAuth "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"strings"
)

func registerProfileRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
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

}
