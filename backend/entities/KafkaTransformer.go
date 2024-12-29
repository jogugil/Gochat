package entities

import (
	"backend/utils"
	"encoding/json"
	"fmt"
	"time"
)

// KafkaTransformer es un transformador que maneja la conversión de mensajes entre el formato interno de la aplicación y el formato de Kafka.
type KafkaTransformer struct{}

// NewKafkaTransformer crea una nueva instancia del KafkaTransformer.
func NewKafkaTransformer() *KafkaTransformer {
	return &KafkaTransformer{}
}
func (k *KafkaTransformer) BuildMessage(key, value string, headers map[string]string) *KafkaMessage {
	return &KafkaMessage{
		Key:     key,
		Value:   value,
		Headers: headers,
	}
}

// TransformFromExternal convierte un mensaje de Kafka (KafkaMessage) a un mensaje interno (Message).
func (k *KafkaTransformer) TransformFromExternal(rawMsg []byte) (*Message, error) {
	// Definir la estructura que representa el formato del mensaje de Kafka
	var kafkaMsg KafkaMessage

	// Intentar deserializar el JSON recibido
	if err := json.Unmarshal(rawMsg, &kafkaMsg); err != nil {
		return nil, fmt.Errorf("error al deserializar el mensaje de Kafka: %w", err)
	}

	// Validar los campos esenciales del mensaje deserializado
	if kafkaMsg.Value == "" {
		return nil, fmt.Errorf("el mensaje de Kafka no contiene texto válido en 'Value'")
	}

	// Mapear el KafkaMessage al formato Message de la aplicación
	// Extraer valores de Headers
	nickname := k.getHeaderValue(kafkaMsg.Headers, "Nickname", "default-nickname")
	roomID := utils.ParseUUID(k.getHeaderValue(kafkaMsg.Headers, "RoomID", ""))
	roomName := k.getHeaderValue(kafkaMsg.Headers, "RoomName", "default-room")
	priority := utils.ParseInt(k.getHeaderValue(kafkaMsg.Headers, "Priority", "1"))
	originalLang := k.getHeaderValue(kafkaMsg.Headers, "OriginalLang", "unknown")

	// Construir el mensaje de la aplicación
	msg := &Message{
		MessageId:   utils.ParseUUID(kafkaMsg.Key), // Key es el ID del mensaje
		MessageType: Text,
		MessageText: kafkaMsg.Value,
		Nickname:    nickname,
		RoomID:      roomID,
		RoomName:    roomName,
		SendDate:    time.Now(),
		Metadata: Metadata{
			Priority:     priority,
			OriginalLang: originalLang,
		},
	}

	return msg, nil
}

// TransformToExternal convierte un mensaje interno (Message) al formato de Kafka (KafkaMessage).
func (k *KafkaTransformer) TransformToExternal(message *Message) ([]byte, error) {
	// Crear el objeto KafkaMessage y mapear los valores correspondientes
	kafkaMsg := KafkaMessage{
		Key:     message.MessageId.String(), // El Key es el MessageId (UUID como string)
		Value:   message.MessageText,        // El Value es el MessageText
		Headers: make(map[string]string),    // Iniciar el mapa de headers
	}

	// Mapear los valores adicionales a los headers
	kafkaMsg.Headers["RoomID"] = message.RoomID.String()                        // RoomID como UUID
	kafkaMsg.Headers["Nickname"] = message.Nickname                             // Nickname
	kafkaMsg.Headers["Priority"] = fmt.Sprintf("%d", message.Metadata.Priority) // Priority como string
	kafkaMsg.Headers["OriginalLang"] = message.Metadata.OriginalLang            // OriginalLang
	kafkaMsg.Headers["SendDate"] = message.SendDate.Format(time.RFC3339)        // Fecha de envío en formato RFC3339

	// Si tienes más campos en Message que necesitas incluir en el header, agrégales aquí
	// Ejemplo: kafkaMsg.Headers["OtroCampo"] = message.OtroCampo

	// Convertimos KafkaMessage a []byte
	rawMsg, err := json.Marshal(kafkaMsg)
	if err != nil {
		return nil, fmt.Errorf("error al serializar el mensaje de Kafka: %v", err)
	}

	// Retornar el mensaje serializado en formato JSON
	return rawMsg, nil
}
func (k *KafkaTransformer) getHeaderValue(headers map[string]string, key, defaultValue string) string {
	if value, exists := headers[key]; exists {
		return value
	}
	return defaultValue
}
