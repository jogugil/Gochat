package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

const apiURL = "http://localhost:8081/login" // Cambia esta URL a la de tu servidor REST

// Estructura para el cuerpo de la solicitud POST al API REST
type LoginRequest struct {
	Nickname string `json:"nickname"`
}

// Definimos el tipo MessageType como int
type MessageType int

// Ahora definimos constantes para los valores posibles de MessageType
const (
	// Los diferentes tipos de mensajes
	Text         MessageType = iota // 0
	Image                           // 1
	Video                           // 2
	Notification                    // 3
)

// Estructura para la respuesta del API REST
/*
	responseData := gin.H{
		"status":   "ok",
		"message":  "login realizado",
		"token":    usuario.Token,
		"nickname": usuario.Nickname,
		"roomid":   usuario.RoomId,   // Sala por defecto
		"roomname": usuario.RoomName, // Nombre de la sala
	}

*/
// KafkaMessage represents the structure of a message in Kafka
type KafkaMessage struct {
	Key     string            `json:"key"`
	Value   string            `json:"value"`
	Headers map[string]string `json:"headers"`
}

// NatsMessage represents the structure of a message in NATS
type NatsMessage struct {
	Subject string            `json:"subject"`
	Data    []byte            `json:"data"`
	Headers map[string]string `json:"headers"`
}

// Clase Mensaje
type Metadata struct {
	AckStatus    bool   `json:"ackstatus"`    // Estado de confirmación
	Priority     int    `json:"priority"`     // Prioridad del mensaje
	OriginalLang string `json:"originallang"` // Idioma original
}
type Message struct {
	MessageId   uuid.UUID   `json:"messageid"`   // Identificador único
	MessageType MessageType `json:"messagetype"` // Tipo de mensaje
	SendDate    time.Time   `json:"senddate"`    // Fecha de envío
	ServerDate  time.Time   `json:"serverdate"`  // Fecha en el servidor
	Nickname    string      `json:"nickname"`    // Nombre del remitente
	Token       string      `json:"tokenjwt"`    // JWT del remitente
	MessageText string      `json:"messagetext"` // Contenido del mensaje
	RoomID      uuid.UUID   `json:"roomid"`      // Sala destino
	RoomName    string      `json:"roomname"`    // Nombre de la sala
	Metadata    Metadata    `json:"metadata"`    // Metadatos adicionales
}

type LoginResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Token    string `json:"token"`
	Nickname string `json:"nickname"`
	Roomid   string `json:"roomid"`
	Roomname string `json:"roomname"`
}

// Función para realizar el login al API REST
func login(nickname string) (*LoginResponse, error) {
	loginReq := LoginRequest{
		Nickname: nickname,
	}

	// Convertir el loginReq a JSON
	reqBody, err := json.Marshal(loginReq)
	if err != nil {
		return nil, fmt.Errorf("error al convertir la solicitud de login a JSON: %v", err)
	}

	// Hacer la solicitud POST al API REST
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error al realizar solicitud POST: %v", err)
	}
	defer resp.Body.Close()

	// Leer la respuesta
	var loginResp LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		return nil, fmt.Errorf("error al leer la respuesta del API REST: %v", err)
	}
	fmt.Printf("Respuesta del login: %s\n", loginResp)
	return &loginResp, nil
}

