package comm

import (
	"backend/api"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Manejador para WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Permitir conexiones de cualquier origen
	},
}

func WebSocketHandler(c *gin.Context) {
	// Establecer la conexión WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocketHandler: Error al establecer WebSocket:", err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Println("WebSocketHandler: Error al cerrar la conexión WebSocket:", err)
		} else {
			log.Println("WebSocketHandler: Conexión WebSocket cerrada correctamente")
		}
	}()
	// Obtener la URL de la solicitud HTTP original
	urlCliente := c.Request.URL.String()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocketHandler: Error al leer mensaje WebSocket:%v", err)
			log.Printf("WebSocketHandler: Error al ReadMessage:%v", msg)

			break
		}
		trimmedMsg := strings.TrimSpace(string(msg))
		if len(trimmedMsg) == 0 {
			log.Println("WebSocketHandler: Mensaje vacío o solo con espacios recibido, ignorando.")
			continue
		}
		if len(msg) == 0 {
			log.Println("WebSocketHandler: Mensaje vacío recibido, ignorando.")
			continue
		}
		var data map[string]string
		err = json.Unmarshal(msg, &data)
		if err != nil {
			log.Printf("WebSocketHandler: Error al deserializar el mensaje:%v", err)
			msgr, _ := json.Marshal(`{"status": "error", "message": "Error al deserializar el mensaje"}`)
			conn.WriteMessage(websocket.TextMessage, msgr)
			continue
		}

		operacion, exists := data["operation"]
		if !exists {
			log.Println("WebSocketHandler: No se encontró la clave 'operation' en el mensaje")
			msgr, _ := json.Marshal(`{"status": "error", "message": "No se encontró la clave 'operation' en el mensaje"}`)
			conn.WriteMessage(websocket.TextMessage, msgr)
			continue
		}

		var response []byte
		switch operacion {
		case "listmenssage":
			response = callPostListHandler(msg, urlCliente)
		case "listusers":
			response = callPostUsersHandler(msg, urlCliente)
		case "heat":
			response = callheatHandler(msg, urlCliente)
		default:
			response, _ = json.Marshal(`{"status": "error", "message": "Comando no reconocido"}`)
		}

		err = conn.WriteMessage(websocket.TextMessage, response)
		if err != nil {
			log.Println("WebSocketHandler: Error al enviar mensaje WebSocket:", err)
		}
	}
}

// Aquí solo retornas una cadena de texto (JSON) que representa la respuesta
func callPostListHandler(msg []byte, urlClient string) []byte {
	// Llamamos a la función correspondiente del API
	//log.Printf("callPostListHandler: %s", string(msg))
	result := api.PostListHandler(msg, urlClient) // Llamar a la función de API pasando la request
	return result
}

func callPostUsersHandler(msg []byte, urlClient string) []byte {
	// Llamamos a la función correspondiente del API
	//log.Printf("callPostUsersHandler: %v", msg)
	result := api.PostUsersHandler(msg, urlClient) // Llamar a la función de API pasando la request
	return result
}

// Tu función que maneja el "heat" y devuelve el código 202
func callheatHandler(msg []byte, urlClient string) []byte {
	log.Printf("WebSocketHandler: callheatHandler procesando solicitud: %v", msg)
	// Si todo va bien, se establece el código de estado 202 y se envía la respuesta
	errorJSON, err := json.Marshal(map[string]interface{}{
		"status":   "OK",
		"x_gochat": urlClient, // Asignamos directamente la variable
		"message":  "Request accepted for processing",
	})
	if err != nil {
		log.Printf("WebSocketHandler: sendErrorResponse: Error al serializar el mensaje de error:%v", err)
		return errorJSON
	}
	return errorJSON
}
