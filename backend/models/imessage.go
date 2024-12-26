package models

import (
	"backend/entities"

	"github.com/google/uuid"
)

// IMessage es la interfaz que maneja las operaciones sobre el mensaje
type IMessage interface {
	Get(attribute string) interface{}
	Set(attribute string, value interface{})
	Keys() []string
	TransformFromExternal(rawMsg []byte) error
	TransformToExternal() ([]byte, error)
	BuildMessage(nickname, messageText, roomName string, roomID uuid.UUID, priority int, originalLang string) entities.Message
}
