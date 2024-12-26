package entities

import (
	"github.com/google/uuid"
)

// MessageBroker es la interfaz que define los métodos comunes
// para interactuar con los brokers de mensajes.
// MessageBroker define las operaciones básicas que un broker de mensajes debería soportar.
type MessageBroker interface {
	// Publica un mensaje en un tópico específico.
	Publish(topic string, message []byte) error

	// Se suscribe a un tópico y recibe mensajes, procesándolos mediante el manejador proporcionado.
	Subscribe(topic string, handler func(message []byte) error) error

	// Se suscribe a un patrón de tópicos y recibe mensajes que coincidan con el patrón, procesándolos con el manejador proporcionado.
	SubscribeWithPattern(pattern string, handler func(message []byte) error) error

	// Confirma la recepción de un mensaje, útil para sistemas que requieren confirmación manual.
	Acknowledge(messageID string) error

	// Reintenta el envío de un mensaje fallido, si el broker soporta reintentos.
	Retry(messageID string) error

	// Obtiene las métricas actuales del broker (por ejemplo, número de mensajes enviados, latencia, etc.).
	GetMetrics() (map[string]interface{}, error)

	// Crea un tópico si el broker lo requiere.
	CreateTopic(topic string) error

	// Retrieves messages from the broker by their ID.
	GetMessagesFromId(roomId string, messageId uuid.UUID) ([]Message, error)

	// Cierra la conexión con el broker, liberando recursos.
	Close() error
}
