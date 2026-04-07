package sms

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/shukisha/backend/internal/platform/config"
)

type TwilioProvider struct {
	accountSid string
	authToken  string
	fromNumber string
}

func NewTwilioProvider() *TwilioProvider {
	return &TwilioProvider{
		accountSid: config.GetEnv("TWILIO_ACCOUNT_SID", ""),
		authToken:  config.GetEnv("TWILIO_AUTH_TOKEN", ""),
		fromNumber: config.GetEnv("TWILIO_FROM_NUMBER", ""),
	}
}

func (p *TwilioProvider) SendSMS(ctx context.Context, to string, message string) error {
	// Skip if dummy SID is used or empty
	if p.accountSid == "" || strings.HasPrefix(p.accountSid, "ACxxxx") {
		fmt.Printf("[MOCK SMS to %s]: %s\n", to, message)
		return nil
	}

	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", p.accountSid)
	
	data := url.Values{}
	data.Set("To", to)
	data.Set("From", p.fromNumber)
	data.Set("Body", message)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(p.accountSid, p.authToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("twilio api returned status: %d", resp.StatusCode)
	}

	return nil
}
