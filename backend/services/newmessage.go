package services

import (
	"backend/entities"
	"log"
)

// Callback para manejar nuevos mensajes
func HandleNewMessages(msg interface{}) {
	message, ok := msg.(*entities.Message)
	if !ok {
		log.Println("Invalid message type received")

	}
	// Lanzar el procesamiento de mensajes en una goroutine para no bloquear la ejecución principal

	go func() {

		// Log para ver los datos recibidos
		log.Printf("HandleNewMessages: Datos recibidos: %+v", message)

		// Obtener la instancia del singleton
		secMod, err := GetChatServerModule()
		if err != nil {
			// Log para mostrar error si el servicio no está disponible
			log.Printf("HandleNewMessages: Error al obtener el servidor chat: %v", err)

			return
		}
		// Llamar al método para enviar el mensaje
		err = secMod.ExecuteSendMessage(message)
		if err != nil {
			// Log para mostrar error al intentar enviar el mensaje
			log.Printf("HandleNewMessages: Error al enviar el mensaje: %v", err)

			return
		}

		// Log para confirmar que el mensaje fue enviado exitosamente
		log.Printf("HandleNewMessages: Mensaje enviado exitosamente a la sala %s: %s", message.RoomName, message.MessageText)

	}()

}
