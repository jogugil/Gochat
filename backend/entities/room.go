package entities

import (
	"time"

	"github.com/google/uuid"
)

type Room struct {
	RoomId           uuid.UUID      `json:"roomid"`
	RoomName         string         `json:"roomname"`
	RoomType         string         `json:"roomtype"`
	Users            []User         `json:"chatusers"`
	ActivityTime     int            `json:"activitytime"`
	Status           bool           `json:"status"`
	LastUpdate       time.Time      `json:"lastupdate"`
	CreationDate     time.Time      `json:"creationdate"`
	DeactivationDate time.Time      `json:"deactivationdate"`
	MessageHistory   *CircularQueue `json:"messagehistory"` // Circular Queue
}
