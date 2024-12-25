package entities

import (
	"backend/utils"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

type CircularQueue struct {
	buffer       []Message         // Buffer circular de mensajes
	head, tail   int               // Índice del primer mensaje y el último
	size         int               // Número actual de mensajes en la cola
	capacity     int               // Capacidad máxima de la cola (se lee desde el archivo .env)
	mu           sync.Mutex        // Mutex para la exclusión mutua, para controlar el flujo de acceso a la cola
	insertMutex  sync.Mutex        // Mutex específico para la inserción de mensajes ern el buffer   circular
	onceQueue    sync.Once         // Bandera para controlar el mutex
	notEmpty     *sync.Cond        // Condición para saber si la cola está vacía
	notFull      *sync.Cond        // Condición para saber si la cola está llena
	isFlushing   bool              // Indica si el vaciado está en curso
	isProcessing bool              //Para procesar el canal
	messageMap   map[uuid.UUID]int // Mapa que asocia un MessageId con su índice en el buffer
	waitChannel  chan Message      // Canal para nuevas inserciones bloqueadas
	persistencia *Persistencia
}

// / Getter para obtener el buffer
func (q *CircularQueue) IsFlushing() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	log.Printf("CircularQueue: IsFlushing: estado actual de isFlushing: %v\n", q.isFlushing)
	return q.isFlushing
}
func (q *CircularQueue) SetFlushingState(state bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.isFlushing = state
	log.Printf("CircularQueue: SetFlushingState: isFlushing cambiado a %v\n", state)
}

// / Getter para obtener el buffer
func (cq *CircularQueue) GetBuffer() []Message {
	return cq.buffer
}

// Getter para obtener la cabeza (head)
func (cq *CircularQueue) GetHead() int {
	return cq.head
}

// Getter para obtener la cola (tail)
func (cq *CircularQueue) GetTail() int {
	return cq.tail
}

// Getter para obtener el tamaño de la cola
func (cq *CircularQueue) GetSize() int {
	return cq.size
}

// Getter para obtener la capacidad de la cola
func (cq *CircularQueue) GetCapacity() int {
	return cq.capacity
}

// Getter para obtener el mapa de mensajes
func (cq *CircularQueue) GetMessageMap() map[uuid.UUID]int {
	return cq.messageMap
}

// Getter para obtener la instancia de Persistencia
func (cq *CircularQueue) GetPersistencia() *Persistencia {
	return cq.persistencia
}

// Método para insertar mensajes en la cola
// NewCircularQueue crea una nueva instancia de ColaCircular
func NewCircularQueue(persistence *Persistencia) *CircularQueue {
	val, err_env := utils.ObtenerVariableDeEntorno("SizeQueue")
	if err_env != nil {
		log.Fatalf("CircularQueue: NewCircularQueue Error al leer EXP_SIZE_QMESSAGE: %v", err_env)
	}
	capacity, err := strconv.Atoi(val)
	if err != nil {
		log.Fatalf("CircularQueue: NewCircularQueue Error al strconv.Atoi (val): %v", err)
	}

	// Crear la cola con la capacidad configurada
	q := CircularQueue{
		buffer:       make([]Message, capacity),
		head:         0,
		tail:         0, // Iniciamos en 0 en lugar de -1
		size:         0,
		capacity:     capacity,
		messageMap:   make(map[uuid.UUID]int, capacity),
		waitChannel:  make(chan Message, capacity),
		persistencia: persistence,
		isFlushing:   false,
	}
	q.notEmpty = sync.NewCond(&q.mu)
	q.notFull = sync.NewCond(&q.mu)
	return &q
}

// Inserción de mensajes en la cola de manera segura
func (q *CircularQueue) insertMessage(msg Message) {

	// Bloqueamos solo al principio para hacer la inserción
	q.insertMutex.Lock()
	defer q.insertMutex.Unlock()
	log.Printf("CircularQueue: insertMessage (Goroutine %d): q.size %d\n", runtime.NumGoroutine(), q.size)

	// Si la cola está vacía, insertamos en el head
	if q.size == 0 {
		q.buffer[q.head] = msg
		q.size++
		q.messageMap[msg.MessageId] = q.head
		log.Printf("CircularQueue: insertMessage (Goroutine %d): Mensaje agregado al índice %d (cola vacía)\n", runtime.NumGoroutine(), q.head)
	} else {
		// Si la cola no está vacía, insertamos en el tail y actualizamos
		q.tail = (q.tail + 1) % q.capacity
		q.buffer[q.tail] = msg
		q.size++
		q.messageMap[msg.MessageId] = q.tail
		log.Printf("Mensaje agregado al índice %d  \n", q.tail)
	}

	log.Printf("CircularQueue: insertMessage (Goroutine %d):q.size %d - tail:%d\n", runtime.NumGoroutine(), q.size, q.tail)
}

