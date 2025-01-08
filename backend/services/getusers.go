package services

import (
	"backend/entities"
	"log"

	"github.com/google/uuid"
)

func HandleGetUsersMessage(msg interface{}) {
	// Decodificar mensaje entrante
	requestData, ok := msg.(*entities.RequestListUsers)
	if !ok {
		log.Println("HandleGetUsersMessage: Invalid message type received")

	}

	nickname := requestData.Nickname
	token := requestData.TokenSesion

	log.Printf("HandleGetUsersMessage: Datos recibidos: %+v - nickname:[%s]\n", requestData, nickname)
	secMod, err := GetChatServerModule()
	if err != nil {
		log.Printf("HandleGetUsersMessage: Error al obtener el modulo de chat: %v", err)
		return
	}

	// Validate if the user is authorized to send the message
	token_user, err := secMod.UserManagement.GetUserToken(nickname)
	if err != nil {
		log.Println("HandleGetUsersMessage: Invalid token or action not allowed")
		return
	}

	if !secMod.ValidateTokenAction(token_user, nickname, "sendMessage") {
		log.Println("HandleGetUsersMessage: Invalid token or action not allowed")
		return
	}

	if token_user != token {
		log.Println("HandleGetUsersMessage: Token mismatch")
		return
	}
	_, err = secMod.UserManagement.FindUserByToken(token_user)
	if err != nil {
		log.Printf("HandleGetUsersMessage:  Error finding user by token: %v\n", err)
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
		sendErrorResponse(secMod, requestData.Topic, "El campo x_gochat no existe.", "")
		return
	}

	users := secMod.UserManagement.GetActiveUsers()
	if users == nil {
		sendErrorResponse(secMod, requestData.Topic, "No se encontraron usuarios activos.", requestData.X_GoChat)
		return
	}

	// Preparar lista de usuarios activos
	var aliveUsers []entities.AliveUsers
	for _, usuario := range users {
		aliveUsers = append(aliveUsers, entities.AliveUsers{
			Nickname:       usuario.Nickname,
			LastActionTime: usuario.LastActionTime.Format("2006-01-02 15:04:05"),
		})
	}

	// Crear respuesta
	response := entities.ResponseListUser{
		Status:      "OK",
		Message:     "Usuarios activos obtenidos.",
		TokenSesion: requestData.TokenSesion,
		Nickname:    requestData.Nickname,
		RoomId:      requestData.RoomId,
		AliveUsers:  aliveUsers,
		X_GoChat:    requestData.X_GoChat,
	}

	err = secMod.RoomManagement.MainRoom.Room.MessageBroker.PublishGetUsers(requestData.Topic, &response)
	if err != nil {
		log.Printf("HandleGetUsersMessage: Error al publicar la respuesta: %v", err)
	} else {
		log.Printf("HandleGetUsersMessage: Respuesta enviada al topic %s ", requestData.Topic)
	}
}

func sendErrorResponse(secMod *ChatServerModule, topic, message, xGoChat string) {
	responseE := entities.ResponseListUser{
		Status:   "NOK",
		Message:  message,
		X_GoChat: xGoChat,
	}

	err := secMod.RoomManagement.MainRoom.Room.MessageBroker.PublishGetUsers(topic, &responseE)
	if err != nil {
		log.Printf("HandleGetUsersMessage: sendErrorResponse: Error al publicar la respuesta de error: %v", err)
	}
}
