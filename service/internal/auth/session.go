package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	sessionKey = "coreSession"

	// AppKey identifies this service when reading app-scoped roles.
	AppKey = "marketplace"

	// Marketplace app roles.
	RoleOwner   = "marketplace.owner"
	RoleManager = "marketplace.manager"
	RoleStaff   = "marketplace.staff"

	// Platform roles leveraged for elevated access to marketplace dashboards.
	PlatformRoleAdmin   = "platform.admin"
	PlatformRoleSupport = "platform.support"
)

// Session captures the subset of Core API session data we care about in downstream services.
type Session struct {
	UUID          string
	PlatformRoles []string
	AppRoles      map[string][]string
	Roles         []string
}

func storeSession(c *fiber.Ctx, session Session) {
	c.Locals(sessionKey, session)
}

// SessionFrom extracts the current request's Core session (if any).
func SessionFrom(c *fiber.Ctx) Session {
	if v := c.Locals(sessionKey); v != nil {
		if s, ok := v.(Session); ok {
			return s
		}
	}
	return Session{}
}

// HasAppRole returns true when the user holds the provided app-scoped role for marketplace.
func HasAppRole(c *fiber.Ctx, role string) bool {
	return SessionFrom(c).hasAppRole(role)
}

// HasAnyAppRole returns true when the user holds any of the provided app roles.
func HasAnyAppRole(c *fiber.Ctx, roles ...string) bool {
	session := SessionFrom(c)
	for _, role := range roles {
		if session.hasAppRole(role) {
			return true
		}
	}
	return false
}

// HasPlatformRole returns true when the user holds the provided platform role.
func HasPlatformRole(c *fiber.Ctx, role string) bool {
	role = strings.TrimSpace(strings.ToLower(role))
	if role == "" {
		return false
	}
	session := SessionFrom(c)
	for _, candidate := range session.PlatformRoles {
		if strings.EqualFold(candidate, role) {
			return true
		}
	}
	return false
}

func (s Session) hasAppRole(role string) bool {
	role = strings.TrimSpace(strings.ToLower(role))
	if role == "" {
		return false
	}
	if len(s.AppRoles) > 0 {
		if roles := s.AppRoles[AppKey]; len(roles) > 0 {
			for _, candidate := range roles {
				if strings.EqualFold(candidate, role) {
					return true
				}
			}
		}
	}
	for _, candidate := range s.Roles {
		if strings.EqualFold(candidate, role) {
			return true
		}
	}
	return false
}
