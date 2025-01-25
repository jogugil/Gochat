package main

import (
	"backend/api"
	"backend/persistence"
	"backend/services"
	"backend/utils"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type FilteredWriter struct {
	allowedClasses []string
	writer         io.Writer
}

func (f *FilteredWriter) Write(p []byte) (n int, err error) {
	message := string(p)
	for _, class := range f.allowedClasses {
		if strings.Contains(message, class) {
			return f.writer.Write(p)
		}
	}
	return len(p), nil // Ignore the log if it doesn't match allowed classes
}
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	utils.LoadEnvironmentVariables()

	filter := &FilteredWriter{
		allowedClasses: []string{"Main", "BrokerNats", "NatsTransformer", "RoomManagement", "ChatServerModule",
			"ChatServerModule", "BrokerKafka", "KafkaTransformer", "UserManagement", "LoginHandler", "LocalRoom", "HandleNewMessages", "HandleGetUsersMessage", "NatsTransformer"},
		writer: io.Discard, // Initially don't log anywhere
	}
	log.SetOutput(filter) // Redirect log output to our custom filter

	consoleWriter := io.Writer(os.Stdout)
	filter.writer = consoleWriter

	// Cargar variables de entorno
	ginMode, err := utils.GetEnvVariable("GIN_MODE")
	if err != nil {
		ginMode = "debug"
	}
	if ginMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	log.Printf("Main: GIN_MODE configurado como: %s\n", ginMode)
	// Configurar la base de datos
	uriMongo, err := utils.GetEnvVariable("URIMongo")
	if err != nil {
		log.Printf("Main: Error cargando URIMongo, usando valor predeterminado: %v\n", err)
		uriMongo = "mongodb://localhost:27017"
	}

	nameMongo, err := utils.GetEnvVariable("NameMongo")
	if err != nil {
		log.Printf("Main: Error cargando NameMongo, usando valor predeterminado: %v\n", err)
		nameMongo = "MongoChat"
	}

	persistence, err := persistence.NewMongoPersistence(uriMongo, nameMongo)
	if err != nil {
		log.Fatalf("Main: Error inicializando MongoPersistence: %v\n", err)
	}

	configFile, err := utils.GetEnvVariable("GOCHAT_CONFIG_FILE")
	if err != nil {
		log.Printf("Main: No se encontró GOCHAT_CONFIG_FILE, usando 'gochat.json' por defecto: %v\n", err)
		configFile = "gochat.json"
	}
	log.Printf("Main: Cargando archivo de configuración: %s\n", configFile)
	secMod := services.CreateChatServerModule(persistence, configFile)
	roomManager := secMod.RoomManagement

	log.Printf("Main: roomManager.MainRoom.Room.RoomName: %s, ID: %s\n", roomManager.MainRoom.Room.RoomName, roomManager.MainRoom.Room.RoomId)

	for id, room := range roomManager.FixedRooms {
		log.Printf("Main: Sala fija: %s, ID: %s\n", room.Room.RoomName, id)
	}

	// Crear un enrutador de Gin
	r := gin.Default()

	// Configuración CORS: permitir todos los orígenes en desarrollo.
	allowedOrigins, _ := utils.GetEnvVariable("ALLOWED_ORIGINS")

	if allowedOrigins == "" {
		log.Println("ALLOWED_ORIGINS no está configurado, usando valores por defecto.")
		allowedOrigins = "http://localhost:5174,http://localhost:5173" // Valores por defecto para desarrollo
	}
	// Dividir los orígenes permitidos en un slice
	allowedOriginsList := strings.Split(allowedOrigins, ",")

	// Permitir cualquier origen con 'localhost' en el dominio, sin importar el puerto
	// Esto es para permitir cualquier puerto en localhost
	var filteredOrigins []string
	for _, origin := range allowedOriginsList {
		parsedOrigin, err := url.Parse(origin)
		if err == nil && parsedOrigin.Hostname() == "localhost" {
			filteredOrigins = append(filteredOrigins, origin)
		}
	}
	// En producción, sóslo restringir a dominios específicos (allowedOrigins)
	if env := os.Getenv("ENV"); env == "production" {
		// Restringir solo a orígenes específicos para producción
		filteredOrigins = []string{
			"http://localhost:5174",
		}
	}
	log.Printf("Main: ALLOWED_ORIGINS configurado como: %s\n", allowedOrigins)
	log.Printf("Main: ALLOWED_ORIGINS configurado como: %s\n", allowedOriginsList)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOriginsList, // --123-- ojo!! Debemos Cambiar  el * esto en producción para especificar orígenes. MEor mediante variabl entorno.
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "x_gochat"},
		AllowCredentials: true,
	}))

	// Middleware para verificar el encabezado x_gochat
	r.Use(func(c *gin.Context) {
		// Excluir ciertas rutas del chequeo (por ejemplo: /status y /metrics)
		if c.Request.URL.Path == "/status" || c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		// Ver todas las cabeceras que llegan
		log.Printf("Main: Cabeceras recibidas: %+v\n", c.Request.Header)

		// Verifica sMAini la cabecera x_gochat está presente
		goChatHeader := c.GetHeader("x_gochat")
		if goChatHeader == "" {
			log.Printf("Main: Cabecera 'x_gochat' no encontrada\n")
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "nok",
				"message": "Cliente no aceptado: Falta 'x_gochat' en el encabezado",
			})
			c.Abort()
			return
		}

		log.Printf("Main: Servidor: goChatHeader recibido: %s\n", goChatHeader)
		c.Next()
	})

	// Crear instancias de los servicios
	statusService := api.NewStatusService()
	metricsService := api.NewMetricsService()
	// Ruta prueba conexión xon el servidor Gochat
	// Definir la ruta para la URL raíz '/'
	r.GET("/", func(c *gin.Context) {
		// Responder con un mensaje JSON
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	// Rutas para monitoreo o métricas
	r.GET("/status", statusService.Status)
	r.GET("/metrics", metricsService.Metrics)

	// Rutas de login y logout
	r.POST("/login", func(c *gin.Context) {
		// Verificar que la solicitud de login está llegando
		log.Printf("Main: Solicitud de login recibida en /login\n")
		log.Printf("Main: Cabeceras recibidas: %+v\n", c.Request.Header)
		// Llamar al controlador de login
		api.LoginHandler(c)
	})
	r.POST("/logout", api.LogoutHandler)

	// Configurar el servidor
	server, err := utils.GetEnvVariable("NameServer")
	if err != nil {
		log.Printf("Main: Error cargando NameServer, usando 'localhost' por defecto: %v\n", err)
		server = "localhost"
	}

	port, err := utils.GetEnvVariable("PortServer")
	if err != nil {
		log.Printf("Main: Error cargando PortServer, usando '8081' por defecto: %v\n", err)
		port = "8081"
	}
	log.Printf("Main: Configuración del servidor - NameServer: %s, PortServer: %s\n", server, port)
	address := fmt.Sprintf("%s:%s", server, port)
	log.Printf("Main: Servidor escuchando en %s\n", address)

	// Iniciar servidor HTTP
	if err := r.Run(address); err != nil {
		log.Fatalf("Error al iniciar el servidor HTTP: %v\n", err)
	}
}
