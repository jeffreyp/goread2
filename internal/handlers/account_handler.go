package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jeffreyp/goread2/internal/auth"
	"github.com/jeffreyp/goread2/internal/services"
)

// AccountHandler serves the signed-in user's data rights requests.
type AccountHandler struct {
	userDataService *services.UserDataService
}

func NewAccountHandler(userDataService *services.UserDataService) *AccountHandler {
	return &AccountHandler{userDataService: userDataService}
}

// ExportData handles GET /api/account/export. It sends the signed-in user's
// personal data as a JSON file download.
func (ah *AccountHandler) ExportData(c *gin.Context) {
	user, exists := auth.GetUserFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "You must be signed in to access this resource."})
		return
	}

	export, err := ah.userDataService.PrepareExport(user.ID)
	if err != nil {
		log.Printf("Failed to prepare data export for user %d: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate the data export. Please try again."})
		return
	}

	filename := fmt.Sprintf("goread2-data-%s.json", export.ExportedAt.Format("2006-01-02"))
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Status(http.StatusOK)

	if err := ah.userDataService.WriteExport(c.Writer, user.ID, export); err != nil {
		// Headers are already sent, so the client receives a truncated file.
		log.Printf("Data export for user %d failed while streaming: %v", user.ID, err)
	}
}
