package entities_test

import (
	"backend/entities"
	"backend/models"
	"backend/persistence"
	"backend/utils"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/rand"
)

func GenerateRandomNickname() string {
	// Generar un nombre aleatorio como un Nickname
	return fmt.Sprintf("Usuario_%d", rand.Intn(1000000))
}

func GenerateRandomToken() string {
	// Generar un token aleatorio (esto es solo un ejemplo, en producción debería ser un JWT válido)
	token := uuid.New().String()
	return fmt.Sprintf("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.%s", token)
}
func InsertMessagesConUsuariosConcurrentes(numMessages int, numUsuarios int) *entities.CircularQueue {
	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()

	// Configuración de los logs
	filter := &FilteredWriter{
		allowedClasses: []string{
			"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue",
		},
		writer: os.Stdout,
	}
	log.SetOutput(filter)

	// Conexión a la base de datos
	var persis, err = persistence.NuevaMongoPersistencia("mongodb://localhost:27017/", "MongoGoChat")
	if err != nil {
		log.Fatal(err)
	}

	// Crear la cola
	cola := entities.NewCircularQueue(persis)
	roomId := uuid.New()
	log.Printf("roomId :[%s]", roomId.String())
	log.SetOutput(filter)

	// Sincronización de goroutines
	var wg sync.WaitGroup

	// Crear usuarios diferentes
	for i := 0; i < numUsuarios; i++ {
		wg.Add(1) // Incrementa el contador del WaitGroup

		go func(userId int) {
			defer wg.Done() // Decrementa el contador cuando termina esta goroutine

			// Crear un nuevo usuario con un Nickname y Token diferentes para cada uno
			roomId := uuid.New()
			usuario := entities.User{
				Nickname: fmt.Sprintf("Usuario%d", userId),
				RoomId:   roomId,
				Token:    fmt.Sprintf("token_%d", userId),
			}

			// Insertar mensajes en paralelo para este usuario
			for j := 1; j <= numMessages; j++ {
				msg := *models.CrearMensajeConFecha(fmt.Sprintf("Mensaje %d de %s", j, usuario.Nickname), usuario, usuario.RoomId, "Sala1", time.Now())
				cola.Enqueue(msg, usuario.RoomId)
				fmt.Printf(" head:[%d] . tail:[%d], size: [%d] \n", cola.GetHead(), cola.GetTail(), cola.GetSize())
			}
		}(i)
	}

	// Esperar a que todas las goroutines terminen
	wg.Wait()

	// Retornar la cola con los mensajes insertados
	return cola
}
func InsertMessagesConConcurrentUsers(numMessages int) *entities.CircularQueue {
	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()

	// Configuración de los logs
	filter := &FilteredWriter{
		allowedClasses: []string{
			"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue",
		},
		writer: os.Stdout,
	}
	log.SetOutput(filter)

	// Conexión a la base de datos
	var persis, err = persistence.NuevaMongoPersistencia("mongodb://localhost:27017/", "MongoGoChat")
	if err != nil {
		log.Fatal(err)
	}

	// Crear la cola
	cola := entities.NewCircularQueue(persis)
	roomId := uuid.New()

	// WaitGroup para esperar a que todas las goroutines terminen
	var wg sync.WaitGroup

	// Función para insertar mensajes en la cola
	insertMessage := func(i int) {
		defer wg.Done() // Señaliza que la goroutine ha terminado

		// Crear un usuario único por cada goroutine
		usuario := entities.User{
			Nickname: GenerateRandomNickname(), // Genera un nickname único
			RoomId:   roomId,
			Token:    GenerateRandomToken(), // Genera un token único
		}

		// Crear y encolar el mensaje
		msg := *models.CrearMensajeConFecha(fmt.Sprintf("Mensaje %d", i), usuario, usuario.RoomId, "Sala1", time.Now())
		cola.Enqueue(msg, usuario.RoomId)
	}

	// Insertar mensajes en paralelo
	for i := 1; i <= numMessages; i++ {
		wg.Add(1)           // Añadir una nueva goroutine al WaitGroup
		go insertMessage(i) // Ejecutar la inserción de mensaje en una goroutine
	}

	// Esperar a que todas las goroutines terminen
	wg.Wait()

	// Devolver la cola después de insertar los mensajes
	return cola
}
func TestTestInsertMsgUsersAsynq(t *testing.T) {
	// Número de mensajes y usuarios concurrentes
	numMessages := 20
	numUsuarios := 4

	// Llamar a la función que inserta mensajes en paralelo
	cola := InsertMessagesConUsuariosConcurrentes(numMessages, numUsuarios)

	// Aquí puedes agregar código para trabajar con la cola, por ejemplo, imprimir el contenido
	fmt.Printf("Mensajes insertados en la cola. roomID: [%s] - len msg {%d}\n", cola.GetBuffer()[0].RoomID.String(), len(cola.GetBuffer()))
}

