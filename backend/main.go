package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"MDM-matrix/api"
	"MDM-matrix/hub"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	dbConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Unable to parse DATABASE_URL: %v\n", err)
	}

	// Supabase's transaction pooler (port 6543) does not support persistent
	// prepared statements. pgx prepares and caches statements by default, so
	// use the unnamed extended-protocol execution mode, which remains compatible
	// with both transaction pooling and ordinary PostgreSQL connections.
	dbConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec

	dbPool, err := pgxpool.NewWithConfig(context.Background(), dbConfig)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}
	fmt.Println("Connected to Supabase successfully!")

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
