package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dandychux/euro-haus/internal/services"
)

// NewsletterSubscription represents a request to subscribe to the newsletter
type NewsletterSubscription struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// MailchimpLink represents a link in the Mailchimp API response
type MailchimpLink struct {
	Rel          string `json:"rel"`
	Href         string `json:"href"`
	Method       string `json:"method"`
	TargetSchema string `json:"target_schema"`
	Schema       string `json:"schema"`
}

// MailchimpListStats represents statistics for a Mailchimp list
type MailchimpListStats struct {
	MemberCount               int     `json:"member_count"`
	TotalContacts             int     `json:"total_contacts"`
	UnsubscribeCount          int     `json:"unsubscribe_count"`
	CleanedCount              int     `json:"cleaned_count"`
	MemberCountSinceSend      int     `json:"member_count_since_send"`
	UnsubscribeCountSinceSend int     `json:"unsubscribe_count_since_send"`
	CleanedCountSinceSend     int     `json:"cleaned_count_since_send"`
	CampaignCount             int     `json:"campaign_count"`
	CampaignLastSent          string  `json:"campaign_last_sent"`
	MergeFieldCount           int     `json:"merge_field_count"`
	AvgSubRate                float64 `json:"avg_sub_rate"`
	AvgUnsubRate              float64 `json:"avg_unsub_rate"`
	TargetSubRate             float64 `json:"target_sub_rate"`
	OpenRate                  float64 `json:"open_rate"`
	ClickRate                 float64 `json:"click_rate"`
	LastSubDate               string  `json:"last_sub_date"`
	LastUnsubDate             string  `json:"last_unsub_date"`
}

// MailchimpContact represents contact information for a Mailchimp list
type MailchimpContact struct {
	Company  string `json:"company"`
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	City     string `json:"city"`
	State    string `json:"state"`
	Zip      string `json:"zip"`
	Country  string `json:"country"`
	Phone    string `json:"phone"`
}

// MailchimpCampaignDefaults represents default campaign settings for a Mailchimp list
type MailchimpCampaignDefaults struct {
	FromName  string `json:"from_name"`
	FromEmail string `json:"from_email"`
	Subject   string `json:"subject"`
	Language  string `json:"language"`
}

// MailchimpList represents a single mailing list in Mailchimp
type MailchimpList struct {
	ID                   string                    `json:"id"`
	WebID                int                       `json:"web_id"`
	Name                 string                    `json:"name"`
	Contact              MailchimpContact          `json:"contact"`
	PermissionReminder   string                    `json:"permission_reminder"`
	UseArchiveBar        bool                      `json:"use_archive_bar"`
	CampaignDefaults     MailchimpCampaignDefaults `json:"campaign_defaults"`
	NotifyOnSubscribe    bool                      `json:"notify_on_subscribe"`
	NotifyOnUnsubscribe  bool                      `json:"notify_on_unsubscribe"`
	DateCreated          string                    `json:"date_created"`
	ListRating           int                       `json:"list_rating"`
	EmailTypeOption      bool                      `json:"email_type_option"`
	SubscribeURLShort    string                    `json:"subscribe_url_short"`
	SubscribeURLLong     string                    `json:"subscribe_url_long"`
	BeamerAddress        string                    `json:"beamer_address"`
	Visibility           string                    `json:"visibility"`
	DoubleOptin          bool                      `json:"double_optin"`
	HasWelcome           bool                      `json:"has_welcome"`
	MarketingPermissions bool                      `json:"marketing_permissions"`
	Modules              []string                  `json:"modules"`
	Stats                MailchimpListStats        `json:"stats"`
	Links                []MailchimpLink           `json:"_links"`
}

// MailchimpListsResponse represents the response from Mailchimp's lists endpoint
type MailchimpListsResponse struct {
	Lists       []MailchimpList `json:"lists"`
	TotalItems  int             `json:"total_items"`
	Constraints struct {
		MayCreate             bool `json:"may_create"`
		MaxInstances          int  `json:"max_instances"`
		CurrentTotalInstances int  `json:"current_total_instances"`
	} `json:"constraints"`
	Links []MailchimpLink `json:"_links"`
}

// SubscribeToNewsletter handles subscribing users to the Mailchimp newsletter
func SubscribeToNewsletter(w http.ResponseWriter, r *http.Request) {
	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var subscription NewsletterSubscription
	err := json.NewDecoder(r.Body).Decode(&subscription)
	if err != nil {
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	// Validate email
	if subscription.Email == "" {
		http.Error(w, "Email address is required", http.StatusBadRequest)
		return
	}

	// Add subscriber to Mailchimp
	err = services.AddMailchimpSubscriber(
		subscription.Email,
		subscription.FirstName,
		subscription.LastName,
	)
	if err != nil {
		log.Printf("Error subscribing to newsletter: %v", err)
		http.Error(w, "Failed to subscribe to newsletter", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Successfully subscribed to newsletter",
	})
}

// GetMailingLists retrieves information about all lists in the account
func GetMailingLists(w http.ResponseWriter, r *http.Request) {
	apiKey := os.Getenv("MAILCHIMP_API_KEY")
	dataCenter := strings.Split(apiKey, "-")[1]

	if apiKey == "" {
		http.Error(w, "Missing Mailchimp API key", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("https://%s.api.mailchimp.com/3.0/lists", dataCenter)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set authentication
	req.SetBasicAuth("apikey", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Mailchimp API error: %s", resp.Status), resp.StatusCode)
		return
	}

	var listsResponse MailchimpListsResponse
	err = json.NewDecoder(resp.Body).Decode(&listsResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listsResponse)
}
