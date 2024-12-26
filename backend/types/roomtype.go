package types

import "github.com/google/uuid"

// Room Type (Main or Temporary)
type RoomType string

const (
	Main      RoomType = "Main"      // Main room, unique
	Temporary RoomType = "Temporary" // Temporary room
	Fixed     RoomType = "Fixed"     // Fixed room (with a fixed ID)
)

// Structure to handle the room type
type Room struct {
	ID       uuid.UUID // Unique UUID of the room
	Name     string    // Name of the room
	RoomType RoomType  // Type of the room (Main, Temporary, Fixed)
}
