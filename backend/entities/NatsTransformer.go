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

// NatsTransformer es un transformador que maneja la conversión de mensajes entre el formato interno de la aplicación y el formato de NATS.
type NatsTransformer struct{}

// NewNatsTransformer crea una nueva instancia del NatsTransformer.
func NewNatsTransformer() *NatsTransformer {
	return &NatsTransformer{}
}
func (n *NatsTransformer) BuildMessage(subject, data string, header map[string]string) *NatsMessage {
	return &NatsMessage{
		Subject: subject,
		Data:    []byte(data),
		Headers: header,
	}
}

// TransformFromExternal convierte un mensaje de NATS (NatsMessage) a un mensaje interno (Message).
/*
	headers := map[string]string{
		"MessageId":    msg.MessageId.String(),
		"MessageType":  string(Notification),
		"SendDate":     msg.SendDate.Format(time.RFC3339),
		"ServerDate":   msg.ServerDate.Format(time.RFC3339),
		"Nickname":     msg.Nickname,
		"Token":        msg.Token,
		"RoomID":       msg.RoomID.String(),
		"RoomName":     msg.RoomName,
		"AckStatus":    fmt.Sprintf("%t", msg.Metadata.AckStatus),
		"Priority":     fmt.Sprintf("%d", msg.Metadata.Priority),
		"OriginalLang": msg.Metadata.OriginalLang,
	}

*/
func (n *NatsTransformer) TransformFromExternal(rawMsg []byte) (*Message, error) {
	// Parsear el mensaje de NATS (rawMsg) a NatsMessage

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
	messageId, err := uuid.Parse(reqMessage["MessageId"])
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear MessageId: %v", err)
	}
	messageType_h := int(reqMessage["MessageType"][0])

	// Convertir MessageType (puedes definir tu tipo MessageType según tu necesidad)
	messageType := MessageType(messageType_h)

	// Convertir SendDate y ServerDate a time.Time
	log.Printf("\nNatsTransformer: TransformFromExternal:  SendDate :[%s]\n", reqMessage["SendDate"])
	sendDate, err := utils.ConvertToRFC3339(reqMessage["SendDate"])
	if err != nil {
		return nil, fmt.Errorf("error al parsear SendDate: %v", err)
	}

	serverDate, err := time.Parse(time.RFC3339, reqMessage["ServerDate"])
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear ServerDate: %v", err)
	}

	// Asignar Nickname, Token, RoomName, y OriginalLang
	nickname := reqMessage["Nickname"]
	token := reqMessage["Token"]
	roomName := reqMessage["RoomName"]
	originalLang := reqMessage["OriginalLang"]

	// Convertir RoomID a uuid.UUID
	roomID, err := uuid.Parse(reqMessage["RoomID"])
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear RoomID: %v", err)
	}

	// Convertir AckStatus a bool
	ackStatus, err := strconv.ParseBool(reqMessage["AckStatus"])
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal: error al parsear AckStatus: %v", err)
	}

	// Convertir Priority a int
	log.Printf("\nNatsTransformer: TransformFromExternal: reqMessage[ Priority ]: [%s]\n", reqMessage["Priority"])
	priority, err := strconv.Atoi(reqMessage["Priority"])
	if err != nil {
		return nil, fmt.Errorf("NatsTransformer: TransformFromExternal:error al parsear Priority: %v", err)
	}

	// Convertir Priority si está presente

	//mensaje interno
	msgapp := &Message{
		MessageId:   messageId,
		MessageType: messageType, // Puedes ajustar esto dependiendo del tipo de mensaje que recibas
		MessageText: messageText, // El cuerpo del mensaje
		RoomID:      roomID,      // Este valor se puede generar aquí o extraer de los headers si está disponible
		RoomName:    roomName,    // El nombre del room
		Nickname:    nickname,    // Nickname desde los headers
		SendDate:    sendDate,    // Fecha de envío
		Token:       token,       // Token desde los headers

		ServerDate: serverDate, // Fecha de servidor actual
		Metadata: Metadata{
			Priority:     priority,     // Asignamos la prioridad desde los headers
			OriginalLang: originalLang, // Asignamos el idioma original desde los headers
			AckStatus:    ackStatus,    // Status de confirmación
		},
	}
	log.Printf("NatsTransformer: TransformFromExternal: msg :%v", msgapp)
	return msgapp, nil
}

// TransformToExternal convierte un mensaje interno (Message) al formato de NATS (NatsMessage).
func (n *NatsTransformer) TransformToExternal(message *Message) ([]byte, error) {
	// Crear el objeto NatsMessage y mapear los valores correspondientes
	natsMsg := NatsMessage{
		Subject: message.RoomName,            // El Subject es el RoomName
		Data:    []byte(message.MessageText), // El Data es el MessageText
		Headers: make(map[string]string),
	}

	// Mapear los valores adicionales a los headers

	headers := map[string]string{
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

	natsMsg.Headers = headers
	// Si tienes más campos en Message que necesitas incluir en el header, agrégalos aquí
	// Ejemplo: natsMsg.Header["OtroCampo"] = message.OtroCampo

	// Serializar el mensaje NATS a JSON
	rawMsg, err := json.Marshal(natsMsg)
	if err != nil {
		return nil, fmt.Errorf("error al serializar el mensaje de NATS: %v", err)
	}

	// Retornar el mensaje serializado en formato JSON
	return rawMsg, nil
}
