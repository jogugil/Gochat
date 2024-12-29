package models

import (
	"backend/entities"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// Alias of the Room type
type LocalRoom struct {
	entities.Room // Composición con un puntero
}

func (room *LocalRoom) AddUser(user entities.User) {
	log.Printf("LocalRoom: AddUser: Adding user %v to room %s\n", user, room.Room.RoomName)

	// Key operation: Add the user to the list
	room.Room.Users = append(room.Room.Users, user)

	log.Printf("LocalRoom:AddUser: User %v added successfully to room %s\n", user, room.Room.RoomName)
}

func (room *LocalRoom) RemoveUser(user entities.User) {
	log.Printf("LocalRoom:RemoveUser: Removing user %v from room %s\n", user, room.Room.RoomName)

	// Iterate through the list of users
	for i, u := range room.Room.Users {
		// If the user is found, remove them from the list
		if u == user {
			log.Printf("LocalRoom:RemoveUser: User %v found at position %d\n", user, i)

			// Key operation: Remove the user while keeping the others
			room.Room.Users = append(room.Room.Users[:i], room.Room.Users[i+1:]...)

			log.Printf("LocalRoom:RemoveUser: User %v removed successfully from room %s\n", user, room.Room.RoomName)
			return
		}
	}

	log.Printf("LocalRoom: RemoveUser: User %v not found in room %s\n", user, room.Room.RoomName)
}

func (room *LocalRoom) SendMessage(user entities.User, message entities.Message) {
	log.Printf("LocalRoom: SendMessage: User %v is sending message %v to room %s\n", user, message, room.Room.RoomName)

	// Key operation: Add the message to the broker
	log.Printf("LocalRoom: SendMessage:  messages from room.ClientTopic %s\n", room.ClientTopic)
	room.Room.MessageBroker.Publish(room.ClientTopic, &message)

	log.Printf("LocalRoom: SendMessage: Message %v added successfully to room %s\n", message, room.Room.RoomName)
}

func (room *LocalRoom) GetRoomMessages() []entities.Message {
	log.Printf("LocalRoom: GetRoomMessages: Fetching unread messages from room %s\n", room.Room.RoomName)
	subject := room.ServerTopic

	// Obtener los mensajes no leídos
	messages, err := room.Room.MessageBroker.GetUnreadMessages(subject)
	if err != nil {
		log.Printf("LocalRoom: GetRoomMessages: Error fetching unread messages from room %s: %v\n", room.Room.RoomName, err)
		return nil
	}

	log.Printf("LocalRoom: GetRoomMessages: Unread messages fetched successfully from room %s\n", room.Room.RoomName)
	return messages
}

func (room *LocalRoom) GetMessagesFromId(messageId uuid.UUID) []entities.Message {
	log.Printf("LocalRoom: GetMessagesFromId: Fetching messages from ID %v for room %s\n", messageId, room.Room.RoomName)

	// Key operation: Get messages from a specific ID
	messages, err := room.Room.MessageBroker.GetMessagesFromId(room.ServerTopic, messageId)
	if err != nil {
		log.Printf("LocalRoom: GetMessagesFromId: Error fetching messages from ID %v in room %s: %v\n", messageId, room.Room.RoomName, err)
		return nil
	}

	log.Printf("LocalRoom: GetMessagesFromId: Messages fetched successfully from ID %v in room %s\n", messageId, room.Room.RoomName)
	return messages
}
func (room *LocalRoom) GetMessagesWithLimit(messageID uuid.UUID, count int) ([]entities.Message, error) {
	log.Printf("LocalRoom: GetMessagesWithLimit: Fetching messages starting from ID %v with a limit of %d for room %s\n", messageID, count, room.Room.RoomName)

	// Generar el subject (topic) de la sala
	subject := room.ServerTopic

	// Obtener los mensajes a partir de un ID específico con un límite en la cantidad de mensajes
	messages, err := room.Room.MessageBroker.GetMessagesWithLimit(subject, messageID, count)
	if err != nil {
		log.Printf("LocalRoom: GetMessagesWithLimit: Error fetching messages with limit for room %s: %v\n", room.Room.RoomName, err)
		return nil, fmt.Errorf("error fetching messages with limit for room %s: %w", room.Room.RoomName, err)
	}

	log.Printf("LocalRoom: GetMessagesWithLimit: Successfully fetched %d messages starting from ID %v in room %s\n", len(messages), messageID, room.Room.RoomName)
	return messages, nil
}
