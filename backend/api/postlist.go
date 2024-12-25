package api

import (
	"backend/entities"
	"backend/models"
	"backend/services"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

// Estructura para recibir los datos de la solicitud
type RequestData struct {
	Operation     string `json:"operation"`
	LastMessageId string `json:"lastmessageid"`
	TokenSesion   string `json:"tokensesion"`
	Nickname      string `json:"nickname"`
	RoomId        string `json:"roomid"`
	X_GoChat      string `json:"x_gochat"`
}

type MessageResponse struct {
	MessageId   uuid.UUID `json:"messageid"`
	Nickname    string    `json:"nickname"`
	MessageText string    `json:"messagetext"`
}

// Estructura para la respuesta
type Response struct {
	Status      string            `json:"status"`
	Message     string            `json:"message"`
	TokenSesion string            `json:"tokenSesion"`
	Nickname    string            `json:"nickname"`
	RoomId      string            `json:"roomid"`
	X_GoChat    string            `json:"x_gochat"`
	ListMessage []MessageResponse `json:"data,omitempty"` // Lista de mensajes si existen
}

// Convertir los mensajes a la estructura de respuesta
func ConvertirMensajes(mensajes []entities.Message) []MessageResponse {
	log.Println("ConvertirMensajes: Iniciando conversión de mensajes.")
	var respuesta []MessageResponse
	for _, mensaje := range mensajes {
		respuesta = append(respuesta, MessageResponse{
			MessageId:   mensaje.MessageId,
			Nickname:    mensaje.Nickname,
			MessageText: mensaje.MessageText,
		})
	}
	log.Printf("ConvertirMensajes: Se han convertido %d mensajes.\n", len(respuesta))
	return respuesta
}

// Manejador para la ruta POST /messagelist
func PostListHandler(msg []byte, urlClient string) []byte {
	log.Println("PostListHandler: Iniciando el manejo de la solicitud POST para la lista de mensajes.")

	// Decodificar el cuerpo de la solicitud
	var requestData RequestData

	// Crear un canal para pasar la respuesta
	respChan := make(chan Response)

	// Ejecutar el manejo de la solicitud en una goroutine
	go func() {
		err := json.Unmarshal(msg, &requestData)
		log.Printf("PostListHandler: Datos de solicitud decodificados: %+v\n", requestData)
		if err != nil {
			log.Printf("PostListHandler: Error al decodificar los datos: %v", err)
			// Enviar error por WebSocket
			respChan <- Response{
				Status:  "NOK",
				Message: "Solicitud incorrecta. cod:00",
			}
			return
		}
		// Obtener la instancia del singleton
		secMod, err := services.GetSecModServidorChat()
		if err != nil {
			// Si el IdSala no es un UUID válido, devolver un error
			log.Printf("Error al obtener el servidor chat. : %v", err)
			respChan <- Response{
				Status:  "NOK",
				Message: "Servicio  chat no disponible",
			}
			return
		}
		log.Println("PostListHandler: Obteniendo usuario por token de sesión.")
		user, err := secMod.GestionUsuarios.BuscarUsuarioPorToken(requestData.TokenSesion)
		if err != nil {
			log.Printf("PostListHandler: Error al buscar el usuario por token: %v", err)
			respChan <- Response{
				Status:  "NOK",
				Message: "Sesión de usuario inválida, por favor inicie sesión nuevamente. cod:01",
			}
			return
		}

		log.Println("PostListHandler: Validando token de sesión.")
		_, err = models.ValidarTokenSesion(requestData.TokenSesion)
		if err != nil {
			log.Printf("PostListHandler: Error al validar el token: %v", err)
			respChan <- Response{
				Status:  "NOK",
				Message: "Sesión de usuario inválida, por favor inicie sesión nuevamente. cod:02",
			}
			return
		}

		tokenUser := user.Token
		if tokenUser != requestData.TokenSesion {
			log.Println("PostListHandler: Error de validación de token: tokenUser != requestData.TokenSesion")
			respChan <- Response{
				Status:  "NOK",
				Message: "Sesión de usuario inválida, por favor inicie sesión nuevamente. cod:03",
			}
			return
		}

		log.Println("PostListHandler: Validando ID de sala.")
		idSalaUUID, err := uuid.Parse(requestData.RoomId)
		if err != nil {
			log.Printf("PostListHandler: Error al validar el IdSala: %v", err)
			respChan <- Response{
				Status:  "NOK",
				Message: "Sala de chat inválida, por favor inicie sesión nuevamente. cod:04",
			}
			return
		}

		log.Printf("PostListHandler: Llamando al servicio para obtener mensajes desde el IdUltimoMensaje:[%s].", requestData.LastMessageId)
		idmensaje, err := uuid.Parse(requestData.LastMessageId)
		if err != nil {
			log.Printf("PostListHandler: Error al validar el idmensaje: %v", err)
			respChan <- Response{
				Status:  "NOK",
				Message: "idmensaje no es un UUID válido",
			}
			return
		}

		mensajes, err := secMod.GestionSalas.ObtenerMensajesDesdeId(idSalaUUID, idmensaje)
		if err != nil {
			// Verificar si el error contiene el mensaje "no se encontraron mensajes en la sala
			log.Printf("PostListHandler:   %v", err)
			if strings.Contains(err.Error(), "no se encontraron mensajes en la sala") {
				// Si el error contiene esa frase, respondemos con OK y un mensaje adecuado
				respChan <- Response{
					Status:      "OK",
					Message:     "No se encontraron mensajes en la sala",
					ListMessage: nil, // Devolvemos null o una lista vacía si lo prefieres
				}
				return
			}

			// Si el error no contiene la frase esperada, procesamos como un error general
			log.Printf("PostListHandler: Error al obtener los mensajes: %v", err)
			respChan <- Response{
				Status:  "NOK",
				Message: fmt.Sprintf("Error al obtener los mensajes: %v", err),
			}
			return
		}
		// Mostrar los mensajes por pantalla
		log.Printf("PostListHandler: Contenido de mensajes: %+v", mensajes)
		// Eliminar el mensaje con el ID igual al de LastMessageId

		for i := 0; i < len(mensajes); {
			if mensajes[i].MessageId.String() == requestData.LastMessageId {
				// Eliminar mensaje del slice
				mensajes = append(mensajes[:i], mensajes[i+1:]...)
				log.Printf(" --> mensajes[%d]: %+v", i, mensajes)
			} else {
				i++
			}
		}

		// Si no hay mensajes nuevos, devolver solo OK
		if len(mensajes) == 0 {
			log.Println("PostListHandler: No hay mensajes nuevos.")
			respChan <- Response{
				Status:  "OK",
				Message: "No hay mensajes nuevos.",
			}
			return
		}

		// Convertir los mensajes a la respuesta adecuada
		log.Println("PostListHandler: Convertiendo mensajes a la estructura de respuesta.")
		mensajesResponse := ConvertirMensajes(mensajes)

		// Si hay mensajes, devolver la lista con un OK
		log.Println("PostListHandler: Devolviendo mensajes obtenidos correctamente.")
		log.Printf("PostListHandler: mensajesResponse: %s", mensajesResponse)
		respChan <- Response{
			Status:      "OK",
			Message:     "Mensajes obtenidos correctamente.",
			TokenSesion: requestData.TokenSesion,
			Nickname:    requestData.Nickname,
			RoomId:      requestData.RoomId,
			X_GoChat:    urlClient,
			ListMessage: mensajesResponse,
		}
	}()

	// Esperar la respuesta del canal
	response := <-respChan

	// Serializar la respuesta a JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Printf("PostListHandler: Error al serializar la respuesta: %v", err)
		respChan <- Response{
			Status:  "NOK",
			Message: "Error interno del servidor. No hay mensajes nuevos.PostListHandler: Error al serializar la respuesta",
		}
		response := <-respChan
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Printf("PostListHandler: Error al serializar la respuesta: %v", err)
		}
		return responseJSON
	}

	// Enviar la respuesta por WebSocket
	log.Printf("Lista mensaje activos. enviando el mensaje: %s", responseJSON)
	return responseJSON
}
