package services

import (
	"backend/entities"
	"backend/models"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
)

type ChatServerModule struct {
	RoomManagement *RoomManagement
	UserManagement *UserManagement
}

var onceServerChat sync.Once
var instanceChatServer *ChatServerModule

// CreateChatServerModule creates an instance of ChatServerModule
func CreateChatServerModule(persistence *entities.Persistence, configRoomFile string) *ChatServerModule {
	onceServerChat.Do(func() {
		// Validación de los parámetros necesarios
		if persistence == nil || configRoomFile == "" {
			log.Fatalf("ChatServerModule: CreateChatServerModule:" +
				"RoomManagement is not configured correctly.Persistence or configRoomFile is invalid.")
		}
		// Si todo está bien, crea la instancia
		instanceChatServer = &ChatServerModule{
			RoomManagement: NewRoomManagement(persistence, configRoomFile),
			UserManagement: NewUserManagement(),
		}
	})
	return instanceChatServer
}

// GetChatServerModule retrieves the existing instance, with no parameters
func GetChatServerModule() (*ChatServerModule, error) {
	if instanceChatServer == nil {
		return nil, errors.New("ChatServerModule: the instance has not been created. You must call CreateChatServerModule first")
	}

	return instanceChatServer, nil
}

// ValidateTokenAction validates if a token is correct and if the nickname is associated with that token
func (chatModule *ChatServerModule) ValidateTokenAction(token, nickname, action string) bool {
	log.Printf("ChatServerModule: ValidateTokenAction: Validating token: %s, nickname: %s, action: %s\n", token, nickname, action)
	user, err := chatModule.UserManagement.FindUserByToken(token)
	if err != nil || user.Nickname != nickname {
		log.Println("ChatServerModule: ValidateTokenAction: Invalid token or nickname")
		return false
	}

	// Action validation (action) can be added here based on the action type
	switch action {
	case "sendMessage":
		log.Println("ChatServerModule: ValidateTokenAction: Valid action: sendMessage")
		return true
	case "viewRoom":
		log.Println("ChatServerModule: ValidateTokenAction: Valid action: viewRoom")
		return true
	default:
		log.Println("ChatServerModule: ValidateTokenAction: Invalid action")
		return false
	}
}

// CreateSessionToken generates a new token for a user when they register
func (chatModule *ChatServerModule) CreateSessionToken(nickname string) string {
	log.Printf("ChatServerModule: CreateSessionToken: Creating session token for user: %s\n", nickname)
	return models.CreateSessionToken(nickname)
}

// ExecuteLogin handles the login process
func (chatModule *ChatServerModule) ExecuteLogin(nickname string) (*entities.User, error) {
	log.Printf("ChatServerModule: ExecuteLogin: Executing login for user: %s\n", nickname)

	// Call RegisterUser to ensure the user is registered
	if !chatModule.UserManagement.VerifyExistingUser(nickname) {
		log.Println("ChatServerModule: ExecuteLogin: CODL00: Nickname already in use")
		return nil, fmt.Errorf("ExecuteLogin: CODL00: nickname already in use")
	}

	token := chatModule.CreateSessionToken(nickname)
	if token == "" {
		log.Println("ChatServerModule: ExecuteLogin: CODL01: Failed to create token")
		return nil, fmt.Errorf("ExecuteLogin: CODL01: failed to create token")
	}

	newUser, err := chatModule.UserManagement.RegisterUser(nickname, token, &chatModule.RoomManagement.MainRoom.Room)
	if err != nil {
		log.Printf("ChatServerModule: ExecuteLogin: CODL02: Error registering user: %v\n", err)
		return nil, fmt.Errorf("ExecuteLogin: CODL02: error registering user %v", err)
	}

	chatModule.RoomManagement.MainRoom.Room.Users = append(chatModule.RoomManagement.MainRoom.Room.Users, *newUser)
	newUser.RoomId = chatModule.RoomManagement.MainRoom.Room.RoomId
	newUser.RoomName = chatModule.RoomManagement.MainRoom.Room.RoomName

	// Show success message
	log.Printf("ChatServerModule: ExecuteLogin: User %s logged in with token %s\n", nickname, newUser.Token)
	log.Printf("ChatServerModule: ExecuteLogin: Main room: %s (ID: %v)\n", chatModule.RoomManagement.MainRoom.Room.RoomName, chatModule.RoomManagement.MainRoom.Room.RoomId)

	// Return the user with room details
	return newUser, nil
}

