package entities_test

import (
	"backend/entities"
	"backend/models"
	"backend/persistence"
	"backend/utils"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// FilteredWriter es un tipo que implementa io.Writer y filtra los logs.
type FilteredWriter struct {
	allowedClasses []string
	writer         io.Writer // Salida de log (puede ser os.Stdout o cualquier otro destino)
}

func (fw *FilteredWriter) Write(p []byte) (n int, err error) {
	// Convertimos el mensaje de log en string
	logMessage := string(p)

	// Verificamos si el mensaje contiene alguna de las clases permitidas
	for _, allowedClass := range fw.allowedClasses {
		if containsClass(logMessage, allowedClass) {
			// Si contiene la clase permitida, escribimos el log
			return fw.writer.Write(p)
		}
	}
	// Si no contiene la clase permitida, descartamos el mensaje
	return len(p), nil
}

func containsClass(logMessage, className string) bool {
	// Verificamos si el logMessage contiene el nombre de la clase
	return strings.Contains(logMessage, className)
}
func TestCircularQueue(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()
	// Filtrar logs de una clase específica
	// Creamos un FilteredWriter que filtra solo las clases permitidas.
	filter := &FilteredWriter{
		allowedClasses: []string{"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue"},
		writer: os.Stdout, // Cambia a `os.Stdout` para ver los logs en la consola
	}
	log.SetOutput(filter)
	// Crear un objeto de Persistencia simulado o falso si es necesario
	// Configuración de Persistencia mock
	var persis, err = persistence.NuevaMongoPersistencia("mongodb://localhost:27017/", "MongoGoChat")
	if err != nil {
		log.Fatal(err)
	}
	// Crear la cola con una capacidad de 5 mensajes
	cola := entities.NewCircularQueue(persis)

	// Crear un usuario de ejemplo para asociar a los mensajes
	usuario := entities.User{
		Nickname: "UsuarioTest",
		Token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX25hbWUiOiJKb2huIERvZSIsImV4cCI6MTY3MDEzNjA0M30.5YQ6wKUMfytcLjB-N9nZQ-k0S2z7Fpl68yV_Fo9G_z8",
	}

	// Crear un mensaje de ejemplo
	msg := models.CrearMensajeConFecha("Mensaje 1", usuario, uuid.New(), "Sala1", time.Now())

	// Probar Enqueue
	cola.Enqueue(*msg, uuid.New())

	// Verificar que la cola tiene el tamaño esperado
	assert.Equal(t, 1, cola.GetSize(), "El tamaño de la cola debería ser 1")

	// Verificar que el mensaje se encuentra en el índice correcto
	assert.Equal(t, msg, &cola.GetBuffer()[cola.GetHead()], "El mensaje debería estar en la posición de la cabeza")

	// Verificar que el índice del mensaje en el mapa es correcto
	assert.Equal(t, cola.GetHead(), cola.GetMessageMap()[msg.MessageId], "El índice del mensaje en el mapa debería coincidir con el índice de la cabeza")
}
func TestSynq(t *testing.T) {

	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()
	// Filtrar logs de una clase específica
	// Creamos un FilteredWriter que filtra solo las clases permitidas.
	filter := &FilteredWriter{
		allowedClasses: []string{"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue"}, // Define las clases que deseas permitir
		writer: os.Stdout, // Inicialmente no escribimos en ningún lado
	}

	// Redirigimos el log estándar al filtro.
	log.SetOutput(filter)
	// Crear un objeto de Persistencia simulado o falso si es necesario
	// Configuración de Persistencia mock
	var persis, err = persistence.NuevaMongoPersistencia("mongodb://localhost:27017/", "MongoGoChat")
	if err != nil {
		log.Fatal(err)
	}
	// Crear la cola con una capacidad de 5 mensajes
	cola := entities.NewCircularQueue(persis)

	// Obtener la capacidad de la cola (asegurándonos de que la cola tenga un campo que almacene esta capacidad)
	capacidad := cola.GetCapacity() // Aquí se debe obtener la capacidad de la cola

	// Crear un usuario de ejemplo
	usuario := entities.User{
		Nickname: "UsuarioTest",
		RoomId:   uuid.New(),
		Token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX25hbWUiOiJKb2huIERvZSIsImV4cCI6MTY3MDEzNjA0M30.5YQ6wKUMfytcLjB-N9nZQ-k0S2z7Fpl68yV_Fo9G_z8",
	}

	// Definir el número de mensajes a insertar (mayor que la capacidad de la cola)
	numMessages := capacidad + 5 // Esto asegura que insertamos más mensajes que la capacidad de la cola

	// Crear y añadir mensajes a la cola
	for i := 1; i <= numMessages; i++ {
		if i == 1024 {
			fmt.Printf("Empezamops a vaciar")
		}
		msg := *models.CrearMensajeConFecha(fmt.Sprintf("Mensaje %d", i), usuario, uuid.New(), "Sala1", time.Now())
		cola.Enqueue(msg, usuario.RoomId) // Usamos un roomId mock
		fmt.Printf(" head:[%d] . tail:[%d], size: [%d] \n", cola.GetHead(), cola.GetTail(), cola.GetSize())
	}

	// Verificar el estado de la cola después de insertar los mensajes
	fmt.Println("Estado de la cola después de insertar los mensajes:")
	fmt.Printf(" head:[%d] . tail:[%d], size: [%d] \n", cola.GetHead(), cola.GetTail(), cola.GetSize())

	// Mostrar todos los mensajes en la cola después de que se haya llenado y vaciado
	mensajesDB, mensajesCola, _ := cola.ObtenerTodosDiv(usuario.RoomId)
	fmt.Println("Mensajes en la cola después de insertar más mensajes que la capacidad:")
	//for _, msg := range mensajes {
	//fmt.Println(msg.MessageText)
	//}
	fmt.Printf("lenDB [%d]: lenCola: [%d]", len(mensajesDB), len(mensajesCola))
	// Verificar si la cola se vació correctamente al superar la capacidad
	// Esto se debe comprobar viendo los mensajes en la cola y asegurándose de que
	// la cantidad de mensajes sea igual a la capacidad de la cola.
	//if len(mensajes) != numMessages {
	//	t.Errorf("Esperábamos %d mensajes en la cola, pero hay %d", numMessages, len(mensajes))
	//}
}

func TestObtenerMensajes22(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()
	// Filtrar logs de una clase específica
	// Creamos un FilteredWriter que filtra solo las clases permitidas.
	filter := &FilteredWriter{
		allowedClasses: []string{"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue"},
		writer: os.Stdout, // Cambia a `os.Stdout` para ver los logs en la consola
	}
	log.SetOutput(filter)
	// Crear un objeto de Persistencia simulado o falso si es necesario
	// Configuración de Persistencia mock
	var persis, err = persistence.NuevaMongoPersistencia("mongodb://localhost:27017/", "MongoGoChat")
	if err != nil {
		log.Fatal(err)
	}
	// Crear la cola con una capacidad de 5 mensajes
	cola := entities.NewCircularQueue(persis)

	roomId := uuid.New()

	// Convertir string a uuid.UUID
	parsedUUID1, err := uuid.Parse("123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		fmt.Printf("Error al convertir string a UUID: %v\n", err)
		return
	}

	// Convertir string a uuid.UUID
	parsedUUID2, err := uuid.Parse("123e4567-e89b-12d3-a456-426614174001")
	if err != nil {
		fmt.Printf("Error al convertir string a UUID: %v\n", err)
		return
	}
	// Crear algunos mensajes de ejemplo
	msg1 := entities.Message{
		MessageId:   parsedUUID1,
		MessageText: "Mensaje 1",
	}
	msg2 := entities.Message{
		MessageId:   parsedUUID2,
		MessageText: "Mensaje 2",
	}
	// Enqueue de los mensajes
	cola.Enqueue(msg1, roomId)
	fmt.Printf(" head:[%d] . tail:[%d], size: [%d] \n", cola.GetHead(), cola.GetTail(), cola.GetSize())
	cola.Enqueue(msg2, roomId)
	fmt.Printf(" head:[%d] . tail:[%d], size: [%d] \n", cola.GetHead(), cola.GetTail(), cola.GetSize())

	// Obtener los mensajes
	mensajes, err := cola.ObtenerMensajes(roomId, msg1.MessageId, 2)
	fmt.Printf(" head:[%d] . tail:[%d], size: [%d] \n", cola.GetHead(), cola.GetTail(), cola.GetSize())
	fmt.Printf(" len(mensajes):[%d]   \n", len(mensajes))
	assert.NoError(t, err, "No debería haber error al obtener los mensajes")
	assert.Equal(t, 2, len(mensajes), "Deberían haberse recuperado 2 mensajes")
	assert.Equal(t, msg1, mensajes[0], "El primer mensaje debería ser 'Mensaje 1'")
	assert.Equal(t, msg2, mensajes[1], "El segundo mensaje debería ser 'Mensaje 2'")

	// Obtener los mensajes
	mensajes, err = cola.ObtenerMensajes(roomId, msg2.MessageId, 2)
	fmt.Printf(" head:[%d] . tail:[%d], size: [%d] \n", cola.GetHead(), cola.GetTail(), cola.GetSize())
	fmt.Printf(" len(mensajes):[%d]   \n", len(mensajes))
	assert.NoError(t, err, "No debería haber error al obtener los mensajes")
	assert.Equal(t, 1, len(mensajes), "Deberían haberse recuperado 2 mensajes")
	assert.Equal(t, msg2, mensajes[0], "El primer mensaje debería ser 'Mensaje 2'")

}

// InsertMessagesIntoQueue inserta el número de mensajes especificado en la cola
func InsertMessagesIntoQueue(numMessages int) *entities.CircularQueue {
	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()

	// Filtrar logs de una clase específica
	// Creamos un FilteredWriter que filtra solo las clases permitidas.
	filter := &FilteredWriter{
		allowedClasses: []string{
			"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue",
		}, // Define las clases que deseas permitir
		writer: os.Stdout, // Inicialmente no escribimos en ningún lado
	}

	// Redirigimos el log estándar al filtro.
	log.SetOutput(filter)

	// Crear un objeto de Persistencia simulado o falso si es necesario
	// Configuración de Persistencia mock
	var persis, err = persistence.NuevaMongoPersistencia("mongodb://localhost:27017/", "MongoGoChat")
	if err != nil {
		log.Fatal(err)
	}

	// Crear la cola con una capacidad de 5 mensajes
	cola := entities.NewCircularQueue(persis)
	roomId := uuid.New()
	// Crear un usuario de ejemplo
	usuario := entities.User{
		Nickname: "UsuarioTest",
		RoomId:   roomId,
		Token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX25hbWUiOiJKb2huIERvZSIsImV4cCI6MTY3MDEzNjA0M30.5YQ6wKUMfytcLjB-N9nZQ-k0S2z7Fpl68yV_Fo9G_z8",
	}

	// Crear y añadir mensajes a la cola
	for i := 1; i <= numMessages; i++ {
		msg := *models.CrearMensajeConFecha(fmt.Sprintf("Mensaje %d", i), usuario, usuario.RoomId, "Sala1", time.Now())
		cola.Enqueue(msg, usuario.RoomId) // Usamos un roomId mock
		fmt.Printf(" head:[%d] . tail:[%d], size: [%d] \n", cola.GetHead(), cola.GetTail(), cola.GetSize())
	}

	// Verificar el estado de la cola después de insertar los mensajes
	fmt.Println("Estado de la cola después de insertar los mensajes:")
	fmt.Printf(" head:[%d] . tail:[%d], size: [%d] \n", cola.GetHead(), cola.GetTail(), cola.GetSize())

	// Devolver el puntero a la cola
	return cola
}

func TestObtenerMensajes1(t *testing.T) {
	// Obtener la capacidad de la cola
	var capacidad = 1024
	var numMessages = capacidad + 5 // Insertar más mensajes que la capacidad

	// Crear la cola con una capacidad de 5 mensajes
	queue := InsertMessagesIntoQueue(numMessages)

	// Simular datos de prueba
	idSala := queue.GetBuffer()[0].RoomID
	idMensaje := queue.GetBuffer()[0].MessageId
	cantidad := 2

	mensajes, err := queue.ObtenerMensajes(idSala, idMensaje, cantidad)
	if err != nil || len(mensajes) != cantidad {
		t.Errorf("Esperaba %d mensajes, pero recibí %d", cantidad, len(mensajes))
	}
	fmt.Printf("  recibí %d mensajes", len(mensajes))
}
func TestObtenerTodos(t *testing.T) {
	// Obtener la capacidad de la cola
	var capacidad = 1024
	var numMessages = capacidad + 5 // Insertar más mensajes que la capacidad

	// Crear la cola con una capacidad de 5 mensajes
	queue := InsertMessagesIntoQueue(numMessages)
	var roomId = queue.GetBuffer()[0].RoomID
	mensajes, err := queue.ObtenerTodos(roomId)
	if err != nil || len(mensajes) == 0 {
		t.Errorf("Esperaba  mensajes, pero recibí %d", len(mensajes))
	}
	fmt.Printf("  recibí %d mensajes", len(mensajes))
}
func TestObtenerTodosDiv(t *testing.T) {
	var capacidad = 1024
	var numMessages = capacidad + 5 // Insertar más mensajes que la capacidad

	// Crear la cola con una capacidad de 5 mensajes
	queue := InsertMessagesIntoQueue(numMessages)
	var roomId = queue.GetBuffer()[0].RoomID
	mensajesBD, mensajesCola, err := queue.ObtenerTodosDiv(roomId)
	if err != nil {
		t.Errorf("Esperaba   mensajes, pero recibí BD: %d, Cola: %d", len(mensajesBD), len(mensajesCola))
	}
	fmt.Printf("  recibí BD: %d, Cola: %d", len(mensajesBD), len(mensajesCola))

}
func TestObtenerMensajesDesdeId(t *testing.T) {
	var capacidad = 1024
	var numMessages = capacidad + 5 // Insertar más mensajes que la capacidad

	// Crear la cola con una capacidad de 5 mensajes
	queue := InsertMessagesIntoQueue(numMessages)

	var roomId = queue.GetBuffer()[0].RoomID
	var roomId3 = queue.GetBuffer()[3].RoomID
	var msgId = queue.GetBuffer()[254].MessageId

	fmt.Printf(" len buffer {%d] - id msgId  :[%s] \n", len(queue.GetBuffer()), msgId.String())
	fmt.Printf(" roomId {%s] - roomId3  :[%s]\n ", roomId.String(), roomId3.String())
	mensajes, err := queue.ObtenerMensajesDesdeId(roomId, msgId)

	if err != nil || len(mensajes) == 0 {
		t.Errorf("Esperaba obtener todos los mensajes, pero recibí %d", len(mensajes))
	}
	fmt.Printf(" len (mensajes) :[%d] - id msgId  :[%s] - id primer mensaje [%s] ", len(mensajes), msgId, mensajes[0].MessageId)

	var msgId0 = queue.GetBuffer()[0].MessageId

	fmt.Printf(" len buffer {%d] - id msgId  :[%s] \n", len(queue.GetBuffer()), msgId.String())
	fmt.Printf(" roomId {%s] - roomId3  :[%s]\n ", roomId.String(), roomId3.String())
	mensajes, err = queue.ObtenerMensajesDesdeId(roomId, msgId0)

	if err != nil || len(mensajes) == 0 {
		t.Errorf("Esperaba obtener todos los mensajes, pero recibí %d", len(mensajes))
	}
	fmt.Printf(" len (mensajes) :[%d] - id msgId  :[%s] - id primer mensaje [%s] ", len(mensajes), msgId0, mensajes[0].MessageId)

}
func TestErrorWebMensajesDesdeId(t *testing.T) {
	log.SetFlags(log.Lshortfile)
	utils.CargarVariablesDeEntorno()
	// Filtrar logs de una clase específica
	// Creamos un FilteredWriter que filtra solo las clases permitidas.
	filter := &FilteredWriter{
		allowedClasses: []string{"PostListHandler", "WebSocketHandler", "GestionSalas",
			"NewMessageHandler", "SecModServidorChat", "CircularQueue"},
		writer: os.Stdout, // Cambia a `os.Stdout` para ver los logs en la consola
	}
	log.SetOutput(filter)
	// Crear un objeto de Persistencia simulado o falso si es necesario
	// Configuración de Persistencia mock
	var persis, err = persistence.NuevaMongoPersistencia("mongodb://localhost:27017/", "MongoGoChat")
	if err != nil {
		log.Fatal(err)
	}
	// Crear la cola con una capacidad de 5 mensajes
	queue := entities.NewCircularQueue(persis)

	// String con el UUID
	uuidString := "86cee21d-ed08-435a-bdb3-e6f443c00514"

	// Convertir string a uuid.UUID
	roomId, err := uuid.Parse(uuidString) //uuid.New()
	if err != nil {
		roomId = uuid.New()
	}
	// Crear un usuario de ejemplo para asociar a los mensajes
	usuario := entities.User{
		Nickname: "UsuarioTest",
		RoomId:   roomId,
		Token:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX25hbWUiOiJKb2huIERvZSIsImV4cCI6MTY3MDEzNjA0M30.5YQ6wKUMfytcLjB-N9nZQ-k0S2z7Fpl68yV_Fo9G_z8",
	}
	numMensajes := 6
	// Iterar para insertar los mensajes
	for i := 1; i <= numMensajes; i++ {
		// Crear un mensaje con fecha y un texto dinámico
		mensajeTexto := fmt.Sprintf("Mensaje %d", i)
		msg := models.CrearMensajeConFecha(mensajeTexto, usuario, roomId, "Sala1", time.Now())
		msgId := msg.MessageId

		// Probar Enqueue (insertar el mensaje en la cola)
		queue.Enqueue(*msg, roomId)

		// Imprimir el estado del búfer y el ID del mensaje
		fmt.Printf("Mensaje %d insertado\n", i)
		fmt.Printf("len buffer: {%d} - id msgId: [%s]\n", len(queue.GetBuffer()), msgId.String())
		fmt.Printf("roomId: {%s}\n", roomId)

		// Obtener los mensajes desde el ID del mensaje insertado
		mensajesRecuperados, err := queue.ObtenerMensajesDesdeId(roomId, msgId)

		// Verificar si los mensajes fueron recuperados correctamente
		if err != nil || len(mensajesRecuperados) == 0 {
			fmt.Printf("Esperaba obtener todos los mensajes, pero recibí %d mensajes\n", len(mensajesRecuperados))
		} else {
			// Imprimir detalles sobre el mensaje recuperado
			fmt.Printf("Detalles de mensajes recuperados:\n")
			for _, mensaje := range mensajesRecuperados {
				fmt.Printf("ID: [%s] - Texto: [%s]\n", mensaje.MessageId.String(), mensaje.MessageText)
			}
		}
	}

}
