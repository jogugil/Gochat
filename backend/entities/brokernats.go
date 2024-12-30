package entities

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// BrokerNats representa un broker basado en NATS con soporte para JetStream.
// Me guardio un vectior de subscripociones para en un futuro poder añadir funciones de administacion de topics
type BrokerNats struct {
	conn    *nats.Conn             // Conexión de NATS
	js      nats.JetStreamContext  // Contexto para operaciones con JetStream
	metrics sync.Map               // Para las métricas
	adapter NatsTransformer        // Transformador Nats (si aplica)
	config  map[string]interface{} // Configuración
}

// NewNatsBroker inicializa un nuevo BrokerNats con JetStream.
func NewNatsBroker(config map[string]interface{}) (MessageBroker, error) {
	// Verificar que "nats" existe en la configuración
	natsConfig, ok := config["nats"]
	if !ok {
		return nil, fmt.Errorf("BrokerNats: NewNatsBroker: 'nats' no está presente en la configuración")
	}

	// Obtener la clave "urls" dentro de la configuración de "nats"
	natsConfigMap, ok := natsConfig.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("BrokerNats: NewNatsBroker: 'nats' no es un mapa válido")
	}

	urlsInterface, ok := natsConfigMap["urls"]
	if !ok {
		return nil, fmt.Errorf("BrokerNats: NewNatsBroker: 'urls' no está presente en la configuración de NATS")
	}

	// Verificar que "urls" sea una lista de cadenas
	urls, ok := urlsInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("BrokerNats: NewNatsBroker: 'urls' no es una lista válida")
	}

	// Conectar a NATS
	url, ok := urls[0].(string)
	if !ok {
		return nil, fmt.Errorf("BrokerNats: NewNatsBroker: la URL proporcionada no es una cadena válida")
	}

	// Conectar a NATS
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("BrokerNats: NewNatsBroker: error al conectar a NATS:=> %w -- url:[%s]", err, url)
	}

	// Inicializar JetStream
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("BrokerNats: NewNatsBroker: error al inicializar JetStream: %w", err)
	}

	// Obtener todos los topics
	topics, err := getTopics(config)
	if err != nil {
		return nil, fmt.Errorf("BrokerNats: NewNatsBroker: error al obtener los topics: %w", err)
	}

	// Crear el stream con todos los topics
	err = createStream(js, topics)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Crear consumidores y productores para cada topic
	var consumerName = "main-client-consumer"
	var streamName = "MYGOCHAT_STREAM"
	for _, topic := range topics {
		// Publicar un mensaje vacío por ahora
		err := createProducer(js, topic, []byte{})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("BrokerNats: error al crear el productor para el tópico %s: %w", topic, err)
		}

		err = createConsumer(js, streamName, consumerName, topic)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("BrokerNats: error al crear el consumidor para el tópico %s: %w", topic, err)
		}
	}

	// Retornar instancia del broker
	return &BrokerNats{
		conn:    conn,
		metrics: sync.Map{},
		adapter: NatsTransformer{},
		js:      js,
		config:  config,
	}, nil
}

