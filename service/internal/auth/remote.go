package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type Options struct{ CoreAPIBase string }

const userKey = "userID"

func Middleware(opts Options) fiber.Handler {
	client := &http.Client{}
	return func(c *fiber.Ctx) error {
		apiURL := strings.TrimRight(opts.CoreAPIBase, "/") + "/v1/auth/verify"
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
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "auth verify failed"})
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			msg := strings.TrimSpace(string(b))
			if msg == "" {
				msg = "unauthorized"
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": msg})
		}
		var body map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
			if data, ok := body["data"].(map[string]any); ok {
				if valid, vok := data["valid"].(bool); vok && valid {
					if uuid, ok := data["uuid"].(string); ok && uuid != "" {
						c.Locals(userKey, uuid)
					}
				}
			}
		}
		if uid := UserID(c); uid == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "unauthorized"})
		}
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
