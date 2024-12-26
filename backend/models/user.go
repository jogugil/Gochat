package models

import (
	"backend/entities"
	"backend/types" // Import the interfaces package
	"log"
	"time"

	"github.com/google/uuid" // Create unique UUIDs
)

// Definición de LocalUser utilizando composición
type LocalUser struct {
	User entities.User
}

// Implementation of the methods for the UserChat interface
func (u *LocalUser) StartSession() bool {
	u.User.LastActionTime = time.Now()
	u.User.State = types.Active
	return true
}

func (u *LocalUser) EndSession() bool {
	u.User.State = types.Inactive
	return true
}

func (u *LocalUser) UpdateStatus() {
	u.User.State = types.Active
}

func (u *LocalUser) JoinRoom(room *entities.Room) {
	u.User.RoomId = room.RoomId
}

func (u *LocalUser) LeaveRoom() {
	roomId, err := uuid.Parse("00000000-0000-0000-0000-000000000000")
	if err != nil {
		log.Printf("error setting room ID to nil: %v", err)
	}
	u.User.RoomId = roomId // Now the room reference is removed
}

func NewGoChatUser(nickname string, room *entities.Room) *entities.User {
	return &entities.User{
		UserId:         "usr-" + uuid.New().String(),
		Nickname:       nickname,
		Token:          CreateSessionToken(nickname),
		Type:           "userchat",
		LastActionTime: time.Now(),
		State:          types.Active,
		RoomId:         room.RoomId,
		RoomName:       room.RoomName,
	}
}