// Crear un consumidor para un tópico específico
func createConsumer(js nats.JetStreamContext, streamName, consumerName, subject string) error {
	// Crear el consumidor con los parámetros proporcionados
	_, err := js.AddConsumer(streamName, &nats.ConsumerConfig{
		Durable:        consumerName,
		DeliverSubject: subject, // El nombre del topic
		AckPolicy:      nats.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("error al crear el consumidor: %w", err)
	}

	log.Printf("Consumidor '%s' creado exitosamente para el stream '%s' con subject '%s'", consumerName, streamName, subject)
	return nil
}

// Crear un productor para un tópico específico
func createProducer(js nats.JetStreamContext, topic string, message []byte) error {
	// Publicar un mensaje en el tópico
	_, err := js.Publish(topic, message)
	if err != nil {
		return fmt.Errorf("BrokerNats: createProducer: error al publicar en el tópico '%s': %w", topic, err)
	}
	return nil
}

func deleteConflictingStreams(js nats.JetStreamContext, topics []string) error {
	streams := js.StreamNames()
	if streams == nil {
		return fmt.Errorf("error al obtener los nombres de los streams: %v", streams)
	}

	for stream := range streams {
		info, err := js.StreamInfo(stream)
		if err != nil {
			log.Printf("No se pudo obtener información del stream %s: %v", stream, err)
			continue
		}

		for _, subject := range info.Config.Subjects {
			for _, topic := range topics {
				if subject == topic {
					log.Printf("Eliminando stream conflictivo: %s (Subjects: %v)", stream, info.Config.Subjects)
					if err := js.DeleteStream(stream); err != nil {
						return fmt.Errorf("error al eliminar el stream %s: %w", stream, err)
					}
					break
				}
			}
		}
	}
	return nil
}

func listStreams(js nats.JetStreamContext) {
	streams := js.StreamNames()
	if streams == nil {
		log.Fatalf("BrokerNats: listStreams: Error al obtener los nombres de los streams: %v", streams)
	}

	log.Println("BrokerNats: listStreams: Streams existentes en JetStream:")
	for stream := range streams {
		info, err := js.StreamInfo(stream)
		if err != nil {
			log.Printf("BrokerNats: listStreams: Error al obtener información del stream %s: %v", stream, err)
			continue
		}

		log.Printf("BrokerNats: listStreams: Stream: %s, Subjects: %v", info.Config.Name, info.Config.Subjects)
	}
}
func checkTopicConflicts(js nats.JetStreamContext, topics []string) ([]string, error) {
	conflictingStreams := []string{}

	// Obtener los nombres de todos los streams
	streamNames := js.StreamNames()
	if streamNames == nil {
		return nil, fmt.Errorf("error al obtener los nombres de los streams")
	}

	// Verificar cada stream
	for streamName := range streamNames {
		info, err := js.StreamInfo(streamName)
		if err != nil {
			log.Printf("No se pudo obtener información del stream %s: %v", streamName, err)
			continue
		}

		// Comparar los subjects del stream con los topics
		for _, subject := range info.Config.Subjects {
			for _, topic := range topics {
				if subject == topic {
					log.Printf("Conflicto encontrado: Topic '%s' ya está en el stream '%s'", topic, streamName)
					conflictingStreams = append(conflictingStreams, streamName)
				}
			}
		}
	}
	return conflictingStreams, nil
}

// Crear el stream con todos los topics. si existe, lo actualizamos
// Crear el stream con todos los topics
func createStream(js nats.JetStreamContext, topics []string) error {
	// Verificar conflictos de topics
	conflictingStreams, err := checkTopicConflicts(js, topics)
	if err != nil {
		return fmt.Errorf("BrokerNats: createStream: error al verificar conflictos de topics: %w", err)
	}

	// Manejar los conflictos (opcional: eliminar streams conflictivos)
	if len(conflictingStreams) > 0 {
		log.Printf("BrokerNats: createStream: conflictos encontrados en streams: %v", conflictingStreams)
		// Aquí puedes decidir eliminar los streams conflictivos
		for _, stream := range conflictingStreams {
			err := js.DeleteStream(stream)
			if err != nil {
				return fmt.Errorf("BrokerNats: createStream: error al eliminar stream conflictivo %s: %w", stream, err)
			}
			log.Printf("Stream conflictivo eliminado: %s", stream)
		}
	}

	// Crear el stream después de resolver conflictos
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "MYGOCHAT_STREAM",
		Subjects: topics,
		Storage:  nats.FileStorage,
		Replicas: 1,
	})
	if err != nil {
		return fmt.Errorf("BrokerNats: createStream: error al crear el stream: %w", err)
	}

	log.Println("BrokerNats: createStream: stream creado exitosamente")
	return nil
}

