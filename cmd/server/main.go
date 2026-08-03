package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/dandychux/euro-haus/internal/handlers"
	"github.com/dandychux/euro-haus/internal/middleware"
	"github.com/dandychux/euro-haus/internal/services"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/stripe/stripe-go/v82"
)

// spaFileServer serves static files from the SvelteKit build output directory.
// If the requested file does not exist on disk, it falls back to serving
// index.html so that SvelteKit's client-side router can handle the route.
type spaFileServer struct {
	staticDir  string
	fileServer http.Handler
}

func newSPAFileServer(staticDir string) *spaFileServer {
	return &spaFileServer{
		staticDir:  staticDir,
		fileServer: http.FileServer(http.Dir(staticDir)),
	}
}

func (s *spaFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the URL path to prevent directory traversal
	urlPath := filepath.Clean(r.URL.Path)

	// Build the full file path on disk
	filePath := filepath.Join(s.staticDir, urlPath)

	// Check if the requested file exists on disk
	info, err := os.Stat(filePath)
	if err == nil && !info.IsDir() {
		// File exists — serve it directly (CSS, JS, images, fonts, etc.)
		s.fileServer.ServeHTTP(w, r)
		return
	}

	// If it's a directory, check for an index.html inside it
	if err == nil && info.IsDir() {
		indexPath := filepath.Join(filePath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			s.fileServer.ServeHTTP(w, r)
			return
		}
	}

	// File doesn't exist — serve the root index.html (SPA fallback)
	http.ServeFile(w, r, filepath.Join(s.staticDir, "index.html"))
}

