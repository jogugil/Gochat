package services

import (
	"backend/entities"
	"backend/models"
	"backend/persistence"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
)

type RoomManagement struct {
	FixedRooms  map[uuid.UUID]*entities.Room // Map of fixed rooms in memory
	MainRoom    *entities.Room               // Single main room
	mu          sync.RWMutex                 // Read/write mutex
	persistence *entities.Persistence        // Persistence to store data
	once        sync.Once                    // To ensure single initialization
}

var instance *RoomManagement // Singleton instance of RoomManagement

// NewRoomManagement creates and returns a Singleton instance of RoomManagement

func NewRoomManagement(persistence *entities.Persistence, configFile string) *RoomManagement {
	// Ensure only one instance is created
	if instance == nil {
		instance = &RoomManagement{
			FixedRooms:  make(map[uuid.UUID]*entities.Room), // Initialize the map
			persistence: persistence,                        // Assign persistence
		}
	}
	// Configuration and data load only once
	instance.once.Do(func() {
		log.Println("RoomManagement:  NewRoomManagement:  Initializing instance configuration.")
		if configFile != "" {
			err := instance.LoadFixedRoomsFromFile(configFile)
			if err != nil {
				log.Fatalf("RoomManagement:  NewRoomManagement: Error loading configuration: %v", err)
			}
		}
		log.Println("RoomManagement:  NewRoomManagement: Configuration completed.")
	})
	return instance
}

func (rm *RoomManagement) LoadFixedRoomsFromFile(configFile string) error {
	log.Printf("RoomManagement: LoadFixedRoomsFromFile: Loading rooms from file: %s", configFile)
	if rm.FixedRooms == nil {
		rm.FixedRooms = make(map[uuid.UUID]*entities.Room)
	}
	var persis, err = persistence.GetDBInstance()

	if err != nil {
		log.Printf("RoomManagement: LoadFixedRoomsFromFile: Error creating MongoPersistence instance:%v", err)
		return err
	}

	rm.MainRoom = &entities.Room{
		RoomId:         uuid.New(),
		RoomName:       "Main Room",
		RoomType:       "Main",
		MessageHistory: entities.NewCircularQueue(persis),
	}

	file, err := os.Open(configFile)
	if err != nil {
		log.Printf("RoomManagement: LoadFixedRoomsFromFile: Error opening file: %v", err)
		return err
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("RoomManagement: LoadFixedRoomsFromFile: Error reading file: %v", err)
		return err
	}

	var rooms []map[string]interface{}
	err = json.Unmarshal(byteValue, &rooms)
	if err != nil {
		log.Printf("RoomManagement: LoadFixedRoomsFromFile: Error parsing JSON: %v", err)
		return err
	}

	// Use RLock to read FixedRooms concurrently.
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for _, roomData := range rooms {
		roomID, err_parse := uuid.Parse(roomData["id"].(string))
		if err_parse != nil {
			log.Printf("RoomManagement: LoadFixedRoomsFromFile: Error parsing ID: %v", err_parse)
			return err_parse
		}
		var persis, err_per = persistence.GetDBInstance()

		if err_per != nil {
			log.Printf("RoomManagement: Error creating MongoPersistence instance:%v", err_per)
			return err_per
		}
		room := &entities.Room{
			RoomId:         roomID,
			RoomName:       roomData["name"].(string),
			RoomType:       "Fixed",
			MessageHistory: entities.NewCircularQueue(persis),
		}
		rm.FixedRooms[roomID] = room
	}
	log.Println("RoomManagement: LoadFixedRoomsFromFile: Load completed.")
	return nil
}

func (rm *RoomManagement) CreateTemporaryRoom(name string) *entities.Room {
	log.Printf("RoomManagement: CreateTemporaryRoom: Creating temporary room with name: %s", name)
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if rm.persistence == nil {
		log.Fatal("RoomManagement: CreateTemporaryRoom: Persistence not initialized.")
	}

	room := &entities.Room{
		RoomId:         uuid.New(),
		RoomName:       name,
		RoomType:       "Temporary",
		MessageHistory: entities.NewCircularQueue(rm.persistence),
	}
	log.Printf("RoomManagement: CreateTemporaryRoom: Temporary room created with ID: %s", room.RoomId)
	return room
}

func (rm *RoomManagement) GetRoomByID(roomID uuid.UUID) (*entities.Room, error) {
	log.Printf("RoomManagement: GetRoomByID: Searching for room with ID: %s", roomID)

	rm.mu.RLock()
	log.Printf("RoomManagement: GetRoomByID: Lock acquired for reading room with ID %s", roomID)
	defer func() {
		rm.mu.RUnlock()
		log.Printf("RoomManagement: GetRoomByID: Lock released for reading room with ID %s", roomID)
	}()

	if rm.MainRoom != nil && rm.MainRoom.RoomId == roomID {
		log.Println("RoomManagement: GetRoomByID: Main room found.")
		return rm.MainRoom, nil
	}

	if room, exists := rm.FixedRooms[roomID]; exists {
		log.Printf("RoomManagement: GetRoomByID: Fixed room found with ID: %s", roomID)
		return room, nil
	}

	log.Printf("RoomManagement: GetRoomByID: Room not found with ID: %s", roomID)
	return nil, fmt.Errorf("the room with ID %s does not exist", roomID)
}

