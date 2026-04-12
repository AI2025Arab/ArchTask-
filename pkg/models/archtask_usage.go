// ArchTask Addition - DO NOT MERGE INTO VIKUNJA CORE
// Tracks AI usage per user for Freemium enforcement.
package models

import (
	"time"

	"code.vikunja.io/api/pkg/db"
	"code.vikunja.io/api/pkg/log"

	"xorm.io/xorm"
)

// ArchTaskAIUsage tracks each AI operation performed by a user.
type ArchTaskAIUsage struct {
	ID            int64     `xorm:"bigint autoincr not null unique pk" json:"id"`
	UserID        int64     `xorm:"bigint not null INDEX" json:"user_id"`
	OperationType string    `xorm:"varchar(50) not null" json:"operation_type"` // voice, text, image, boq, suggest
	TokensUsed    int       `xorm:"int null default 0" json:"tokens_used"`
	Created       time.Time `xorm:"created not null" json:"created"`
}

// TableName returns the table name for ArchTaskAIUsage.
func (*ArchTaskAIUsage) TableName() string {
	return "archtask_ai_usage"
}

// FreeMonthlyLimit is the maximum number of free AI operations per user per month.
const FreeMonthlyLimit = 50

// ArchTaskUsageStats holds aggregated usage statistics for a user.
type ArchTaskUsageStats struct {
	TotalThisMonth int   `json:"total_this_month"`
	RemainingFree  int   `json:"remaining_free"`
	CanUseAI       bool  `json:"can_use_ai"`
	UserID         int64 `json:"user_id"`
}

// GetMonthlyUsage returns the number of AI operations a user has performed in the current month.
func GetMonthlyUsage(s *xorm.Session, userID int64) (int, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	count, err := s.
		Where("user_id = ?", userID).
		And("created >= ?", startOfMonth).
		Count(&ArchTaskAIUsage{})
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// CanUserUseAI checks whether a user has remaining free operations this month.
func CanUserUseAI(s *xorm.Session, userID int64) (bool, int, error) {
	used, err := GetMonthlyUsage(s, userID)
	if err != nil {
		return false, 0, err
	}

	remaining := FreeMonthlyLimit - used
	if remaining < 0 {
		remaining = 0
	}

	return remaining > 0, remaining, nil
}

// RecordAIUsage saves a new AI usage record for a user.
func RecordAIUsage(s *xorm.Session, userID int64, operationType string, tokensUsed int) error {
	record := &ArchTaskAIUsage{
		UserID:        userID,
		OperationType: operationType,
		TokensUsed:    tokensUsed,
	}

	_, err := s.Insert(record)
	if err != nil {
		log.Errorf("[ArchTask] Failed to record AI usage for user %d: %v", userID, err)
		return err
	}

	return nil
}

// GetUsageStats returns a summary of AI usage for a user this month.
func GetUsageStats(userID int64) (*ArchTaskUsageStats, error) {
	s := db.NewSession()
	defer s.Close()

	used, err := GetMonthlyUsage(s, userID)
	if err != nil {
		return nil, err
	}

	remaining := FreeMonthlyLimit - used
	if remaining < 0 {
		remaining = 0
	}

	return &ArchTaskUsageStats{
		TotalThisMonth: used,
		RemainingFree:  remaining,
		CanUseAI:       remaining > 0,
		UserID:         userID,
	}, nil
}
