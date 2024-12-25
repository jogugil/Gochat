package services

import (
	"backend/entities"
	"backend/models"
	"backend/persistence"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
)

type GestionSalas struct {
	SalasFijas    map[uuid.UUID]*entities.Room // Mapa de salas fijas en memoria
	SalaPrincipal *entities.Room               // Sala principal única
	mu            sync.RWMutex                 // Mutex lectura/excritura
	persistencia  *entities.Persistencia       // Persistencia para almacenar los datos
	once          sync.Once                    // Para garantizar inicialización única
}

var instancia *GestionSalas // Instancia única de GestionSalas

// NuevaGestionSalas crea y devuelve una instancia Singleton de GestionSalas

func NuevaGestionSalas(persistencia *entities.Persistencia, configFile string) *GestionSalas {
	// Asegura que solo se cree una instancia
	if instancia == nil {
		instancia = &GestionSalas{
			SalasFijas:   make(map[uuid.UUID]*entities.Room), // Inicialización del mapa
			persistencia: persistencia,                       // Asignación de persistencia
		}
	}
	// Configuración y carga de datos solo una vez
	instancia.once.Do(func() {
		log.Println("GestionSalas:  NuevaGestionSalas:  Iniciando la configuración de la instancia.")
		if configFile != "" {
			err := instancia.CargarSalasFijasDesdeArchivo(configFile)
			if err != nil {
				log.Fatalf("GestionSalas:  NuevaGestionSalas: Error al cargar configuración: %v", err)
			}
		}
		log.Println("GestionSalas:  NuevaGestionSalas: Configuración completada.")
	})
	return instancia
}

func (sm *GestionSalas) CargarSalasFijasDesdeArchivo(configFile string) error {
	log.Printf("GestionSalas: CargarSalasFijasDesdeArchivo: Cargando salas desde archivo: %s", configFile)
	if sm.SalasFijas == nil {
		sm.SalasFijas = make(map[uuid.UUID]*entities.Room)
	}
	var persis, err = persistence.ObtenerInstanciaDB()

	if err != nil {
		log.Printf("GestionSalas: CargarSalasFijasDesdeArchivo: Error al crear la instancia de MongoPersistencia:%v", err)
		return err
	}

	sm.SalaPrincipal = &entities.Room{
		RoomId:         uuid.New(),
		RoomName:       "Sala Principal",
		RoomType:       "Principal",
		MessageHistory: entities.NewCircularQueue(persis),
	}

	file, err := os.Open(configFile)
	if err != nil {
		log.Printf("GestionSalas: CargarSalasFijasDesdeArchivo: Error al abrir archivo: %v", err)
		return err
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		log.Printf("GestionSalas: CargarSalasFijasDesdeArchivo: Error al leer archivo: %v", err)
		return err
	}

	var salas []map[string]interface{}
	err = json.Unmarshal(byteValue, &salas)
	if err != nil {
		log.Printf("GestionSalas: CargarSalasFijasDesdeArchivo: Error al parsear JSON: %v", err)
		return err
	}

	// Usar RLock para leer de SalasFijas de forma concurrente.
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, salaData := range salas {
		salaID, err_par := uuid.Parse(salaData["id"].(string))
		if err_par != nil {
			log.Printf("GestionSalas: CargarSalasFijasDesdeArchivo: Error al parsear ID: %v", err_par)
			return err_par
		}
		var persis, err_per = persistence.ObtenerInstanciaDB()

		if err_per != nil {
			log.Printf("GestionSalas: Error al crear la instancia de MongoPersistencia:%v", err_per)
			return err_per
		}
		sala := &entities.Room{
			RoomId:         salaID,
			RoomName:       salaData["nombre"].(string),
			RoomType:       "Fija",
			MessageHistory: entities.NewCircularQueue(persis),
		}
		sm.SalasFijas[salaID] = sala
	}
	log.Println("GestionSalas: CargarSalasFijasDesdeArchivo: Carga completada.")
	return nil
}

func (sm *GestionSalas) CrearSalaTemporal(nombre string) *entities.Room {
	log.Printf("GestionSalas: CrearSalaTemporal: Creando sala temporal con nombre: %s", nombre)
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.persistencia == nil {
		log.Fatal("GestionSalas: CrearSalaTemporal: Persistencia no inicializada.")
	}

	sala := &entities.Room{
		RoomId:         uuid.New(),
		RoomName:       nombre,
		RoomType:       "Temporal",
		MessageHistory: entities.NewCircularQueue(sm.persistencia),
	}
	log.Printf("GestionSalas: CrearSalaTemporal: Sala temporal creada con ID: %s", sala.RoomId)
	return sala
}