// flushToDatabaseInBackground: Vacia la cola principal y lanza el procesamiento del canal
func (q *CircularQueue) flushToDatabaseInBackground(roomId uuid.UUID) {
	var bufferCopia []Message
	log.Printf("CircularQueue: flushToDatabaseInBackground: inicio vaciado - antes de lock")

	// Copiar el contenido actual del buffer de manera eficiente
	if q.head < q.tail {
		bufferCopia = append(bufferCopia, q.buffer[q.head:q.tail]...)
	} else if q.size > 0 {
		bufferCopia = append(bufferCopia, q.buffer[q.head:]...)
		bufferCopia = append(bufferCopia, q.buffer[:q.tail]...)
	}

	// Reinicializamos la cola después de hacer la copia
	q.buffer = make([]Message, q.capacity)
	q.messageMap = make(map[uuid.UUID]int)
	q.head = 0
	q.tail = 0
	q.size = 0
	q.isFlushing = false
	q.notFull.Broadcast() // Notificamos que hay espacio disponible

	// Lanzamos la goroutine para guardar los datos en la base de datos
	go func(datos []Message) {
		log.Printf("CircularQueue: flushToDatabaseInBackground: Guardando mensajes en la base de datos...")
		err := (*q.persistencia).GuardarMensajesEnBaseDeDatos(datos, roomId)
		if err != nil {
			log.Printf("CircularQueue: flushToDatabaseInBackground: Error al guardar los mensajes: %v", err)
		}
	}(bufferCopia)

	// Siempre que haya nuevos mensajes en el canal, comenzamos el procesamiento
	if len(q.waitChannel) > 0 && !q.isProcessing {
		q.isProcessing = true
		log.Printf("CircularQueue: flushToDatabaseInBackground: Iniciando procesamiento del canal de espera")
		go q.processWaitChannel()
	}

	log.Printf("CircularQueue: flushToDatabaseInBackground: buffer limpio")
}

// Goroutine que procesa mensajes en el canal de espera
// Goroutine que procesa mensajes en el canal de espera
// Goroutine que procesa mensajes en el canal de espera
func (q *CircularQueue) processWaitChannel() {
	// Procesamos mensajes en el canal mientras haya mensajes
	goroutineID := runtime.NumGoroutine()
	log.Printf("CircularQueue: processWaitChannel (Goroutine %d): Comenzando a procesar el canal", goroutineID)

	for msg := range q.waitChannel {
		// Si hay espacio en la cola, inserta el mensaje
		for {
			if q.size < q.capacity {
				q.insertMessage(msg)
				break
			}
			time.Sleep(10 * time.Millisecond) // Espera antes de intentar de nuevo
		}
	}

	// Si el canal está vacío, procesamos más
	if len(q.waitChannel) == 0 {
		// Aquí, revisamos si hay nuevos mensajes en el canal antes de poner q.isProcessing en false
		if len(q.waitChannel) == 0 {
			q.isProcessing = false
			log.Printf("CircularQueue: processWaitChannel  (Goroutine %d): Canal vacío, deteniendo procesamiento", goroutineID)
		}
	} else {
		// Si llegaron más mensajes, volvemos a activar el procesamiento
		q.isProcessing = true
		log.Printf("CircularQueue: processWaitChannel  (Goroutine %d): Nuevos mensajes, reactivando procesamiento", goroutineID)
		go q.processWaitChannel()
	}
}

