package models

import (
	"backend/entities"
	"log"

	"github.com/google/uuid"
)

// Alias del tipo Sala
type LocalSala entities.Room

func (sala *LocalSala) AgregarUsuario(usuario entities.User) {
	log.Printf("AgregarUsuario: Entrando con usuario %v a la sala %s", usuario, sala.RoomName)

	// Operación clave: Añadir el usuario a la lista
	sala.Users = append(sala.Users, usuario)

	log.Printf("AgregarUsuario: Usuario %v agregado correctamente a la sala %s", usuario, sala.RoomName)
}

func (sala *LocalSala) QuitarUsuario(usuario entities.User) {
	log.Printf("QuitarUsuario: Entrando con usuario %v para quitar de la sala %s", usuario, sala.RoomName)

	// Recorre la lista de usuarios
	for i, u := range sala.Users {
		// Si se encuentra al usuario, lo elimina de la lista
		if u == usuario {
			log.Printf("QuitarUsuario: Usuario %v encontrado en la posición %d", usuario, i)

			// Operación clave: Elimina el usuario manteniendo los demás
			sala.Users = append(sala.Users[:i], sala.Users[i+1:]...)

			log.Printf("QuitarUsuario: Usuario %v eliminado correctamente de la sala %s", usuario, sala.RoomName)
			return
		}
	}

	log.Printf("QuitarUsuario: Usuario %v no encontrado en la sala %s", usuario, sala.RoomName)
}

func (sala *LocalSala) EnviarMensaje(usuario entities.User, mensaje entities.Message) {
	log.Printf("EnviarMensaje: Entrando con usuario %v y mensaje %v en la sala %s", usuario, mensaje, sala.RoomName)

	// Operación clave: Añadir el mensaje a la cola
	sala.MessageHistory.Enqueue(mensaje, usuario.RoomId)

	log.Printf("EnviarMensaje: Mensaje %v añadido correctamente a la cola de la sala %s", mensaje, sala.RoomName)
}

func (sala *LocalSala) ObtenerMensajesSala() []entities.Message {
	log.Printf("ObtenerMensajesSala: Entrando para obtener mensajes de la sala %s", sala.RoomName)

	// Operación clave: Obtener todos los mensajes
	mensajes, err := sala.MessageHistory.ObtenerTodos(sala.RoomId)
	if err != nil {
		log.Printf("ObtenerMensajesSala: Error al obtener mensajes de la sala %s: %v", sala.RoomName, err)

		return nil
	}

	log.Printf("ObtenerMensajesSala: Mensajes obtenidos correctamente de la sala %s", sala.RoomName)
	return mensajes
}

func (sala *LocalSala) ObtenerMensajesdesdeId(idMensaje uuid.UUID) []entities.Message {
	log.Printf("ObtenerMensajesdesdeId: Entrando con idMensaje %v para la sala %s", idMensaje, sala.RoomName)
	log.Printf("ObtenerMensajesdesdeId: sala.MessageHistory: %v", sala.MessageHistory)

	// Operación clave: Obtener mensajes desde un ID específico
	mensajes, err := sala.MessageHistory.ObtenerMensajesDesdeId(sala.RoomId, idMensaje)
	if err != nil {
		log.Printf("ObtenerMensajesdesdeId: Error al obtener mensajes desde ID %v en la sala %s: %v", idMensaje, sala.RoomName, err)
		return nil
	}
	log.Printf("ObtenerMensajesdesdeId: mensajes: %v", mensajes)

	log.Printf("ObtenerMensajesdesdeId: Mensajes obtenidos correctamente desde ID %v en la sala %s", idMensaje, sala.RoomName)
	return mensajes
}
