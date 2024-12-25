package persistence

import (
	"backend/entities"
	"backend/utils"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Configuración de MongoDB (puede ser configurada en la propia estructura de MongoPersistencia)

// Configuración de correo

var mongoInstance *MongoPersistencia
var onceMongodb sync.Once

type MongoPersistencia struct {
	client *mongo.Client
	db     *mongo.Database
}

// NuevaMongoPersistencia crea o devuelve una instancia de MongoPersistencia con pool de conexiones
func NuevaMongoPersistencia(uri, dbName string) (*entities.Persistencia, error) {
	log.Printf("MongoPersistencia:  NuevaMongoPersistencia: Creando nueva instancia de MongoPersistencia con URI: %s y DB: %s\n", uri, dbName)
	onceMongodb.Do(func() {
		// Configura el cliente MongoDB con un pool de conexiones (máximo 10 conexiones).
		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri).SetMaxPoolSize(20))
		if err != nil {
			utils.LogCriticalError(fmt.Sprintf("MongoPersistencia:  NuevaMongoPersistencia: Error de conexión con MongoDB: %v\n", err))
			log.Printf("MongoPersistencia:  NuevaMongoPersistencia: Error de conexión con MongoDB: %v\n", err)
			return
		}
		// Verifica la conexión
		err = client.Ping(context.Background(), nil)
		if err != nil {
			log.Printf("MongoPersistencia:  NuevaMongoPersistencia: Error al hacer ping a MongoDB: %v\n", err)
			return
		}
		db := client.Database(dbName)
		mongoInstance = &MongoPersistencia{client: client, db: db}
	})

	if mongoInstance == nil {
		return nil, fmt.Errorf("MongoPersistencia:  NuevaMongoPersistencia: no se pudo crear la instancia de MongoPersistencia")
	}

	var persistenciaPersist entities.Persistencia = mongoInstance
	log.Println("MongoPersistencia:  NuevaMongoPersistencia: Instancia de MongoPersistencia creada correctamente")
	return &persistenciaPersist, nil
}

// ObtenerInstanciaDB obtiene la instancia de la base de datos
func ObtenerInstanciaDB() (*entities.Persistencia, error) {
	log.Println("MongoPersistencia:  ObtenerInstanciaDB: ObtenerInstanciaDB: Obteniendo instancia de base de datos MongoDB...")
	if mongoInstance == nil {
		// Regresamos un error si la instancia no ha sido inicializada
		return nil, errors.New("MongoPersistencia:  ObtenerInstanciaDB: ObtenerInstanciaDB: la instancia de MongoPersistencia no ha sido inicializada")
	}

	// Devolvemos mongoInstance como un tipo Persistencia, que implementa la interfaz Persistencia
	var persistenciaPersist entities.Persistencia = mongoInstance
	return &persistenciaPersist, nil
}
func (mp *MongoPersistencia) handleNoDocumentsError(err error, idSala uuid.UUID) error {
	if err == mongo.ErrNoDocuments {
		// Si no encontramos documentos, verificamos si la sala existe
		log.Printf("MongoPersistencia:  handleNoDocumentsError: No se encontraron mensajes para la sala: %v", idSala)
		// Retornar el resultado de VerificarSalaExistente
		existe, err := mp.VerificarSalaExistente(idSala)
		if err != nil {
			log.Printf("MongoPersistencia: Error al verificar si la sala existe: %v\n", err)
			return err // Retornar el error si ocurre uno en VerificarSalaExistente
		}
		if !existe {
			log.Printf("MongoPersistencia: La sala con id [%v] no existe.\n", idSala)
			return fmt.Errorf("MongoPersistencia:  handleNoDocumentsError: la sala con id %v no existe", idSala)
		} else {
			return nil
		}
	}
	log.Printf("MongoPersistencia: Error al obtener el mensaje base en la BD: %v\n", err)
	return fmt.Errorf("error al obtener el mensaje base: %v", err) // Si el error no es mongo.ErrNoDocuments, devolver el error original
}
func (mp *MongoPersistencia) ObtenerMensajesAnteriores(idMensaje uuid.UUID, cantidadRestante int) ([]entities.Message, error) {
	collection := mp.db.Collection("mensajes")

	// Buscar el mensaje con el id dado para obtener su índice
	var mensaje entities.Message
	filtro := bson.M{"idMensaje": idMensaje}
	err := collection.FindOne(context.TODO(), filtro).Decode(&mensaje)
	if err != nil {
		log.Printf("MongoPersistencia : ObtenerMensajesAnteriores: No se pudo encontrar el mensaje con id %s: %v", idMensaje, err)
		return nil, nil
	}

	// Obtener los mensajes anteriores al índice del mensaje encontrado
	indiceObjetivo := mensaje.MessageId // Asumo que hay un campo "Indice" en tu estructura que representa la posición
	filtroMensajesAnteriores := bson.M{
		"indice":    bson.M{"$lt": indiceObjetivo}, // Índices menores al índice del mensaje objetivo
		"idMensaje": bson.M{"$ne": idMensaje},      // Excluir el mensaje con el id dado
	}
	opciones := options.Find().
		SetSort(bson.M{"indice": -1}).    // Ordenar por índice descendente
		SetLimit(int64(cantidadRestante)) // Limitar la cantidad de resultados

	// Buscar los mensajes anteriores
	cursor, err := collection.Find(context.TODO(), filtroMensajesAnteriores, opciones)
	if err != nil {
		return nil, fmt.Errorf("Error al buscar mensajes anteriores en MongoDB: %v", err)
	}
	defer cursor.Close(context.TODO())

	// Decodificar los mensajes encontrados
	var mensajes []entities.Message
	for cursor.Next(context.TODO()) {
		var mensajeAnterior entities.Message
		if err := cursor.Decode(&mensajeAnterior); err != nil {
			return nil, fmt.Errorf("Error al decodificar un mensaje: %v", err)
		}
		mensajes = append(mensajes, mensajeAnterior)
	}

	// Verificar errores del cursor
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("Error en el cursor de MongoDB: %v", err)
	}

	return mensajes, nil
}

