package entities

import (
	"backend/utils"
	"encoding/json"
	"fmt"
	"log"
	"strings"
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

	log.Printf("\nNatsTransformer: TransformFromExternalToGetMessage: messageText:[%s]\n", messageText)
	// Extraemos el MessageId del header o generamos uno nuevo
	// Convertir MessageId a uuid.UUID

	roomId, err := uuid.Parse(n.getHeaderValue(msg.Headers, "roomid", ""))
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternalToGetMessage: error al parsear roomid: %v", err)
	}

	lastmessageid, err := uuid.Parse(n.getHeaderValue(msg.Headers, "lastmessageid", ""))
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternalToGetMessage: error al parsear lastmessageid: %v", err)
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
	log.Printf("NatsTransformer: TransformFromExternalToGetMessage: msg :%v", msgapp)
	return msgapp, nil
}

func (n *NatsTransformer) TransformFromExternalToGetUsers(rawMsg []byte) (*RequestListUsers, error) {
	var msg NatsMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil, err
	}
	// Decodificar
	log.Printf("NatsTransformer: TransformFromExternalToGetUsers: msg:[%s]\n", msg)
	messageText := string(msg.Data)
	reqMessage := msg.Headers
	log.Printf("\nNatsTransformer: TransformFromExternalToGetUsers: messageText:[%s]\n", messageText)
	log.Printf("NatsTransformer: TransformFromExternalToGetUsers: reqMessage:[%s]\n", reqMessage)
	// Extraemos el MessageId del header o generamos uno nuevo
	// Convertir MessageId a uuid.UUID
	roomId, err := uuid.Parse(n.getHeaderValue(msg.Headers, "RoomId", ""))
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternalToGetUsers: error al parsear RoomId: %v", err)
	}

	nickname := n.getHeaderValue(msg.Headers, "Nickname", "")
	token := n.getHeaderValue(msg.Headers, "TokenSesion", "")
	topic := n.getHeaderValue(msg.Headers, "Topic", "")
	request := n.getHeaderValue(msg.Headers, "request", "")
	X_GoChat := n.getHeaderValue(msg.Headers, "x_gochat", "")

	//mensaje interno
	msgapp := &RequestListUsers{
		RoomId:      roomId,
		TokenSesion: token, //
		Nickname:    nickname,
		Request:     request,
		Topic:       topic,
		X_GoChat:    X_GoChat,
	}
	log.Printf("NatsTransformer: TransformFromExternalToGetUsers: msg :%v", msgapp)
	return msgapp, nil
}

