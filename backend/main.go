package main

import (
	"log"
	"os"

	"interactive-scraper/internal/api"
	"interactive-scraper/internal/database"
	"interactive-scraper/internal/scraper"
	"interactive-scraper/internal/service"
)

func main() {
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.InitSchema(db); err != nil {
		log.Fatalf("Failed to initialize database schema: %v", err)
	}

	dataService := service.NewDataService(db)
	authService := service.NewAuthService()
	authService.SetDB(db)

	scraperService := scraper.NewScraperService(db)
	go scraperService.Start()

	router := api.SetupRouter(dataService, authService, scraperService)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

