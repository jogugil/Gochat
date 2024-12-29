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
		log.Println("UserManagement: NewUserManagement:  Initializing the User Management instance.")
		// Here you could load users from a database or file if necessary
		// Example of loading users (depends on your implementation and needs)
		// instance.LoadUsersFromFile("users.txt")
		log.Println("UserManagement: NewUserManagement:  Initialization completed.")
	})
	return userInstance
}

func (management *UserManagement) FindUserByToken(token string) (entities.User, error) {
	log.Println("UserManagement:FindUserByToken: Starting search for user by token:", token)
	for _, user := range management.Users {
		if user.Token == token {
			log.Println("User found:", user.Nickname)
			return *user, nil
		}
	}
	log.Println("UserManagement:FindUserByToken: User not found for token:", token)
	return entities.User{}, errors.New("user not found")
}

func (management *UserManagement) GetUserToken(nickname string) (string, error) {
	log.Println("UserManagement: GetUserToken: Getting token for user:", nickname)
	log.Printf("UserManagement: GetUserToken: Getting token for user: %d - nickname : %s", len(management.Users), management.Users[0].Nickname)
	for _, user := range management.Users {
		if user.Nickname == nickname {
			log.Println("UserManagement:GetUserToken: Token found for user:", user.Token)
			return user.Token, nil
		}
	}
	log.Println("UserManagement: GetUserToken: User not found for nickname:", nickname)
	return "", errors.New("user not found")
}

// VerifyExistingUser checks if a user with the given nickname is already registered
func (management *UserManagement) VerifyExistingUser(nickname string) bool {
	log.Println("UserManagement: VerifyExistingUser: Verifying if user already exists:", nickname)
	for _, user := range management.Users {
		if user.Nickname == nickname {
			log.Println("UserManagement: VerifyExistingUser: User already registered:", nickname)
			return false // User already registered
		}
	}
	log.Println("UserManagement: VerifyExistingUser: User not registered:", nickname)
	return true // User not registered
}

// Function that registers a user and saves it in the database
func (management *UserManagement) RegisterUser(nickname string, token string, room *entities.Room) (*entities.User, error) {
	log.Println("UserManagement: RegisterUser: Registering new user:", nickname)

	// Create a new user with the provided room
	newUser := models.NewGoChatUser(nickname, room)
	log.Println("UserManagement: RegisterUser: New user created:", newUser.Nickname)

	// Create a new instance of MongoPersistence
	mongoPersistence, err := persistence.GetDBInstance()
	if err != nil {
		log.Println("UserManagement: RegisterUser: Error creating MongoPersistence instance:", err)
		return nil, err
	}

	// Save the user in the database
	log.Println("UserManagement: RegisterUser: Saving user to the database...")
	err = (*mongoPersistence).SaveUser(newUser)
	if err != nil {
		log.Println("UserManagement: RegisterUser: Error saving the user:", err)
		return nil, err
	} else {
		log.Println("UserManagement: RegisterUser: User saved successfully to the database")
	}

	// Add the user to the in-memory list of users
	management.Users = append(management.Users, newUser)
	log.Println("UserManagement: RegisterUser: User added to in-memory list:", newUser.Nickname)
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
func (management *UserManagement) Logout(token string) error {
	log.Println("UserManagement: Logout: Intentando logout para el usuario con token:", token)

	// Buscar el usuario por token
	user, err := management.FindUserByToken(token)
	if err != nil {
		log.Println("UserManagement: Logout: Error al encontrar usuario:", err)
		return err
	}

	// Cambiar el estado del usuario a Inactive
	user.State = types.Inactive
	log.Println("UserManagement: Logout: Estado del usuario cambiado a Inactive:", user.Nickname)

	// Guardar los cambios en la base de datos
	mongoPersistence, err := persistence.GetDBInstance()
	if err != nil {
		log.Println("UserManagement: Logout: Error al obtener instancia de persistencia:", err)
		return err
	}

	err = (*mongoPersistence).SaveUser(&user)
	if err != nil {
		log.Println("UserManagement: Logout: Error al guardar usuario en la base de datos:", err)
		return err
	}

	log.Println("UserManagement: Logout: Logout exitoso para el usuario:", user.Nickname)
	return nil
}