func (rm *RoomManagement) SendMessage(roomID uuid.UUID, nickname, message string, user entities.User) error {
	log.Printf("RoomManagement: SendMessage: Sending message to room with ID: %s", roomID)

	// First, we get the room with RLock, as we are reading
	room, err := rm.GetRoomByID(roomID)
	if err != nil {
		log.Printf("RoomManagement: SendMessage: Error getting room: %v", err)
		return err
	}

	newMessage := models.CreateMessageWithDate(message, user, roomID, room.RoomName, user.LastActionTime)
	log.Printf("RoomManagement: New message created: %+v\n", newMessage)

	// Modify message history
	room.MessageHistory.Enqueue(*newMessage, room.RoomId)
	log.Printf("RoomManagement: SendMessage: Message sent to room with ID: %s - %v \n", roomID, newMessage)
	return nil
}

// Function to get messages from a room
func (rm *RoomManagement) GetMessagesFromId(roomID uuid.UUID, messageID uuid.UUID) ([]entities.Message, error) {
	log.Printf("RoomManagement: GetMessagesFromId: Getting messages from ID %s in room %s", messageID, roomID)
	rm.mu.RLock() // Reading, can be done concurrently
	defer rm.mu.RUnlock()

	room, err := rm.GetRoomByID(roomID)
	if err != nil {
		log.Printf("RoomManagement: GetMessagesFromId: Error getting room: %v", err)
		return nil, fmt.Errorf("the room with ID %s does not exist", roomID)
	}
	if room == nil {
		log.Printf("RoomManagement: GetMessagesFromId: Error getting room: room is nil")
		return nil, fmt.Errorf("RoomManagement: room not found with ID: %s", roomID)
	}
	queue := room.MessageHistory
	// Check if the queue is nil
	if queue == nil {
		log.Println("RoomManagement: GetMessagesFromId: Error: The queue is nil.")
		return nil, fmt.Errorf("RoomManagement: GetMessagesFromId: the queue is nil in room with ID %s ", roomID)
	}

	messages, err := queue.GetMessagesFromId(roomID, messageID)
	if err != nil {
		log.Printf("RoomManagement: GetMessagesFromId: Error getting messages: %v", err)
		return nil, fmt.Errorf("RoomManagement: error in GetByLastMessageId: %v", err)
	}

	log.Printf("RoomManagement: GetMessagesFromId: Messages obtained successfully (%d messages)", len(messages))
	return messages, nil
}

// Function to get all messages from a room
func (rm *RoomManagement) GetMessages(roomID uuid.UUID) ([]entities.Message, error) {
	log.Printf("RoomManagement: GetMessages: Getting all messages from room %s", roomID)
	rm.mu.RLock() // Reading, can be done concurrently
	defer rm.mu.RUnlock()

	room, err := rm.GetRoomByID(roomID)
	if err != nil {
		log.Printf("RoomManagement: GetMessages: Error getting room: %v", err)
		return nil, fmt.Errorf("the room with ID %s does not exist", roomID)
	}
	queue := room.MessageHistory
	messages, err := queue.GetAll(roomID)
	if err != nil {
		log.Printf("RoomManagement: GetMessages: Error getting messages: %v", err)
		return nil, fmt.Errorf("RoomManagement: error in GetMessages: %v", err)
	}

	log.Printf("RoomManagement: GetMessages: Messages obtained successfully (%d messages)", len(messages))
	return messages, nil
}

func (rm *RoomManagement) GetMessageCount(roomID uuid.UUID, messageID uuid.UUID, count int) ([]entities.Message, error) {
	log.Printf("RoomManagement: GetMessageCount: Getting %d messages from room %s", count, roomID)
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Pre-validation
	if roomID == uuid.Nil || messageID == uuid.Nil || count == 0 {
		log.Printf("RoomManagement: GetMessageCount: Invalid request, missing roomID, messageID, or count.")
		return nil, fmt.Errorf("missing parameters")
	}

	// Get room
	room, err := rm.GetRoomByID(roomID)
	if err != nil {
		log.Printf("RoomManagement: GetMessageCount: Error getting room: %v", err)
		return nil, err
	}

	// Get messages
	messages, err := room.MessageHistory.GetMessagesWithLimit(messageID, count, room.RoomId)
	if err != nil {
		log.Printf("RoomManagement: GetMessageCount: Error getting messages with limit: %v", err)
		return nil, fmt.Errorf("error in GetMessagesWithLimit: %v", err)
	}

	log.Printf("RoomManagement: GetMessageCount: Retrieved %d messages", len(messages))
	return messages, nil
}
