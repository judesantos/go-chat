// Package config provides a centralized access to application initialization configuration variables
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func init() {

	// Using dotenv interface
	var err error = nil
	// Get app. config from .env file
	envFile := os.Getenv("ENV_FILE") // ENV_FILE specified in command
	if envFile != "" {
		err = godotenv.Load(envFile)
	} else {
		err = godotenv.Load() // Get default .env file in root
	}

	if err != nil {
		log.Fatalf("config.Initialize() - Error loading .env file: %s", err.Error())
	}
}

func GetValue(keyName string) string {
	return os.Getenv(keyName)
}
