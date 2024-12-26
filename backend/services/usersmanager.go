package services

import (
	"backend/entities"
	"backend/models"
	"backend/persistence"
	"backend/types"
	"errors"
	"log"
	"sync"
)

type UserManagement struct {
	Users    []*entities.User // List of users
	onceUser sync.Once        // Ensures the instance is initialized only once
}

var userInstance *UserManagement // Singleton instance of UserManagement

// NewUserManagement creates and returns a Singleton instance of UserManagement
func NewUserManagement() *UserManagement {
	// If the instance hasn't been created yet, create a new one
	if userInstance == nil {
		userInstance = &UserManagement{
			Users: make([]*entities.User, 0), // Initialize an empty user list
		}
	}
	// Use once.Do to ensure the setup is performed only once
	userInstance.onceUser.Do(func() {
		log.Println("NewUserManagement: Initializing the User Management instance.")
		// Here you could load users from a database or file if necessary
		// Example of loading users (depends on your implementation and needs)
		// instance.LoadUsersFromFile("users.txt")
		log.Println("NewUserManagement: Initialization completed.")
	})
	return userInstance
}

func (management *UserManagement) FindUserByToken(token string) (entities.User, error) {
	log.Println("Starting search for user by token:", token)
	for _, user := range management.Users {
		if user.Token == token {
			log.Println("User found:", user.Nickname)
			return *user, nil
		}
	}
	log.Println("User not found for token:", token)
	return entities.User{}, errors.New("user not found")
}

func (management *UserManagement) GetUserToken(nickname string) (string, error) {
	log.Println("Getting token for user:", nickname)
	for _, user := range management.Users {
		if user.Nickname == nickname {
			log.Println("Token found for user:", user.Token)
			return user.Token, nil
		}
	}
	log.Println("User not found for nickname:", nickname)
	return "", errors.New("user not found")
}

// VerifyExistingUser checks if a user with the given nickname is already registered
func (management *UserManagement) VerifyExistingUser(nickname string) bool {
	log.Println("Verifying if user already exists:", nickname)
	for _, user := range management.Users {
		if user.Nickname == nickname {
			log.Println("User already registered:", nickname)
			return false // User already registered
		}
	}
	log.Println("User not registered:", nickname)
	return true // User not registered
}

// Function that registers a user and saves it in the database
func (management *UserManagement) RegisterUser(nickname string, token string, room *entities.Room) (*entities.User, error) {
	log.Println("Registering new user:", nickname)

	// Create a new user with the provided room
	newUser := models.NewGoChatUser(nickname, room)
	log.Println("New user created:", newUser.Nickname)

	// Create a new instance of MongoPersistence
	mongoPersistence, err := persistence.GetDBInstance()
	if err != nil {
		log.Println("Error creating MongoPersistence instance:", err)
		return nil, err
	}

	// Save the user in the database
	log.Println("Saving user to the database...")
	err = (*mongoPersistence).SaveUser(newUser)
	if err != nil {
		log.Println("Error saving the user:", err)
		return nil, err
	} else {
		log.Println("User saved successfully to the database")
	}

	// Add the user to the in-memory list of users
	management.Users = append(management.Users, newUser)
	log.Println("User added to in-memory list:", newUser.Nickname)
	return newUser, nil
}

// For other versions, pass the room UUID and return the users in that room
// The room object contains the list of active users in the room
// Method GetActiveUsers that returns a list of active users
func (g *UserManagement) GetActiveUsers() []*entities.User {
	var activeUsers []*entities.User
	for _, user := range g.Users {
		if user.State == types.Active {
			activeUsers = append(activeUsers, user)
		}
	}
	return activeUsers
}
