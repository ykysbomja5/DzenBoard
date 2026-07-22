package main

import (
	"log"
	"net/http"
	"os"

	"nasha_igrushka/database"
	"nasha_igrushka/handlers"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Initialize database
	if err := database.InitDB(); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	// Ensure upload and temp folders exist
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	tempDir := os.Getenv("TEMP_DIR")
	if tempDir == "" {
		tempDir = "./temp"
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create router using modern Go ServeMux (supports methods)
	mux := http.NewServeMux()

	// Static assets routes
	fsStatic := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fsStatic))

	fsUploads := http.FileServer(http.Dir(uploadDir))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", fsUploads))

	// Frontend Page routes
	mux.HandleFunc("GET /{$}", handlers.HomeHandler)
	mux.HandleFunc("GET /catalog", handlers.CatalogHandler)
	mux.HandleFunc("GET /product", handlers.ProductHandler)
	mux.HandleFunc("GET /cart", handlers.CartHandler)
	mux.HandleFunc("GET /checkout", handlers.CheckoutHandler)
	mux.HandleFunc("GET /blog", handlers.BlogHandler)
	mux.HandleFunc("GET /blog-post", handlers.BlogPostHandler)
	mux.HandleFunc("GET /contacts", handlers.ContactsHandler)

	// Frontend AJAX APIs
	mux.HandleFunc("POST /api/order", handlers.SubmitOrderHandler)
	mux.HandleFunc("POST /api/review", handlers.SubmitReviewHandler)

	// Admin views & APIs
	mux.HandleFunc("GET /admin", handlers.AdminDashboardHandler)
	mux.HandleFunc("GET /api/admin/stats", handlers.AdminStatsHandler)
	mux.HandleFunc("GET /api/admin/orders", handlers.AdminOrdersHandler)
	mux.HandleFunc("POST /api/admin/orders/status", handlers.AdminUpdateOrderStatusHandler)

	// 1C CommerceML Exchange Endpoint
	// Direct mapping to /export/exchange1c.php for 1C client compatibility
	mux.HandleFunc("GET /export/exchange1c.php", handlers.Exchange1cHandler)
	mux.HandleFunc("POST /export/exchange1c.php", handlers.Exchange1cHandler)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s...", port)
	log.Printf("Web URL: http://localhost:%s/", port)
	log.Printf("Admin panel: http://localhost:%s/admin", port)
	log.Printf("1C Sync endpoint: http://localhost:%s/export/exchange1c.php", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
