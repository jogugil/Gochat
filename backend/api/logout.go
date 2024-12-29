package api

import (
	"backend/services"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// LogoutHandler maneja la solicitud de logout
func LogoutHandler(c *gin.Context) {
	// Definir estructura para recibir datos de la solicitud
	var requestData struct {
		Nickname string `json:"nickname"`
		Token    string `json:"token"`
		RoomId   string `json:"roomid"`
		RoomName string `json:"roomname"`
	}

	// Log de entrada de la solicitud
	log.Println("Recibiendo datos de solicitud de logout...")

	// Decodificar los datos JSON de la solicitud
	if err := c.ShouldBindJSON(&requestData); err != nil {
		// Log de error en la decodificación de datos
		log.Printf("Error al decodificar el JSON de la solicitud: %v", err)

		// Si hay un error en el body de la solicitud, devolver un error HTTP 400
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "nok",
			"message": err.Error(),
		})
		return
	}

	// Log de los datos recibidos
	log.Println("Datos recibidos de la solicitud de logout:", requestData)

	// Obtener la instancia del servidor de chat
	secMod, err := services.GetChatServerModule()
	if err != nil {
		// Si no se puede obtener el servicio, devolver error
		log.Printf("Error al obtener el servidor chat. : %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "nok",
			"message": "Servicio de chat no disponible",
		})
		return
	}

	// Llamar a EjecutarLogout con los datos recibidos
	log.Printf("Ejecutando logout para el usuario: %s en la sala %s", requestData.Nickname, requestData.RoomName)
	err = secMod.ExecuteLogout(requestData.Token)
	if err != nil {
		// Log del error en el proceso de logout
		log.Printf("Error al ejecutar logout para el usuario: %v", err)

		// Responder con el error adecuado
		responseData := gin.H{
			"status":  "nok",
			"message": err.Error(),
		}
		c.JSON(http.StatusBadRequest, responseData)
		return
	}

	// Log de la acción de logout exitosa
	log.Printf("Logout exitoso para el usuario: %s en la sala: %s", requestData.Nickname, requestData.RoomName)

	// Responder con un JSON de éxito si el logout es exitoso
	responseData := gin.H{
		"status":   "ok",
		"message":  "logout realizado con éxito",
		"nickname": requestData.Nickname,
		"idsale":   requestData.RoomId,
		"roomname": requestData.RoomName,
	}

	// Log de la respuesta enviada
	log.Println("Enviando respuesta de logout:", responseData)

	c.JSON(http.StatusOK, responseData)
}
