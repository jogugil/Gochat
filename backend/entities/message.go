package entities

import (
	"time"

	"github.com/google/uuid"
)

// Tipo de mensaje
type MessageType int

const (
	Ordinario MessageType = iota + 1
	Administracion
	Info
)

// Clase Mensaje
type Message struct {
	MessageId   uuid.UUID   `json:"messageid"`
	MessageType MessageType `json:"messagetype"`
	SendDate    time.Time   `json:"senddate"`
	ServerDate  time.Time   `json:"serverdate"`
	Nickname    string      `json:"nickname"`
	Token       string      `json:"tokenjwt"`
	MessageText string      `json:"messagetext"`
	RoomID      uuid.UUID   `json:"roomid"`
	RoomName    string      `json:"roomname"`
}

// Actualizar el tipo de mensaje
func (m *Message) ActualizarTipo(messageType MessageType) {
	m.MessageType = messageType
}