// Crea una sala con el idSala que se le pasa
func (mp *MongoPersistencia) CrearSala(sala entities.Room) error {
	salasCollection := mp.db.Collection("salas")
	_, err := salasCollection.UpdateOne(
		context.TODO(),
		bson.M{"id": sala.RoomId},
		bson.D{
			{Key: "$set", Value: sala}, // Esto usa claves etiquetadas
		},
		options.Update().SetUpsert(true),
	)
	return err
}

// Implementa GuardarSala
func (m *MongoPersistencia) GuardarSala(sala entities.Room) error {
	log.Printf("MongoPersistencia:  GuardarSala: Guardando sala: %+v\n", sala)
	collection := m.db.Collection("salas")
	_, err := collection.InsertOne(context.TODO(), sala)
	if err != nil {
		log.Printf("MongoPersistencia:  GuardarSala: Error al guardar la sala: %v\n", err)
	}
	return err
}

// Implementa ObtenerSala
func (m *MongoPersistencia) ObtenerSala(id uuid.UUID) (entities.Room, error) {
	log.Printf("MongoPersistencia:  ObtenerSala: Obteniendo sala con ID: %s\n", id)
	var sala entities.Room
	collection := m.db.Collection("salas")
	err := collection.FindOne(context.TODO(), bson.M{"id": id}).Decode(&sala)
	if err != nil {
		log.Printf("MongoPersistencia:  ObtenerSala: Error al obtener la sala: %v\n", err)
		return entities.Room{}, err
	}
	log.Printf("MongoPersistencia:  ObtenerSala: Sala obtenida: %+v\n", sala)
	return sala, nil
}

// GuardarMensaje guarda un solo mensaje en la base de datos MongoDB
func (mp *MongoPersistencia) GuardarMensaje(mensaje *entities.Message) error {
	log.Printf("MongoPersistencia:  GuardarMensaje: Guardando mensaje: %+v\n", mensaje)
	collection := mp.db.Collection("mensajes")
	_, err := collection.InsertOne(context.TODO(), mensaje)
	if err != nil {
		log.Printf("MongoPersistencia:  GuardarMensaje: Error al guardar el mensaje en MongoDB: %v\n", err)
		return fmt.Errorf("MongoPersistencia:  GuardarMensaje: error al guardar el mensaje en MongoDB: %v", err)
	}
	log.Println("MongoPersistencia:  GuardarMensaje: Mensaje guardado correctamente")
	return nil
}

// GuardarMensajesEnBaseDeDatos guarda una lista de mensajes en MongoDB
func (mp *MongoPersistencia) GuardarMensajesEnBaseDeDatos(mensajes []entities.Message, roomId uuid.UUID) error {
	log.Printf("MongoPersistencia:  GuardarMensajesEnBaseDeDatos: Guardando lista de %d mensajes\n", len(mensajes))
	collection := mp.db.Collection("mensajes")

	// Convertimos los mensajes a una lista de interfaces{}
	var documentos []interface{}
	for _, mensaje := range mensajes {
		documentos = append(documentos, bson.D{
			{Key: "nick_usuario", Value: mensaje.Nickname},
			{Key: "fecha_envio", Value: mensaje.SendDate},
			{Key: "id_sala", Value: roomId},
			{Key: "nombre_sala", Value: mensaje.RoomName},
		})
	}

	// Insertamos todos los documentos en la colección
	_, err := collection.InsertMany(context.TODO(), documentos)
	if err != nil {
		log.Printf("MongoPersistencia:  GuardarMensajesEnBaseDeDatos: Error al guardar los mensajes en MongoDB: %v\n", err)
		return fmt.Errorf("MongoPersistencia:  GuardarMensajesEnBaseDeDatos: error al guardar los mensajes en MongoDB: %v", err)
	}

	// Todo salió bien
	log.Println("MongoPersistencia:  GuardarMensajesEnBaseDeDatos: Mensajes guardados correctamente")
	return nil
}