// Función para obtener todos los topics a partir de la configuración
func getTopics(config map[string]interface{}) ([]string, error) {
	var topics []string

	// Extraer los topics de "mainroom" y "operations"
	goChatConfig, ok := config["gochat"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("BrokerNats: getTopics: 'gochat' no es un mapa válido")
	}

	mainroomConfig, ok := goChatConfig["mainroom"].(map[string]interface{})
	if ok {
		// Agregar topics de mainroom
		if serverTopic, ok := mainroomConfig["server_topic"].(string); ok {
			topics = append(topics, serverTopic)
		}
		if clientTopic, ok := mainroomConfig["client_topic"].(string); ok {
			topics = append(topics, clientTopic)
		}
	}

	operationsConfig, ok := goChatConfig["operations"].(map[string]interface{})
	if ok {
		// Agregar topics de operations
		if getUsersTopic, ok := operationsConfig["get_users"].(string); ok {
			topics = append(topics, getUsersTopic)
		}
		if getMessagesTopic, ok := operationsConfig["get_messages"].(string); ok {
			topics = append(topics, getMessagesTopic)
		}
	}

	// Extraer los topics de las salas
	salasConfig, ok := config["salas"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("BrokerNats: getTopics: 'salas' no es un array válido")
	}

	for _, sala := range salasConfig {
		salaConfig, ok := sala.(map[string]interface{})
		if ok {
			if serverTopic, ok := salaConfig["server_topic"].(string); ok {
				topics = append(topics, serverTopic)
			}
			if clientTopic, ok := salaConfig["client_topic"].(string); ok {
				topics = append(topics, clientTopic)
			}
		}
	}

	return topics, nil
}
func (b *BrokerNats) OnMessage(topic string, callback func(interface{})) error {
	// Suscripción al topic
	_, err := b.conn.Subscribe(topic, func(m *nats.Msg) {

		// Construir el mensaje NATS personalizado
		log.Printf("BrokerNats: OnMessage:  m.Subject: %s \n", m.Subject)
		natsMsg := &NatsMessage{
			Subject: m.Subject,
			Data:    m.Data,
		}
		log.Printf("BrokerNats: OnMessage:  natsMsg.Subject: %s \n", natsMsg.Subject)
		log.Printf("BrokerNats: OnMessage:  natsMsg.Data: %s \n", natsMsg.Data)

		message, err := b.adapter.TransformFromExternal(natsMsg.Data)

		if err != nil {
			// Manejar el error si es necesario
			fmt.Println("Error transformando el mensaje:", err)
			return
		}
		log.Printf("BrokerNats: OnMessage:  message: %v\n", message)
		// Llamar al callback con el mensaje deserializado
		callback(message)
	})

	return err
}

func (b *BrokerNats) OnGetUsers(topic string, callback func(interface{})) error {
	// Suscripción al topic
	_, err := b.conn.Subscribe(topic, func(m *nats.Msg) {

		// Construir el mensaje NATS personalizado
		log.Printf("BrokerNats: OnGetUsers:  m.Subject: %s \n", m.Subject)
		natsMsg := &NatsMessage{
			Subject: m.Subject,
			Data:    m.Data,
		}
		log.Printf("BrokerNats: OnGetUsers:  natsMsg.Subject: %s \n", natsMsg.Subject)
		log.Printf("BrokerNats: OnGetUsers:  natsMsg.Data: %s \n", natsMsg.Data)

		message, err := b.adapter.TransformFromExternalToGetUsers(natsMsg.Data)

		if err != nil {
			// Manejar el error si es necesario
			fmt.Println("Error transformando el mensaje:", err)
			return
		}
		log.Printf("BrokerNats: OnGetUsers:  message: %v\n", message)
		// Llamar al callback con el mensaje deserializado
		callback(message)
	})

	return err
}