// ExecuteLogout handles the logout process
func (chatModule *ChatServerModule) ExecuteLogout(token string) error {
	log.Printf("ChatServerModule: ExecuteLogout: Executing logout for token: %s\n", token)

	// Verify that the token is associated with a valid user
	user, err := chatModule.UserManagement.FindUserByToken(token)
	if err != nil {
		log.Printf("ChatServerModule: ExecuteLogout: Error finding user for token: %s. Error: %v\n", token, err)
		return fmt.Errorf("ExecuteLogout: COLG00: user not found or invalid token")
	}

	// Perform the logout operation
	err = chatModule.UserManagement.Logout(user.Token)
	if err != nil {
		log.Printf("ChatServerModule: ExecuteLogout: Error during logout for user: %s. Error: %v\n", user.Nickname, err)
		return fmt.Errorf("ExecuteLogout: COLG01: logout operation failed")
	}

	log.Printf("ChatServerModule: ExecuteLogout: User %s logged out successfully\n", user.Nickname)

	// Return a success message or status
	return nil
}

// ExecuteSendMessage allows a user to send a message
func (chatModule *ChatServerModule) ExecuteSendMessage(msg *entities.Message) error {

	nickname := msg.Nickname
	token := msg.Token
	message := msg.MessageText
	roomId := msg.RoomID
	log.Printf("ChatServerModule: ExecuteSendMessage: Executing send message: %s for user: %s in room: %v\n", message, nickname, roomId)

	// Validate if the user is authorized to send the message
	token_user, err := chatModule.UserManagement.GetUserToken(nickname)
	if err != nil {
		log.Println("ChatServerModule: ExecuteSendMessage: Invalid token or action not allowed")
		return errors.New("ChatServerModule: ExecuteSendMessage: CODM00: invalid token or action not allowed")
	}

	if !chatModule.ValidateTokenAction(token_user, nickname, "sendMessage") {
		log.Println("ChatServerModule: ExecuteSendMessage: Invalid token or action not allowed")
		return errors.New("ChatServerModule: ExecuteSendMessage: CODM01: invalid token or action not allowed")
	}
	log.Printf("ChatServerModule: ExecuteSendMessage: token_user [%s]  \n", token_user)
	log.Printf("ChatServerModule: ExecuteSendMessage: token [%s]  \n", token)

	if token_user != token {
		log.Println("ChatServerModule: ExecuteSendMessage: Token mismatch")
		return errors.New("ChatServerModule: ExecuteSendMessage: CODM02: invalid token or action not allowed")
	}

	user, err := chatModule.UserManagement.FindUserByToken(token_user)
	if err != nil {
		log.Printf("ChatServerModule: ExecuteSendMessage: CODM03: Error finding user by token: %v\n", err)
		return err
	}
	// Call the logic to send the message
	err = chatModule.RoomManagement.SendMessage(msg, user)
	if err != nil {
		log.Printf("ChatServerModule: ExecuteSendMessage: CODM04: Error sending message: %v\n", err)
		return err
	}

	log.Printf("ChatServerModule: ExecuteSendMessage: CODM05: Message sent by %s: %s\n", nickname, message)
	return nil
}

// ExecuteViewRoom allows a user to view the message list in a room
func (chatModule *ChatServerModule) ExecuteViewRoom(nickname string, roomId uuid.UUID) error {
	log.Printf("ChatServerModule: ExecuteViewRoom: Executing view room: %v for user: %s\n", roomId, nickname)

	// Validate if the user is authorized to view the room
	token, err := chatModule.UserManagement.GetUserToken(nickname)
	if err != nil {
		log.Println("ChatServerModule: ExecuteViewRoom: Invalid token or action not allowed")
		return errors.New("ChatServerModule: ExecuteViewRoom: invalid token or action not allowed")
	}

	if !chatModule.ValidateTokenAction(token, nickname, "viewRoom") {
		log.Println("ChatServerModule: ExecuteViewRoom: Invalid token or action not allowed")
		return errors.New("ChatServerModule: ExecuteViewRoom: invalid token or action not allowed")
	}

	// Call the logic to view the room's messages
	messages, err := chatModule.RoomManagement.GetMessages(roomId)
	if err != nil {
		log.Printf("ChatServerModule: ExecuteViewRoom: Error getting room messages: %v\n", err)
		return fmt.Errorf("ChatServerModule: ExecuteViewRoom: invalid token or action not allowed %v", err)
	}

	// Display the messages
	log.Println("ChatServerModule: ExecuteViewRoom: Messages in the room:")
	for _, msg := range messages {
		log.Printf("[%s]: %s\n", msg.Nickname, msg.MessageText)
	}

	return nil
}
