package models

import (
	"backend/entities"
	"log"
	"time"

	"github.com/google/uuid"
)

// Implementación del método BuildMessage para cumplir con la interfaz IMessage
func BuildMessage(nickname, messageText, roomName string, roomID uuid.UUID, priority int, originalLang string) *entities.Message {
	log.Println("Messag: BuildMessage: Creando un mensaje de aplicación")
	return &entities.Message{
		MessageId:   uuid.New(),
		MessageType: entities.Text, // O puedes asignar el tipo que necesites
		SendDate:    time.Now(),
		ServerDate:  time.Now(),
		Nickname:    nickname,
		Token:       "sample-token", // Aquí deberías asignar un token real
		MessageText: messageText,
		RoomID:      roomID,
		RoomName:    roomName,
		Metadata: entities.Metadata{
			AckStatus:    false,
			Priority:     priority,
			OriginalLang: originalLang,
		},
	}
}
