package utils

import (
	"encoding/json"
	"log"
	"os"
	"strconv"

	"github.com/google/uuid"
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
func ParseUUID(value string) uuid.UUID {
	if value == "" {
		return uuid.New() // Generar nuevo UUID si no se proporciona
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.New() // Generar nuevo UUID si el valor es inválido
	}
	return parsed
}
func ParseInt(value string) int {
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0 // Retornar 0 si el valor no es un entero válido
	}
	return parsed
}