// TransformFromExternal convierte un mensaje de NATS (NatsMessage) a un mensaje interno (Message).
func (n *NatsTransformer) TransformFromExternal(rawMsg []byte) (*Message, error) {
	// Log para ver el rawMsg recibido
	log.Printf("NatsTransformer: TransformFromExternal: rawMsg recibido: [%s]", string(rawMsg))

	// Parsear el mensaje de NATS (rawMsg) a NatsMessage
	var natsMsg NatsMessage
	if err := json.Unmarshal(rawMsg, &natsMsg); err != nil {
		log.Printf("NatsTransformer: TransformFromExternal: error unmarshaling rawMsg: %v", err)
		// Agregar el rawMsg que causó el error para facilitar la depuración
		log.Printf("NatsTransformer: TransformFromExternal: rawMsg con error de deserialización: [%s]\n", string(rawMsg))
		return nil, err
	}

	// Log de la data después de deserializar
	log.Printf("\nNatsTransformer: TransformFromExternal: messageText (deserialized) [%s]\n", string(natsMsg.Data))

	// Extraemos el MessageId del header o generamos uno nuevo
	messageID := n.getHeaderValue(natsMsg.Headers, "MessageId", "")
	log.Printf("NatsTransformer: TransformFromExternal: Extracted MessageId: [%s]\n", messageID)
	if messageID == "" {
		messageID = uuid.New().String() // Si no hay MessageId, generamos uno nuevo
		log.Printf("NatsTransformer: TransformFromExternal: Generated new MessageId: [%s]\n", messageID)
	}
	messageUUID := utils.ParseUUID(messageID) // Utilizamos la función ParseUUID para convertir el string a UUID

	// Extraemos otros valores de los headers
	nickname := n.getHeaderValue(natsMsg.Headers, "Nickname", "")
	log.Printf("NatsTransformer: TransformFromExternal: Nickname extracted: [%s]\n", nickname)
	token := n.getHeaderValue(natsMsg.Headers, "Token", "")
	log.Printf("NatsTransformer: TransformFromExternal: Token extracted: [%s]\n", token)
	roomName := natsMsg.Subject
	log.Printf("NatsTransformer: TransformFromExternal: RoomName (subject) extracted: [%s]\n", roomName)

	priorityStr := n.getHeaderValue(natsMsg.Headers, "PriorityStr", "")
	log.Printf("NatsTransformer: TransformFromExternal: PriorityStr extracted: [%s]\n", priorityStr)

	originalLang := n.getHeaderValue(natsMsg.Headers, "OriginalLang", "")
	log.Printf("NatsTransformer: TransformFromExternal: OriginalLang extracted: [%s]\n", originalLang)

	roomId, err := uuid.Parse(n.getHeaderValue(natsMsg.Headers, "RoomID", ""))
	if err != nil {
		log.Printf("NatsTransformer: TransformFromExternal: Error parsing RoomID: %v\n", err)
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear roomid: %v\n", err)
	}
	log.Printf("NatsTransformer: TransformFromExternal: RoomID parsed: [%s]\n", roomId)

	snd := n.getHeaderValue(natsMsg.Headers, "SendDate", "")
	log.Printf("NatsTransformer: TransformFromExternal: SendDate extracted: [%s]\n", snd)

	sendDateT, err := utils.ConvertToRFC3339(snd)
	if err != nil {
		log.Printf("NatsTransformer: TransformFromExternal: Error converting SendDate: %v\n", err)
		sendDateT = time.Now()
		log.Printf("NatsTransformer: TransformFromExternal: Using current time as SendDate: [%s]\n", sendDateT)
	}
	log.Printf("NatsTransformer: TransformFromExternal: SendDate converted: [%s]\n", sendDateT)

	// Convertir Priority si está presente
	priority := utils.ParseInt(priorityStr)
	log.Printf("NatsTransformer: TransformFromExternal: Priority converted: [%d]\n", priority)

	// Crear el mensaje interno
	msg := Message{
		MessageId:   messageUUID,
		MessageType: Text,                 // Puedes ajustar esto dependiendo del tipo de mensaje que recibas
		Token:       token,                // Token de sesión de usuario
		MessageText: string(natsMsg.Data), // El cuerpo del mensaje
		RoomID:      roomId,               // Este valor se puede generar aquí o extraer de los headers
		RoomName:    roomName,             // Usamos el Subject como RoomName
		Nickname:    nickname,             // Nickname desde los headers
		SendDate:    sendDateT,            // Fecha de envío
		ServerDate:  time.Now(),
		Metadata: Metadata{
			Priority:     priority,     // Asignamos la prioridad desde los headers
			OriginalLang: originalLang, // Asignamos el idioma original desde los headers
		},
	}

	// Log del mensaje final antes de devolverlo
	log.Printf("\nNatsTransformer: TransformFromExternal: Final Message:[%v]\n", msg)

	return &msg, nil
}

// TransformToExternal convierte un mensaje interno (Message) al formato de NATS (NatsMessage).
func (n *NatsTransformer) TransformToExternal(message *Message) ([]byte, error) {
	// Crear el objeto NatsMessage y mapear los valores correspondientes
	natsMsg := NatsMessage{
		Subject:   message.RoomName,            // El Subject es el RoomName
		Data:      []byte(message.MessageText), // El Data es el MessageText
		Timestamp: time.Now(),
		Headers:   make(map[string]interface{}),
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
		Subject:   topic,                   // El Subject es el RoomName
		Data:      []byte(message.Message), // El Data es el MessageText
		Timestamp: time.Now(),
		Headers:   make(map[string]interface{}),
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
		Subject:   message.RoomId.String(), // El Subject es el RoomName
		Data:      []byte(message.Message), // El Data es el MessageText
		Timestamp: time.Now(),
		Headers:   make(map[string]interface{}),
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
	// Convertir la clave de búsqueda a minúsculas
	key = strings.ToLower(key)

	// Crear un nuevo mapa con las claves en minúsculas
	lowerHeaders := make(map[string]interface{})
	for k, v := range headers {
		lowerHeaders[strings.ToLower(k)] = v
	}

	// Buscar la clave en el mapa de headers con claves en minúsculas
	if value, exists := lowerHeaders[key]; exists {
		log.Printf("NatsTransformer: getHeaderValue: value :%s", value)
		return value.(string)
	}

	log.Printf("NatsTransformer: getHeaderValue: defaultValue :%s", defaultValue)
	return defaultValue
}