func (b *BrokerNats) OnGetMessage(topic string, callback func(interface{})) error {
	// Suscripción al topic
	_, err := b.conn.Subscribe(topic, func(m *nats.Msg) {

		// Construir el mensaje NATS personalizado
		log.Printf("BrokerNats: OnMessage:  m.Subject: %s \n", m.Subject)
		natsMsg := &NatsMessage{
			Subject: m.Subject,
			Data:    m.Data,
		}
		log.Printf("BrokerNats: OnMessage:  natsMsg.Subject: %s \n", natsMsg.Subject)
		log.Printf("BrokerNats: OnMessage:  natsMsg.Data: %s \n", natsMsg.Data)

		message, err := b.adapter.TransformFromExternal(natsMsg.Data)

		if err != nil {
			// Manejar el error si es necesario
			fmt.Println("Error transformando el mensaje:", err)
			return
		}
		log.Printf("BrokerNats: OnMessage:  message: %v\n", message)
		// Llamar al callback con el mensaje deserializado
		callback(message)
	})

	return err
}

// GetMessagesFromId obtiene mensajes desde un MessageId específico en JetStream.
func (b *BrokerNats) GetMessagesFromId(topic string, messageId uuid.UUID) ([]Message, error) {
	// Construye el subject con el formato adecuado (por ejemplo, "chat.<roomId>").
	subject := fmt.Sprintf("chat.%s", topic)

	// Crea un consumer temporal para recuperar mensajes desde un punto específico.
	consumerName := fmt.Sprintf("consumer-%s", uuid.New().String())
	streamName := topic // Suponiendo que el stream tiene el mismo nombre que roomId.

	// Configuración del consumer.
	consumerConfig := &nats.ConsumerConfig{
		Durable:        consumerName,
		FilterSubject:  subject,
		AckPolicy:      nats.AckExplicitPolicy,
		DeliverPolicy:  nats.DeliverByStartSequencePolicy,
		OptStartSeq:    uint64(messageId.ID()), // Mapear el UUID a secuencia si corresponde.
		ReplayPolicy:   nats.ReplayInstantPolicy,
		MaxAckPending:  100,
		DeliverSubject: fmt.Sprintf("_deliver.%s", consumerName),
	}

	// Crear el consumer.
	_, err := b.js.AddConsumer(streamName, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("BrokerNats: GetMessagesFromId: error al crear consumer: %w", err)
	}

	// Suscribirse al subject del consumer.
	sub, err := b.js.PullSubscribe(subject, consumerName)
	if err != nil {
		return nil, fmt.Errorf("BrokerNats: GetMessagesFromId: error en PullSubscribe: %w", err)
	}

	// Recuperar los mensajes.
	messages, err := sub.Fetch(100) // Cambia el número según lo que necesites.
	if err != nil {
		return nil, fmt.Errorf("BrokerNats: GetMessagesFromId: error al recuperar mensajes: %w", err)
	}

	// Convertir mensajes a tu tipo `entities.Message`.
	var result []Message
	for _, msg := range messages {
		// Procesa el mensaje.
		entityMsg := Message{
			MessageId:   uuid.MustParse(string(msg.Header.Get("MessageId"))), // Suponiendo que MessageId está en el header.
			MessageText: string(msg.Data),
		}
		result = append(result, entityMsg)

		// Aceptar el mensaje.
		msg.Ack()
	}

	return result, nil
}

// Publish publica un mensaje en un tópico utilizando JetStream.

func (b *BrokerNats) Publish(topic string, message *Message) error {
	// Transforma el mensaje a su formato externo.
	msgData, err := b.adapter.TransformToExternal(message)
	if err != nil {
		log.Printf("BrokerNats: Publish: Error al transformar el mensaje de la app al formato externo: %s", err)
		return err
	}
	log.Printf("BrokerNats: Publish: topic [%s] --- msgData:[%v]", topic, msgData)
	// Publica el mensaje usando JetStream.
	ack, err := b.js.Publish(topic, msgData)
	if err != nil {
		log.Printf("BrokerNats: Publish: topic [%s] --- msgData:[%v]", topic, msgData)
		log.Printf("BrokerNats: Publish: Error al publicar mensaje en JetStream: %s", err)
		return err
	}

	log.Printf("BrokerNats: Publish: Mensaje publicado en JetStream. Stream: %s, Secuencia: %d", ack.Stream, ack.Sequence)
	return nil
}

