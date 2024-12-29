package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StatusService maneja la verificación del estado de NATS y Kafka
type StatusService struct{}

// NewStatusService crea una nueva instancia de StatusService
func NewStatusService() *StatusService {
	return &StatusService{}
}

// Status verifica el estado de NATS y Kafka
func (s *StatusService) Status(c *gin.Context) {
	// Verificar estado de NATS y Kafka
	isNATSConnected := checkNATSConnection()
	isKafkaConnected := checkKafkaConnection()

	// Responder con estado según las conexiones
	if isNATSConnected && isKafkaConnected {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "NATS y Kafka están funcionando correctamente",
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "not ok",
			"message": "Problema de conexión con NATS o Kafka",
		})
	}
}

// Simulación de verificación de conexión NATS
func checkNATSConnection() bool {
	// Aquí deberías agregar la lógica para verificar realmente la conexión a NATS
	return true // Simulado, siempre asume que NATS está conectado
}

// Simulación de verificación de conexión Kafka
func checkKafkaConnection() bool {
	// Aquí deberías agregar la lógica para verificar realmente la conexión a Kafka
	return true // Simulado, siempre asume que Kafka está conectado
}
