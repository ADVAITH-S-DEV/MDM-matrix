package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"MDM-matrix/api"
	"MDM-matrix/hub"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func bootstrapAdmin(ctx context.Context, dbPool *pgxpool.Pool) error {
	username := os.Getenv("ADMIN_USERNAME")
	password := os.Getenv("ADMIN_PASSWORD")

	// Existing database-managed credentials remain supported. When either
	// variable is set, require both so a partial deployment configuration does
	// not silently create an unusable admin account.
	if username == "" && password == "" {
		log.Println("ADMIN_USERNAME and ADMIN_PASSWORD are not set; using the existing admin_user record")
		return nil
	}
	if username == "" || password == "" {
		return fmt.Errorf("ADMIN_USERNAME and ADMIN_PASSWORD must both be set")
	}

	if _, err := dbPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS admin_user (
			username TEXT PRIMARY KEY,
			password_hash TEXT NOT NULL
		)`); err != nil {
		return fmt.Errorf("create admin_user table: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	if _, err := dbPool.Exec(ctx, `
		INSERT INTO admin_user (username, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash
	`, username, string(hash)); err != nil {
		return fmt.Errorf("upsert admin user: %w", err)
	}

	log.Printf("Admin credentials configured for username %q", username)
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}
	fmt.Println("Connected to Supabase successfully!")

	if err := bootstrapAdmin(context.Background(), dbPool); err != nil {
		log.Fatalf("Unable to configure admin credentials: %v", err)
	}

	h := hub.NewHub()
	adminHub := hub.NewAdminHub() // NEW

	// Setup routes
	http.HandleFunc("/health", api.CorsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Server is healthy"))
	}))

	http.HandleFunc("/login", api.CorsMiddleware(api.HandleLogin(dbPool)))
	http.HandleFunc("/enroll", api.CorsMiddleware(api.HandleEnroll(dbPool)))

	// UPDATED: Pass adminHub to the device WS handler
	http.HandleFunc("/ws", api.CorsMiddleware(api.HandleWS(dbPool, h, adminHub)))

	// NEW: Admin WS route
	http.HandleFunc("/admin/ws", api.CorsMiddleware(api.HandleAdminWS(adminHub)))

	http.HandleFunc("/devices", api.CorsMiddleware(api.AuthMiddleware(api.HandleGetDevices(dbPool))))
	http.HandleFunc("/devices/", api.CorsMiddleware(api.AuthMiddleware(api.HandleDispatchCommand(dbPool, h))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	address := ":" + port
	fmt.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(address, nil); err != nil {
		log.Fatalf("Server failed: %v\n", err)
	}
}
