package main

import (
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
	} else {
		log.Println("Redis initialized successfully")
	}

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

		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
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
	r := mux.NewRouter()

	// API Routes
	api := r.PathPrefix("/api").Subrouter()

	// Healthcheck
	api.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	// Products endpoints
	api.HandleFunc("/products", handlers.GetProducts).Methods("GET")
	api.HandleFunc("/products/{id}", handlers.GetProduct).Methods("GET")
	api.HandleFunc("/products/{id}/prices", handlers.GetProductPrices).Methods("GET", "OPTIONS")

	// Auth endpoints
	api.HandleFunc("/auth/login", handlers.Login).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/logout", handlers.Logout).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/validate", handlers.ValidateToken).Methods("GET", "OPTIONS")

	// Payment endpoints
	api.HandleFunc("/create-payment-intent", handlers.CreatePaymentIntent).Methods("POST")
	api.HandleFunc("/create-checkout-session", handlers.CreateCheckoutSession).Methods("POST")

	// Content placement endpoints
	api.HandleFunc("/content-placements", handlers.GetAllContentPlacements).Methods("GET", "OPTIONS")
	api.HandleFunc("/content-placements/register", handlers.RegisterContentPlacement).Methods("POST", "OPTIONS")
	api.HandleFunc("/content-placements/dynamic", handlers.GetDynamicContentPlacements).Methods("GET", "OPTIONS")
	api.HandleFunc("/content-placements/{id}", handlers.GetContentPlacement).Methods("GET", "OPTIONS")
	api.HandleFunc("/content-placements/{id}", handlers.UpdateContentPlacement).Methods("PUT", "OPTIONS")
	api.HandleFunc("/content-placements/by-media/{key}", handlers.GetContentPlacementsByMedia).Methods("GET", "OPTIONS")

	// Admin endpoints (requires authentication)
	api.HandleFunc("/admin/create-product", handlers.CreateProduct).Methods("POST", "OPTIONS")
	api.HandleFunc("/admin/update-product/{id}", handlers.UpdateProduct).Methods("PUT", "OPTIONS")
	api.HandleFunc("/admin/delete-product/{id}", handlers.DeleteProduct).Methods("DELETE", "OPTIONS")

	// Media management endpoints (requires authentication)
	api.HandleFunc("/admin/media", handlers.ListMedia).Methods("GET", "OPTIONS")
	api.HandleFunc("/admin/media/upload", handlers.UploadMedia).Methods("POST", "OPTIONS")
	api.HandleFunc("/admin/media/delete", handlers.DeleteMedia).Methods("DELETE", "OPTIONS")

	api.HandleFunc("/admin/create-price", handlers.CreatePrice).Methods("POST", "OPTIONS")
	api.HandleFunc("/admin/update-price/{id}", handlers.UpdatePrice).Methods("PUT", "OPTIONS")
	api.HandleFunc("/admin/archive-price/{id}", handlers.ArchivePrice).Methods("PUT", "OPTIONS")
	api.HandleFunc("/admin/set-default-price/{id}", handlers.SetDefaultPrice).Methods("PUT", "OPTIONS")

	// Webhook endpoint (no CORS needed)
	r.HandleFunc("/webhook", handlers.HandleWebhook).Methods("POST")

	// Setup CORS
	allowedOrigins := []string{
		"http://localhost:3000",
		"https://theeurohaus.com",
	}
	corsMiddleware := middleware.SetupCORS(allowedOrigins)

	// Apply CORS to all routes except webhooks
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/webhook" {
			r.ServeHTTP(w, req)
		} else {
			corsMiddleware.Handler(r).ServeHTTP(w, req)
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
