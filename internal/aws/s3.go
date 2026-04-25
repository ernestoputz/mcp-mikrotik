// Package aws provides a minimal S3 client using stdlib only (AWS SigV4).
package aws

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"
)

// S3Config holds credentials and bucket info for S3 uploads.
type S3Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	Bucket          string
	Prefix          string // key prefix, e.g. "mikrotik-backups/"
}

// S3Client uploads objects to S3 using AWS Signature Version 4.
type S3Client struct {
	cfg  S3Config
	http *http.Client
}

// NewS3Client returns a ready-to-use S3Client.
func NewS3Client(cfg S3Config) *S3Client {
	return &S3Client{
		cfg:  cfg,
		http: &http.Client{Timeout: 120 * time.Second},
	}
}

// PutObject uploads data under the given key (prefix is prepended automatically).
// contentType should be "application/octet-stream" for binary files.
func (c *S3Client) PutObject(key string, data []byte, contentType string) error {
	if c.cfg.Prefix != "" {
		key = c.cfg.Prefix + key
	}

	host := fmt.Sprintf("%s.s3.%s.amazonaws.com", c.cfg.Bucket, c.cfg.Region)
	endpoint := "https://" + host + "/" + key

	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	payloadHash := hexSHA256(data)

	// Canonical headers (sorted alphabetically by name)
	canonHeaders := fmt.Sprintf(
		"content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		contentType, host, payloadHash, amzDate,
	)
	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"

	canonRequest := fmt.Sprintf("%s\n/%s\n\n%s\n%s\n%s",
		http.MethodPut, key, canonHeaders, signedHeaders, payloadHash)

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, c.cfg.Region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate, credentialScope, hexSHA256([]byte(canonRequest)))

	sigKey := deriveSigKey(c.cfg.SecretAccessKey, dateStamp, c.cfg.Region, "s3")
	signature := hex.EncodeToString(hmacSHA256(sigKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s,SignedHeaders=%s,Signature=%s",
		c.cfg.AccessKeyID, credentialScope, signedHeaders, signature,
	)

	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build s3 request: %w", err)
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Host", host)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("x-amz-date", amzDate)
	req.ContentLength = int64(len(data))

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("s3 put: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("s3 put failed (HTTP %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// S3URL returns the public (or path-style) URL of an object.
func (c *S3Client) S3URL(key string) string {
	if c.cfg.Prefix != "" {
		key = c.cfg.Prefix + key
	}
	return fmt.Sprintf("s3://%s/%s", c.cfg.Bucket, key)
}

func hexSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func deriveSigKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte("aws4_request"))
}