func TestCircularQueueAsynq(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()

	// Configuración de los logs
	filter := &FilteredWriter{
		allowedClasses: []string{
			"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue",
		},
		writer: os.Stdout,
	}
	log.SetOutput(filter)

	// Conexión a la base de datos
	var persis, err = persistence.NuevaMongoPersistencia("mongodb://localhost:27017/", "MongoGoChat")
	if err != nil {
		log.Fatal(err)
	}

	// Crear la cola
	cola := entities.NewCircularQueue(persis)
	roomId := uuid.New()
	log.Printf("roomId :[%s]", roomId.String())
	log.SetOutput(filter)

	usuario := entities.User{
		Nickname: "UsuarioTest",
		Token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX25hbWUiOiJKb2huIERvZSIsImV4cCI6MTY3MDEzNjA0M30.5YQ6wKUMfytcLjB-N9nZQ-k0S2z7Fpl68yV_Fo9G_z8",
	}
	var wg sync.WaitGroup

	// Goroutine para insertar mensajes
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Crear un usuario de ejemplo para asociar a los mensajes
			// Crear un mensaje de ejemplo
			msg := models.CrearMensajeConFecha("Mensaje "+strconv.Itoa(i), usuario, uuid.New(), "Sala1", time.Now())

			cola.Enqueue(*msg, uuid.New()) // Usar un roomId mock
			log.Printf("Mensaje %d encolado: %s", i, msg.MessageText)
		}(i)
	}
	// Goroutine para leer mensajes
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Esperamos un poco para simular el procesamiento
		// y leemos los mensajes después
		mensajes, _ := cola.ObtenerTodos(uuid.New())
		log.Println("Mensajes en la cola:")
		for _, msg := range mensajes {
			log.Println(msg.MessageText)
		}
	}()

	// Esperamos a que todas las goroutines terminen
	wg.Wait()

	// Ver si la cola funciona correctamente después de la concurrencia
	mensajes, _ := cola.ObtenerTodos(uuid.New())
	fmt.Println("Mensajes en la cola después de las operaciones concurrentes:")
	for _, msg := range mensajes {
		fmt.Println(msg.MessageText)
	}
}
func TestCircularQueueAsynq1(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()
	filter := &FilteredWriter{
		allowedClasses: []string{"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue"},
		writer: os.Stdout,
	}
	log.SetOutput(filter)
	var persis, err = persistence.NuevaMongoPersistencia("mongodb://localhost:27017/", "MongoGoChat")
	if err != nil {
		log.Fatal(err)
	}
	cola := entities.NewCircularQueue(persis)
	roomID := uuid.New()
	usuario := entities.User{
		Nickname: "UsuarioTest",
		Token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX25hbWUiOiJKb2huIERvZSIsImV4cCI6MTY3MDEzNjA0M30.5YQ6wKUMfytcLjB-N9nZQ-k0S2z7Fpl68yV_Fo9G_z8",
	}
	msg := models.CrearMensajeConFecha("Mensaje 1", usuario, roomID, "Sala1", time.Now())

	// Goroutine para encolar y verificar
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		cola.Enqueue(*msg, roomID)
		assert.Equal(t, 1, cola.GetSize(), "El tamaño de la cola debería ser 1")
		assert.Equal(t, msg, &cola.GetBuffer()[cola.GetHead()], "El mensaje debería estar en la posición de la cabeza")
	}()

	wg.Wait() // Esperamos que la goroutine termine

	assert.Equal(t, msg, &cola.GetBuffer()[cola.GetHead()], "El mensaje debería estar en la posición de la cabeza")
	assert.Equal(t, cola.GetHead(), cola.GetMessageMap()[msg.MessageId], "El índice del mensaje en el mapa debería coincidir con el índice de la cabeza")
}

