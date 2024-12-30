package entities

import (
	"backend/utils"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// NatsTransformer es un transformador que maneja la conversión de mensajes entre el formato interno de la aplicación y el formato de NATS.
type NatsTransformer struct{}

// NewNatsTransformer crea una nueva instancia del NatsTransformer.
func NewNatsTransformer() *NatsTransformer {
	return &NatsTransformer{}
}
func (n *NatsTransformer) BuildMessage(subject, data string, header map[string]interface{}) *NatsMessage {
	return &NatsMessage{
		Subject: subject,
		Data:    []byte(data),
		Headers: header,
	}
}
func (n *NatsTransformer) TransformFromExternalToGetMessage(rawMsg []byte) (*RequestLisMessages, error) {
	var msg NatsMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil, err
	}
	// Decodificar
	messageText := string(msg.Data)
	reqMessage := msg.Headers
	log.Printf("\nNatsTransformer: TransformFromExternal: messageText:[%s]\n", messageText)
	// Extraemos el MessageId del header o generamos uno nuevo
	// Convertir MessageId a uuid.UUID
	roomId, err := uuid.Parse(reqMessage["roomid"].(string))
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear roomid: %v", err)
	}

	lastmessageid, err := uuid.Parse(reqMessage["lastmessageid"].(string))
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear lastmessageid: %v", err)
	}
	operation := reqMessage["operation"].(string)

	nickname := reqMessage["nickname"].(string)
	token := reqMessage["tokensesion"].(string)
	topic := reqMessage["topic"].(string)
	X_GoChat := reqMessage["x_gochat"].(string)

	//mensaje interno
	msgapp := &RequestLisMessages{
		RoomId:        roomId,
		TokenSesion:   token, //
		Nickname:      nickname,
		Operation:     operation,
		LastMessageId: lastmessageid,
		Topic:         topic,
		X_GoChat:      X_GoChat,
	}
	log.Printf("NatsTransformer: TransformFromExternal: msg :%v", msgapp)
	return msgapp, nil
}

func (n *NatsTransformer) TransformFromExternalToGetUsers(rawMsg []byte) (*RequestLisUsers, error) {
	var msg NatsMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil, err
	}
	// Decodificar
	messageText := string(msg.Data)
	reqMessage := msg.Headers
	log.Printf("\nNatsTransformer: TransformFromExternal: messageText:[%s]\n", messageText)
	// Extraemos el MessageId del header o generamos uno nuevo
	// Convertir MessageId a uuid.UUID
	roomId, err := uuid.Parse(reqMessage["RoomId"].(string))
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear RoomId: %v", err)
	}

	nickname := reqMessage["nickname"].(string)
	token := reqMessage["tokensesion"].(string)
	topic := reqMessage["topic"].(string)
	X_GoChat := reqMessage["x_gochat"].(string)

	//mensaje interno
	msgapp := &RequestLisUsers{
		RoomId:      roomId,
		TokenSesion: token, //
		Nickname:    nickname,
		Topic:       topic,
		X_GoChat:    X_GoChat,
	}
	log.Printf("NatsTransformer: TransformFromExternal: msg :%v", msgapp)
	return msgapp, nil
}

// TransformFromExternal convierte un mensaje de NATS (NatsMessage) a un mensaje interno (Message).
func (n *NatsTransformer) TransformFromExternal(rawMsg []byte) (*Message, error) {
	// Parsear el mensaje de NATS (rawMsg) a NatsMessage
	var natsMsg NatsMessage
	if err := json.Unmarshal(rawMsg, &natsMsg); err != nil {
		return nil, err
	}

	// Extraemos el MessageId del header o generamos uno nuevo
	messageID := natsMsg.Headers["MessageId"].(string)
	if messageID == "" {
		messageID = uuid.New().String() // Si no hay MessageId, generamos uno nuevo
	}
	messageUUID := utils.ParseUUID(messageID) // Utilizamos la función ParseUUID para convertir el string a UUID

	// Extraemos otros valores de los headers
	nickname := natsMsg.Headers["Nickname"].(string) // Nickname desde los headers
	roomName := natsMsg.Subject                      // Subject es el RoomName
	priorityStr := natsMsg.Headers["Priority"].(string)
	originalLang := natsMsg.Headers["OriginalLang"].(string)

	// Convertir Priority si está presente
	priority := utils.ParseInt(priorityStr)
	//mensaje interno
	msg := Message{
		MessageId:   messageUUID,
		MessageType: Text,                 // Puedes ajustar esto dependiendo del tipo de mensaje que recibas
		MessageText: string(natsMsg.Data), // El cuerpo del mensaje
		RoomID:      uuid.New(),           // Este valor se puede generar aquí o extraer de los headers si está disponible
		RoomName:    roomName,             // Usamos el Subject como RoomName
		Nickname:    nickname,             // Nickname desde los headers
		SendDate:    time.Now(),           // Fecha de envío actual
		Metadata: Metadata{
			Priority:     priority,     // Asignamos la prioridad desde los headers
			OriginalLang: originalLang, // Asignamos el idioma original desde los headers
		},
	}

	return &msg, nil
}

