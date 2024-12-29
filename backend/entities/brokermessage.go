package entities

import (
	"fmt"
	"log"

	"github.com/google/uuid"
)

type BrokerType string

const (
	KafkaBrokerType BrokerType = "kafka"
	NATSBrokerType  BrokerType = "nats"
)

// Factory para crear el broker adecuado
func MessageBrokerFactory(config map[string]interface{}) (MessageBroker, error) {
	log.Printf("MessageBroker: MessageBrokerFactory:  config :%v", config)
	// Obtener el valor del tipo de broker
	brokerTypeInterface, ok := config["broker_type"]
	if !ok {
		log.Printf("MessageBroker:MessageBrokerFactory: 'broker_type' no está presente en la configuración")
		return nil, fmt.Errorf("'broker_type' no está presente en la configuración")
	}

	// Asegurarse de que brokerType sea una cadena
	brokerType, ok := brokerTypeInterface.(string)
	if !ok {
		log.Printf("MessageBroker:MessageBrokerFactory: 'broker_type' no es una cadena")
		return nil, fmt.Errorf("'broker_type' no es una cadena")
	}
	log.Printf("MessageBroker: MessageBrokerFactory:   BrokerType(brokerType) :%v", brokerType)
	// Convertir brokerType a BrokerType
	switch BrokerType(brokerType) {
	case KafkaBrokerType:
		msgBroker, err := NewKafkaBroker(config)

		return msgBroker, err
	case NATSBrokerType:
		msgBroker, err := NewNatsBroker(config)

		return msgBroker, err
	default:
		log.Printf("MessageBroker:MessageBrokerFactory: tipo de broker desconocido: %s", brokerType)
		return nil, fmt.Errorf("tipo de broker desconocido: %s", brokerType)
	}
}

// MessageBroker es la interfaz que define los métodos comunes
// para interactuar con los brokers de mensajes.
// MessageBroker define las operaciones básicas que un broker de mensajes debería soportar.
type MessageBroker interface {
	//subscribir las funbcines de callback al topic correspondiente
	OnMessage(topic string, callback func(interface{})) error
	// Publica un mensaje en un tópico específico.
	Publish(topic string, message *Message) error

	// Se suscribe a un tópico y recibe mensajes, procesándolos mediante el manejador proporcionado.
	Subscribe(topic string, handler func(message *Message) error) error

	// Se suscribe a un patrón de tópicos y recibe mensajes que coincidan con el patrón, procesándolos con el manejador proporcionado.
	SubscribeWithPattern(pattern string, handler func(message *Message) error) error

	// Confirma la recepción de un mensaje, útil para sistemas que requieren confirmación manual.
	Acknowledge(messageID uuid.UUID) error

	// Reintenta el envío de un mensaje fallido, si el broker soporta reintentos.
	Retry(messageID uuid.UUID) error

	// Obtiene las métricas actuales del broker (por ejemplo, número de mensajes enviados, latencia, etc.).
	GetMetrics() (map[string]interface{}, error)

	// Crea un tópico si el broker lo requiere.
	CreateTopic(topic string) error

	// Obtiene los mensajes no leídos de un tópico (basado en roomId).
	GetUnreadMessages(topic string) ([]Message, error)

	// Retrieves messages from the broker by their ID.
	GetMessagesFromId(topic string, messageId uuid.UUID) ([]Message, error)

	// Devuelve count entities.Message del rtopic roomId
	GetMessagesWithLimit(topic string, messageId uuid.UUID, count int) ([]Message, error)
	// Cierra la conexión con el broker, liberando recursos.
	Close() error
}
