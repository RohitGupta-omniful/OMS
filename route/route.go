package route

import (
	"github.com/RohitGupta-omniful/OMS/internal/handlers"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, h *handlers.Handler) {
	r.POST("/api/orders/upload", h.UploadCSV)

}
