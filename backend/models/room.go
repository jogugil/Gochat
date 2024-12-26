package models

import (
	"backend/entities"
	"log"

	"github.com/google/uuid"
)

// Alias of the Room type
type LocalRoom entities.Room

func (room *LocalRoom) AddUser(user entities.User) {
	log.Printf("AddUser: Adding user %v to room %s", user, room.RoomName)

	// Key operation: Add the user to the list
	room.Users = append(room.Users, user)

	log.Printf("AddUser: User %v added successfully to room %s", user, room.RoomName)
}

func (room *LocalRoom) RemoveUser(user entities.User) {
	log.Printf("RemoveUser: Removing user %v from room %s", user, room.RoomName)

	// Iterate through the list of users
	for i, u := range room.Users {
		// If the user is found, remove them from the list
		if u == user {
			log.Printf("RemoveUser: User %v found at position %d", user, i)

			// Key operation: Remove the user while keeping the others
			room.Users = append(room.Users[:i], room.Users[i+1:]...)

			log.Printf("RemoveUser: User %v removed successfully from room %s", user, room.RoomName)
			return
		}
	}

	log.Printf("RemoveUser: User %v not found in room %s", user, room.RoomName)
}

func (room *LocalRoom) SendMessage(user entities.User, message entities.Message) {
	log.Printf("SendMessage: User %v is sending message %v to room %s", user, message, room.RoomName)

	// Key operation: Add the message to the broker
	room.MessageBroker.Publish(user.RoomId.String(), entities.Message)

	log.Printf("SendMessage: Message %v added successfully to room %s", message, room.RoomName)
}

func (room *LocalRoom) GetRoomMessages() []entities.Message {
	log.Printf("GetRoomMessages: Fetching messages from room %s", room.RoomName)

	// Key operation: Get all messages
	messages, err := room.MessageBroker.GetMessages(room.RoomId.String())
	if err != nil {
		log.Printf("GetRoomMessages: Error fetching messages from room %s: %v", room.RoomName, err)
		return nil
	}

	log.Printf("GetRoomMessages: Messages fetched successfully from room %s", room.RoomName)
	return messages
}

func (room *LocalRoom) GetMessagesFromId(messageId uuid.UUID) []entities.Message {
	log.Printf("GetMessagesFromId: Fetching messages from ID %v for room %s", messageId, room.RoomName)

	// Key operation: Get messages from a specific ID
	messages, err := room.MessageBroker.GetMessagesFromId(room.RoomId.String(), messageId)
	if err != nil {
		log.Printf("GetMessagesFromId: Error fetching messages from ID %v in room %s: %v", messageId, room.RoomName, err)
		return nil
	}

	log.Printf("GetMessagesFromId: Messages fetched successfully from ID %v in room %s", messageId, room.RoomName)
	return messages
}