// Función para conectarse a NATS, enviar un mensaje y recibir una respuesta
func connectToNATS(token, roomId, roomName, nickname string) {
	// Conectarse al servidor NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Error al conectar a NATS: %v", err)
	}
	defer nc.Close()

	// Generar mensajes
	msg1 := GenerateMessage1(nickname, token, roomId, roomName)
	msg2 := GenerateMessage2(nickname, token, roomId, roomName)

	// Convertir mensajes a NatsMessage
	natsMsg1, err := ConvertToNatsMessage("principal.server", msg1)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	natsMsg2, err := ConvertToNatsMessage("principal.server", msg2)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Conectado a NATS\n")
	// Convertir el mensaje 1 a JSON
	msgBytes1, err := json.Marshal(natsMsg1)
	if err != nil {
		log.Fatalf("Error al convertir el mensaje a JSON: %v", err)
	}

	// Enviar el mensaje a 'principal.server'
	err = nc.Publish("principal.server", msgBytes1)
	if err != nil {
		log.Fatalf("Error al enviar el mensaje a NATS: %v", err)
	}
	fmt.Println("Mensaje enviado a principal.server:", string(msgBytes1))

	// Convertir el mensaje 2 a JSON
	msgBytes2, err := json.Marshal(natsMsg2)
	if err != nil {
		log.Fatalf("Error al convertir el mensaje a JSON: %v", err)
	}

	// Enviar el mensaje a 'principal.server'
	err = nc.Publish("principal.server", msgBytes2)
	if err != nil {
		log.Fatalf("Error al enviar el mensaje a NATS: %v", err)
	}
	fmt.Println("Mensaje enviado a principal.server:", string(msgBytes2))

	// Suscribirse a 'principal.client' para recibir respuestas
	_, err = nc.Subscribe("principal.client", func(m *nats.Msg) {
		fmt.Printf("Respuesta recibida en principal.client: %s\n", string(m.Data))
	})
	if err != nil {
		log.Fatalf("Error al suscribirse a principal.client: %v", err)
	}

	// Mantener el cliente NATS escuchando por mensajes
	select {}
}

// Función para convertir Message a NatsMessage
func ConvertToNatsMessage(topic string, msg Message) (NatsMessage, error) {
	// Crear el mapa de headers
	headers := map[string]string{
		"MessageId":    msg.MessageId.String(),
		"MessageType":  string(rune(Notification)),
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

	// Convertir el texto del mensaje a bytes
	data := []byte(msg.MessageText)

	// Crear el objeto NatsMessage
	natsMsg := NatsMessage{
		Subject: topic,
		Data:    data,
		Headers: headers,
	}

	return natsMsg, nil
}

// Función para generar un mensaje de ejemplo 1
func GenerateMessage1(nickname, token, roomId, roomName string) Message {
	return Message{
		MessageId:   uuid.New(),
		MessageType: Notification,
		SendDate:    time.Now(),
		ServerDate:  time.Now(),
		Nickname:    nickname,
		Token:       token,
		MessageText: "Este es un mensaje de ejemplo 1.",
		RoomID:      uuid.MustParse(roomId),
		RoomName:    roomName,
		Metadata: Metadata{
			AckStatus:    true,
			Priority:     1,
			OriginalLang: "es",
		},
	}
}

// Función para generar un mensaje de ejemplo 2
func GenerateMessage2(nickname, token, roomId, roomName string) Message {
	return Message{
		MessageId:   uuid.New(),
		MessageType: Notification,
		SendDate:    time.Now(),
		ServerDate:  time.Now(),
		Nickname:    nickname,
		Token:       token,
		MessageText: "Este es un mensaje de ejemplo 2.",
		RoomID:      uuid.MustParse(roomId),
		RoomName:    roomName,
		Metadata: Metadata{
			AckStatus:    false,
			Priority:     2,
			OriginalLang: "en",
		},
	}
}
func main() {
	// Datos de ejemplo para el login
	nickname := "usuario123"

	// Realizar login y obtener token, roomId y roomName
	loginResp, err := login(nickname)
	if err != nil {
		log.Fatalf("Error durante el login: %v", err)
	}
	if loginResp.Status == "ok" {
		// Mostrar la información del login
		fmt.Printf("Login exitoso: NickName: %s Token: %s, RoomID: %s, RoomName: %s\n", loginResp.Nickname, loginResp.Token, loginResp.Roomid, loginResp.Roomname)

		// Conectar a NATS y enviar/recibir mensajes
		connectToNATS(loginResp.Token, loginResp.Roomid, loginResp.Roomname, nickname)
	} else {
		fmt.Println("Error durante el login. Verifique su nickname y contraseña. Si el problema persiste")
		fmt.Println("E:", loginResp.Message)
	}

}