func (sm *GestionSalas) ObtenerSalaPorID(idSala uuid.UUID) (*entities.Room, error) {
	log.Printf("GestionSalas: ObtenerSalaPorID: Buscando sala con ID: %s", idSala)

	sm.mu.RLock()
	log.Printf("GestionSalas: ObtenerSalaPorID: Lock adquirido para lectura de la sala con ID %s", idSala)
	defer func() {
		sm.mu.RUnlock()
		log.Printf("GestionSalas: ObtenerSalaPorID: Lock liberado para lectura de la sala con ID %s", idSala)
	}()

	if sm.SalaPrincipal != nil && sm.SalaPrincipal.RoomId == idSala {
		log.Println("GestionSalas: ObtenerSalaPorID: Sala principal encontrada.")
		return sm.SalaPrincipal, nil
	}

	if sala, existe := sm.SalasFijas[idSala]; existe {
		log.Printf("GestionSalas: ObtenerSalaPorID: Sala fija encontrada con ID: %s", idSala)
		return sala, nil
	}

	log.Printf("GestionSalas: ObtenerSalaPorID: Sala no encontrada con ID: %s", idSala)
	return nil, fmt.Errorf("la sala con ID %s no existe", idSala)
}

func (sm *GestionSalas) EnviarMensaje(idSala uuid.UUID, nickname, mensaje string, usuario entities.User) error {
	log.Printf("GestionSalas: EnviarMensaje: Enviando mensaje a la sala con ID: %s", idSala)

	// Primero obtenemos la sala con RLock, ya que estamos haciendo una lectura
	sala, err := sm.ObtenerSalaPorID(idSala)
	if err != nil {
		log.Printf("GestionSalas: EnviarMensaje: Error al obtener sala: %v", err)
		return err
	}

	nuevoMensaje := models.CrearMensajeConFecha(mensaje, usuario, idSala, sala.RoomName, usuario.LastActionTime)
	log.Printf("GestionSalas: Nuevo mensaje creado: %+v\n", nuevoMensaje)

	// Modificamos el historial de mensajes
	sala.MessageHistory.Enqueue(*nuevoMensaje, sala.RoomId)
	log.Printf("GestionSalas: EnviarMensaje: Mensaje enviado a la sala con ID: %s - %v \n", idSala, nuevoMensaje)
	return nil
}

// Función para obtener mensajes de una sala
func (sm *GestionSalas) ObtenerMensajesDesdeId(idSala uuid.UUID, idMensaje uuid.UUID) ([]entities.Message, error) {
	log.Printf("GestionSalas: ObtenerMensajesDesdeId: Obteniendo mensajes desde ID %s en la sala %s", idMensaje, idSala)
	sm.mu.RLock() // Lectura, se puede hacer concurrentemente
	defer sm.mu.RUnlock()

	sala, err := sm.ObtenerSalaPorID(idSala)
	if err != nil {
		log.Printf("GestionSalas: ObtenerMensajesDesdeId: Error al obtener sala: %v", err)
		return nil, fmt.Errorf("la sala con ID %s no existe", idSala)
	}
	if sala == nil {
		log.Printf("GestionSalas: ObtenerMensajesDesdeId: Error al obtener sala: sala es nil")
		return nil, fmt.Errorf("GestionSalas: sala no encontrada con ID: %s", idSala)
	}
	cola := sala.MessageHistory
	// Verificar si la cola es nula
	if cola == nil {
		log.Println("GestionSalas: ObtenerMensajesDesdeId: Error: La cola es nula.")
		return nil, fmt.Errorf("GestionSalas: ObtenerMensajesDesdeId: la cola es nula en la sala con ID %s ", idSala)
	}

	mensajes, err := cola.ObtenerMensajesDesdeId(idSala, idMensaje)
	if err != nil {
		log.Printf("GestionSalas: ObtenerMensajesDesdeId: Error al obtener mensajes: %v", err)
		return nil, fmt.Errorf("GestionSalas: error en ObtenerPorUltimoIdMensaje: %v", err)
	}

	log.Printf("GestionSalas: ObtenerMensajesDesdeId: Mensajes obtenidos correctamente (%d mensajes)", len(mensajes))
	return mensajes, nil
}

// Función para obtener todos los mensajes de una sala
func (sm *GestionSalas) ObtenerMensajes(idSala uuid.UUID) ([]entities.Message, error) {
	log.Printf("GestionSalas: ObtenerMensajes: Obteniendo todos los mensajes de la sala %s", idSala)
	sm.mu.RLock() // Lectura, se puede hacer concurrentemente
	defer sm.mu.RUnlock()

	sala, err := sm.ObtenerSalaPorID(idSala)
	if err != nil {
		log.Printf("GestionSalas: ObtenerMensajes: Error al obtener sala: %v", err)
		return nil, fmt.Errorf("la sala con ID %s no existe", idSala)
	}
	cola := sala.MessageHistory
	mensajes, err := cola.ObtenerTodos(idSala)
	if err != nil {
		log.Printf("GestionSalas: ObtenerMensajes: Error al obtener mensajes: %v", err)
		return nil, fmt.Errorf("GestionSalas: error en ObtenerMensajes: %v", err)
	}

	log.Printf("GestionSalas: ObtenerMensajes: Mensajes obtenidos correctamente (%d mensajes)", len(mensajes))
	return mensajes, nil
}

