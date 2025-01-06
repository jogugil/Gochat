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
func (n *NatsTransformer) TransformFromExternalToGetMessage(rawMsg []byte) (*RequestListMessages, error) {
	var msg NatsMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil, err
	}
	// Decodificar
	messageText := string(msg.Data)

	log.Printf("\nNatsTransformer: TransformFromExternal: messageText:[%s]\n", messageText)
	// Extraemos el MessageId del header o generamos uno nuevo
	// Convertir MessageId a uuid.UUID

	roomId, err := uuid.Parse(n.getHeaderValue(msg.Headers, "roomid", ""))
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear roomid: %v", err)
	}

	lastmessageid, err := uuid.Parse(n.getHeaderValue(msg.Headers, "lastmessageid", ""))
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear lastmessageid: %v", err)
	}
	operation := n.getHeaderValue(msg.Headers, "operation", "")

	nickname := n.getHeaderValue(msg.Headers, "nickname", "")
	token := n.getHeaderValue(msg.Headers, "tokensesion", "")
	topic := n.getHeaderValue(msg.Headers, "topic", "")
	X_GoChat := n.getHeaderValue(msg.Headers, "x_gochat", "")

	//mensaje interno
	msgapp := &RequestListMessages{
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

func (n *NatsTransformer) TransformFromExternalToGetUsers(rawMsg []byte) (*RequestListUsers, error) {
	var msg NatsMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil, err
	}
	// Decodificar
	log.Printf("NatsTransformer: TransformFromExternal: msg:[%s]\n", msg)
	messageText := string(msg.Data)
	reqMessage := msg.Headers
	log.Printf("\nNatsTransformer: TransformFromExternal: messageText:[%s]\n", messageText)
	log.Printf("NatsTransformer: TransformFromExternal: reqMessage:[%s]\n", reqMessage)
	// Extraemos el MessageId del header o generamos uno nuevo
	// Convertir MessageId a uuid.UUID
	roomId, err := uuid.Parse(n.getHeaderValue(msg.Headers, "RoomId", ""))
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear RoomId: %v", err)
	}

	nickname := n.getHeaderValue(msg.Headers, "Nickname", "")
	token := n.getHeaderValue(msg.Headers, "TokenSesion", "")
	topic := n.getHeaderValue(msg.Headers, "Topic", "")
	X_GoChat := n.getHeaderValue(msg.Headers, "x_gochat", "")

	//mensaje interno
	msgapp := &RequestListUsers{
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
	messageID := n.getHeaderValue(natsMsg.Headers, "MessageId", "")
	if messageID == "" {
		messageID = uuid.New().String() // Si no hay MessageId, generamos uno nuevo
	}
	messageUUID := utils.ParseUUID(messageID) // Utilizamos la función ParseUUID para convertir el string a UUID

	// Extraemos otros valores de los headers
	nickname := n.getHeaderValue(natsMsg.Headers, "Nickname", "") // Nickname desde los headers
	token := n.getHeaderValue(natsMsg.Headers, "Token", "")       // Token de sesión de usuario, si no existe en la petición no puede jececutar ninguna operación
	roomName := natsMsg.Subject                                   // Subject es el RoomName
	priorityStr := n.getHeaderValue(natsMsg.Headers, "PriorityStr", "")
	originalLang := n.getHeaderValue(natsMsg.Headers, "OriginalLang", "")
	roomId, err := uuid.Parse(n.getHeaderValue(natsMsg.Headers, "RoomID", ""))
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear roomid: %v", err)
	}
	snd := n.getHeaderValue(natsMsg.Headers, "SendDate", "")
	sendDateT, err := utils.ConvertToRFC3339(snd)
	if err != nil {
		fmt.Println("Error al convertir la fecha:", err)
		sendDateT = time.Now()
	}
	// Convertir Priority si está presente
	priority := utils.ParseInt(priorityStr)
	//mensaje interno
	msg := Message{
		MessageId:   messageUUID,
		MessageType: Text,                 // Puedes ajustar esto dependiendo del tipo de mensaje que recibas
		Token:       token,                // Token de sesión de usuario, si no existe en la petición no puede jececutar ninguna operación
		MessageText: string(natsMsg.Data), // El cuerpo del mensaje
		RoomID:      roomId,               // Este valor se puede generar aquí o extraer de los headers si está disponible
		RoomName:    roomName,             // Usamos el Subject como RoomName
		Nickname:    nickname,             // Nickname desde los headers
		SendDate:    sendDateT,            // Fecha de envío actual
		ServerDate:  time.Now(),
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
		Timestamp: time.Now(),
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
func (n *NatsTransformer) TransformToExternalUsers(topic string, message *ResponseListUser) ([]byte, error) {
	// Crear el objeto NatsMessage y mapear los valores correspondientes
	natsMsg := NatsMessage{
		Subject: topic,                   // El Subject es el RoomName
		Data:    []byte(message.Message), // El Data es el MessageText
		Timestamp: time.Now(),
		Headers: make(map[string]interface{}),
	}
	// Convertir `AliveUsers` a JSON y agregarlo a los headers
	aliveUsersJSON, err := json.Marshal(message.AliveUsers)
	if err != nil {
		log.Fatalf("Error al serializar AliveUsers: %v", err)
	}

	natsMsg.Headers["Status"] = message.Status
	natsMsg.Headers["Message"] = message.Message
	natsMsg.Headers["TokenSesion"] = message.TokenSesion
	natsMsg.Headers["Nickname"] = message.Nickname
	natsMsg.Headers["RoomId"] = message.RoomId
	natsMsg.Headers["X_GoChat"] = message.X_GoChat
	// Aquí agregamos el JSON de `AliveUsers` como un string en los headers
	natsMsg.Headers["AliveUsers"] = string(aliveUsersJSON)

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
		Timestamp: time.Now(),
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
func (n *NatsTransformer) getHeaderValue(headers map[string]interface{}, key, defaultValue string) string {
	if value, exists := headers[key]; exists {
		log.Printf("NatsTransformer: getHeaderValue: value :%s", value)
		return value.(string)
	}
	log.Printf("NatsTransformer: getHeaderValue: defaultValue :%s", defaultValue)
	return defaultValue
}