// Método para insertar mensajes en la cola
// Enqueue: Inserta un mensaje en la cola o el canal de espera
func (q *CircularQueue) Enqueue(msg Message, roomId uuid.UUID) {
	// Usar el mutex para proteger el acceso a isFlushing
	q.mu.Lock()
	defer q.mu.Unlock()
	log.Printf("CircularQueue: Enqueue: Iniciamos Enqueue q.size:[%d] - tail:[%d] - isFlushing :[%v].", q.size, q.tail, q.isFlushing)

	// Lanzar el vaciado si está lleno y no se está vaciando
	if q.size == q.capacity && !q.isFlushing {
		q.isFlushing = true
		log.Printf("CircularQueue: Enqueue: go q.flushToDatabaseInBackground(roomId).")

		// Liberar el mutex antes de iniciar la goroutine

		go func() {

			q.flushToDatabaseInBackground(roomId)
		}()

	}

	// Si la cola está llena, enviar el mensaje al canal de espera
	if q.size == q.capacity {
		log.Printf("CircularQueue: Enqueue: Cola llena, mensaje enviado al canal.")
		q.waitChannel <- msg
		return
	}
	log.Printf("CircularQueue: Enqueue: q.insertMessage(msg).")
	// Insertar el mensaje en la cola
	q.insertMessage(msg)
	log.Printf("CircularQueue: Enqueue: Mensaje agregado en el índice %d, tamaño %d\n", q.tail, q.size)
}

