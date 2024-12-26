package persistence

import (
	"log"
	"time"

	"backend/entities"
	"backend/types"

	"github.com/google/uuid"
)

func TestYourMongoDBFunction() {
	// Simulating a MongoPersistence instance
	uri := "mongodb://localhost:27017" // Change this according to your MongoDB URI
	dbName := "chatDB"
	persistence, err := NewMongoPersistence(uri, dbName)
	if err != nil {
		log.Fatalf("Error creating the persistence instance: %v", err)
	}
	roomId, err := uuid.Parse("10000000-0000-0000-0000-000000000010") // Using a room ID
	if err != nil {
		log.Printf("Error setting room for user: %v", err)
	}
	// Create a test user
	user := &entities.User{
		UserId:         uuid.New().String(),
		Nickname:       "test_user",
		Token:          "token_123",
		LastActionTime: time.Now(),
		State:          types.Active,
		Type:           "regular",
		RoomId:         roomId, // Assuming it could be nil
		RoomName:       "RoomId",
	}

	// 1. Test SaveUser
	log.Println("Starting SaveUser test...")
	startTime := time.Now()
	err = (*persistence).SaveUser(user)
	duration := time.Since(startTime)
	if err != nil {
		log.Printf("Error in SaveUser: %v\n", err)
	} else {
		log.Printf("SaveUser completed in %v\n", duration)
	}

	// 2. Test SaveRoom
	room := entities.Room{
		RoomId:   uuid.New(),
		RoomName: "TestRoom",
	}
	log.Println("Starting SaveRoom test...")
	startTime = time.Now()
	err = (*persistence).SaveRoom(room)
	duration = time.Since(startTime)
	if err != nil {
		log.Printf("Error in SaveRoom: %v\n", err)
	} else {
		log.Printf("SaveRoom completed in %v\n", duration)
	}

	// 3. Test GetRoom
	log.Println("Starting GetRoom test...")
	startTime = time.Now()
	_, err = (*persistence).GetRoom(room.RoomId)
	duration = time.Since(startTime)
	if err != nil {
		log.Printf("Error in GetRoom: %v\n", err)
	} else {
		log.Printf("GetRoom completed in %v\n", duration)
	}

	// 4. Test SaveMessage
	message := &entities.Message{
		MessageId:   uuid.New(),
		Nickname:    "test_user",
		SendDate:    time.Now(),
		RoomID:      room.RoomId,
		RoomName:    room.RoomName,
		Token:       "token_123",
		MessageText: "This is a test message",
	}
	log.Println("Starting SaveMessage test...")
	startTime = time.Now()
	err = (*persistence).SaveMessage(message)
	duration = time.Since(startTime)
	if err != nil {
		log.Printf("Error in SaveMessage: %v\n", err)
	} else {
		log.Printf("SaveMessage completed in %v\n", duration)
	}

	// 5. Test GetMessagesFromRoom
	log.Println("Starting GetMessagesFromRoom test...")
	startTime = time.Now()
	_, err = (*persistence).GetMessagesFromRoom(room.RoomId)
	duration = time.Since(startTime)
	if err != nil {
		log.Printf("Error in GetMessagesFromRoom: %v\n", err)
	} else {
		log.Printf("GetMessagesFromRoom completed in %v\n", duration)
	}

	// 6. Test GetMessagesFromRoomById
	log.Println("Starting GetMessagesFromRoomById test...")
	startTime = time.Now()
	_, err = (*persistence).GetMessagesFromRoomById(room.RoomId, message.MessageId)
	duration = time.Since(startTime)
	if err != nil {
		log.Printf("Error in GetMessagesFromRoomById: %v\n", err)
	} else {
		log.Printf("GetMessagesFromRoomById completed in %v\n", duration)
	}
}