func init() {
	// Load environment variables from .env file in development
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	if env == "development" {
		if err := godotenv.Load(".env"); err != nil {
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

	// Initialize Postgres
	services.InitDB()

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
	api.HandleFunc("/products/{id}/prices", handlers.GetProductPrices).Methods("GET")

	// Event endpoints - public validation, admin check-in
	api.HandleFunc("/events/ticket/validate", handlers.ValidateTicket).Methods("POST")
	api.HandleFunc("/events/updates", handlers.HandleEventUpdates)
	api.HandleFunc("/events", handlers.GetEvents).Methods("GET")
	api.HandleFunc("/events/{id}", handlers.GetEventByID).Methods("GET")
	api.HandleFunc("/events/{id}/prices", handlers.GetEventPrices).Methods("GET")
	api.HandleFunc("/events/{id}/linked-products", handlers.GetEventLinkedProducts).Methods("GET")
	api.HandleFunc("/events/{id}/tickets", handlers.GetEventTickets).Methods("GET")

	// Vehicle submission endpoints
	api.HandleFunc("/submissions", handlers.CreateSubmission).Methods("POST")
	api.HandleFunc("/submissions/{submissionId}", handlers.GetSubmission).Methods("GET")
	api.HandleFunc("/create-participant-checkout", handlers.CreateParticipantCheckout).Methods("POST")
	api.HandleFunc("/checkout/submission", handlers.CreateSubmissionCheckout).Methods("POST")

	// Newsletter endpoints
	api.HandleFunc("/newsletter/subscribe", handlers.SubscribeToNewsletter).Methods("POST")
	api.HandleFunc("/newsletter/lists", handlers.GetMailingLists).Methods("GET")

	// Auth endpoints
	// api.HandleFunc("/auth/register", handlers.Register).Methods("POST")
	api.HandleFunc("/auth/login", handlers.Login).Methods("POST")
	api.HandleFunc("/auth/logout", handlers.Logout).Methods("POST")
	api.HandleFunc("/auth/validate", handlers.ValidateToken).Methods("GET")

	// Payment endpoints
	api.HandleFunc("/create-payment-intent", handlers.CreatePaymentIntent).Methods("POST")
	api.HandleFunc("/calculate-tax-shipping", handlers.CalculateTaxAndShipping).Methods("POST")
	api.HandleFunc("/shipping-rates", handlers.GetShippingRates).Methods("GET")
	api.HandleFunc("/create-checkout-session", handlers.CreateCheckoutSession).Methods("POST")
	api.HandleFunc("/checkout-session", handlers.GetCheckoutSession).Methods("GET")

	// Content placement endpoints
	api.HandleFunc("/content-placements", handlers.GetAllContentPlacements).Methods("GET")
	api.HandleFunc("/content-placements/register", handlers.RegisterContentPlacement).Methods("POST")
	api.HandleFunc("/content-placements/dynamic", handlers.GetDynamicContentPlacements).Methods("GET")
	api.HandleFunc("/content-placements/{id}", handlers.GetContentPlacement).Methods("GET")
	api.HandleFunc("/content-placements/{id}", handlers.UpdateContentPlacement).Methods("PUT")
	api.HandleFunc("/content-placements/by-media/{key}", handlers.GetContentPlacementsByMedia).Methods("GET")

	// Public discount validation endpoint
	api.HandleFunc("/validate-promotion-code", handlers.ValidatePromotionCode).Methods("POST")

	// Public media management endpoint
	api.HandleFunc("/media", handlers.ListMedia).Methods("GET")

	// Admin endpoints (requires authentication)
	// Product management
	api.Handle("/admin/create-product", middleware.RequireAdminAuth(http.HandlerFunc(handlers.CreateProduct))).Methods("POST")
	api.Handle("/admin/update-product/{id}", middleware.RequireAdminAuth(http.HandlerFunc(handlers.UpdateProduct))).Methods("PUT")
	api.Handle("/admin/delete-product/{id}", middleware.RequireAdminAuth(http.HandlerFunc(handlers.DeleteProduct))).Methods("DELETE")

	// Media management endpoints (requires authentication)
	api.Handle("/admin/media/upload", middleware.RequireAdminAuth(http.HandlerFunc(handlers.UploadMedia))).Methods("POST")
	api.Handle("/admin/media/delete", middleware.RequireAdminAuth(http.HandlerFunc(handlers.DeleteMedia))).Methods("DELETE")
	api.Handle("/admin/events/folders", middleware.RequireAdminAuth(http.HandlerFunc(handlers.ListEventFolders))).Methods("GET")
	api.Handle("/admin/events/gallery/upload", middleware.RequireAdminAuth(http.HandlerFunc(handlers.UploadEventGalleryBatch))).Methods("POST")

	// Price management endpoints (requires authentication)
	api.Handle("/admin/create-price", middleware.RequireAdminAuth(http.HandlerFunc(handlers.CreatePrice))).Methods("POST")
	api.Handle("/admin/update-price/{id}", middleware.RequireAdminAuth(http.HandlerFunc(handlers.UpdatePrice))).Methods("PUT")

	api.Handle("/admin/archive-price/{id}", middleware.RequireAdminAuth(http.HandlerFunc(handlers.ArchivePrice))).Methods("PUT")
	api.Handle("/admin/set-default-price", middleware.RequireAdminAuth(http.HandlerFunc(handlers.SetDefaultPrice))).Methods("POST")

	// Discount management endpoints (requires authentication)
	api.Handle("/admin/coupons", middleware.RequireAdminAuth(http.HandlerFunc(handlers.CreateCoupon))).Methods("POST")
	api.Handle("/admin/coupons", middleware.RequireAdminAuth(http.HandlerFunc(handlers.ListCoupons))).Methods("GET")
	api.Handle("/admin/coupons/{id}", middleware.RequireAdminAuth(http.HandlerFunc(handlers.DeleteCoupon))).Methods("DELETE")
	api.Handle("/admin/promotion-codes", middleware.RequireAdminAuth(http.HandlerFunc(handlers.CreatePromotionCode))).Methods("POST")
	api.Handle("/admin/promotion-codes", middleware.RequireAdminAuth(http.HandlerFunc(handlers.ListPromotionCodes))).Methods("GET")

	api.Handle(
		"/admin/products/{productId}/variants",
		middleware.RequireAdminAuth(
			http.HandlerFunc(handlers.GetProductVariants),
		),
	).Methods("GET")
	api.Handle(
		"/admin/products/{productId}/variants/stock",
		middleware.RequireAdminAuth(
			http.HandlerFunc(handlers.UpdateVariantStock),
		),
	).Methods("PUT")

	// Admin event endpoints (requires authentication)
	api.Handle(
		"/admin/events",
		middleware.RequireAdminAuth(
			http.HandlerFunc(handlers.CreateEvent),
		),
	).Methods("POST")

	api.Handle(
		"/admin/events/{id}",
		middleware.RequireAdminAuth(
			http.HandlerFunc(handlers.UpdateEvent),
		),
	).Methods("PUT")
	api.HandleFunc("/events/{id}/recommendations", handlers.GetEventMerchandiseRecommendations).Methods("GET")
	api.HandleFunc("/create-event-checkout-session", handlers.CreateEventCheckoutSession).Methods("POST")
	api.Handle("/admin/events/ticket/check-in", middleware.RequireAdminAuth(http.HandlerFunc(handlers.CheckInTicket))).Methods("POST")
	api.Handle("/admin/events/{id}/cleanup-tickets", middleware.RequireAdminAuth(http.HandlerFunc(handlers.CleanupEventTickets))).Methods("POST")
	api.Handle("/admin/events/{id}/attendees", middleware.RequireAdminAuth(http.HandlerFunc(handlers.GetEventAttendees))).Methods("GET")
	api.Handle("/admin/events/{id}/link-products", middleware.RequireAdminAuth(http.HandlerFunc(handlers.LinkProductsToEvent))).Methods("POST")
	api.Handle("/admin/events/{id}/products/{productId}", middleware.RequireAdminAuth(http.HandlerFunc(handlers.RemoveProductFromEvent))).Methods("DELETE")
	api.Handle(
		"/admin/events/{eventId}/tiers/{priceId}/products",
		middleware.RequireAdminAuth(
			http.HandlerFunc(handlers.AddProductsToTier),
		),
	).Methods("PUT")

	// Fulfillment management (admin only)
	api.Handle("/admin/fulfillments/pending", middleware.RequireAdminAuth(http.HandlerFunc(handlers.GetPendingFulfillments))).Methods("GET")
	api.Handle("/admin/fulfillments/{id}/status", middleware.RequireAdminAuth(http.HandlerFunc(handlers.UpdateFulfillmentStatus))).Methods("PUT")

	// Admin submission endpoints (requires authentication)
	api.Handle("/admin/submissions/pending-count", middleware.RequireAdminAuth(http.HandlerFunc(handlers.GetPendingSubmissionsCount))).Methods("GET")
	api.Handle("/admin/submissions/issues", middleware.RequireAdminAuth(http.HandlerFunc(handlers.GetAllSubmissionsWithIssues))).Methods("GET") // MOVED UP
	api.Handle("/admin/submissions/{id}", middleware.RequireAdminAuth(http.HandlerFunc(handlers.GetEventSubmissions))).Methods("GET")
	api.Handle("/admin/submissions/{submissionId}/approve", middleware.RequireAdminAuth(http.HandlerFunc(handlers.ApproveSubmission))).Methods("PUT")
	api.Handle("/admin/submissions/{submissionId}/deny", middleware.RequireAdminAuth(http.HandlerFunc(handlers.DenySubmission))).Methods("PUT")
	api.Handle("/admin/submissions/{submissionId}/payment-status", middleware.RequireAdminAuth(http.HandlerFunc(handlers.GetSubmissionPaymentStatus))).Methods("GET")
	api.Handle("/admin/submissions/{submissionId}/create-payment", middleware.RequireAdminAuth(http.HandlerFunc(handlers.CreateSubmissionPayment))).Methods("POST")
	api.Handle("/admin/submissions/{submissionId}/resend-email", middleware.RequireAdminAuth(http.HandlerFunc(handlers.ResendApprovalEmail))).Methods("POST")
	api.Handle("/admin/submissions/{submissionId}/update-email", middleware.RequireAdminAuth(http.HandlerFunc(handlers.UpdateSubmissionEmail))).Methods("PUT")
	api.Handle("/admin/submissions/{submissionId}/revoke", middleware.RequireAdminAuth(http.HandlerFunc(handlers.RevokeSubmission))).Methods("POST")

	// Webhook endpoint (no CORS needed)
	api.HandleFunc("/webhook", handlers.HandleWebhook).Methods("POST")

	// Serve static files from the SvelteKit build output
	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		staticDir = "./ui/build"
	}
	log.Println("Serving static files from: ", staticDir)

	router.PathPrefix("/").Handler(
		middleware.StaticCacheMiddleware(newSPAFileServer(staticDir)),
	).Methods("GET")

	// Setup CORS
	baseURL := os.Getenv("BASE_URL")
	allowedOrigins := []string{
		"http://localhost:3000",
		"http://localhost:5173",
	}
	if baseURL != "" {
		allowedOrigins = append(allowedOrigins, baseURL)
	}

	corsMiddleware := middleware.SetupCORS(allowedOrigins)
	corsHandler := corsMiddleware.Handler(router)

	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/webhook" {
			router.ServeHTTP(w, r)
			return
		}
		corsHandler.ServeHTTP(w, r)
	})

	handler = loggingMiddleware(handler)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}
