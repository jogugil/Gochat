// interfaces/persistencia.go
package entities

import (
	"github.com/google/uuid"
)

type Persistencia interface {
	GuardarUsuario(usuario *User) error
	GuardarSala(sala Room) error
	ObtenerSala(id uuid.UUID) (Room, error)
	ObtenerMensajesAnteriores(idMensaje uuid.UUID, cantidadRestante int) ([]Message, error)
	GuardarMensaje(mensaje *Message) error
	GuardarMensajesEnBaseDeDatos(mensajes []Message, roomId uuid.UUID) error
	ObtenerMensajesDesdeSala(idSala uuid.UUID) ([]Message, error)
	ObtenerMensajesDesdeSalaPorId(idSala uuid.UUID, idMensaje uuid.UUID) ([]Message, error)
}