func (mp *MongoPersistencia) ObtenerMensajesDesdeSala(idSala uuid.UUID) ([]entities.Message, error) {
	log.Printf("MongoPersistencia: ObtenerMensajesDesdeSala: Buscando mensajes para la sala [%s]", idSala)

	collection := mp.db.Collection("mensajes")

	// Filtro para buscar todos los mensajes de la sala con id_sala
	filter := bson.D{{Key: "id_sala", Value: idSala}}

	// Buscar todos los mensajes con ese id_sala
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		log.Printf("MongoPersistencia: ObtenerMensajesDesdeSala: Error al buscar los mensajes en MongoDB: %v\n", err)
		return nil, fmt.Errorf("Error al buscar mensajes en MongoDB: %v", err)
	}
	defer cursor.Close(context.TODO())

	// Iterar sobre los documentos devueltos y mapearlos a una lista de entidades Message
	var mensajes []entities.Message
	for cursor.Next(context.TODO()) {
		var mensaje entities.Message
		if err := cursor.Decode(&mensaje); err != nil {
			log.Printf("MongoPersistencia: ObtenerMensajesDesdeSala: Error al decodificar el mensaje: %v", err)
			continue
		}
		mensajes = append(mensajes, mensaje)
	}

	// Verificamos si hubo errores en el cursor
	if err := cursor.Err(); err != nil {
		log.Printf("MongoPersistencia: ObtenerMensajesDesdeSala: Error durante la iteración del cursor: %v", err)
		return nil, err
	}

	// Devolver la lista de mensajes
	log.Printf("MongoPersistencia: ObtenerMensajesDesdeSala: Se encontraron %d mensajes para la sala [%s]", len(mensajes), idSala)
	return mensajes, nil
}

