package models

import (
	"backend/entities"
	"encoding/json"

	"github.com/google/uuid"
)

// NatsTransformer es un transformador que maneja la conversión de mensajes entre el formato interno de la aplicación y el formato de NATS.
type NatsTransformer struct{}

// NewNatsTransformer crea una nueva instancia del NatsTransformer.
func NewNatsTransformer() *NatsTransformer {
	return &NatsTransformer{}
}

// TransformFromExternal convierte un mensaje de NATS (NatsMessage) a un mensaje interno (Message).
func (n *NatsTransformer) TransformFromExternal(rawMsg []byte) (IMessage, error) {
	// Aquí transformamos el mensaje de NATS (rawMsg) al formato de la aplicación.
	// Deberías hacer un parse de rawMsg a NatsMessage y luego mapearlo a un Message.
	var natsMsg entities.NatsMessage
	if err := json.Unmarshal(rawMsg, &natsMsg); err != nil {
		return nil, err
	}

	// Mapear NatsMessage a Message (o cualquier tipo que implementes IMessage)
	msg := entities.Message{
		MessageId:   uuid.New(),
		MessageType: entities.Text, // Esto puede depender de los datos en natsMsg
		MessageText: natsMsg.Data,
		RoomID:      uuid.New(), // Dependería de los datos recibidos
		// Agregar otros campos según sea necesario
	}
	return &msg, nil
}

// TransformToExternal convierte un mensaje interno (Message) al formato de NATS (NatsMessage).
func (n *NatsTransformer) TransformToExternal(msg IMessage) ([]byte, error) {
	// Aquí transformamos un mensaje de la aplicación (Message) a NatsMessage
	message := msg.(*Message)

	natsMsg := entities.NatsMessage{
		Subject: message.RoomName,
		Data:    message.MessageText,
		Header:  map[string]string{"room": message.RoomID.String()},
	}

	// Convertimos NatsMessage a []byte
	return json.Marshal(natsMsg)
}