// Publica un mensaje en un tópico específico.
func (b *BrokerNats) PublishGetUSers(topic string, message *ResponseListUser) error {
	// Transforma el mensaje a su formato externo.
	msgData, err := b.adapter.TransformToExternalUsers(message)
	if err != nil {
		log.Printf("BrokerNats: Publish: Error al transformar el mensaje de la app al formato externo: %s", err)
		return err
	}

	// Publica el mensaje usando JetStream.
	ack, err := b.js.Publish(topic, msgData)
	if err != nil {
		log.Printf("BrokerNats: Publish: Error al publicar mensaje en JetStream: %s", err)
		return err
	}

	log.Printf("BrokerNats: Publish: Mensaje publicado en JetStream. Stream: %s, Secuencia: %d", ack.Stream, ack.Sequence)
	return nil
}

// Publica un mensaje en un tópico específico.
func (b *BrokerNats) PublishGetMessages(topic string, message *ResponseListMessages) error {
	// Transforma el mensaje a su formato externo.
	msgData, err := b.adapter.TransformToExternalMessages(message)
	if err != nil {
		log.Printf("BrokerNats: Publish: Error al transformar el mensaje de la app al formato externo: %s", err)
		return err
	}

	// Publica el mensaje usando JetStream.
	ack, err := b.js.Publish(topic, msgData)
	if err != nil {
		log.Printf("BrokerNats: Publish: Error al publicar mensaje en JetStream: %s", err)
		return err
	}

	log.Printf("BrokerNats: Publish: Mensaje publicado en JetStream. Stream: %s, Secuencia: %d", ack.Stream, ack.Sequence)
	return nil
}

// Subscribe se suscribe a un tópico y procesa mensajes utilizando JetStream.
func (b *BrokerNats) Subscribe(topic string, handler func(message *Message) error) error {
	sub, err := b.js.PullSubscribe(topic, "durable-consumer")
	if err != nil {
		log.Printf("error al crear suscripción: %s", err)
		return err
	}

	// Procesar los mensajes en un goroutine
	go func() {
		for {
			messages, err := sub.Fetch(10) // Configura la cantidad de mensajes que se obtendrán
			if err != nil {
				log.Printf("error al recuperar mensajes: %s", err)
				continue
			}

			for _, msg := range messages {
				msgApp, err := b.adapter.TransformFromExternal(msg.Data)
				if err != nil {
					log.Printf("error al transformar el mensaje: %s", err)
					msg.Nak()
					continue
				}

				// Procesar el mensaje
				if err := handler(msgApp); err != nil {
					log.Printf("error al procesar el mensaje: %s", err)
					msg.Nak()
				} else {
					log.Printf("mensaje procesado con éxito")
					msg.Ack()
				}
			}
		}
	}()

	return nil
}

// Publish publica un mensaje en un tópico.
func (b *BrokerNats) Publish_native(topic string, message *Message) error {
	msgK, err := b.adapter.TransformToExternal(message)
	if err != nil {
		log.Printf("BrokerNats: Publish: Un error al transformar el mensaje de mi app al mensaje de kafka: %s", err)
		return err
	}
	return b.conn.Publish(topic, msgK)
}