func (sm *GestionSalas) ObtenerMensajesCantidad(idSala uuid.UUID, idMensaje uuid.UUID, cantidad int) ([]entities.Message, error) {
	log.Printf("GestionSalas: ObtenerMensajesCantidad: Obteniendo %d mensajes de la sala %s", cantidad, idSala)
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validación previa
	sala, err := sm.ObtenerSalaPorID(idSala)
	if err != nil {
		log.Printf("GestionSalas: ObtenerMensajesCantidad: Error al obtener sala: %v", err)
		return nil, fmt.Errorf("la sala con ID %s no existe", idSala)
	}

	// Mejorar rendimiento si la cantidad es mayor que los mensajes disponibles
	cola := sala.MessageHistory
	mensajes, err := cola.ObtenerMensajes(idSala, idMensaje, cantidad)
	if err != nil {
		log.Printf("GestionSalas: ObtenerMensajesCantidad: Error al obtener mensajes: %v", err)
		return nil, fmt.Errorf("error en ObtenerMensajes: %v", err)
	}

	log.Printf("GestionSalas: ObtenerMensajesCantidad: Mensajes obtenidos correctamente (%d mensajes)", len(mensajes))
	return mensajes, nil
}

func (sm *GestionSalas) UnirseASala(usuarioId string, salaFijaID string) (*entities.Room, error) {
	log.Printf("GestionSalas: UnirseASala: Usuario %s intentando unirse a la sala fija %s", usuarioId, salaFijaID)

	if salaFijaID != "" {
		salaUUID, err := uuid.Parse(salaFijaID)
		if err != nil {
			log.Printf("GestionSalas: UnirseASala: Error al parsear ID de sala fija: %v", err)
			return nil, fmt.Errorf("el ID de la sala fija %s no es un UUID válido: %v", salaFijaID, err)
		}

		// Mejorar la búsqueda con un "if" temprano
		sala, existe := sm.SalasFijas[salaUUID]
		if !existe {
			log.Printf("GestionSalas: UnirseASala: Sala fija no encontrada con ID: %s", salaFijaID)
			return nil, fmt.Errorf("la sala fija con ID %s no existe", salaFijaID)
		}

		log.Printf("GestionSalas: UnirseASala: Usuario %s unido a la sala fija %s", usuarioId, salaFijaID)
		return sala, nil
	}

	log.Printf("GestionSalas: UnirseASala: Usuario %s unido a la sala principal", usuarioId)
	return sm.SalaPrincipal, nil
}

func (gestion *GestionSalas) ObtenerTodosLosMensajes(idSala uuid.UUID) ([]entities.Message, error) {
	log.Printf("GestionSalas: ObtenerTodosLosMensajes: Obteniendo todos los mensajes únicos de la sala %s", idSala)

	if gestion.persistencia == nil {
		log.Println("GestionSalas: ObtenerTodosLosMensajes: Persistencia no inicializada")
		return nil, fmt.Errorf("persistencia no inicializada")
	}

	mensajesBD, err := (*gestion.persistencia).ObtenerMensajesDesdeSala(idSala)
	if err != nil {
		log.Printf("GestionSalas: ObtenerTodosLosMensajes: Error al obtener mensajes de la base de datos: %v", err)
		return nil, fmt.Errorf("error al obtener mensajes de la base de datos: %v", err)
	}

	sala, err := gestion.ObtenerSalaPorID(idSala)
	if err != nil {
		log.Printf("GestionSalas: ObtenerTodosLosMensajes: Error al obtener sala: %v", err)
		return nil, fmt.Errorf("la sala con ID %s no existe", idSala)
	}

	mensajesCola := sala.MessageHistory.ObtenerElementos()
	uniqueMessages := make(map[uuid.UUID]entities.Message)

	// Usando un map para eliminar duplicados
	for _, mensaje := range mensajesCola {
		uniqueMessages[mensaje.MessageId] = mensaje
	}
	for _, mensaje := range mensajesBD {
		uniqueMessages[mensaje.MessageId] = mensaje
	}

	var mensajesUnicos []entities.Message
	for _, mensaje := range uniqueMessages {
		mensajesUnicos = append(mensajesUnicos, mensaje)
	}

	log.Printf("GestionSalas: ObtenerTodosLosMensajes: Mensajes únicos obtenidos (%d mensajes)", len(mensajesUnicos))
	return mensajesUnicos, nil
}

func (gestion *GestionSalas) ObtenerMensajesDesdeCola(idSala uuid.UUID) ([]entities.Message, error) {
	log.Printf("GestionSalas: ObtenerMensajesDesdeCola: Obteniendo mensajes desde la cola de la sala %s", idSala)

	sala, err := gestion.ObtenerSalaPorID(idSala)
	if err != nil {
		log.Printf("GestionSalas: ObtenerMensajesDesdeCola: Error al obtener sala: %v", err)
		return nil, fmt.Errorf("la sala con ID %s no existe", idSala)
	}

	mensajesCola := sala.MessageHistory.ObtenerElementos()
	log.Printf("GestionSalas: ObtenerMensajesDesdeCola: Mensajes obtenidos desde la cola (%d mensajes)", len(mensajesCola))
	return mensajesCola, nil
}
