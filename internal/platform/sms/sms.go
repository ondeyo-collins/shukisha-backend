package sms

import "context"

// Provider defines the interface for sending SMS messages
type Provider interface {
	SendSMS(ctx context.Context, to string, message string) error
}
