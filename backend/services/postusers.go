package services

import (
	"backend/models"
	"encoding/json"
	"log"

	"github.com/google/uuid"
)

// Estructura para los usuarios activos
type AliveUsers struct {
	Nickname       string `json:"nickname"`
	LastActionTime string `json:"lastactiontime"`
}

// Estructura para la respuesta general
type ResponseUser struct {
	Status      string       `json:"status"`
	Message     string       `json:"message"`
	TokenSesion string       `json:"tokenSesion"`
	Nickname    string       `json:"nickname"`
	RoomId      string       `json:"roomId"`
	X_GoChat    string       `json:"x_gochat"`
	AliveUsers  []AliveUsers `json:"data,omitempty"`
}

func PostUsersHandler(msg []byte, urlClient string) []byte {
	var requestData struct {
		RoomId      string `json:"roomid"`
		TokenSesion string `json:"tokensesion"`
		Nickname    string `json:"nickname"`
		X_GoChat    string `json:"x_gochat"`
	}

	// Crear un canal para pasar la respuesta
	respChan := make(chan ResponseUser)

	// Validar el token de sesión en una goroutine para no bloquear la respuesta
	go func() {
		// Decodificar la solicitud
		if err := json.Unmarshal(msg, &requestData); err != nil {
			log.Printf("PostUsersHandler: Error al decodificar la solicitud: %v", err)
			respChan <- ResponseUser{
				Status:  "NOK",
				Message: "Solicitud incorrecta. Error al decodificar la solicitud cod:00",
			}
			return
		}
		log.Printf("PostUsersHandler: Datos de solicitud decodificados: %+v\n", requestData)

		// Obtener la instancia del singleton
		secMod, err := GetChatServerModule()
		if err != nil {
			log.Printf("PostUsersHandler: Error al obtener el servidor chat. : %v", err)
			respChan <- ResponseUser{
				Status:   "NOK",
				Message:  "Servicio chat no disponible",
				X_GoChat: urlClient,
			}
			return
		}
		log.Printf("PostUsersHandler: Singleton de servidor de chat obtenido: %v", secMod)

		// Validar IdSala
		idSala, err := uuid.Parse(requestData.RoomId)
		if err != nil {
			log.Printf("Error al parsear IdSala: %v", err)
			respChan <- ResponseUser{
				Status:   "NOK",
				Message:  "Error al parsear IdSala cod:00",
				X_GoChat: urlClient,
			}
			return
		}
		log.Printf("PostUsersHandler: IdSala validado correctamente: %v", idSala)
		log.Println("PostUsersHandler:  Validando token de sesión.")
		_, err = models.ValidateSessionToken(requestData.TokenSesion)
		if err != nil {
			log.Printf("PostUsersHandler: Error al validar el token: %v", err)
			respChan <- ResponseUser{
				Status:   "NOK",
				Message:  "Sesión de usuario inválida, por favor inicie sesión nuevamente.",
				X_GoChat: urlClient,
			}
			return
		}

		// Obtener usuarios activos
		usuarios := secMod.UserManagement.GetActiveUsers()
		if usuarios == nil {
			respChan <- ResponseUser{
				Status:   "NOK",
				Message:  "Error al obtener usuarios activos",
				X_GoChat: urlClient,
			}

			return
		}
		log.Printf("PostUsersHandler: Usuarios activos obtenidos para la sala %v: %v", idSala, usuarios)

		// Construir la respuesta con la información de la sala y los usuarios activos
		var aliveUsers []AliveUsers
		for _, usuario := range usuarios {
			aliveUsers = append(aliveUsers, AliveUsers{
				Nickname:       usuario.Nickname,
				LastActionTime: usuario.LastActionTime.Format("2006-01-02 15:04:05"), // Formato estándar de fecha y hora
			})
		}

		// Preparar la respuesta
		respChan <- ResponseUser{
			Status:      "OK",
			Message:     "Usuarios activos obtenidos correctamente.",
			TokenSesion: requestData.TokenSesion,
			Nickname:    requestData.Nickname,
			RoomId:      requestData.RoomId,
			AliveUsers:  aliveUsers,
			X_GoChat:    urlClient,
		}
	}()

	// Esperar la respuesta del canal
	response := <-respChan

	// Serializar la respuesta a JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Printf("PostUsersHandler: Error al serializar la respuesta: %v", err)
		// Manejo de error de serialización
		errorResponse := ResponseUser{
			Status:  "NOK",
			Message: "Error interno del servidor. No se pudo procesar la respuesta.",
		}
		responseJSON, err = json.Marshal(errorResponse)
		if err != nil {
			log.Printf("PostUsersHandler: Error al serializar la respuesta de error: %v", err)
		}
	}
	log.Printf("PostUsersHandler: Usuarios activos enviando el mensaje: %s", responseJSON)
	// Retornar la respuesta
	return responseJSON
}