func (q *CircularQueue) ObtenerMensajes(idSala uuid.UUID, idMensaje uuid.UUID, cantidad int) ([]Message, error) {
	// Bloqueamos el mutex al inicio de la función
	q.mu.Lock()
	defer q.mu.Unlock()

	// Caso 1: Si la cola está vacía, obtenemos los mensajes desde la base de datos.
	if q.size == 0 {
		log.Printf("CircularQueue: ObtenerMensajes: Cola vacía, obteniendo mensajes desde la base de datos.")
		mensajes, err := (*q.persistencia).ObtenerMensajesDesdeSalaPorId(idSala, idMensaje)
		if err != nil {
			log.Printf("Error al obtener mensajes: %v", err)
			return nil, err
		}
		if len(mensajes) > cantidad {
			mensajes = mensajes[:cantidad]
		}
		return mensajes, nil
	}

	// Caso 2: Si la cola tiene mensajes, buscamos el idMensaje en la cola circular
	_, indice, err := q.buscarMensajeEnColaSinLock(idMensaje)
	if err != nil {
		log.Printf("CircularQueue: ObtenerMensajes: Mensaje no encontrado en la cola, obteniendo desde la base de datos.")
		mensajes, err := (*q.persistencia).ObtenerMensajesDesdeSalaPorId(idSala, idMensaje)
		if err != nil {
			return nil, fmt.Errorf("error al obtener mensajes desde la base de datos: %v", err)
		}
		return mensajes, nil
	}

	// Recuperar mensajes
	var mensajes []Message

	// Caso 3: Si la cola tiene suficientes mensajes después del índice
	if q.size-indice >= cantidad {
		// Si tenemos suficiente espacio en la cola desde el índice
		for i := 0; i < cantidad; i++ {
			mensajes = append(mensajes, q.buffer[(q.head+indice+i)%q.capacity])
		}
	} else {
		// Caso 4: Si la cola tiene menos mensajes que la cantidad solicitada
		// Primero, agregamos todos los mensajes desde el índice hasta el final de la cola
		for i := 0; i < q.size-indice; i++ {
			mensajes = append(mensajes, q.buffer[(q.head+indice+i)%q.capacity])
		}

		// Ahora calculamos cuántos mensajes faltan y tratamos de obtenerlos desde la base de datos
		cantidadRestante := cantidad - len(mensajes)
		mensajesDesdeBD, err := (*q.persistencia).ObtenerMensajesAnteriores(idMensaje, cantidadRestante)
		if err != nil {
			return nil, fmt.Errorf("CircularQueue: ObtenerMensajes: error al obtener mensajes desde la base de datos: %v", err)
		}
		if mensajesDesdeBD != nil {
			mensajes = append(mensajes, mensajesDesdeBD...)
		}
	}

	return mensajes, nil
}
func (q *CircularQueue) ObtenerTodos(idSala uuid.UUID) ([]Message, error) {
	q.mu.Lock()
	defer q.mu.Unlock() // Defers la liberación del bloqueo cuando se termina la función

	var mensajes []Message

	// Consultamos la base de datos si la cola está vacía
	if q.size == 0 {
		mensajesEnBD, err := (*q.persistencia).ObtenerMensajesDesdeSala(idSala)
		if err != nil {
			log.Printf("CircularQueue: ObtenerTodos: Error al obtener mensajes desde la BD: %v", err)
			return nil, err
		}
		if mensajesEnBD != nil {
			return mensajesEnBD, nil
		}
		return nil, nil
	}

	// Si la cola no está vacía, obtenemos los mensajes de la cola
	mensajesDeLaCola := make([]Message, q.size)
	for i := 0; i < q.size; i++ { // El límite debe ser 'i < q.size'
		indice := (q.head + i) % q.capacity
		mensajesDeLaCola[i] = q.buffer[indice]
	}

	// Consultamos la base de datos para obtener los mensajes si es necesario
	mensajesEnBD, err := (*q.persistencia).ObtenerMensajesDesdeSala(idSala)
	if err != nil {
		log.Printf("CircularQueue: ObtenerTodos: Error al obtener mensajes desde la BD: %v", err)
		return nil, err
	}

	// Concatenamos los mensajes de la base de datos y la cola si es necesario
	if mensajesEnBD != nil && len(mensajesDeLaCola) > 0 {
		mensajes = append(mensajesEnBD, mensajesDeLaCola...)
	} else if mensajesEnBD != nil {
		mensajes = mensajesEnBD
	} else if len(mensajesDeLaCola) > 0 {
		mensajes = mensajesDeLaCola
	}

	log.Printf("%v", mensajes)
	return mensajes, nil
}
func (q *CircularQueue) ObtenerTodosDiv(idSala uuid.UUID) ([]Message, []Message, error) {
	var alreadyLocked bool

	q.onceQueue.Do(func() {
		q.mu.Lock()
		alreadyLocked = true
	})
	if !alreadyLocked {
		defer q.mu.Unlock()
	}

	var mensajes []Message

	// Consultamos la base de datos si la cola está vacía
	if q.size == 0 {
		mensajesEnBD, err := (*q.persistencia).ObtenerMensajesDesdeSala(idSala)
		if err != nil {
			log.Printf("CircularQueue: ObtenerTodos: Error al obtener mensajes desde la BD: %v", err)
			return nil, nil, err
		}
		// Si no hay mensajes en la base de datos, devolvemos nil
		if mensajesEnBD != nil {
			return mensajesEnBD, nil, nil
		}
		return nil, nil, nil
	}

	// Si la cola no está vacía, obtenemos los mensajes de la cola
	mensajesDeLaCola := make([]Message, q.size)
	for i := 0; i < q.size; i++ {
		indice := (q.head + i) % q.capacity
		mensajesDeLaCola[i] = q.buffer[indice]
	}

	// Consultamos la base de datos para obtener los mensajes si es necesario
	mensajesEnBD, err := (*q.persistencia).ObtenerMensajesDesdeSala(idSala)
	if err != nil {
		log.Printf("CircularQueue: ObtenerTodos: Error al obtener mensajes desde la BD: %v", err)
		return nil, nil, err
	}

	// Si hay mensajes en la base de datos y en la cola, concatenamos los dos
	if mensajesEnBD != nil && len(mensajesDeLaCola) > 0 {
		mensajes = append(mensajesEnBD, mensajesDeLaCola...)
	} else if mensajesEnBD != nil {
		mensajes = mensajesEnBD
	} else if len(mensajesDeLaCola) > 0 {
		mensajes = mensajesDeLaCola
	}
	log.Printf("%v", mensajes)
	return mensajesEnBD, mensajesDeLaCola, nil
}

