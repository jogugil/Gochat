package entities

import (
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// IMessage es la interfaz que maneja las operaciones sobre el mensaje
type IMessage interface {
	Get(attribute string) interface{}
	Set(attribute string, value interface{})
	Keys() []string
	TransformFromExternal(rawMsg []byte) error
	TransformToExternal() ([]byte, error)
	BuildMessage(nickname string, messageText string, roomID uuid.UUID, roomName string, priority int, originalLang string) Message
	SendMessageToNATS(nc *nats.Conn, message Message) error
	ReadMessageFromNATS(nc *nats.Conn, roomID uuid.UUID) ([]Message, error)
}
