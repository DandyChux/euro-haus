package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"euro-haus-api/handlers"
	"euro-haus-api/middleware"
	"euro-haus-api/services"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v82"
)

func init() {
	// Load environment variables from .env file in development
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	if env == "development" {
		if err := godotenv.Load("../.env"); err != nil {
			log.Println("No .env file found")
		}
	}

	// Initialize stripe
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		log.Fatal("STRIPE_SECRET_KEY environment variable not set")
	}

	// Initialize S3 client
	services.InitS3Client()

	// Initialize Redis
	if err := services.InitRedis(); err != nil {
		log.Printf("Redis not available, using in-memory storage: %v", err)
	}

	// Initialize event listeners
	handlers.InitEventListeners()

}

// loggingMiddleware logs each request with method, path, status, and duration
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the ResponseWriter to capture the status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		// Use fmt.Printf to write to stdout for successful requests
		if wrapped.statusCode < 400 {
			fmt.Printf("%s %s %d %v\n", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
		} else {
			// Use log.Printf (stderr) for errors
			log.Printf("ERROR: %s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	router := mux.NewRouter()

	// API Routes
	api := router.PathPrefix("/api").Subrouter()

	// Healthcheck
	api.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	// Public endpoints (no auth required)
	api.HandleFunc("/products", handlers.GetProducts).Methods("GET")
	api.HandleFunc("/products/{id}", handlers.GetProduct).Methods("GET")
	api.HandleFunc("/products/{id}/prices", handlers.GetProductPrices).Methods("GET", "OPTIONS")

	// Event endpoints - public validation, admin check-in
	api.HandleFunc("/events/ticket/validate", handlers.ValidateTicket).Methods("POST", "OPTIONS")
	api.HandleFunc("/events/updates", handlers.HandleEventUpdates)
	api.HandleFunc("/events/{eventId}/tickets", handlers.GetEventTickets).Methods("GET", "OPTIONS")
	api.HandleFunc("/events/{slug}", handlers.GetEventBySlug).Methods("GET", "OPTIONS")
	api.HandleFunc("/events/{eventId}/linked-products", handlers.GetEventLinkedProducts).Methods("GET", "OPTIONS")

	// Vehicle submission endpoints
	api.HandleFunc("/submissions", handlers.CreateSubmission).Methods("POST", "OPTIONS")
	api.HandleFunc("/submissions/{submissionId}", handlers.GetSubmission).Methods("GET", "OPTIONS")
	api.HandleFunc("/create-participant-checkout", handlers.CreateParticipantCheckout).Methods("POST", "OPTIONS")
	api.HandleFunc("/checkout/submission", handlers.CreateSubmissionCheckout).Methods("POST", "OPTIONS")

	// Newsletter endpoints
	api.HandleFunc("/newsletter/subscribe", handlers.SubscribeToNewsletter).Methods("POST", "OPTIONS")
	api.HandleFunc("/newsletter/lists", handlers.GetMailingLists).Methods("GET")

	// Auth endpoints
	api.HandleFunc("/auth/login", handlers.Login).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/logout", handlers.Logout).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/validate", handlers.ValidateToken).Methods("GET", "OPTIONS")

	// Payment endpoints
	api.HandleFunc("/create-payment-intent", handlers.CreatePaymentIntent).Methods("POST")
	api.HandleFunc("/create-checkout-session", handlers.CreateCheckoutSession).Methods("POST")
	api.HandleFunc("/checkout-session", handlers.GetCheckoutSession).Methods("GET", "OPTIONS")

	// Content placement endpoints
	api.HandleFunc("/content-placements", handlers.GetAllContentPlacements).Methods("GET", "OPTIONS")
	api.HandleFunc("/content-placements/register", handlers.RegisterContentPlacement).Methods("POST", "OPTIONS")
	api.HandleFunc("/content-placements/dynamic", handlers.GetDynamicContentPlacements).Methods("GET", "OPTIONS")
	api.HandleFunc("/content-placements/{id}", handlers.GetContentPlacement).Methods("GET", "OPTIONS")
	api.HandleFunc("/content-placements/{id}", handlers.UpdateContentPlacement).Methods("PUT", "OPTIONS")
	api.HandleFunc("/content-placements/by-media/{key}", handlers.GetContentPlacementsByMedia).Methods("GET", "OPTIONS")

	// Public discount validation endpoint
	api.HandleFunc("/validate-promotion-code", handlers.ValidatePromotionCode).Methods("POST", "OPTIONS")

	// Event merchandise management
	api.Handle("/admin/events/{eventId}/tiers/{priceId}/products", middleware.RequireAuth(http.HandlerFunc(handlers.UpdateTierIncludedProducts))).Methods("PUT", "OPTIONS")
	api.HandleFunc("/events/{eventId}/recommendations", handlers.GetEventMerchandiseRecommendations).Methods("GET", "OPTIONS")
	api.HandleFunc("/create-event-checkout-session", handlers.CreateEventCheckoutSession).Methods("POST", "OPTIONS")

	// Admin endpoints (requires authentication)
	// Product management
	api.Handle("/admin/create-product", middleware.RequireAuth(http.HandlerFunc(handlers.CreateProduct))).Methods("POST", "OPTIONS")
	api.Handle("/admin/update-product/{id}", middleware.RequireAuth(http.HandlerFunc(handlers.UpdateProduct))).Methods("PUT", "OPTIONS")
	api.Handle("/admin/delete-product/{id}", middleware.RequireAuth(http.HandlerFunc(handlers.DeleteProduct))).Methods("DELETE", "OPTIONS")

	// Media management endpoints (requires authentication)
	api.Handle("/admin/media", middleware.RequireAuth(http.HandlerFunc(handlers.ListMedia))).Methods("GET", "OPTIONS")
	api.Handle("/admin/media/upload", middleware.RequireAuth(http.HandlerFunc(handlers.UploadMedia))).Methods("POST", "OPTIONS")
	api.Handle("/admin/media/delete", middleware.RequireAuth(http.HandlerFunc(handlers.DeleteMedia))).Methods("DELETE", "OPTIONS")

	// Price management endpoints (requires authentication)
	api.Handle("/admin/create-price", middleware.RequireAuth(http.HandlerFunc(handlers.CreatePrice))).Methods("POST", "OPTIONS")
	api.Handle("/admin/update-price/{id}", middleware.RequireAuth(http.HandlerFunc(handlers.UpdatePrice))).Methods("PUT", "OPTIONS")
	api.Handle("/admin/archive-price/{id}", middleware.RequireAuth(http.HandlerFunc(handlers.ArchivePrice))).Methods("PUT", "OPTIONS")
	api.Handle("/admin/set-default-price/{id}", middleware.RequireAuth(http.HandlerFunc(handlers.SetDefaultPrice))).Methods("PUT", "OPTIONS")

	// Discount management endpoints (requires authentication)
	api.Handle("/admin/coupons", middleware.RequireAuth(http.HandlerFunc(handlers.CreateCoupon))).Methods("POST", "OPTIONS")
	api.Handle("/admin/coupons", middleware.RequireAuth(http.HandlerFunc(handlers.ListCoupons))).Methods("GET", "OPTIONS")
	api.Handle("/admin/coupons/{id}", middleware.RequireAuth(http.HandlerFunc(handlers.DeleteCoupon))).Methods("DELETE", "OPTIONS")
	api.Handle("/admin/promotion-codes", middleware.RequireAuth(http.HandlerFunc(handlers.CreatePromotionCode))).Methods("POST", "OPTIONS")

	// Admin event endpoints (requires authentication)
	api.Handle("/admin/events/ticket/check-in", middleware.RequireAuth(http.HandlerFunc(handlers.CheckInTicket))).Methods("POST", "OPTIONS")
	api.Handle("/admin/events/{eventId}/attendees", middleware.RequireAuth(http.HandlerFunc(handlers.GetEventAttendees))).Methods("GET", "OPTIONS")
	api.Handle("/admin/events/{eventId}/link-products", middleware.RequireAuth(http.HandlerFunc(handlers.LinkProductsToEvent))).Methods("POST", "OPTIONS")
	api.Handle("/admin/events/{eventId}/products/{productId}", middleware.RequireAuth(http.HandlerFunc(handlers.RemoveProductFromEvent))).Methods("DELETE", "OPTIONS")
	api.Handle("/admin/tiers/add-products", middleware.RequireAuth(http.HandlerFunc(handlers.AddProductsToTier))).Methods("POST", "OPTIONS")

	// Fulfillment management (admin only)
	api.Handle("/admin/fulfillments/pending", middleware.RequireAuth(http.HandlerFunc(handlers.GetPendingFulfillments))).Methods("GET", "OPTIONS")
	api.Handle("/admin/fulfillments/{id}/status", middleware.RequireAuth(http.HandlerFunc(handlers.UpdateFulfillmentStatus))).Methods("PUT", "OPTIONS")

	// Admin submission endpoints (requires authentication)
	api.Handle("/admin/submissions/pending-count", middleware.RequireAuth(http.HandlerFunc(handlers.GetPendingSubmissionsCount))).Methods("GET", "OPTIONS")
	api.Handle("/admin/submissions/issues", middleware.RequireAuth(http.HandlerFunc(handlers.GetAllSubmissionsWithIssues))).Methods("GET", "OPTIONS") // MOVED UP
	api.Handle("/admin/submissions/{eventId}", middleware.RequireAuth(http.HandlerFunc(handlers.GetEventSubmissions))).Methods("GET", "OPTIONS")
	api.Handle("/admin/submissions/{submissionId}/approve", middleware.RequireAuth(http.HandlerFunc(handlers.ApproveSubmission))).Methods("PUT", "OPTIONS")
	api.Handle("/admin/submissions/{submissionId}/deny", middleware.RequireAuth(http.HandlerFunc(handlers.DenySubmission))).Methods("PUT", "OPTIONS")
	api.Handle("/admin/submissions/{submissionId}/payment-status", middleware.RequireAuth(http.HandlerFunc(handlers.GetSubmissionPaymentStatus))).Methods("GET", "OPTIONS")
	api.Handle("/admin/submissions/{submissionId}/create-payment", middleware.RequireAuth(http.HandlerFunc(handlers.CreateSubmissionPayment))).Methods("POST", "OPTIONS")
	api.Handle("/admin/submissions/{submissionId}/resend-email", middleware.RequireAuth(http.HandlerFunc(handlers.ResendApprovalEmail))).Methods("POST", "OPTIONS")

	// Webhook endpoint (no CORS needed)
	api.HandleFunc("/webhook", handlers.HandleWebhook).Methods("POST")

	// Setup CORS
	baseURL := os.Getenv("BASE_URL")
	allowedOrigins := []string{
		"http://localhost:3000",
		baseURL,
	}
	corsMiddleware := middleware.SetupCORS(allowedOrigins)

	// Apply CORS to all routes except webhooks
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/webhook" {
			router.ServeHTTP(w, req)
		} else {
			corsMiddleware.Handler(router).ServeHTTP(w, req)
		}
	})

	// Apply logging middleware
	handler = loggingMiddleware(handler)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
