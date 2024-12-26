// /backend/utilities/env.go
package utils

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

// Structure to store the loaded environment variables
var envVars map[string]string
var once sync.Once

// ReloadEnvironmentVariables reloads the environment variables from the .env file in real-time
func ReloadEnvironmentVariables() {
	// Reload the environment variables from the .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error reloading the .env file: %w", err)
	}

	// Clear the previous variables map
	envVars = make(map[string]string)

	// Reload the environment variables one by one and assign custom names
	envVars["SecretKey"] = os.Getenv("SECRET_KEY")
	envVars["PortServer"] = os.Getenv("EXP_PORT")
	envVars["NameServer"] = os.Getenv("NAME_SERVER")
	envVars["URIMongo"] = os.Getenv("MONGODB_URI")
	envVars["NameMongo"] = os.Getenv("NAME_MONGO")
	envVars["SizeQueue"] = os.Getenv("EXP_SIZE_QMESSAGE")
	envVars["LogFile"] = os.Getenv("LOGFILE_GOCHAT")
	envVars["EmailAdmin"] = os.Getenv("EMAIL_ADMIN")
	envVars["GMAIL_USER"] = os.Getenv("GMAIL_USER")
	envVars["GMAIL_PASS"] = os.Getenv("GMAIL_PASS")
	envVars["GIN_MODE"] = os.Getenv("GIN_MODE")
}

// LoadEnvironmentVariables loads the environment variables from a .env file
func LoadEnvironmentVariables() {
	once.Do(func() {
		// Load the .env file
		if err := godotenv.Load(); err != nil {
			log.Fatal("LoadEnvironmentVariables: Error loading the .env file")
		}

		// Initialize the map to store environment variables
		envVars = make(map[string]string)

		// Load all environment variables into memory
		ReloadEnvironmentVariables()
	})
}

// GetEnvironmentVariable returns the value of a loaded environment variable
func GetEnvVariable(variableName string) (string, error) {
	value, exists := envVars[variableName]
	if !exists {
		return "", fmt.Errorf("the environment variable %s is not configured", variableName)
	}
	return value, nil
}

// splitEnv splits an environment variable into its name and value
func splitEnv(env string) []string {
	parts := make([]string, 2)
	equalsIndex := -1
	for i, c := range env {
		if c == '=' {
			equalsIndex = i
			break
		}
	}
	if equalsIndex != -1 {
		parts[0] = env[:equalsIndex]
		parts[1] = env[equalsIndex+1:]
	}
	return parts
}
