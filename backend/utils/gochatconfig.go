package utils

import (
	"encoding/json"
	"log"
	"os"
)

// Configuración global
type Config struct {
	MessagingType string `json:"messagingType"` // Ejemplo: "NATS" o "Kafka"
}

var AppConfig Config

// Cargar configuración desde el archivo
func LoadConfig(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error al abrir el archivo de configuración: %v", err)
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&AppConfig)
	if err != nil {
		log.Printf("Error al leer la configuración: %v", err)
		return err
	}
	return nil
}