// TransformToExternal convierte un mensaje interno (Message) al formato de NATS (NatsMessage).
func (n *NatsTransformer) TransformToExternal(message *Message) ([]byte, error) {
	// Crear el objeto NatsMessage y mapear los valores correspondientes
	natsMsg := NatsMessage{
		Subject: message.RoomName,            // El Subject es el RoomName
		Data:    []byte(message.MessageText), // El Data es el MessageText
		Headers: make(map[string]interface{}),
	}

	// Mapear los valores adicionales a los headers
	natsMsg.Headers["MessageId"] = message.MessageId.String()                  // MessageId como UUID
	natsMsg.Headers["Nickname"] = message.Nickname                             // Nickname desde el mensaje
	natsMsg.Headers["Priority"] = fmt.Sprintf("%d", message.Metadata.Priority) // Priority como string
	natsMsg.Headers["OriginalLang"] = message.Metadata.OriginalLang            // OriginalLang desde el mensaje
	natsMsg.Headers["SendDate"] = message.SendDate.Format(time.RFC3339)        // Fecha de envío en formato RFC3339

	// Serializar el mensaje NATS a JSON
	rawMsg, err := json.Marshal(natsMsg)
	if err != nil {
		return nil, fmt.Errorf("error al serializar el mensaje de NATS: %v", err)
	}

	// Retornar el mensaje serializado en formato JSON
	return rawMsg, nil
}

// TransformToExternal convierte un mensaje interno (Message) al formato de NATS (NatsMessage).
func (n *NatsTransformer) TransformToExternalUsers(message *ResponseListUser) ([]byte, error) {
	// Crear el objeto NatsMessage y mapear los valores correspondientes
	natsMsg := NatsMessage{
		Subject: message.RoomId.String(), // El Subject es el RoomName
		Data:    []byte(message.Message), // El Data es el MessageText
		Headers: make(map[string]interface{}),
	}

	// Mapear los valores adicionales a los headers
	natsMsg.Headers["AliveUsers"] = message.AliveUsers

	// Serializar el mensaje NATS a JSON
	rawMsg, err := json.Marshal(natsMsg)
	if err != nil {
		return nil, fmt.Errorf("error al serializar el mensaje de NATS: %v", err)
	}

	// Retornar el mensaje serializado en formato JSON
	return rawMsg, nil
}

// TransformToExternal convierte un mensaje interno (Message) al formato de NATS (NatsMessage).
func (n *NatsTransformer) TransformToExternalMessages(message *ResponseListMessages) ([]byte, error) {
	// Crear el objeto NatsMessage y mapear los valores correspondientes
	natsMsg := NatsMessage{
		Subject: message.RoomId.String(), // El Subject es el RoomName
		Data:    []byte(message.Message), // El Data es el MessageText
		Headers: make(map[string]interface{}),
	}

	// Mapear los valores adicionales a los headers
	natsMsg.Headers["MessageResponse"] = message.ListMessage

	// Serializar el mensaje NATS a JSON
	rawMsg, err := json.Marshal(natsMsg)
	if err != nil {
		return nil, fmt.Errorf("error al serializar el mensaje de NATS: %v", err)
	}

	// Retornar el mensaje serializado en formato JSON
	return rawMsg, nil
}
