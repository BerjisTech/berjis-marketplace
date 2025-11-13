package server

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	srvAuth "github.com/berjistech/berjis-ecosystem/marketplace/service/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func registerTeamRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
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

}
