// entities/persistence.go
package entities

import (
	"github.com/google/uuid"
)

type Persistence interface {
	SaveUser(user *User) error
	SaveRoom(room Room) error
	GetRoom(id uuid.UUID) (Room, error)
	GetPreviousMessages(messageId uuid.UUID, remainingCount int) ([]Message, error)
	SaveMessage(message *Message) error
	SaveMessagesToDatabase(messages []Message, roomId uuid.UUID) error
	GetMessagesFromRoom(roomId uuid.UUID) ([]Message, error)
	GetMessagesFromRoomById(roomId uuid.UUID, messageId uuid.UUID) ([]Message, error)
}
