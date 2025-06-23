package server

import (
	"context"

	"github.com/RohitGupta-omniful/OMS/internal/handlers"
	"github.com/RohitGupta-omniful/OMS/route"
	"github.com/omniful/go_commons/config"
	"github.com/omniful/go_commons/http"
)

// Initialize sets up the routes and returns the initialized server
func Initialize(ctx context.Context, h *handlers.Handler) *http.Server {
	server := http.InitializeServer(
		config.GetString(ctx, "server.port"),
		config.GetDuration(ctx, "server.read_timeout"),
		config.GetDuration(ctx, "server.write_timeout"),
		config.GetDuration(ctx, "server.idle_timeout"),
		false, // CORS disabled
	)

	route.RegisterRoutes(server.Engine, h)
	return server
}
