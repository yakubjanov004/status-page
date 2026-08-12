package config

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	DBPath        string
	AdminUsername string
	AdminPassword string
	SessionSecret string
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

	adminUser := os.Getenv("ADMIN_USERNAME")
	if adminUser == "" {
		adminUser = "admin"
	}

	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "admin"
	}

	return &Config{
		Port:          port,
		DBPath:        dbPath,
		AdminUsername: adminUser,
		AdminPassword: adminPass,
		SessionSecret: generateSessionSecret(),
	}
}

func generateSessionSecret() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "fallback-secret-for-errors-only-12345"
	}
	return base64.StdEncoding.EncodeToString(b)
}
