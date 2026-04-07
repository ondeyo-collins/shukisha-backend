package parcels

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shukisha/backend/internal/platform/config"
	"github.com/shukisha/backend/internal/platform/sms"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type NotificationService struct {
	db  *gorm.DB
	sms sms.Provider
}

func NewNotificationService(db *gorm.DB, smsProvider sms.Provider) *NotificationService {
	return &NotificationService{db: db, sms: smsProvider}
}

func (s *NotificationService) ProcessOnce(ctx context.Context, batchSize int) error {
	if batchSize <= 0 || batchSize > 50 {
		batchSize = 20
	}

	// Safety default: do not send real driver notifications unless enabled.
	// OTP SMS in the identity flow is separate.
	sendEnabled := config.GetEnv("DRIVER_NOTIFY_SEND_ENABLED", "false") == "true"

	now := time.Now()

	for i := 0; i < batchSize; i++ {
		var job NotificationJob
		err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
				Where("status IN ? AND next_attempt_at <= ? AND attempts < max_attempts", []string{"QUEUED", "FAILED"}, now).
				Order("next_attempt_at asc").
				Take(&job).Error; err != nil {
				return err
			}

			if strings.TrimSpace(job.To) == "" || strings.TrimSpace(job.Message) == "" {
				return errors.New("invalid notification job")
			}

			if !sendEnabled {
				sentAt := time.Now()
				return tx.Model(&NotificationJob{}).Where("id = ?", job.ID).Updates(map[string]any{
					"status":          "SENT",
					"sent_at":         &sentAt,
					"last_error":      "",
					"next_attempt_at": sentAt,
				}).Error
			}

			if err := s.sms.SendSMS(ctx, job.To, job.Message); err != nil {
				next, status, lastErr, attempts := nextRetry(job.Attempts+1, job.MaxAttempts, err)
				return tx.Model(&NotificationJob{}).Where("id = ?", job.ID).Updates(map[string]any{
					"status":          status,
					"attempts":        attempts,
					"last_error":      lastErr,
					"next_attempt_at": next,
				}).Error
			}

			sentAt := time.Now()
			return tx.Model(&NotificationJob{}).Where("id = ?", job.ID).Updates(map[string]any{
				"status":          "SENT",
				"sent_at":         &sentAt,
				"attempts":        job.Attempts + 1,
				"last_error":      "",
				"next_attempt_at": sentAt,
			}).Error
		})

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
	}

	return nil
}

func nextRetry(attempts int, maxAttempts int, sendErr error) (time.Time, string, string, int) {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	status := "FAILED"
	if attempts >= maxAttempts {
		status = "FAILED"
	}

	backoff := time.Minute
	switch attempts {
	case 1:
		backoff = 30 * time.Second
	case 2:
		backoff = 2 * time.Minute
	case 3:
		backoff = 5 * time.Minute
	case 4:
		backoff = 15 * time.Minute
	default:
		backoff = 30 * time.Minute
	}

	return time.Now().Add(backoff), status, fmt.Sprintf("%v", sendErr), attempts
}
