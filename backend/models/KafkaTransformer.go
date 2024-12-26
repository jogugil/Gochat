package models

import (
	"backend/entities"
	"encoding/json"

	"github.com/google/uuid"
)

// KafkaTransformer es un transformador que maneja la conversión de mensajes entre el formato interno de la aplicación y el formato de Kafka.
type KafkaTransformer struct{}

// NewKafkaTransformer crea una nueva instancia del KafkaTransformer.
func NewKafkaTransformer() *KafkaTransformer {
	return &KafkaTransformer{}
}

// TransformFromExternal convierte un mensaje de Kafka (KafkaMessage) a un mensaje interno (Message).
func (k *KafkaTransformer) TransformFromExternal(rawMsg []byte) (IMessage, error) {
	// Aquí transformamos el mensaje de Kafka (rawMsg) al formato de la aplicación.
	// Deberías hacer un parse de rawMsg a KafkaMessage y luego mapearlo a un Message.
	var kafkaMsg entities.KafkaMessage
	// Parsear el mensaje de Kafka (por ejemplo, usando json.Unmarshal)
	if err := json.Unmarshal(rawMsg, &kafkaMsg); err != nil {
		return nil, err
	}

	// Mapear KafkaMessage a Message (o cualquier tipo que implementes IMessage)
	msg := entities.Message{
		MessageId:   uuid.New(),
		MessageType: entities.Text, // Esto puede depender de los datos en kafkaMsg
		MessageText: kafkaMsg.Value,
		RoomID:      uuid.New(), // Dependería de los datos recibidos
		// Agregar otros campos según sea necesario
	}
	return &msg, nil
}

// TransformToExternal convierte un mensaje interno (Message) al formato de Kafka (KafkaMessage).
func (k *KafkaTransformer) TransformToExternal(msg IMessage) ([]byte, error) {
	// Aquí transformamos un mensaje de la aplicación (Message) a KafkaMessage
	message := msg.(*Message)

	kafkaMsg := entities.KafkaMessage{
		Key:     message.MessageId.String(),
		Value:   message.MessageText,
		Headers: map[string]string{"room": message.RoomID.String()},
	}

	// Convertimos KafkaMessage a []byte
	return json.Marshal(kafkaMsg)
}
