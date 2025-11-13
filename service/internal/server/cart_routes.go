package server

import (
	"github.com/gofiber/fiber/v2"
)

func registerCartRoutes(app *fiber.App, opts Options, requireAuth fiber.Handler) {
	app.Get("/v1/cart", requireAuth, func(c *fiber.Ctx) error { return getCart(c, opts.DB) })
	app.Put("/v1/cart", requireAuth, func(c *fiber.Ctx) error { return replaceCart(c, opts.DB) })
	app.Post("/v1/cart/items", requireAuth, func(c *fiber.Ctx) error { return addCartItem(c, opts.DB) })
	app.Put("/v1/cart/items/:id", requireAuth, func(c *fiber.Ctx) error { return updateCartItem(c, opts.DB) })
	app.Delete("/v1/cart/items/:id", requireAuth, func(c *fiber.Ctx) error { return deleteCartItem(c, opts.DB) })
	app.Delete("/v1/cart", requireAuth, func(c *fiber.Ctx) error { return clearCart(c, opts.DB) })
	app.Post("/v1/cart/preview", requireAuth, func(c *fiber.Ctx) error { return previewCartPricing(c, opts.DB) })

	// Wishlist endpoints
	app.Get("/v1/wishlist", requireAuth, func(c *fiber.Ctx) error { return getWishlist(c, opts.DB) })
	app.Put("/v1/wishlist", requireAuth, func(c *fiber.Ctx) error { return replaceWishlist(c, opts.DB) })

	// Orders minimal endpoints
	app.Post("/v1/orders", requireAuth, func(c *fiber.Ctx) error { return createOrderFromCart(c, opts.DB) })
	app.Get("/v1/orders", requireAuth, func(c *fiber.Ctx) error { return listOrders(c, opts.DB) })
	app.Get("/v1/orders/:id", requireAuth, func(c *fiber.Ctx) error { return getOrder(c, opts.DB) })

	// Unified search (owner-scoped). Returns sections for products, orders, and placeholders for others.
}
