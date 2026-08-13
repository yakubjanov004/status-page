package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port     string
	DBPath   string
	SiteName string
	SiteURL  string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/status.db"
	}

	siteName := os.Getenv("SITE_NAME")
	if siteName == "" {
		siteName = "Darrov Status"
	}

	siteURL := os.Getenv("SITE_URL")
	if siteURL == "" {
		siteURL = "https://status.darrov.uz"
	}

	return &Config{
		Port:     port,
		DBPath:   dbPath,
		SiteName: siteName,
		SiteURL:  siteURL,
	}
}