// ObtenerMensajesDesdeId obtiene mensajes desde un id específico.
func (q *CircularQueue) ObtenerMensajesDesdeId(idSala uuid.UUID, idMensaje uuid.UUID) ([]Message, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	log.Printf("CircularQueue:  ObtenerMensajesDesdeId:  Parámetros de entrada: idSala=%s, idMensaje=%s\n", idSala, idMensaje)

	// Si idMensaje es uuid.Nil, devolvemos todos los mensajes de la base de datos y la cola
	if idMensaje == uuid.Nil {
		log.Printf("CircularQueue: ObtenerMensajesDesdeId: Recuperando todos los mensajes para la sala [%s] porque idMensaje es UUID de ceros...", idSala)

		mensajesDesdeBD, errbd := (*q.persistencia).ObtenerMensajesDesdeSala(idSala)
		mensajesDesdeCola, errcl := q.obtenerTodosLosMensajesSinLock()
		if errbd != nil {
			log.Printf("No esxiste mensajes al obtener los mensajes de la BD para la sala [%s]: %v", idSala, errbd)
		}

		if errcl != nil {
			log.Printf("No existen mensaje al obtener los mensajes de la cola para la sala [%s]: %v", idSala, errcl)

		}

		var todosLosMensajes []Message
		if mensajesDesdeBD != nil {
			todosLosMensajes = append(todosLosMensajes, mensajesDesdeBD...)
		}
		if mensajesDesdeCola != nil {
			todosLosMensajes = append(todosLosMensajes, mensajesDesdeCola...)
		}

		log.Printf("CircularQueue: ObtenerMensajesDesdeId: Devueltos %d mensajes para la sala [%s]", len(todosLosMensajes), idSala)
		return todosLosMensajes, nil
	}

	// Si la cola está vacía, obtenemos los mensajes desde la persistencia
	if q.size == 0 {
		return q.obtenerMensajesDesdePersistencia(idSala, idMensaje)
	}

	// Buscar el mensaje en la cola
	msg, index, err := q.buscarMensajeEnColaSinLock(idMensaje)
	if err != nil {
		log.Printf("CircularQueue: ObtenerMensajesDesdeId:  El mensaje con id [%s] no se encuentra en la cola, buscando en la BD", idMensaje)
		return q.obtenerMensajeDesdePersistencia(idSala, idMensaje)
	}

	// Recopilar los mensajes desde el índice encontrado
	return q.recopilarMensajesDesdeIndice(index, msg)
}

// Función para obtener mensajes desde la persistencia
func (q *CircularQueue) obtenerMensajesDesdePersistencia(idSala uuid.UUID, idMensaje uuid.UUID) ([]Message, error) {
	if idMensaje == uuid.Nil {
		log.Println("CircularQueue:  obtenerMensajesDesdePersistencia: idMensaje es UUID de ceros, obteniendo todos los mensajes de la sala...")
		return (*q.persistencia).ObtenerMensajesDesdeSala(idSala)
	}
	log.Println("CircularQueue:  obtenerMensajesDesdePersistencia: idMensaje no es UUID de ceros, obteniendo el mensaje con el id especificado...")
	return (*q.persistencia).ObtenerMensajesDesdeSalaPorId(idSala, idMensaje)
}

func (q *CircularQueue) obtenerTodosLosMensajesSinLock() ([]Message, error) {
	// Esta función no requiere bloquear el mutex, por lo que no lo usamos aquí
	log.Println("CircularQueue: obtenerTodosLosMensajesSinLock: Recuperando todos los mensajes de la cola sin bloqueo...")

	if q.size == 0 {
		log.Println("CircularQueue: obtenerTodosLosMensajesSinLock: La cola está vacía.")
		return nil, nil
	}

	var mensajes []Message
	for i := 0; i < q.size; i++ {
		index := (q.head + i) % q.capacity // Índice circular
		mensajes = append(mensajes, q.buffer[index])
	}

	log.Printf("CircularQueue: obtenerTodosLosMensajesSinLock: Todos los mensajes en la cola: %v\n", mensajes)
	return mensajes, nil
}
func (q *CircularQueue) obtenerTodosLosMensajesConLock() ([]Message, error) {
	q.mu.Lock()         // Aseguramos el bloqueo del mutex al inicio
	defer q.mu.Unlock() // Y lo desbloqueamos al final
	log.Println("CircularQueue: obtenerTodosLosMensajesConLock: Recuperando todos los mensajes de la cola con bloqueo...")

	msg, err := q.obtenerTodosLosMensajesSinLock()
	return msg, err
}

// Función para buscar un mensaje en la cola
func (q *CircularQueue) buscarMensajeEnColaSinLock(idMensaje uuid.UUID) (Message, int, error) {
	log.Println("circularQueue: buscarMensajeEnColaSinLock: Buscando mensaje en la cola...")
	indice, existe := q.messageMap[idMensaje]
	if !existe {
		return Message{}, -1, fmt.Errorf("circularQueue: buscarMensajeEnColaSinLock: Mensaje no encontrado en la cola")
	}
	mensaje := q.buffer[(q.head+indice)%q.capacity]
	return mensaje, indice, nil
}

