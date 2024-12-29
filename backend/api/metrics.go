package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MetricsService maneja las métricas del servicio
type MetricsService struct{}

// NewMetricsService crea una nueva instancia de MetricsService
func NewMetricsService() *MetricsService {
	return &MetricsService{}
}

// Metrics responde con un mensaje indicando que el servicio está temporalmente parado
func (m *MetricsService) Metrics(c *gin.Context) {
	// Mensaje de servicio parado
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"status":  "service unavailable",
		"message": "Servicio temporalmente parado. Disculpe las molestias.",
	})
}
