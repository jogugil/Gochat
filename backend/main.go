package main

import (
	"backend/api"
	"backend/persistence"
	"backend/services"
	"backend/utils"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

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
	log.SetFlags(log.Lshortfile)
	utils.LoadEnvironmentVariables()

	filter := &FilteredWriter{
		allowedClasses: []string{"Main", "PostListHandler", "WebSocketHandler", "RoomManagement", "NewMessageHandler",
			"ChatServerModule", "LocalRoom", "RoomManagement"},
		writer: io.Discard, // Initially don't log anywhere
	}

	log.SetOutput(filter) // Redirect log output to our custom filter

	ginMode, err := utils.GetEnvVariable("GIN_MODE")
	if err != nil {
		ginMode = "debug"
	}

	if ginMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	consoleWriter := io.Writer(os.Stdout)
	filter.writer = consoleWriter
	// Cargar variables de entorno
	utils.LoadEnvironmentVariables()

	// Establecer el modo de Gin
	ginMode, err_v := utils.GetEnvVariable("GIN_MODE")
	if err_v != nil {
		log.Printf("Main: No se pudo cargar GIN_MODE, usando 'debug' por defecto: %v", err_v)
		ginMode = "debug"
	}

	if ginMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Configurar la base de datos
	uriMongo, err := utils.GetEnvVariable("URIMongo")
	if err != nil {
		log.Printf("Main: Error cargando URIMongo, usando valor predeterminado: %v", err)
		uriMongo = "mongodb://localhost:27017"
	}

	nameMongo, err := utils.GetEnvVariable("NameMongo")
	if err != nil {
		log.Printf("Main: Error cargando NameMongo, usando valor predeterminado: %v", err)
		nameMongo = "MongoChat"
	}

	persistence, err := persistence.NewMongoPersistence(uriMongo, nameMongo)
	if err != nil {
		log.Fatalf("Main: Error inicializando MongoPersistence: %v", err)
	}

	// Leer el nombre del archivo de configuración desde la variable de entorno
	configFile, err := utils.GetEnvVariable("GOCHAT_CONFIG_FILE")
	if err != nil {
		log.Printf("Main: No se encontró GOCHAT_CONFIG_FILE, usando 'gochat.json' por defecto: %v", err)
		configFile = "gochat.json"
	}
	log.Printf("Main:  Cargando archivo de configuración: %s", configFile)
	secMod := services.CreateChatServerModule(persistence, configFile)
	roomManager := secMod.RoomManagement

	log.Printf("Sala principal: %s, ID: %s\n", roomManager.MainRoom.Room.RoomName, roomManager.MainRoom.Room.RoomId)

	for id, room := range roomManager.FixedRooms {
		log.Printf("Main: Sala fija: %s, ID: %s\n", room.RoomName, id)
	}

	// Crear un enrutador de Gin
	r := gin.Default()

	// Crear instancias de los servicios
	statusService := api.NewStatusService()
	metricsService := api.NewMetricsService()

	// Rutas para monitoreo o métricas
	r.GET("/status", statusService.Status)
	r.GET("/metrics", metricsService.Metrics)
	// Rutas de login y logout
	r.POST("/login", api.LoginHandler)   // Rutas de Login
	r.POST("/logout", api.LogoutHandler) // Rutas de Logout
	// Rutas de chat
	// Configurar el servidor
	server, err := utils.GetEnvVariable("NameServer")
	if err != nil {
		log.Printf("Main: Error cargando NameServer, usando 'localhost' por defecto: %v", err)
		server = "localhost"
	}

	port, err := utils.GetEnvVariable("PortServer")
	if err != nil {
		log.Printf("Main: Error cargando PortServer, usando '8081' por defecto: %v", err)
		port = "8081"
	}

	address := fmt.Sprintf("%s:%s", server, port)
	log.Printf("Main: Servidor escuchando en %s", address)

	// Iniciar servidor HTTP
	if err := r.Run(address); err != nil {
		log.Fatalf("Error al iniciar el servidor HTTP: %v", err)
	}
}
