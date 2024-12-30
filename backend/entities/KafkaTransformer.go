package entities

import (
	"backend/utils"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// KafkaTransformer es un transformador que maneja la conversión de mensajes entre el formato interno de la aplicación y el formato de Kafka.
type KafkaTransformer struct{}

// NewKafkaTransformer crea una nueva instancia del KafkaTransformer.
func NewKafkaTransformer() *KafkaTransformer {
	return &KafkaTransformer{}
}
func (k *KafkaTransformer) BuildMessage(key, value string, headers map[string]interface{}) *KafkaMessage {
	return &KafkaMessage{
		Key:     key,
		Value:   value,
		Headers: headers,
	}
}
func (k *KafkaTransformer) TransformFromExternalToGetMessage(rawMsg []byte) (*RequestLisMessages, error) {
	var msg KafkaMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil, err
	}
	// Decodificar
	messageText := string(msg.Value)
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

func (k *KafkaTransformer) TransformFromExternalToGetUsers(rawMsg []byte) (*RequestLisUsers, error) {
	var msg KafkaMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil, err
	}
	// Decodificar
	messageText := string(msg.Value)
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
	sendDate := k.getHeaderValue(kafkaMsg.Headers, "SendDate", "")
	sendDateT, err := utils.ConvertToRFC3339(sendDate)
	if err != nil {
		fmt.Println("Error al convertir la fecha:", err)
		sendDateT = time.Now()
	}
	messageType := k.getHeaderValue(kafkaMsg.Headers, "MessageType", "Text")
	token := k.getHeaderValue(kafkaMsg.Headers, "Token", "")
	ackStatus := k.getHeaderValue(kafkaMsg.Headers, "AckStatus", "false") == "true"
	priority := k.getHeaderValue(kafkaMsg.Headers, "Priority", "1")
	originalLang := k.getHeaderValue(kafkaMsg.Headers, "OriginalLang", "unknown")

	// Convertir Priority si está presente
	var priorityInt int
	if p, err := strconv.Atoi(priority); err == nil {
		priorityInt = p
	} else {
		fmt.Println("Error al convertir priority, usando valor predeterminado:", err)
	}
	var messageType_i int
	if p, err := strconv.Atoi(messageType); err == nil {
		messageType_i = p
	} else {
		fmt.Println("Error al convertir priority, usando valor predeterminado:", err)
	}
	// Construir el mensaje de la aplicación
	msg := &Message{
		MessageId:   utils.ParseUUID(kafkaMsg.Key), // Key es el ID del mensaje (UUID como string)
		MessageType: MessageType(messageType_i),    // El tipo de mensaje
		MessageText: kafkaMsg.Value,                // El cuerpo del mensaje
		RoomID:      roomID,                        // RoomID como UUID
		RoomName:    roomName,                      // Nombre del room
		Nickname:    nickname,                      // Nickname desde los headers
		SendDate:    sendDateT,                     // Fecha de envío
		Token:       token,                         // Token desde los headers

		ServerDate: time.Now(), // Fecha del servidor actual
		Metadata: Metadata{
			Priority:     priorityInt,  // Asignamos la prioridad desde los headers
			OriginalLang: originalLang, // Asignamos el idioma original desde los headers
			AckStatus:    ackStatus,    // Status de confirmación
		},
	}

	return msg, nil
}

// TransformToExternal convierte un mensaje interno (Message) al formato de Kafka (KafkaMessage).
func (k *KafkaTransformer) TransformToExternal(message *Message) ([]byte, error) {
	// Crear el objeto KafkaMessage y mapear los valores correspondientes
	kafkaMsg := KafkaMessage{
		Key:     message.MessageId.String(),   // El Key es el MessageId (UUID como string)
		Value:   message.MessageText,          // El Value es el MessageText
		Headers: make(map[string]interface{}), // Iniciar el mapa de headers
	}

	// Mapear los valores adicionales a los headers
	headers := map[string]interface{}{
		"MessageId":    message.MessageId.String(),
		"MessageType":  string(rune(Notification)),
		"SendDate":     message.SendDate.Format(time.RFC3339),
		"ServerDate":   message.ServerDate.Format(time.RFC3339),
		"Nickname":     message.Nickname,
		"Token":        message.Token,
		"RoomID":       message.RoomID.String(),
		"RoomName":     message.RoomName,
		"AckStatus":    fmt.Sprintf("%t", message.Metadata.AckStatus),
		"Priority":     fmt.Sprintf("%d", message.Metadata.Priority),
		"OriginalLang": message.Metadata.OriginalLang,
	}
	kafkaMsg.Headers = headers

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
func (k *KafkaTransformer) getHeaderValue(headers map[string]interface{}, key, defaultValue string) string {
	if value, exists := headers[key]; exists {
		return value.(string)
	}
	return defaultValue
}

// TransformToExternal convierte un mensaje interno (Message) al formato de NATS (NatsMessage).
func (k *KafkaTransformer) TransformToExternalUsers(message *ResponseListUser) ([]byte, error) {
	// Crear el objeto NatsMessage y mapear los valores correspondientes
	kfkMsg := KafkaMessage{
		Key:     message.RoomId.String(), // El Subject es el RoomName
		Value:   message.Message,         // El Data es el MessageText
		Headers: make(map[string]interface{}),
	}

	// Mapear los valores adicionales a los headers
	kfkMsg.Headers["AliveUsers"] = message.AliveUsers

	// Serializar el mensaje NATS a JSON
	rawMsg, err := json.Marshal(kfkMsg)
	if err != nil {
		return nil, fmt.Errorf("error al serializar el mensaje de NATS: %v", err)
	}

	// Retornar el mensaje serializado en formato JSON
	return rawMsg, nil
}

// TransformToExternal convierte un mensaje interno (Message) al formato de NATS (NatsMessage).
func (k *KafkaTransformer) TransformToExternalMessages(message *ResponseListMessages) ([]byte, error) {
	// Crear el objeto KafkaMessage y mapear los valores correspondientes
	kfkMsg := KafkaMessage{
		Key:     message.RoomId.String(), // El Subject es el RoomName
		Value:   message.Message,         // El Data es el MessageText
		Headers: make(map[string]interface{}),
	}

	// Mapear los valores adicionales a los headers
	kfkMsg.Headers["MessageResponse"] = message.ListMessage

	// Serializar el mensaje NATS a JSON
	rawMsg, err := json.Marshal(kfkMsg)
	if err != nil {
		return nil, fmt.Errorf("error al serializar el mensaje de NATS: %v", err)
	}

	// Retornar el mensaje serializado en formato JSON
	return rawMsg, nil
}
