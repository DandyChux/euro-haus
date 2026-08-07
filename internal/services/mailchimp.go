package services

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// MailchimpSubscriber represents a subscriber in Mailchimp
type MailchimpSubscriber struct {
	Email       string            `json:"email_address"`
	Status      string            `json:"status"`
	MergeFields map[string]string `json:"merge_fields,omitempty"`
}

// AddMailchimpSubscriber adds a subscriber to a Mailchimp list
func AddMailchimpSubscriber(email string, firstName, lastName string) error {
	apiKey := os.Getenv("MAILCHIMP_API_KEY")
	listID := os.Getenv("MAILCHIMP_LIST_ID")
	dataCenter := strings.Split(apiKey, "-")[1]

	if apiKey == "" || listID == "" {
		return fmt.Errorf("missing Mailchimp configuration")
	}

	// Create the subscriber data
	subscriber := MailchimpSubscriber{
		Email:  email,
		Status: "subscribed",
	}

	// Add merge fields if provided
	if firstName != "" || lastName != "" {
		subscriber.MergeFields = make(map[string]string)

		if firstName != "" {
			subscriber.MergeFields["FNAME"] = firstName
		}

		if lastName != "" {
			subscriber.MergeFields["LNAME"] = lastName
		}
	}

	// Convert to JSON
	jsonData, err := json.Marshal(subscriber)
	if err != nil {
		return fmt.Errorf("error marshaling subscriber data: %w", err)
	}

	// Create MD5 hash of lowercase email for Mailchimp API
	emailHash := md5.Sum([]byte(strings.ToLower(email)))
	emailHashStr := fmt.Sprintf("%x", emailHash)

	// Create the request URL (using PUT to handle both new subscribers and updates)
	url := fmt.Sprintf("https://%s.api.mailchimp.com/3.0/lists/%s/members/%s",
		dataCenter, listID, emailHashStr)

	// Create the request
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("apikey", apiKey)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to Mailchimp: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %w", err)
	}

	// Check for successful response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Mailchimp API error: %s - %s", resp.Status, string(body))
	}

	return nil
}
