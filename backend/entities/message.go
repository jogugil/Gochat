package entities

import (
	"time"

	"github.com/google/uuid"
)

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
	Headers map[string]string `json:"header"`
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

/*
	1. priority: Indica la importancia del mensaje.
			Mensajes generales de chat: 1.
			Notificaciones importantes: 2.
			Alertas críticas: 3.
	2. originalLang: Representa el idioma original del mensaje. Utiliza códigos ISO 639-1 de dos letras para especificar el idioma.
			en: Inglés.
			es: Español.
			fr: Francés.
			de: Alemán.
			zh: Chino.
*/
