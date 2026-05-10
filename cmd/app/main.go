package main

import (
	"fmt"
	"log"
	"net/http"
	"shopDashboard/app"
	"shopDashboard/app/config"
	"shopDashboard/app/db"

	"github.com/anthdm/superkit/kit"
	"github.com/go-chi/chi/v5"
)

func main() {
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
