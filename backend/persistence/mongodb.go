package persistence

import (
	"backend/entities"
	"backend/utils"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB Configuration (can be set up in the MongoPersistence structure itself)

// Email Configuration

var mongoInstance *MongoPersistence
var onceMongodb sync.Once

type MongoPersistence struct {
	client *mongo.Client
	db     *mongo.Database
}

// NewMongoPersistence creates or returns a MongoPersistence instance with a connection pool
func NewMongoPersistence(uri, dbName string) (*entities.Persistence, error) {
	log.Printf("MongoPersistence: NewMongoPersistence: Creating a new MongoPersistence instance with URI: %s and DB: %s\n", uri, dbName)
	onceMongodb.Do(func() {
		// Set up the MongoDB client with a connection pool (maximum 20 connections).
		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri).SetMaxPoolSize(20))
		if err != nil {
			utils.LogCriticalError(fmt.Sprintf("MongoPersistence: NewMongoPersistence: MongoDB connection error: %v\n", err))
			log.Printf("MongoPersistence: NewMongoPersistence: MongoDB connection error: %v\n", err)
			return
		}
		// Verify the connection
		err = client.Ping(context.Background(), nil)
		if err != nil {
			log.Printf("MongoPersistence: NewMongoPersistence: Error pinging MongoDB: %v\n", err)
			return
		}
		db := client.Database(dbName)
		mongoInstance = &MongoPersistence{client: client, db: db}
	})

	if mongoInstance == nil {
		return nil, fmt.Errorf("MongoPersistence: NewMongoPersistence: unable to create MongoPersistence instance")
	}

	var persistence entities.Persistence = mongoInstance
	log.Println("MongoPersistence: NewMongoPersistence: MongoPersistence instance successfully created")
	return &persistence, nil
}

// GetDBInstance retrieves the database instance
func GetDBInstance() (*entities.Persistence, error) {
	log.Println("MongoPersistence: GetDBInstance: Retrieving MongoDB database instance...")
	if mongoInstance == nil {
		// Return an error if the instance hasn't been initialized
		return nil, errors.New("MongoPersistence: GetDBInstance: MongoPersistence instance has not been initialized")
	}

	// Return mongoInstance as a Persistence type, which implements the Persistence interface
	var persistence entities.Persistence = mongoInstance
	return &persistence, nil
}

func (mp *MongoPersistence) handleNoDocumentsError(err error, roomId uuid.UUID) error {
	if err == mongo.ErrNoDocuments {
		// If no documents are found, check if the room exists
		log.Printf("MongoPersistence: handleNoDocumentsError: No messages found for room: %v", roomId)
		// Return the result of CheckRoomExists
		exists, err := mp.CheckRoomExists(roomId)
		if err != nil {
			log.Printf("MongoPersistence: Error verifying if the room exists: %v\n", err)
			return err // Return the error if one occurs in CheckRoomExists
		}
		if !exists {
			log.Printf("MongoPersistence: The room with id [%v] does not exist.\n", roomId)
			return fmt.Errorf("MongoPersistence: handleNoDocumentsError: the room with id %v does not exist", roomId)
		} else {
			return nil
		}
	}
	log.Printf("MongoPersistence: Error retrieving the base message from the DB: %v\n", err)
	return fmt.Errorf("error retrieving the base message: %v", err) // If the error is not mongo.ErrNoDocuments, return the original error
}