func TestSynq1(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()
	filter := &FilteredWriter{
		allowedClasses: []string{"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue"},
		writer: os.Stdout,
	}
	log.SetOutput(filter)

	var persis, err = persistence.NuevaMongoPersistencia("mongodb://localhost:27017/", "MongoGoChat")
	if err != nil {
		log.Fatal(err)
	}
	cola := entities.NewCircularQueue(persis)
	capacidad := cola.GetCapacity()

	usuario := entities.User{
		Nickname: "UsuarioTest",
		RoomId:   uuid.New(),
		Token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX25hbWUiOiJKb2huIERvZSIsImV4cCI6MTY3MDEzNjA0M30.5YQ6wKUMfytcLjB-N9nZQ-k0S2z7Fpl68yV_Fo9G_z8",
	}

	numMessages := capacidad + 5
	var wg sync.WaitGroup

	// Insertamos mensajes de manera concurrente
	for i := 1; i <= numMessages; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			msg := *models.CrearMensajeConFecha(fmt.Sprintf("Mensaje %d", i), usuario, uuid.New(), "Sala1", time.Now())
			cola.Enqueue(msg, usuario.RoomId)
			fmt.Printf(" head:[%d] . tail:[%d], size: [%d] \n", cola.GetHead(), cola.GetTail(), cola.GetSize())
		}(i)
	}

	wg.Wait() // Esperamos que todas las goroutines terminen

	fmt.Println("Estado final de la cola después de insertar los mensajes:")
	fmt.Printf(" head:[%d] . tail:[%d], size: [%d] \n", cola.GetHead(), cola.GetTail(), cola.GetSize())
}
func TestObtenerMensajes2(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()

	// Filtrar logs de una clase específica
	filter := &FilteredWriter{
		allowedClasses: []string{"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue"},
		writer: os.Stdout,
	}
	log.SetOutput(filter)

	var persis, err = persistence.NuevaMongoPersistencia("mongodb://localhost:27017/", "MongoGoChat")
	if err != nil {
		log.Fatal(err)
	}

	cola := entities.NewCircularQueue(persis)
	roomId := uuid.New()

	parsedUUID1, err := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		fmt.Printf("Error al convertir string a UUID: %v\n", err)
		return
	}

	parsedUUID2, err := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
	if err != nil {
		fmt.Printf("Error al convertir string a UUID: %v\n", err)
		return
	}

	msg1 := entities.Message{
		MessageId:   parsedUUID1,
		MessageText: "Mensaje 1",
	}
	msg2 := entities.Message{
		MessageId:   parsedUUID2,
		MessageText: "Mensaje 2",
	}

	cola.Enqueue(msg1, roomId)
	cola.Enqueue(msg2, roomId)

	mensajes, err := cola.ObtenerMensajes(roomId, msg1.MessageId, 2)
	assert.NoError(t, err, "No debería haber error al obtener los mensajes")
	assert.Equal(t, 2, len(mensajes), "Deberían haberse recuperado 2 mensajes")
	assert.Equal(t, msg1, mensajes[0], "El primer mensaje debería ser 'Mensaje 1'")
	assert.Equal(t, msg2, mensajes[1], "El segundo mensaje debería ser 'Mensaje 2'")

	mensajes, err = cola.ObtenerMensajes(roomId, msg2.MessageId, 2)
	assert.NoError(t, err, "No debería haber error al obtener los mensajes")
	assert.Equal(t, 1, len(mensajes), "Deberían haberse recuperado 2 mensajes")
	assert.Equal(t, msg2, mensajes[0], "El primer mensaje debería ser 'Mensaje 2'")
}

func TestObtenerMensajes21(t *testing.T) {
	var capacidad = 1024
	var numMessages = capacidad + 5

	queue := InsertMessagesIntoQueue(numMessages)

	idSala := queue.GetBuffer()[0].RoomID
	idMensaje := queue.GetBuffer()[0].MessageId
	cantidad := 2

	mensajes, err := queue.ObtenerMensajes(idSala, idMensaje, cantidad)
	if err != nil || len(mensajes) != cantidad {
		t.Errorf("Esperaba %d mensajes, pero recibí %d", cantidad, len(mensajes))
	}
}

func TestObtenerTodos1(t *testing.T) {
	var capacidad = 1024
	var numMessages = capacidad + 5

	queue := InsertMessagesIntoQueue(numMessages)
	var roomId = queue.GetBuffer()[0].RoomID
	mensajes, err := queue.ObtenerTodos(roomId)
	if err != nil || len(mensajes) == 0 {
		t.Errorf("Esperaba mensajes, pero recibí %d", len(mensajes))
	}
}

func TestObtenerTodosDiv1(t *testing.T) {
	var capacidad = 1024
	var numMessages = capacidad + 5

	queue := InsertMessagesIntoQueue(numMessages)
	var roomId = queue.GetBuffer()[0].RoomID
	mensajesBD, mensajesCola, err := queue.ObtenerTodosDiv(roomId)
	if err != nil {
		t.Errorf("Esperaba mensajes, pero recibí BD: %d, Cola: %d", len(mensajesBD), len(mensajesCola))
	}
}

func TestObtenerMensajesDesdeId1(t *testing.T) {
	var capacidad = 1024
	var numMessages = capacidad + 5

	queue := InsertMessagesIntoQueue(numMessages)

	var roomId = queue.GetBuffer()[0].RoomID
	var msgId = queue.GetBuffer()[254].MessageId

	mensajes, err := queue.ObtenerMensajesDesdeId(roomId, msgId)
	if err != nil || len(mensajes) == 0 {
		t.Errorf("Esperaba obtener todos los mensajes, pero recibí %d", len(mensajes))
	}
}
