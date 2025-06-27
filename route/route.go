package route

import (
	"github.com/RohitGupta-omniful/OMS/internal/handlers"
	"github.com/RohitGupta-omniful/OMS/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, h *handlers.Handler) {
	protected := r.Group("/api/orders", middleware.AuthMiddleware())
	{
		protected.POST("/upload", h.UploadCSV)
	}
}