// Suscripción al tópico con procesamiento del mensaje
func (b *BrokerNats) Subscribe_native(topic string, handler func(message []byte) error) error {
	_, err := b.conn.Subscribe(topic, func(msg *nats.Msg) {
		// Llamamos a TransformFromExternal pasando los datos del mensaje (msg.Data)
		msgApp, err := b.adapter.TransformFromExternal(msg.Data)
		if err != nil {
			log.Printf("BrokerNats: Subscribe: Error al transformar el mensaje de kafka a mi app: %s", err)
			return
		}

		// Llamada al handler con los datos originales del mensaje (msg.Data)
		if err := handler(msg.Data); err != nil {
			log.Printf("BrokerNats: Subscribe:  Un error al procesar el mensaje: %s", err)
		} else {
			// Si no hay error, mostramos el mensaje de NATS
			log.Printf("BrokerNats: Subscribe:  Mensaje recibido de NATS: %s", string(msg.Data))

			// Logueamos los detalles del mensaje transformado (de la app)
			// Aquí puedes personalizar el log dependiendo de la estructura de tu entidad Message
			log.Printf("BrokerNats: Subscribe: Mensaje procesado: ID: %d, Name: %s", msgApp.MessageId, msgApp.Nickname)
		}
	})
	return err
}

// GetUnreadMessages obtiene los mensajes no leídos desde un subject de NATS.
func (b *BrokerNats) GetUnreadMessages(topic string) ([]Message, error) {
	// Establecer la conexión con JetStream
	js, err := b.conn.JetStream()
	if err != nil {
		log.Printf("BrokerNats: GetUnreadMessages: error conectando a JetStream: %v", err)
		return nil, fmt.Errorf("error conectando a JetStream: %w", err)
	}
	subject := topic
	// Suscripción durable al subject
	sub, err := js.SubscribeSync(subject, nats.Durable("message-consumer"))
	if err != nil {
		log.Printf("BrokerNats: GetUnreadMessages: error suscribiendo al subject %s: %v", subject, err)
		return nil, fmt.Errorf("error suscribiendo al subject %s: %w", subject, err)
	}
	defer sub.Drain() // Cierra la suscripción al final

	// Obtener mensajes no leídos
	var messages []Message
	for {
		// Espera hasta 5 segundos por un mensaje
		msg, err := sub.NextMsg(5 * time.Second)
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				// Salir del bucle si no hay más mensajes dentro del tiempo límite
				break
			}
			log.Printf("BrokerNats: GetUnreadMessages: Error al obtener mensaje: %v", err)
			continue
		}

		// Transformar el mensaje de NATS a la entidad Message
		var message Message
		err = json.Unmarshal(msg.Data, &message)
		if err != nil {
			log.Printf("BrokerNats: GetUnreadMessages: Error al desempaquetar el mensaje: %v", err)
			// NAK el mensaje para volver a intentarlo
			msg.Nak()
			continue
		}

		// ACK del mensaje después de procesarlo
		err = msg.Ack()
		if err != nil {
			log.Printf("BrokerNats: GetUnreadMessages: Error al enviar ACK: %v", err)
			continue
		}

		messages = append(messages, message)
	}

	return messages, nil
}
func (b *BrokerNats) GetMessagesWithLimit(topic string, messageId uuid.UUID, count int) ([]Message, error) {
	subject := topic
	consumerName := fmt.Sprintf("consumer-%s", uuid.New().String())
	streamName := topic

	consumerConfig := &nats.ConsumerConfig{
		Durable:        consumerName,
		FilterSubject:  subject,
		AckPolicy:      nats.AckExplicitPolicy,
		DeliverPolicy:  nats.DeliverByStartSequencePolicy,
		OptStartSeq:    uint64(messageId.ID()),
		ReplayPolicy:   nats.ReplayInstantPolicy,
		MaxAckPending:  100,
		DeliverSubject: fmt.Sprintf("_deliver.%s", consumerName),
	}

	_, err := b.js.AddConsumer(streamName, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("BrokerNats: GetMessagesWithLimit: error al crear consumer: %w", err)
	}

	sub, err := b.js.PullSubscribe(subject, consumerName)
	if err != nil {
		return nil, fmt.Errorf("BrokerNats: GetMessagesWithLimit: error en PullSubscribe: %w", err)
	}

	var result []Message
	fetchedCount := 0

	for fetchedCount < count {
		messages, err := sub.Fetch(count - fetchedCount)
		if err != nil {
			return nil, fmt.Errorf("BrokerNats: GetMessagesWithLimit: error al recuperar mensajes: %w", err)
		}

		for _, msg := range messages {
			entityMsg := Message{
				MessageId:   uuid.MustParse(string(msg.Header.Get("MessageId"))),
				MessageText: string(msg.Data),
			}
			result = append(result, entityMsg)

			msg.Ack()
			fetchedCount++
			if fetchedCount >= count {
				break
			}
		}

		if fetchedCount >= count {
			break
		}
	}

	return result, nil
}

