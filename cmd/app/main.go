package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"shopDashboard/app"
	"shopDashboard/app/config"
	"shopDashboard/app/db"

	"github.com/anthdm/superkit/kit"
	"github.com/go-chi/chi/v5"
)

func main() {
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		if secret := os.Getenv("SUPERKIT_SECRET"); secret != "" {
			os.WriteFile(".env", []byte("SUPERKIT_SECRET="+secret+"\n"), 0644)
		}
	}

	kit.Setup()

	cfg := config.Get()

	if cfg.DBHost != "" || cfg.DatabaseURL != "" {
		if err := db.Connect(cfg); err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()
	}

	router := chi.NewMux()

	app.InitializeMiddleware(router)
	app.InitializeRoutes(router)

	fmt.Printf("Shop Dashboard running on http://localhost%s\n", cfg.ListenAddr)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, router))
}