func (mp *MongoPersistence) GetPreviousMessages(messageId uuid.UUID, remainingCount int) ([]entities.Message, error) {
	collection := mp.db.Collection("messages")

	// Find the message with the given id to get its index
	var message entities.Message
	filter := bson.M{"messageId": messageId}
	err := collection.FindOne(context.TODO(), filter).Decode(&message)
	if err != nil {
		log.Printf("MongoPersistence : GetPreviousMessages: Unable to find the message with id %s: %v", messageId, err)
		return nil, nil
	}

	// Get messages prior to the index of the found message
	targetIndex := message.MessageId // Assuming there's a "Index" field in your structure representing the position
	previousMessagesFilter := bson.M{
		"index":     bson.M{"$lt": targetIndex}, // Indices less than the target message index
		"messageId": bson.M{"$ne": messageId},   // Exclude the message with the given id
	}
	options := options.Find().
		SetSort(bson.M{"index": -1}).   // Sort by index in descending order
		SetLimit(int64(remainingCount)) // Limit the number of results

	// Find the previous messages
	cursor, err := collection.Find(context.TODO(), previousMessagesFilter, options)
	if err != nil {
		return nil, fmt.Errorf("Error searching for previous messages in MongoDB: %v", err)
	}
	defer cursor.Close(context.TODO())

	// Decode the found messages
	var messages []entities.Message
	for cursor.Next(context.TODO()) {
		var previousMessage entities.Message
		if err := cursor.Decode(&previousMessage); err != nil {
			return nil, fmt.Errorf("Error decoding a message: %v", err)
		}
		messages = append(messages, previousMessage)
	}

	// Check cursor errors
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("Error in MongoDB cursor: %v", err)
	}

	return messages, nil
}

// Creates a room with the given roomId
func (mp *MongoPersistence) CreateRoom(room entities.Room) error {
	roomsCollection := mp.db.Collection("rooms")
	_, err := roomsCollection.UpdateOne(
		context.TODO(),
		bson.M{"id": room.RoomId},
		bson.D{
			{Key: "$set", Value: room}, // This uses labeled keys
		},
		options.Update().SetUpsert(true),
	)
	return err
}

// Implements SaveRoom
func (m *MongoPersistence) SaveRoom(room entities.Room) error {
	log.Printf("MongoPersistence: SaveRoom: Saving room: %+v\n", room)
	collection := m.db.Collection("rooms")
	_, err := collection.InsertOne(context.TODO(), room)
	if err != nil {
		log.Printf("MongoPersistence: SaveRoom: Error saving the room: %v\n", err)
	}
	return err
}

// Implements GetRoom
func (m *MongoPersistence) GetRoom(id uuid.UUID) (entities.Room, error) {
	log.Printf("MongoPersistence: GetRoom: Getting room with ID: %s\n", id)
	var room entities.Room
	collection := m.db.Collection("rooms")
	err := collection.FindOne(context.TODO(), bson.M{"id": id}).Decode(&room)
	if err != nil {
		log.Printf("MongoPersistence: GetRoom: Error getting the room: %v\n", err)
		return entities.Room{}, err
	}
	log.Printf("MongoPersistence: GetRoom: Room retrieved: %+v\n", room)
	return room, nil
}

// SaveMessage saves a single message in the MongoDB database
func (mp *MongoPersistence) SaveMessage(message *entities.Message) error {
	log.Printf("MongoPersistence: SaveMessage: Saving message: %+v\n", message)
	collection := mp.db.Collection("messages")
	_, err := collection.InsertOne(context.TODO(), message)
	if err != nil {
		log.Printf("MongoPersistence: SaveMessage: Error saving the message in MongoDB: %v\n", err)
		return fmt.Errorf("MongoPersistence: SaveMessage: error saving the message in MongoDB: %v", err)
	}
	log.Println("MongoPersistence: SaveMessage: Message saved successfully")
	return nil
}

// SaveMessagesToDatabase saves a list of messages in MongoDB
func (mp *MongoPersistence) SaveMessagesToDatabase(messages []entities.Message, roomId uuid.UUID) error {
	log.Printf("MongoPersistence: SaveMessagesToDatabase: Saving list of %d messages\n", len(messages))
	collection := mp.db.Collection("messages")

	// Convert messages to a list of interfaces{}
	var documents []interface{}
	for _, message := range messages {
		documents = append(documents, bson.D{
			{Key: "user_nickname", Value: message.Nickname},
			{Key: "send_date", Value: message.SendDate},
			{Key: "room_id", Value: roomId},
			{Key: "room_name", Value: message.RoomName},
		})
	}

	// Insert all documents into the collection
	_, err := collection.InsertMany(context.TODO(), documents)
	if err != nil {
		log.Printf("MongoPersistence: SaveMessagesToDatabase: Error saving the messages in MongoDB: %v\n", err)
		return fmt.Errorf("MongoPersistence: SaveMessagesToDatabase: error saving the messages in MongoDB: %v", err)
	}

	// Everything went well
	log.Println("MongoPersistence: SaveMessagesToDatabase: Messages saved successfully")
	return nil
}

