package entities

import (
	"backend/types" // Importamos el paquete interfaces
	"time"

	"github.com/google/uuid"
	// Creación de uuid's únicos
)

type User struct {
	UserId         string
	Nickname       string
	Token          string
	LastActionTime time.Time
	State          types.EstadoUsuario
	Type           string
	RoomId         uuid.UUID
	RoomName       string
}
