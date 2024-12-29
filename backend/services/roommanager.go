package services

import (
	"backend/entities"
	"backend/models"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
)

type RoomManagement struct {
	FixedRooms  map[uuid.UUID]*models.LocalRoom // Map of fixed rooms in memory
	MainRoom    *models.LocalRoom               // Single main room
	mu          sync.RWMutex                    // Read/write mutex
	persistence *entities.Persistence           // Persistence to store data
	once        sync.Once                       // To ensure single initialization
}

var instance *RoomManagement // Singleton instance of RoomManagement

// NewRoomManagement creates and returns a Singleton instance of RoomManagement

func NewRoomManagement(persistence *entities.Persistence, configFile string) *RoomManagement {
	// Ensure only one instance is created
	if instance == nil {
		instance = &RoomManagement{
			FixedRooms:  make(map[uuid.UUID]*models.LocalRoom), // Initialize the map
			persistence: persistence,                           // Assign persistence
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

	// Asegurarse de que FixedRooms esté inicializado
	if rm.FixedRooms == nil {
		rm.FixedRooms = make(map[uuid.UUID]*models.LocalRoom)
	}

	// Abrir el archivo de configuración
	file, err_op := os.Open(configFile)
	if err_op != nil {
		log.Printf("RoomManagement: LoadFixedRoomsFromFile: Error opening file: %v", err_op)
		return err_op
	}
	defer file.Close()

	// Decodificar el JSON en un mapa genérico
	var config map[string]interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Printf("RoomManagement: LoadFixedRoomsFromFile: Error decoding JSON: %v", err)
		return err
	}

	// Usar la fábrica para obtener el MessageBroker adecuado, pasando la configuración cargada
	msgBroker, err_c := entities.MessageBrokerFactory(config)
	if err_c != nil {
		log.Printf("RoomManagement: LoadFixedRoomsFromFile: Error creating MessageBroker: %v", err_c)
		return err_c
	}

	// Continuar con la carga de salas como antes...

	gochat, ok := config["gochat"].(map[string]interface{})
	if !ok {
		log.Println("RoomManagement: LoadFixedRoomsFromFile: Error - 'gochat' no es un mapa.")
		return fmt.Errorf("RoomManagement: LoadFixedRoomsFromFile: error - 'gochat' no es un mapa")
	}

	mainroom, ok := gochat["mainroom"].(map[string]interface{})
	if !ok {
		log.Println("RoomManagement: LoadFixedRoomsFromFile: Error - 'mainroom' no es un mapa.")
		return fmt.Errorf("RoomManagement: LoadFixedRoomsFromFile: error - 'mainroom' no es un mapa")
	}

	name, ok := mainroom["name"].(string)
	if !ok {
		log.Println("RoomManagement: LoadFixedRoomsFromFile: Error - 'name' no es un mapa.")
		return fmt.Errorf("RoomManagement: LoadFixedRoomsFromFile: error - 'name' no es un mapa")
	}
	server_topic, ok := mainroom["server_topic"].(string)
	if !ok {
		log.Println("RoomManagement: LoadFixedRoomsFromFile: Error - 'server_topic' no es un mapa.")
		return fmt.Errorf("RoomManagement: LoadFixedRoomsFromFile: error - 'server_topic' no es un mapa")
	}
	client_topic, ok := mainroom["client_topic"].(string)
	if !ok {
		log.Println("RoomManagement: LoadFixedRoomsFromFile: Error - 'client_topic' no es un mapa.")
		return fmt.Errorf("RoomManagement: LoadFixedRoomsFromFile: error - 'client_topic' no es un mapa")
	}

	// Mostrar el valor del nombre
	log.Println("RoomManagement: LoadFixedRoomsFromFile: Nombre de la sala:", name)

	// Crear la sala con los datos obtenidos
	rm.MainRoom = &models.LocalRoom{
		Room: entities.Room{
			RoomId:        uuid.New(),
			RoomName:      name,
			RoomType:      "Fixed",
			MessageBroker: msgBroker, // Usar el broker creado
			ServerTopic:   server_topic,
			ClientTopic:   client_topic,
		},
	}
	// Registrar el callback para el topic
	topic := server_topic // Este sería el topic donde se reciben los mensajes
	msgBroker.OnMessage(topic, HandleNewMessages)
	//  carga la sala principal y la agregar al mapa FixedRooms
	rm.FixedRooms[rm.MainRoom.RoomId] = rm.MainRoom

	salas, ok := config["salas"].([]interface{})
	if !ok {
		log.Printf("RoomManagement: LoadFixedRoomsFromFile: Error: 'salas' is not an array")
		return fmt.Errorf("'salas' is not an array")
	}

	// Usar Lock para acceder de manera segura a FixedRooms
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Iterar sobre las salas y cargarlas en el mapa FixedRooms
	for _, roomDataInterface := range salas {
		roomData, ok := roomDataInterface.(map[string]interface{})

		if !ok {
			log.Printf("RoomManagement: LoadFixedRoomsFromFile: Error: roomData is not a map")
			continue
		}

		// Parsear el ID de la sala
		roomID, err := uuid.Parse(roomData["id"].(string))
		if err != nil {
			log.Printf("RoomManagement: LoadFixedRoomsFromFile: Error parsing ID: %v -- roomName:%s", err, roomData["name"].(string))
			continue
		}
		name, ok = roomData["name"].(string)
		if !ok {
			log.Println("RoomManagement: LoadFixedRoomsFromFile: Error - 'name' no es un mapa.")
			return fmt.Errorf("RoomManagement: LoadFixedRoomsFromFile: Error - 'name' no es un mapa")
		}
		server_topic, ok = roomData["server_topic"].(string)
		if !ok {
			log.Println("RoomManagement: LoadFixedRoomsFromFile: Error - 'server_topic' no es un mapa.")
			return fmt.Errorf("RoomManagement: LoadFixedRoomsFromFile: Error - 'server_topic' no es un mapa")
		}
		client_topic, ok = roomData["client_topic"].(string)
		if !ok {
			log.Println("RoomManagement: LoadFixedRoomsFromFile: Error - 'client_topic' no es un mapa.")
			return fmt.Errorf("RoomManagement: LoadFixedRoomsFromFile: Error - 'client_topic' no es un mapa")
		}
		// Crear la sala con los datos obtenidos
		room := &models.LocalRoom{
			Room: entities.Room{
				RoomId:        roomID,
				RoomName:      name,
				RoomType:      "Fixed",
				MessageBroker: msgBroker, // Usar el broker creado
				ServerTopic:   server_topic,
				ClientTopic:   client_topic,
			},
		}

		// Agregar la sala al mapa de salas fijas
		rm.FixedRooms[roomID] = room
	}

	log.Println("RoomManagement: LoadFixedRoomsFromFile: Load completed.")
	return nil
}
func (rm *RoomManagement) CreateTemporaryRoom(name string) *models.LocalRoom {
	log.Printf("RoomManagement: CreateTemporaryRoom: Creating temporary room with name: %s", name)
	rm.mu.Lock()
	defer rm.mu.Unlock()

	room := &models.LocalRoom{
		Room: entities.Room{
			RoomId:        uuid.New(),
			RoomName:      name,
			RoomType:      "Temporary",
			MessageBroker: rm.MainRoom.Room.MessageBroker, // Usar el broker creado
		},
	}

	log.Printf("RoomManagement: CreateTemporaryRoom: Temporary room created with ID: %s", room.Room.RoomId)
	return room
}

func (rm *RoomManagement) GetRoomByID(roomID uuid.UUID) (*models.LocalRoom, error) {
	log.Printf("RoomManagement: GetRoomByID: Searching for room with ID: %s", roomID)

	rm.mu.RLock()
	log.Printf("RoomManagement: GetRoomByID: Lock acquired for reading room with ID %s", roomID)
	defer func() {
		rm.mu.RUnlock()
		log.Printf("RoomManagement: GetRoomByID: Lock released for reading room with ID %s", roomID)
	}()

	if rm.MainRoom != nil && rm.MainRoom.Room.RoomId == roomID {
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

func (rm *RoomManagement) SendMessage(newMessage *entities.Message, user entities.User) error {

	roomID := newMessage.RoomID
	log.Printf("RoomManagement: SendMessage: Sending message to room with ID: %s", roomID)

	// First, we get the room with RLock, as we are reading
	room, err := rm.GetRoomByID(roomID)
	if err != nil {
		log.Printf("RoomManagement: SendMessage: Error getting room: %v", err)
		return err
	}

	log.Printf("RoomManagement: New message created: %+v\n", newMessage)

	// Modify message history
	room.SendMessage(user, *newMessage)
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

	messages := room.GetMessagesFromId(messageID)
	if messages == nil {
		log.Printf("RoomManagement: GetMessagesFromId: Error getting messages")
		return nil, fmt.Errorf("RoomManagement: error in GetByLastMessageId")
	}

	log.Printf("RoomManagement: GetMessagesFromId: Messages obtained successfully (%d messages)", len(messages))
	return messages, nil
}

// Function to get all messages from a room
func (rm *RoomManagement) GetMessages(roomID uuid.UUID) ([]entities.Message, error) {
	log.Printf("RoomManagement: GetMessages: Getting all messages from room %s", roomID.String())
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
	log.Printf("RoomManagement: GetMessages: Getting all messages from roomName %s -- topic: %s", room.RoomName, room.ServerTopic)
	messages := room.GetRoomMessages()
	if messages == nil {
		log.Printf("RoomManagement: GetMessages: Error getting messages")
		return nil, fmt.Errorf("RoomManagement: error in GetMessages")
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
	messages, err := room.GetMessagesWithLimit(messageID, count)
	if err != nil {
		log.Printf("RoomManagement: GetMessageCount: Error getting messages with limit: %v", err)
		return nil, fmt.Errorf("error in GetMessagesWithLimit: %v", err)
	}

	log.Printf("RoomManagement: GetMessageCount: Retrieved %d messages", len(messages))
	return messages, nil
}
