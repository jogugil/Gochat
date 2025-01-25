package entities

import (
	"backend/utils"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
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
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
		Headers:   headers,
	}
}
func (k *KafkaTransformer) TransformFromExternalToGetMessage(rawMsg []byte) (*RequestListMessages, error) {
	var msg KafkaMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil, err
	}
	// Decodificar
	messageText := string(msg.Value)
	reqMessage := msg.Headers
	log.Printf("KafkaTransformer: TransformFromExternalToGetMessage: messageText:[%s]\n", messageText)
	log.Printf("KafkaTransformer: TransformFromExternalToGetMessage: reqMessage:[%s]\n", reqMessage)
	if reqMessage != nil && (len(msg.Headers) > 0) {
		// Extraemos el MessageId del header o generamos uno nuevo
		// Convertir MessageId a uuid.UUID
		roomId, err := uuid.Parse(k.getHeaderValue(reqMessage, "roomid", ""))
		if err != nil {
			return nil, fmt.Errorf("KafkaTransformer: TransformFromExternalToGetMessage: error al parsear roomid: %v", err)
		}

		lastmessageid, err := uuid.Parse(k.getHeaderValue(reqMessage, "lastmessageid", ""))
		if err != nil {
			return nil, fmt.Errorf("KafkaTransformer: TransformFromExternalToGetMessage: error al parsear lastmessageid: %v", err)
		}
		operation := k.getHeaderValue(reqMessage, "operation", "")

		nickname := k.getHeaderValue(reqMessage, "nickname", "")

		token := k.getHeaderValue(reqMessage, "tokensesion", "")
		topic := k.getHeaderValue(reqMessage, "topic", "")
		X_GoChat := k.getHeaderValue(reqMessage, "x_gochat", "")

		//mensaje interno
		msgapp := &RequestListMessages{
			RoomId:      roomId,
			TokenSesion: token, //
			Nickname:    nickname,
			Operation:   operation,

			LastMessageId: lastmessageid,
			Topic:         topic,
			X_GoChat:      X_GoChat,
		}
		log.Printf("KafkaTransformer: TransformFromExternalToGetMessage: msg :%v", msgapp)
		return msgapp, nil
	} else {
		return nil, fmt.Errorf("KafkaTransformer: TransformFromExternalToGetMessage: CMD100 :Heads no encontrado, no se devuelve mensaje")
	}
}

func (k *KafkaTransformer) TransformFromExternalToGetUsers(rawMsg []byte) (*RequestListUsers, error) {
	var msg KafkaMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil, err
	}
	// Decodificar
	log.Printf("KafkaTransformer: TransformFromExternalToGetUsers: msg:[%s]\n", msg)
	messageText := string(msg.Value)
	reqMessage := msg.Headers
	log.Printf("KafkaTransformer: TransformFromExternalToGetUsers: messageText:[%s]\n", messageText)
	log.Printf("KafkaTransformer: TransformFromExternalToGetUsers: reqMessage:[%v]\n", reqMessage)
	if reqMessage != nil && (len(msg.Headers) > 0) {
		// Extraemos el MessageId del header o generamos uno nuevo
		// Convertir MessageId a uuid.UUID
		roomId, err := uuid.Parse(k.getHeaderValue(reqMessage, "RoomId", ""))
		if err != nil {
			return nil, fmt.Errorf("KafkaTransformer: TransformFromExternalToGetUsers: error al parsear RoomId: %v", err)
		}

		nickname := k.getHeaderValue(reqMessage, "Nickname", "")
		token := k.getHeaderValue(reqMessage, "TokenSesion", "")
		request := k.getHeaderValue(reqMessage, "request", "")
		topic := k.getHeaderValue(reqMessage, "Topic", "")
		X_GoChat := k.getHeaderValue(reqMessage, "x_gochat", "")

		//mensaje interno
		msgapp := &RequestListUsers{
			RoomId:      roomId,
			TokenSesion: token, //
			Nickname:    nickname,
			Request:     request,
			Topic:       topic,
			X_GoChat:    X_GoChat,
		}
		log.Printf("KafkaTransformer: TransformFromExternalToGetUsers: msg :%v", msgapp)
		return msgapp, nil
	} else {
		return nil, fmt.Errorf("KafkaTransformer: TransformFromExternalToGetUsers: CMD100 :Heads no encontrado, no se devuelve mensaje")
	}
}

// TransformFromExternal convierte un mensaje de Kafka (KafkaMessage) a un mensaje interno (Message).
func (k *KafkaTransformer) TransformFromExternal(rawMsg []byte) (*Message, error) {
	// Definir la estructura que representa el formato del mensaje de Kafka
	var kafkaMsg KafkaMessage

	// Intentar deserializar el JSON recibido
	if err := json.Unmarshal(rawMsg, &kafkaMsg); err != nil {
		return nil, fmt.Errorf("KafkaTransformer: TransformFromExternal: error al deserializar el mensaje de Kafka: %w", err)
	}

	// Validar los campos esenciales del mensaje deserializado
	if kafkaMsg.Value == "" {
		return nil, fmt.Errorf("KafkaTransformer: TransformFromExternal: el mensaje de Kafka no contiene texto válido en 'Value'")
	}
	if kafkaMsg.Headers != nil && (len(kafkaMsg.Headers) > 0) {
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
			fmt.Println("KafkaTransformer: TransformFromExternal: Error al convertir priority, usando valor predeterminado:", err)
		}
		var messageType_i int
		if p, err := strconv.Atoi(messageType); err == nil {
			messageType_i = p
		} else {
			fmt.Println("KafkaTransformer: TransformFromExternal: Error al convertir priority, usando valor predeterminado:", err)
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
			ServerDate:  time.Now(),                    // Fecha del servidor actual
			Metadata: Metadata{
				Priority:     priorityInt,  // Asignamos la prioridad desde los headers
				OriginalLang: originalLang, // Asignamos el idioma original desde los headers
				AckStatus:    ackStatus,    // Status de confirmación
			},
		}

		return msg, nil
	} else {
		return nil, fmt.Errorf("KafkaTransformer: TransformFromExternal: CMD100 :Heads no encontrado, no se devuelve mensaje")
	}
}

// TransformToExternal convierte un mensaje interno (Message) al formato de Kafka (KafkaMessage).
func (k *KafkaTransformer) TransformToExternal(message *Message) ([]byte, error) {
	// Crear el objeto KafkaMessage y mapear los valores correspondientes
	kafkaMsg := KafkaMessage{
		Key:       message.MessageId.String(), // El Key es el MessageId (UUID como string)
		Value:     message.MessageText,        // El Value es el MessageText
		Timestamp: time.Now(),
		Headers:   make(map[string]interface{}), // Iniciar el mapa de headers
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

// TransformToExternal convierte un mensaje interno (Message) al formato de NATS (NatsMessage).
func (k *KafkaTransformer) TransformToExternalUsers(topic string, message *ResponseListUser) ([]byte, error) {
	// Crear el objeto NatsMessage y mapear los valores correspondientes
	kfkMsg := KafkaMessage{
		Key:       topic,           // El Subject es el RoomName
		Value:     message.Message, // El Data es el MessageText
		Timestamp: time.Now(),
		Headers:   make(map[string]interface{}),
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
		Key:       message.RoomId.String(), // El Subject es el RoomName
		Value:     message.Message,         // El Data es el MessageText
		Timestamp: time.Now(),
		Headers:   make(map[string]interface{}),
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