func (mp *MongoPersistencia) VerificarSalaExistente(idSala uuid.UUID) (bool, error) {
	log.Printf("MongoPersistencia: VerificarSalaExistente: Verificando si la sala existe con ID: %s\n", idSala)

	// Acceso a la colección de salas
	collection := mp.db.Collection("salas")
	filter := bson.M{"_id": idSala}

	var result bson.M
	err := collection.FindOne(context.TODO(), filter).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("MongoPersistencia: VerificarSalaExistente: La sala con ID %s no existe.\n", idSala)
			return false, nil // La sala no existe
		}
		log.Printf("MongoPersistencia: VerificarSalaExistente: Error al buscar la sala: %v\n", err)
		return false, err // Otro error
	}

	log.Printf("MongoPersistencia: VerificarSalaExistente: La sala con ID %s existe.\n", idSala)
	return true, nil // La sala existe
}
func (mp *MongoPersistencia) ObtenerMensajesDesdeSalaPorId(idSala uuid.UUID, idMensaje uuid.UUID) ([]entities.Message, error) {
	log.Printf("MongoPersistencia:  ObtenerMensajesDesdeId: Obteniendo mensajes desde la sala con ID: %s y mensaje con ID: %s\n", idSala, idMensaje)
	collection := mp.db.Collection("mensajes")

	// Buscar el mensaje base
	var mensajeBase entities.Message
	filterBase := bson.D{{Key: "id_mensaje", Value: idMensaje}, {Key: "id_sala", Value: idSala}}
	err := collection.FindOne(context.TODO(), filterBase).Decode(&mensajeBase)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("MongoPersistencia: ObtenerMensajesDesdeSalaPorId: No se encontró el mensaje con id_mensaje: %v en la sala: %v\n", idMensaje, idSala)

			// Verificar si la sala existe
			existeSala, err := mp.VerificarSalaExistente(idSala)
			if err != nil {
				log.Printf("MongoPersistencia: ObtenerMensajesDesdeSalaPorId: Error al verificar si la sala existe: %v\n", err)
				return nil, fmt.Errorf("error al verificar si la sala existe: %v", err)
			}

			if !existeSala {
				log.Printf("MongoPersistencia: ObtenerMensajesDesdeSalaPorId: La sala con id [%v] no existe. Creando nueva sala...\n", idSala)

				// Crear la nueva sala en la BD
				nuevaSala := entities.Room{RoomId: idSala}
				err := mp.CrearSala(nuevaSala) // Método para crear la sala en la colección de salas
				if err != nil {
					log.Printf("MongoPersistencia: ObtenerMensajesDesdeSalaPorId: Error al crear la nueva sala: %v\n", err)
					return nil, fmt.Errorf("error al crear la nueva sala: %v", err)
				}

				log.Printf("MongoPersistencia: ObtenerMensajesDesdeSalaPorId: Se ha creado una nueva sala con id [%v] sin mensajes.\n", idSala)

				// Devolver nil ya que no hay mensajes
				return nil, nil
			}

			// Si la sala existe pero no tiene mensajes
			log.Printf("MongoPersistencia: La sala con id [%v] existe pero no tiene mensajes.\n", idSala)
			return nil, nil
		}

		// Error al intentar obtener el mensaje base
		log.Printf("MongoPersistencia: Error al obtener el mensaje base en la BD: %v\n", err)
		return nil, fmt.Errorf("error al obtener el mensaje base: %v", err)
	}
	// Filtro para obtener mensajes posteriores al idMensaje
	filter := bson.D{
		{Key: "id_sala", Value: idSala},
		{Key: "fecha_envio", Value: bson.D{{Key: "$gte", Value: mensajeBase.SendDate}}}, // Asegurarse de que los mensajes sean posteriores a fecha_envio del idMensaje
	}

	// Configuración para ordenar los mensajes por fecha de envío, con el más antiguo primero.
	findOptions := options.Find().SetSort(bson.D{{Key: "fecha_envio", Value: 1}}) // Ascendente, el más antiguo primero.

	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		log.Printf("MongoPersistencia:  ObtenerMensajesDesdeId: Error al ejecutar la consulta: %v\n", err)
		return nil, fmt.Errorf("error al ejecutar la consulta: %v", err)
	}
	defer cursor.Close(context.TODO())

	var mensajes []entities.Message
	// Iterar sobre los mensajes obtenidos en el cursor
	for cursor.Next(context.TODO()) {
		var mensaje entities.Message
		// Decodificar cada mensaje en la estructura adecuada
		if err := cursor.Decode(&mensaje); err != nil {
			log.Printf("MongoPersistencia:  ObtenerMensajesDesdeId: Error al decodificar el mensaje: %v\n", err)
			return nil, fmt.Errorf("error al decodificar el mensaje: %v", err)
		}
		// Agregar el mensaje a la lista de mensajes
		mensajes = append(mensajes, mensaje)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("MongoPersistencia:  ObtenerMensajesDesdeId: Error al iterar sobre el cursor: %v\n", err)
		return nil, fmt.Errorf("error al iterar sobre el cursor: %v", err)
	}

	// Loguear la cantidad de mensajes obtenidos
	log.Printf("MongoPersistencia:  ObtenerMensajesDesdeId: Mensajes obtenidos después del ID de mensaje: %d\n", len(mensajes))

	// Retornar la lista de mensajes obtenidos
	return mensajes, nil
}
func (mp *MongoPersistencia) GuardarUsuario(usuario *entities.User) error {
	log.Printf("MongoPersistencia:  GuardarUsuario: Iniciando el guardado del usuario con ID: %s\n", usuario.UserId)

	// Crear un documento BSON a partir del usuario
	documento := bson.D{
		{Key: "id_usuario", Value: usuario.UserId},
		{Key: "nickname", Value: usuario.Nickname},
		{Key: "token", Value: usuario.Token},
		{Key: "hora_ultima_accion", Value: usuario.LastActionTime},
		{Key: "estado", Value: usuario.State},
		{Key: "tipo", Value: usuario.Type},
		{Key: "idsala", Value: usuario.RoomId},     // La sala puede ser nil
		{Key: "namesala", Value: usuario.RoomName}, // La sala puede ser nil
	}
	log.Printf("MongoPersistencia:  GuardarUsuario: Documento BSON para guardar usuario: %+v\n", documento)

	// Insertar el documento en la colección
	collection := mp.db.Collection("usuarios")
	log.Printf("MongoPersistencia:  GuardarUsuario: Colección recogida de usuarios: %+v\n", documento)
	_, err := collection.InsertOne(context.Background(), documento)
	if err != nil {
		log.Printf("MongoPersistencia:  GuardarUsuario: Error al guardar el usuario en MongoDB: %v\n", err)
		return fmt.Errorf("MongoPersistencia:  GuardarUsuario: error al guardar el usuario en MongoDB: %v", err)
	}

	// Confirmación de éxito
	log.Printf("MongoPersistencia:  GuardarUsuario: Usuario con ID: %s guardado correctamente en MongoDB\n", usuario.UserId)
	return nil
}
