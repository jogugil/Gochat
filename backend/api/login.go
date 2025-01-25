package api

import (
	"backend/services"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func LoginHandler(c *gin.Context) {
	// Definir estructura para recibir datos de la solicitud
	var requestData struct {
		Nickname string `json:"nickname"`
	}

	// Log de entrada de la solicitud
	log.Printf("LoginHandler: Recibiendo datos de solicitud...[%v]\n", requestData)

	// Decodificar los datos JSON de la solicitud
	if err := c.ShouldBindJSON(&requestData); err != nil {
		// Log de error en la decodificación de datos
		log.Printf("LoginHandler: Error al decodificar el JSON de la solicitud: %v\n", err)

		// Si hay un error en el body de la solicitud, devolver un error HTTP 400
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "nok",
			"message": err.Error(),
		})
		return
	}

	// Log de los datos recibidos
	log.Printf("LoginHandler: Datos recibidos de la solicitud:[%v]\n", requestData)

	// Obtener la instancia del singleton
	secMod, err := services.GetChatServerModule()
	if err != nil {
		// Si el RoomId no es un UUID válido, devolver un error
		log.Printf("LoginHandler: Error al obtener el servidor chat. : %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "nok",
			"message": "Servicio  chat no disponible",
		})
		return
	}
	// Llamar a EjecutarLogin con el nickname recibido
	log.Printf("LoginHandler: Ejecutando login para el usuario:[%s]\n", requestData.Nickname)
	usuario, err := secMod.ExecuteLogin(requestData.Nickname)

	if err != nil {
		// Log del error en el proceso de login
		log.Printf("LoginHandler: Error al ejecutar login para el usuario: %v\n", err)

		// Crear un objeto de respuesta con campos vacíos
		responseData := gin.H{
			"status":   "",
			"message":  "",
			"token":    "",
			"nickname": "",
			"roomId":   "",
			"roomName": "",
		}

		// Verificar si el error contiene el código específico de nickname en uso
		if strings.Contains(err.Error(), "CODL00") {
			// Actualizar el objeto de respuesta para el caso de nickname en uso
			responseData["status"] = "nok"
			responseData["message"] = "El nickname ya está en uso"
			c.JSON(http.StatusOK, responseData)
		} else {
			// Actualizar el objeto de respuesta para errores genéricos
			responseData["status"] = "nok"
			responseData["message"] = err.Error()
			c.JSON(http.StatusBadRequest, responseData)
		}

		return
	}

	// Log de los datos del usuario después del login
	log.Printf("LoginHandler: Login exitoso. Datos del usuario: Token: %s, Nickname: %s, Sala ID: %s, Sala Name: %s\n",
		usuario.Token, usuario.Nickname, usuario.RoomId.String(), usuario.RoomName)

	// Responder con un JSON de éxito si el login es exitoso
	responseData := gin.H{
		"status":   "ok",
		"message":  "login realizado",
		"token":    usuario.Token,
		"nickname": usuario.Nickname,
		"roomid":   usuario.RoomId,   // Sala por defecto
		"roomname": usuario.RoomName, // Nombre de la sala
	}

	// Log de la respuesta enviada
	log.Println("LoginHandler: Enviando respuesta:", responseData)

	c.JSON(http.StatusOK, responseData)
}
