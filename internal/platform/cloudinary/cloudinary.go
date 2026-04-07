package cloudinary

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"
)

type UploadResult struct {
	PublicID     string `json:"public_id"`
	Version      int    `json:"version"`
	ResourceType string `json:"resource_type"`
	Type         string `json:"type"`
	Format       string `json:"format"`
	Bytes        int64  `json:"bytes"`
	SecureURL    string `json:"secure_url"`
}

func UploadAuthenticated(ctx context.Context, cloudName, apiKey, apiSecret, folder, fileName, mimeType string, content []byte) (*UploadResult, error) {
	if strings.TrimSpace(cloudName) == "" || strings.TrimSpace(apiKey) == "" || strings.TrimSpace(apiSecret) == "" {
		return nil, errors.New("cloudinary credentials missing")
	}
	if len(content) == 0 {
		return nil, errors.New("empty upload content")
	}

	url := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/auto/upload", cloudName)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// For authenticated delivery, upload with type=authenticated.
	_ = writer.WriteField("type", "authenticated")
	if strings.TrimSpace(folder) != "" {
		_ = writer.WriteField("folder", folder)
	}

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(apiKey, apiSecret)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if strings.TrimSpace(mimeType) != "" {
		req.Header.Set("X-File-Content-Type", mimeType)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cloudinary upload failed: %s", strings.TrimSpace(string(respBytes)))
	}

	var out UploadResult
	if err := json.Unmarshal(respBytes, &out); err != nil {
		return nil, err
	}
	if out.PublicID == "" || out.ResourceType == "" || out.Version == 0 {
		return nil, errors.New("cloudinary upload returned incomplete response")
	}
	return &out, nil
}

// AuthenticatedURL builds a signed Cloudinary authenticated delivery URL.
// This uses URL signing with the s--SIGN-- component (no expiry).
//
// Signature algorithm (per Cloudinary docs):
//
//	signature = base64url(sha1(toSign + api_secret))[:8]
//
// where toSign is the path portion after the signature component, e.g.:
//
//	"v1312461204/folder/my_public_id.jpg"
func AuthenticatedURL(cloudName, apiSecret string, resourceType string, version int, publicID string, format string) (string, error) {
	cloudName = strings.TrimSpace(cloudName)
	apiSecret = strings.TrimSpace(apiSecret)
	resourceType = strings.TrimSpace(resourceType)
	publicID = strings.TrimSpace(publicID)
	format = strings.TrimSpace(format)
	if cloudName == "" || apiSecret == "" || resourceType == "" || publicID == "" || version <= 0 {
		return "", errors.New("missing fields for authenticated url")
	}

	assetPath := fmt.Sprintf("v%d/%s", version, publicID)
	if format != "" {
		assetPath = assetPath + "." + format
	}

	h := sha1.Sum([]byte(assetPath + apiSecret))
	sig := base64.RawURLEncoding.EncodeToString(h[:])
	if len(sig) > 8 {
		sig = sig[:8]
	}

	// Construct: /<resource_type>/authenticated/s--sig--/<assetPath>
	p := path.Join(resourceType, "authenticated", "s--"+sig+"--", assetPath)
	return "https://res.cloudinary.com/" + cloudName + "/" + p, nil
}

// VerifyUploadSignature is a helper for debugging; not used in the app flow.
func VerifyUploadSignature(publicID string, version int, apiSecret string, signature string) bool {
	toSign := fmt.Sprintf("public_id=%s&version=%d%s", publicID, version, apiSecret)
	h := sha1.Sum([]byte(toSign))
	return hex.EncodeToString(h[:]) == signature
}
