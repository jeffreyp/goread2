package services

import (
	"encoding/json"
	"time"

	"github.com/jeffreyp/goread2/internal/database"
)

type AuditService struct {
	db database.Database
}

func NewAuditService(db database.Database) *AuditService {
	return &AuditService{
		db: db,
	}
}

// LogAdminAction creates an audit log entry for an admin action
func (s *AuditService) LogAdminAction(
	adminUserID int,
	adminEmail string,
	operationType string,
	targetUserID int,
	targetUserEmail string,
	details map[string]interface{},
	ipAddress string,
	result string,
	errorMessage string,
) error {
	// Convert details to JSON
	detailsJSON := ""
	if details != nil {
		jsonBytes, err := json.Marshal(details)
		if err != nil {
			// If we can't marshal the details, log without them
			detailsJSON = "{}"
		} else {
			detailsJSON = string(jsonBytes)
		}
	}

	log := &database.AuditLog{
		Timestamp:        time.Now(),
		AdminUserID:      adminUserID,
		AdminEmail:       adminEmail,
		OperationType:    operationType,
		TargetUserID:     targetUserID,
		TargetUserEmail:  targetUserEmail,
		OperationDetails: detailsJSON,
		IPAddress:        ipAddress,
		Result:           result,
		ErrorMessage:     errorMessage,
	}

	return s.db.CreateAuditLog(log)
}

// LogSuccess is a convenience method for logging successful operations
func (s *AuditService) LogSuccess(
	adminUserID int,
	adminEmail string,
	operationType string,
	targetUserID int,
	targetUserEmail string,
	details map[string]interface{},
	ipAddress string,
) error {
	return s.LogAdminAction(
		adminUserID,
		adminEmail,
		operationType,
		targetUserID,
		targetUserEmail,
		details,
		ipAddress,
		"success",
		"",
	)
}

// LogFailure is a convenience method for logging failed operations
func (s *AuditService) LogFailure(
	adminUserID int,
	adminEmail string,
	operationType string,
	targetUserID int,
	targetUserEmail string,
	details map[string]interface{},
	ipAddress string,
	errorMessage string,
) error {
	return s.LogAdminAction(
		adminUserID,
		adminEmail,
		operationType,
		targetUserID,
		targetUserEmail,
		details,
		ipAddress,
		"failure",
		errorMessage,
	)
}

// GetAuditLogs retrieves audit logs with optional filters
func (s *AuditService) GetAuditLogs(limit, offset int, filters map[string]interface{}) ([]database.AuditLog, error) {
	return s.db.GetAuditLogs(limit, offset, filters)
}
