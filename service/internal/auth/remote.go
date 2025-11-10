package auth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	coreauth "github.com/berjistech/berjis-ecosystem/shared/coreauth"
)

type Options struct {
	CoreAPIBase string
	HTTPClient  *http.Client
	Verifier    *coreauth.Verifier
}

const userKey = "userID"

func Middleware(opts Options) fiber.Handler {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	return func(c *fiber.Ctx) error {
		token := ""
		if authz := c.Get("Authorization"); strings.HasPrefix(strings.ToLower(authz), "bearer ") {
			token = strings.TrimSpace(authz[7:])
		}
		if token == "" {
			token = strings.TrimSpace(c.Cookies("access", ""))
		}
		var session *Session
		if opts.Verifier != nil && token != "" {
			if claims, err := opts.Verifier.Verify(token); err == nil {
				session = &Session{UUID: claims.UUID}
			} else if errors.Is(err, coreauth.ErrTokenInvalid) || errors.Is(err, coreauth.ErrTokenExpired) || errors.Is(err, coreauth.ErrTokenMissing) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "unauthorized"})
			}
		}

		if strings.TrimSpace(opts.CoreAPIBase) == "" {
			if session == nil || strings.TrimSpace(session.UUID) == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "auth verify failed"})
			}
			c.Locals(userKey, session.UUID)
			storeSession(c, *session)
			return c.Next()
		}

		if remoteSession, err := verifyWithCoreAPI(client, strings.TrimRight(opts.CoreAPIBase, "/"), c); err == nil {
			session = remoteSession
		} else if session == nil || strings.TrimSpace(session.UUID) == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "auth verify failed"})
		}

		if session == nil || strings.TrimSpace(session.UUID) == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "unauthorized"})
		}
		c.Locals(userKey, session.UUID)
		storeSession(c, *session)
		return c.Next()
	}
}

func UserID(c *fiber.Ctx) string {
	if v := c.Locals(userKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

type verifyResponse struct {
	Success bool           `json:"success"`
	Data    *verifyPayload `json:"data"`
}

type verifyPayload struct {
	Valid         bool                `json:"valid"`
	UUID          string              `json:"uuid"`
	Roles         []string            `json:"roles"`
	PlatformRoles []string            `json:"platformRoles"`
	AppRoles      map[string][]string `json:"appRoles"`
}

func verifyWithCoreAPI(client *http.Client, base string, c *fiber.Ctx) (*Session, error) {
	apiURL := base + "/v1/auth/verify"
	req, _ := http.NewRequest(http.MethodGet, apiURL, nil)
	if v := c.Get("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}
	if v := c.Get("Cookie"); v != "" {
		req.Header.Set("Cookie", v)
	}
	if v := c.Get("Origin"); v != "" {
		req.Header.Set("Origin", v)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = "unauthorized"
		}
		return nil, errors.New(msg)
	}
	var body verifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if body.Data == nil || !body.Data.Valid || strings.TrimSpace(body.Data.UUID) == "" {
		return nil, errors.New("unauthorized")
	}
	return &Session{
		UUID:          body.Data.UUID,
		Roles:         append([]string{}, body.Data.Roles...),
		PlatformRoles: append([]string{}, body.Data.PlatformRoles...),
		AppRoles:      cloneAppRoles(body.Data.AppRoles),
	}, nil
}

func cloneAppRoles(src map[string][]string) map[string][]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string][]string, len(src))
	for k, v := range src {
		dst[k] = append([]string{}, v...)
	}
	return dst
}