// Implementación del método SubscribeWithPattern para NATS.
func (b *BrokerNats) SubscribeWithPattern(pattern string, handler func(message *Message) error) error {
	// NATS soporta patrones de suscripción directamente usando wildcards.
	sub, err := b.conn.Subscribe(pattern, func(msg *nats.Msg) {
		msgApp, err := b.adapter.TransformFromExternal(msg.Data)
		if err != nil {
			log.Printf("BrokerNats: Subscribe: Error al transformar el mensaje de kafka a mi app: %s", err)
			return
		}

		if err := handler(msgApp); err != nil {
			log.Printf("Error procesando mensaje: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("BrokerNats: SubscribeWithPattern: error suscribiendo al patrón %s: %w", pattern, err)
	}

	// El sub está ahora suscrito a los mensajes del patrón.
	defer sub.Unsubscribe()
	return nil
}

// Implementación del método CreateTopic para BrokerNats
func (b *BrokerNats) CreateTopic(topic string) error {
	// NATS no requiere una creación explícita de temas, pero si necesitas
	// alguna lógica adicional, puedes agregarla aquí.
	// A modo de ejemplo, podríamos intentar publicar algo en el tema
	// para asegurarnos de que existe o configurarlo de alguna forma.

	// Intentar publicar un mensaje vacío (en la práctica no se requiere esto para NATS).
	_, err := b.js.PublishAsync(topic, nil)
	if err != nil {
		log.Printf("BrokerNats: CreateTopic: Error creating topic %s: %v", topic, err)
		return err
	}
	log.Printf("BrokerNats: CreateTopic: Topic %s created (or checked) successfully", topic)
	return nil
}

// Implementación de Acknowledge para NATS.
func (b *BrokerNats) Acknowledge(messageID uuid.UUID) error {
	// NATS JetStream requiere ACK explícitos para garantizar la entrega de los mensajes.
	err := b.conn.Publish(messageID.String(), []byte("ACK"))
	if err != nil {
		return fmt.Errorf("BrokerNats: Acknowledge: error al enviar ACK: %w", err)
	}
	return nil
}

// Implementación de Retry para NATS.
func (b *BrokerNats) Retry(messageID uuid.UUID) error {
	// NATS no tiene un mecanismo de reintentos automático, pero se puede simular reenviando el mensaje.
	// Aquí se reenvía el mensaje como ejemplo.
	return nil // Personaliza este método según tus necesidades.
}

// Implementación de GetMetrics para NATS.
func (b *BrokerNats) GetMetrics() (map[string]interface{}, error) {
	// Obtener métricas de JetStream.
	stats, err := b.js.StreamInfo("stream_name") // Reemplaza "stream_name" con el nombre del stream real.
	if err != nil {
		return nil, fmt.Errorf("BrokerNats: GetMetrics: error obteniendo métricas: %w", err)
	}

	metrics := map[string]interface{}{
		"stream_state": stats.State,
		// Agregar más métricas según sea necesario
	}
	return metrics, nil
}

// Close cierra la conexión con el broker.
func (b *BrokerNats) Close() error {
	b.conn.Close()
	return nil
}
