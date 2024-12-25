package models

import (
	"log"
	"time"

	"backend/entities"

	"github.com/google/uuid"
)

// Crear un nuevo mensaje con los datos proporcionados
func CrearMensaje(mensajeText string, usuario entities.User, idSala uuid.UUID, nombreSala string) *entities.Message {
	return &entities.Message{
		MessageId:   generarIDMensaje(), // Función para generar IDs únicos. Generado en el Servidor
		MessageType: entities.Ordinario, // Tipo predeterminado: Ordinario.En esta versión siempre es ordinario
		SendDate:    time.Now(),         // Fecha del cliente. Se genera en el cleinte antes de enviar el mensaje
		ServerDate:  time.Now(),         // Fecha del servidor. Se genera en el srvidor antes de giardalo en el biffer
		Nickname:    usuario.Nickname,   // Viene del cliente
		Token:       usuario.Token,      // Viene del cliente
		MessageText: mensajeText,        // Viene del cliente
		RoomID:      idSala,             // Viene del cliente
		RoomName:    nombreSala,         //"Sala Principal",   //Sólo para esta versión, cambiarla para principal --123---
	}
}

// Crear un nuevo mensaje con datos enviados por el cliente
func CrearMensajeConFecha(mensajeText string, usuario entities.User, idSala uuid.UUID, nombreSala string, fechaEnvio time.Time) *entities.Message {
	return &entities.Message{
		MessageId:   generarIDMensaje(),
		MessageType: entities.Ordinario, // Tipo predeterminado: Ordinario
		SendDate:    fechaEnvio,         // Fecha enviada por el cliente
		ServerDate:  time.Now(),         // Fecha generada por el servidor
		Nickname:    usuario.Nickname,
		Token:       usuario.Token,
		MessageText: mensajeText,
		RoomID:      idSala,     // Viene del cliente
		RoomName:    nombreSala, // "Sala Principal", //Sólo para esta versión, cambiarla para principal --123---
	}
}

// Generador de ID de mensajes
func generarIDMensaje() uuid.UUID {
	newUUID := uuid.New()
	log.Println("** CrearMensajeConFecha. generarIDMensaje : Nuevo UUID generado:", newUUID) // Imprimir para verificar
	return newUUID
}
