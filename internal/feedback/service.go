package feedback

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FeedbackService struct {
	db *gorm.DB
}

func NewFeedbackService(db *gorm.DB) *FeedbackService {
	return &FeedbackService{db: db}
}

func (s *FeedbackService) SubmitRating(ctx context.Context, raterID, targetID uuid.UUID, score int, comment string) (*Rating, error) {
	rating := Rating{
		RaterID:  raterID,
		TargetID: targetID,
		Score:    score,
		Comment:  comment,
	}
	err := s.db.Create(&rating).Error
	return &rating, err
}

func (s *FeedbackService) SubmitReport(ctx context.Context, reporterID uuid.UUID, ref, reason, desc string) (*Report, error) {
	report := Report{
		ReporterID:  reporterID,
		ReferenceID: ref,
		Reason:      reason,
		Description: desc,
	}
	err := s.db.Create(&report).Error
	return &report, err
}
