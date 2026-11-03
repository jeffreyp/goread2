package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/services"
)

// AdminStatsHandler serves usage statistics to admins.
type AdminStatsHandler struct {
	statsService *services.StatsService
}

func NewAdminStatsHandler(statsService *services.StatsService) *AdminStatsHandler {
	return &AdminStatsHandler{statsService: statsService}
}

// GetStats handles GET /admin/stats
func (h *AdminStatsHandler) GetStats(c *gin.Context) {
	stats, err := h.statsService.GetStats()
	if err != nil {
		log.Printf("Failed to compute admin stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Usage statistics could not be loaded."})
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, stats)
}