func (q *CircularQueue) buscarMensajeEnColaConLock(idMensaje uuid.UUID) (Message, int, error) {
	// Bloqueamos el mutex al inicio de la función
	log.Println("circularQueue: buscarMensajeEnColaConLock: Buscando mensaje en la cola...")
	q.mu.Lock()
	defer q.mu.Unlock()
	msge, n, err := q.buscarMensajeEnColaSinLock(idMensaje)
	return msge, n, err
}

// Función para obtener el mensaje desde la persistencia
func (q *CircularQueue) obtenerMensajeDesdePersistencia(idSala uuid.UUID, idMensaje uuid.UUID) ([]Message, error) {
	return q.obtenerMensajesDesdePersistencia(idSala, idMensaje)
}

func (q *CircularQueue) recopilarMensajesDesdeIndice(index int, mensaje Message) ([]Message, error) {
	// Preasignar el slice con un tamaño máximo posible (en caso de estar lleno)
	mensajes := make([]Message, 0)

	// Primero, agregar el mensaje inicial si no está vacío
	// if mensaje.MessageId != uuid.Nil {
	// mensajes = append(mensajes, mensaje)
	// }

	// Verificar cuántos mensajes válidos hay desde el índice hasta el tail sin desbordar la cola
	numMensajes := q.size - index // El número de mensajes disponibles desde el índice
	log.Printf(" CircularQueue : recopilarMensajesDesdeIndice : numMensajes:[%d] - tail:[%d]", numMensajes, q.tail)
	// Iniciar el recorrido desde el índice
	for i := 0; i < numMensajes; i++ {
		// Calcular la posición real en la cola circular
		realIndex := (index + i) % q.capacity

		// Solo agregar mensajes válidos (que no estén vacíos)
		if q.buffer[realIndex].MessageId != uuid.Nil {
			mensajes = append(mensajes, q.buffer[realIndex])
		}
	}

	// Log para los mensajes recopilados
	log.Printf("CircularQueue: recopilarMensajesDesdeIndice: %d mensajes recopilados desde el índice %d\n", len(mensajes), index)

	return mensajes, nil
}

// ObtenerElementos devuelve un slice con los mensajes almacenados en la cola.
func (q *CircularQueue) ObtenerElementos() []Message {
	var alreadyLocked bool

	q.onceQueue.Do(func() {
		q.mu.Lock()
		alreadyLocked = true
	})
	if !alreadyLocked {
		defer q.mu.Unlock()
	}

	// Si la cola está vacía, devolvemos un slice vacío.
	if q.size == 0 {
		return []Message{}
	}

	// Crear un slice para almacenar los mensajes en el orden correcto.
	elementos := make([]Message, 0, q.size)

	// Recorrer desde `head` hasta el final, manejando la circularidad.
	for i := 0; i < q.size; i++ {
		index := (q.head + i) % q.capacity
		elementos = append(elementos, q.buffer[index])
	}

	return elementos
}

// ComprobarYVisualizarMensajes verifica la cola y muestra los mensajes si existen.
func (q *CircularQueue) ComprobarYVisualizarMensajes() (bool, []Message, string) {

	// Verificar si la cola es nula
	if q == nil {
		log.Println("CircularQueue:  ComprobarYVisualizarMensajes: Error: La cola es nula.")
		return false, nil, "CircularQueue:  ComprobarYVisualizarMensajes: La cola es nula."
	}

	var alreadyLocked bool

	q.onceQueue.Do(func() {
		q.mu.Lock()
		alreadyLocked = true
	})
	if !alreadyLocked {
		defer q.mu.Unlock()
	}
	// Verificar si la cola tiene mensajes
	if q.size == 0 {
		log.Println("CircularQueue:  ComprobarYVisualizarMensajes:  La cola está vacía.")
		return false, nil, "CircularQueue:  ComprobarYVisualizarMensajes: La cola está vacía."
	}

	// Obtener y visualizar los mensajes
	mensajes := q.ObtenerElementos()
	log.Println("CircularQueue:  ComprobarYVisualizarMensajes: La cola contiene los siguientes mensajes:")
	for i, mensaje := range mensajes {
		log.Printf("CircularQueue:  ComprobarYVisualizarMensajes: Mensaje %d: %+v\n", i+1, mensaje)
	}

	return true, mensajes, "CircularQueue:  ComprobarYVisualizarMensajes: La cola contiene mensajes."
}