func (mp *MongoPersistence) GetMessagesFromRoom(roomId uuid.UUID) ([]entities.Message, error) {
	log.Printf("MongoPersistence: GetMessagesFromRoom: Searching messages for room [%s]", roomId)

	collection := mp.db.Collection("messages")

	// Filter to search for all messages from the room with room_id
	filter := bson.D{{Key: "room_id", Value: roomId}}

	// Find all messages with that room_id
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		log.Printf("MongoPersistence: GetMessagesFromRoom: Error searching for messages in MongoDB: %v\n", err)
		return nil, fmt.Errorf("Error searching for messages in MongoDB: %v", err)
	}
	defer cursor.Close(context.TODO())

	// Iterate over the returned documents and map them to a list of Message entities
	var messages []entities.Message
	for cursor.Next(context.TODO()) {
		var message entities.Message
		if err := cursor.Decode(&message); err != nil {
			log.Printf("MongoPersistence: GetMessagesFromRoom: Error decoding message: %v", err)
			continue
		}
		messages = append(messages, message)
	}

	// Check for errors in the cursor
	if err := cursor.Err(); err != nil {
		log.Printf("MongoPersistence: GetMessagesFromRoom: Error during cursor iteration: %v", err)
		return nil, err
	}

	// Return the list of messages
	log.Printf("MongoPersistence: GetMessagesFromRoom: Found %d messages for room [%s]", len(messages), roomId)
	return messages, nil
}
func (mp *MongoPersistence) CheckRoomExists(idRoom uuid.UUID) (bool, error) {
	log.Printf("MongoPersistence: CheckRoomExists: Checking if the room exists with ID: %s\n", idRoom)

	// Access the rooms collection
	collection := mp.db.Collection("rooms")
	filter := bson.M{"_id": idRoom}

	var result bson.M
	err := collection.FindOne(context.TODO(), filter).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("MongoPersistence: CheckRoomExists: The room with ID %s does not exist.\n", idRoom)
			return false, nil // The room doesn't exist
		}
		log.Printf("MongoPersistence: CheckRoomExists: Error while searching for the room: %v\n", err)
		return false, err // Other error
	}

	log.Printf("MongoPersistence: CheckRoomExists: The room with ID %s exists.\n", idRoom)
	return true, nil // The room exists
}

