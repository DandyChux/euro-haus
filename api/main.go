package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"euro-haus-api/handlers"
	"euro-haus-api/middleware"

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

	// Products endpoints
	api.HandleFunc("/products", handlers.GetProducts).Methods("GET")
	api.HandleFunc("/products/{id}", handlers.GetProduct).Methods("GET")

	// Auth endpoints
	api.HandleFunc("/auth/login", handlers.Login).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/logout", handlers.Logout).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/validate", handlers.ValidateToken).Methods("GET", "OPTIONS")

	// Payment endpoints
	api.HandleFunc("/create-payment-intent", handlers.CreatePaymentIntent).Methods("POST")
	api.HandleFunc("/create-checkout-session", handlers.CreateCheckoutSession).Methods("POST")

	// Admin endpoints (requires authentication)
	api.HandleFunc("/admin/create-product", handlers.CreateProduct).Methods("POST", "OPTIONS")
	api.HandleFunc("/admin/update-product/{id}", handlers.UpdateProduct).Methods("PUT", "OPTIONS")
	api.HandleFunc("/admin/delete-product/{id}", handlers.DeleteProduct).Methods("DELETE", "OPTIONS")

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
