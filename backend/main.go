package main

import (
	"backend/api"
	"backend/comm"
	"backend/persistence"
	"backend/services"
	"backend/utils"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	_ "net/http/pprof" // Activar pprof

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Para manejar las conexiones WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Asegúrate de que esta configuración se adapte a tus necesidades de seguridad
		return true
	},
}

// FilteredWriter es un tipo que implementa io.Writer y filtra los logs.
type FilteredWriter struct {
	allowedClasses []string
	writer         io.Writer
}

// Implementa el método Write para que se filtre según las clases permitidas.
func (f *FilteredWriter) Write(p []byte) (n int, err error) {
	message := string(p)
	for _, class := range f.allowedClasses {
		if strings.Contains(message, class) {
			return f.writer.Write(p) // Si el mensaje contiene una clase permitida, lo escribe.
		}
	}
	return len(p), nil // Si no es una clase permitida, lo ignora.
}

// OJO!! TEnemos que arrancar el servidor de mongodb antes, sino no funcionará
//  sudo docker run --name mongodb -d -p 27017:27017 mongo:latest

func main() {
	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()
	// Filtrar logs de una clase específica
	// Creamos un FilteredWriter que filtra solo las clases permitidas.
	filter := &FilteredWriter{
		allowedClasses: []string{"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue"}, // Define las clases que deseas permitir
		writer: io.Discard, // Inicialmente no escribimos en ningún lado
	}

	// Redirigimos el log estándar al filtro.
	log.SetOutput(filter)

	ginMode, err := utils.ObtenerVariableDeEntorno("GIN_MODE")
	if err != nil {
		ginMode = "debug" // Valor por defecto si no se configura
	}

	if ginMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Configuramos un writer que sí imprimirá en consola.
	consoleWriter := io.Writer(os.Stdout)
	filter.writer = consoleWriter // Redirige los logs permitidos a la consola

	// Configuración de la base de datos
	uriMongo, err := utils.ObtenerVariableDeEntorno("URIMongo")
	if err != nil {
		log.Fatalf("Error al iniciar el servidor de BD.  URIMongo no configurado: %v", err)
		uriMongo = "mongodb://localhost:27017" // Valor por defecto si no se configura
	}

	nameMongo, err := utils.ObtenerVariableDeEntorno("NameMongo")
	if err != nil {
		log.Fatalf("Error al iniciar el servidor. Nombre del servidor no configurado: %v", err)
		nameMongo = "MongoChat" // Valor por defecto si no se configura
	}

	persistencia, err := persistence.NuevaMongoPersistencia(uriMongo, nameMongo)
	if err != nil {
		log.Fatalf("Error al iniciar el servidor de Base de datos. Pool de conexiones fallida: %v", err)
		return
	}

	secMod := services.CrearSecModServidorChat(persistencia, "salas_config.json")
	salasManager := secMod.GestionSalas

	// Imprimimos la sala principal
	log.Printf("Sala Principal: %s, ID: %s\n", salasManager.SalaPrincipal.RoomName, salasManager.SalaPrincipal.RoomId)

	// Imprimimos las salas fijas
	for id, sala := range salasManager.SalasFijas {
		log.Printf("Sala Fija: %s, ID: %s\n", sala.RoomName, id)
	}

	r := gin.Default()

	// Middleware CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "x-gochat"},
		AllowCredentials: true,
	}))

	// Middleware para comprobar el encabezado "x-gochat" en cada solicitud
	r.Use(func(c *gin.Context) {
		// Si es una solicitud WebSocket, no verificamos el encabezado
		if c.Request.Method == "GET" && c.Request.URL.Path == "/ws" {
			// Esto es WebSocket, no verificamos el encabezado aquí
			c.Next()
			return
		}

		// Si no es WebSocket, verificar el encabezado "x-gochat"
		goChatHeader := c.GetHeader("x-gochat")
		if goChatHeader == "" {
			// Si no está presente, devolver error y detener la ejecución
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "nok",
				"message": "Falta el parámetro 'x-gochat' en la solicitud",
			})
			c.Abort() // Detener el procesamiento de la solicitud
			return
		}
		log.Printf("Servidor goChatHeader en %s", goChatHeader)
		c.Next()
	})

	// Configurar las rutas para la API REST
	r.POST("/login", api.LoginHandler)
	r.POST("/newmessage", api.NewMessageHandler)

	// Configurar WebSocket para manejar las conexiones
	r.GET("/ws", comm.WebSocketHandler)

	// Configuración del servidor
	server, err := utils.ObtenerVariableDeEntorno("NameServer")
	if err != nil {
		log.Fatalf("Error al iniciar el servidor. Nombre del servidor no configurado: %v", err)
		server = "localhost" // Valor por defecto si no se configura
	}

	port, err := utils.ObtenerVariableDeEntorno("PortServer")
	if err != nil {
		port = "8081"
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}

	// Dirección completa para iniciar el servidor
	address := fmt.Sprintf("%s:%s", server, port)
	log.Printf("Servidor escuchando en %s", address)

	// Iniciar el servidor de forma concurrente, y manejar tanto HTTP como WebSocket
	go func() {
		err := r.Run(address)
		if err != nil {
			log.Fatalf("Error al iniciar el servidor HTTP: %v", err)
		}
	}()
	// Iniciar servidor para pprof en un goroutine
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	// Mantener el servidor activo
	select {} // Mantener el servidor activo
}