func (mp *MongoPersistence) GetMessagesFromRoomById(idRoom uuid.UUID, idMessage uuid.UUID) ([]entities.Message, error) {
	log.Printf("MongoPersistence: GetMessagesFromId: Getting messages from the room with ID: %s and message with ID: %s\n", idRoom, idMessage)
	collection := mp.db.Collection("messages")

	// Search for the base message
	var baseMessage entities.Message
	filterBase := bson.D{{Key: "id_message", Value: idMessage}, {Key: "id_room", Value: idRoom}}
	err := collection.FindOne(context.TODO(), filterBase).Decode(&baseMessage)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("MongoPersistence: GetMessagesFromRoomById: No message found with id_message: %v in room: %v\n", idMessage, idRoom)

			// Verify if the room exists
			roomExists, err := mp.CheckRoomExists(idRoom)
			if err != nil {
				log.Printf("MongoPersistence: GetMessagesFromRoomById: Error verifying if the room exists: %v\n", err)
				return nil, fmt.Errorf("error verifying if the room exists: %v", err)
			}

			if !roomExists {
				log.Printf("MongoPersistence: GetMessagesFromRoomById: The room with id [%v] does not exist. Creating a new room...\n", idRoom)

				// Create the new room in the database
				newRoom := entities.Room{RoomId: idRoom}
				err := mp.CreateRoom(newRoom) // Method to create the room in the rooms collection
				if err != nil {
					log.Printf("MongoPersistence: GetMessagesFromRoomById: Error creating the new room: %v\n", err)
					return nil, fmt.Errorf("error creating the new room: %v", err)
				}

				log.Printf("MongoPersistence: GetMessagesFromRoomById: A new room with id [%v] has been created with no messages.\n", idRoom)

				// Return nil since there are no messages
				return nil, nil
			}

			// If the room exists but has no messages
			log.Printf("MongoPersistence: The room with id [%v] exists but has no messages.\n", idRoom)
			return nil, nil
		}

		// Error attempting to fetch the base message
		log.Printf("MongoPersistence: Error fetching the base message from the DB: %v\n", err)
		return nil, fmt.Errorf("error fetching the base message: %v", err)
	}
	// Filter to get messages after the base message idMessage
	filter := bson.D{
		{Key: "id_room", Value: idRoom},
		{Key: "send_date", Value: bson.D{{Key: "$gte", Value: baseMessage.SendDate}}}, // Ensure messages are after baseMessage's send_date
	}

	// Set up options to sort messages by send date, with the oldest first
	findOptions := options.Find().SetSort(bson.D{{Key: "send_date", Value: 1}}) // Ascending, oldest first.

	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		log.Printf("MongoPersistence: GetMessagesFromId: Error executing the query: %v\n", err)
		return nil, fmt.Errorf("error executing the query: %v", err)
	}
	defer cursor.Close(context.TODO())

	var messages []entities.Message
	// Iterate through the messages obtained from the cursor
	for cursor.Next(context.TODO()) {
		var message entities.Message
		// Decode each message into the appropriate structure
		if err := cursor.Decode(&message); err != nil {
			log.Printf("MongoPersistence: GetMessagesFromId: Error decoding the message: %v\n", err)
			return nil, fmt.Errorf("error decoding the message: %v", err)
		}
		// Add the message to the list of messages
		messages = append(messages, message)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("MongoPersistence: GetMessagesFromId: Error iterating over the cursor: %v\n", err)
		return nil, fmt.Errorf("error iterating over the cursor: %v", err)
	}

	// Log the number of messages obtained
	log.Printf("MongoPersistence: GetMessagesFromId: Messages obtained after the message ID: %d\n", len(messages))

	// Return the list of obtained messages
	return messages, nil
}

func (mp *MongoPersistence) SaveUser(user *entities.User) error {
	log.Printf("MongoPersistence: SaveUser: Starting to save the user with ID: %s\n", user.UserId)

	// Create a BSON document from the user
	document := bson.D{
		{Key: "user_id", Value: user.UserId},
		{Key: "nickname", Value: user.Nickname},
		{Key: "token", Value: user.Token},
		{Key: "last_action_time", Value: user.LastActionTime},
		{Key: "state", Value: user.State},
		{Key: "type", Value: user.Type},
		{Key: "room_id", Value: user.RoomId},     // The room can be nil
		{Key: "room_name", Value: user.RoomName}, // The room can be nil
	}
	log.Printf("MongoPersistence: SaveUser: BSON document for saving user: %+v\n", document)

	// Insert the document into the collection
	collection := mp.db.Collection("users")
	log.Printf("MongoPersistence: SaveUser: Users collection fetched: %+v\n", document)
	_, err := collection.InsertOne(context.Background(), document)
	if err != nil {
		log.Printf("MongoPersistence: SaveUser: Error saving the user in MongoDB: %v\n", err)
		return fmt.Errorf("MongoPersistencia: SaveUser: error saving the user in MongoDB: %v", err)
	}

	// Success confirmation
	log.Printf("MongoPersistence: SaveUser: User with ID: %s saved successfully in MongoDB\n", user.UserId)
	return nil
}
