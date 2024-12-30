package services

import (
	"backend/entities"

	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

// Convertir los mensajes a la estructura de respuesta
func ConvertirMensajes(mensajes []entities.Message) []entities.MessageResponse {
	log.Println("ConvertirMensajes: Iniciando conversión de mensajes.")
	var respuesta []entities.MessageResponse
	for _, mensaje := range mensajes {
		respuesta = append(respuesta, entities.MessageResponse{
			MessageId:   mensaje.MessageId,
			Nickname:    mensaje.Nickname,
			MessageText: mensaje.MessageText,
		})
	}
	log.Printf("ConvertirMensajes: Se han convertido %d mensajes.\n", len(respuesta))
	return respuesta
}

// Manejador para la ruta POST /messagelist
func HandleGetMessage(msg interface{}) {
	// Decodificar el cuerpo de la solicitud
	log.Println("HandleGetMessage: Iniciando el manejo de la solicitud POST para la lista de mensajes.")
	requestData, ok := msg.(*entities.RequestLisMessages)
	if !ok {
		log.Println("Invalid message type received")

	}
	nickname := requestData.Nickname
	token := requestData.TokenSesion

	// Crear un canal para pasar la respuesta
	respChan := make(chan entities.ResponseListMessages)
	secMod, err := GetChatServerModule()
	if err != nil {
		log.Printf("HandleGetUsersMessage: Error al obtener el modulo de chat: %v", err)
		return
	}
	// Ejecutar el manejo de la solicitud en una goroutine
	go func() {

		log.Printf("HandleGetUsersMessage: Datos recibidos: %+v", requestData)

		// Validate if the user is authorized to send the message
		token_user, err := secMod.UserManagement.GetUserToken(nickname)
		if err != nil {
			log.Println("ChatServerModule: ExecuteSendMessage: Invalid token or action not allowed")
			return
		}

		if !secMod.ValidateTokenAction(token_user, nickname, "sendMessage") {
			log.Println("ChatServerModule: ExecuteSendMessage: Invalid token or action not allowed")
			return
		}

		if token_user != token {
			log.Println("ChatServerModule: ExecuteSendMessage: Token mismatch")
			return
		}
		_, err = secMod.UserManagement.FindUserByToken(token_user)
		if err != nil {
			log.Printf("ChatServerModule: ExecuteSendMessage: CODM03: Error finding user by token: %v\n", err)
			return
		}

		// Validar RoomId
		_, err = uuid.Parse(requestData.RoomId.String())
		if err != nil {

			sendErrorResponse(secMod, requestData.Topic, "RoomId inválido.", requestData.X_GoChat)

			return
		}

		xGoChat := requestData.X_GoChat
		if xGoChat == "" {
			sendErrorResponse(secMod, requestData.Topic, "El campo x-gochat no existe.", "")
			return
		}

		log.Printf("HandleGetMessage: Llamando al servicio para obtener mensajes desde el IdUltimoMensaje:[%s].", requestData.LastMessageId)
		_, err = uuid.Parse(requestData.LastMessageId.String())
		if err != nil {
			log.Printf("PostListHandler: Error al validar el idmensaje: %v", err)
			sendErrorResponse(secMod, requestData.Topic, "idmensaje no es un UUID válido", requestData.X_GoChat)
			return
		}

		mensajes, err := secMod.RoomManagement.GetMessagesFromId(requestData.RoomId, requestData.LastMessageId)
		if err != nil {
			log.Printf("HandleGetMessage: Error al obtener los mensajes: %v", err)
			if strings.Contains(err.Error(), "no se encontraron mensajes en la sala") {
				respChan <- entities.ResponseListMessages{
					Status:      "OK",
					Message:     "No se encontraron mensajes en la sala",
					ListMessage: nil,
				}
				return
			}
			sendErrorResponse(secMod, requestData.Topic, fmt.Sprintf("Error al obtener los mensajes: %v", err), requestData.X_GoChat)
			return
		}

		// Eliminar el mensaje con el ID igual al de LastMessageId
		for i := 0; i < len(mensajes); {
			if mensajes[i].MessageId.String() == requestData.LastMessageId.String() {
				mensajes = append(mensajes[:i], mensajes[i+1:]...)
			} else {
				i++
			}
		}

		if len(mensajes) == 0 {
			log.Println("HandleGetMessage: No hay mensajes nuevos.")
			respChan <- entities.ResponseListMessages{
				Status:  "OK",
				Message: "No hay mensajes nuevos.",
			}

			return
		}

		// Convertir los mensajes a la respuesta adecuada
		mensajesResponse := ConvertirMensajes(mensajes)

		// Enviar la respuesta a través del canal
		respChan <- entities.ResponseListMessages{
			Status:      "OK",
			Message:     "Mensajes obtenidos correctamente.",
			TokenSesion: requestData.TokenSesion,
			Nickname:    requestData.Nickname,
			RoomId:      requestData.RoomId,
			X_GoChat:    requestData.X_GoChat,
			ListMessage: mensajesResponse,
		}
	}()

	// Esperar la respuesta del canal
	response := <-respChan

	err = secMod.RoomManagement.MainRoom.Room.MessageBroker.PublishGetMessages(requestData.Topic, &response)
	if err != nil {
		log.Printf("HandleGetUsersMessage: Error al publicar la respuesta: %v", err)
	} else {
		log.Printf("HandleGetUsersMessage: Respuesta enviada al topic %s ", requestData.Topic)
	}
}
