package entities

import (
	"backend/utils"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
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
	jsnats  bool                   // indica que usamos jetastrean en ntas par acontrol de persistencia en vez de nats bñasico
	metrics sync.Map               // Para las métricas
	adapter NatsTransformer        // Transformador Nats (si aplica)
	config  map[string]interface{} // Configuración
}

// NewNatsBroker inicializa un nuevo BrokerNats con JetStream.
func NewNatsBroker(config map[string]interface{}) (MessageBroker, error) {
	// Verificar que "nats" existe en la configuración
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

	// Conectar a NATS
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
	topics, err := utils.GetTopics(config)
	if err != nil {
		return nil, fmt.Errorf("BrokerNats: NewNatsBroker: error al obtener los topics: %w", err)
	}
	prefixStreamName, ok := natsConfigMap["prefixStreamName"].(string)
	if !ok {
		return nil, fmt.Errorf("BrokerNats: NewNatsBroker: 'prefixStreamName' no está presente en la configuración de NATS")
	}
	// Crear un stream para cada topic
	for _, topic := range topics {
		err := utils.CreateStreamForTopic(js, prefixStreamName, topic) // Crear stream único por cada topic
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("BrokerNats: NewNatsBroker: error al crear el stream para el topic %s: %w", topic, err)
		}
	}

	log.Printf("[*]BrokerNats: dentro de for con los Tópicos '%v'  \n", topics)

	// Crear consumidores y productores para cada topic

	// Limpiar recursos antes de crear nuevos productores y consumidores
	/*err = utils.CleanUpNatsResources(js, "MYGOCHAT_STREAM", topics)
	if err != nil {
		log.Fatalf("Error limpiando recursos: %v", err)
	}*/

	log.Printf("[*]BrokerNats: dentro de for con el Tópics '%v'  \n", topics)

	// Crear consumidores y productores para cada topic
	for _, topic := range topics {
		streamName := prefixStreamName + "_" + strings.ReplaceAll(topic, ".", "_")
		log.Printf("[*]BrokerNats: dentro de for con el Tópico '%s' agregado al stream '%s'\n", topic, streamName)
		if strings.HasSuffix(topic, ".client") {
			// Comprobar si el tópico está configurado en el stream
			streamInfo, err := js.StreamInfo(streamName)
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("BrokerNats: error al obtener información del stream '%s': %w", streamName, err)
			}

			// Verificar si el tópico está presente en el stream
			topicExists := false
			for _, subj := range streamInfo.Config.Subjects {
				if subj == topic {
					topicExists = true
					break
				}
			}

			// Si el tópico no existe, agregarlo al stream
			if !topicExists {
				streamInfo.Config.Subjects = append(streamInfo.Config.Subjects, topic)
				_, err = js.UpdateStream(&streamInfo.Config)
				if err != nil {
					conn.Close()
					return nil, fmt.Errorf("BrokerNats: error al actualizar el stream para agregar el tópico '%s': %w", topic, err)
				}
				log.Printf("BrokerNats: Tópico '%s' agregado al stream '%s'\n", topic, streamName)
			}

			// Crear un productor para el tópico
			err = utils.CreateProducer(js, topic, []byte{})
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("BrokerNats: error al crear el productor para el tópico %s: %w ", topic, err)
			}
			log.Printf("BrokerNats: CreateProducer: Tópico '%s' agregado al stream '%s'\n", topic, streamName)
		} else if strings.HasSuffix(topic, ".server") {
			prefix := strings.TrimSuffix(topic, ".server")
			consumerName := fmt.Sprintf("%s-user-consumer", prefix)
			log.Printf("BrokerNats: CreateConsumer '%s' : Tópico '%s' agregado al stream '%s'\n", consumerName, topic, streamName)
			// Crear un consumidor solo para los temas terminados en .server
			err := utils.CreateConsumer(js, streamName, consumerName, topic)
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("BrokerNats: error al crear el consumidor para el tópico %s: %w ", topic, err)
			}
		} else {
			log.Printf("BrokerNats: NewNatsBroker:  El tópico %s no corresponde a ninguna categoría conocida (.client o .server)\n", topic)
		}
	}

	// Retornar instancia del broker
	// ponemos en esta version jsnats:  true,  por defecto. Facilmente se peude añadir el fichero JSON

	brock := &BrokerNats{
		conn:    conn,
		metrics: sync.Map{},
		adapter: NatsTransformer{},
		js:      js,
		jsnats:  true,
		config:  config,
	}
	log.Printf("BrokerNats: NewNatsBroker: Salgo de NewNatsBroker [%v]\n", brock)
	return brock, nil
}

func (b *BrokerNats) OnMessage(topic string, callback func(interface{})) error {
	log.Printf("BrokerNats: OnMessage: topic :[%s] - callback ", topic)
	if b.jsnats {
		return b.OnMessageJetStream(topic, callback)
	} else {
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
				if strings.Contains(err.Error(), "CMD100") {
					// Ignorar CMD100 y continuar
					log.Println("BrokerNats: OnMessage: CMD100 detectado, ignorando y continuando.")
				} else {
					// Manejar otros errores
					log.Println("BrokerNats: OnMessage: Error transformando el mensaje:", err)
					return
				}
			} else {
				log.Printf("BrokerNats: OnMessage:  message: %v\n", message)
				// Llamar al callback con el mensaje deserializado
				callback(message)
			}

		})

		return err
	}

}

func (b *BrokerNats) OnGetUsers(topic string, callback func(interface{})) error {
	if b.jsnats {
		return b.OnGetUsersJetStream(topic, callback)
	} else {
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
				if strings.Contains(err.Error(), "CMD100") {
					// Ignorar CMD100 y continuar
					log.Println("BrokerNats: OnGetUsers: CMD100 detectado, ignorando y continuando.")
				} else {
					// Manejar otros errores
					log.Println("BrokerNats: OnGetUsers: Error transformando el mensaje:", err)
					return
				}
			} else {
				log.Printf("BrokerNats: OnGetUsers:  message: %v\n", message)
				// Llamar al callback con el mensaje deserializado
				callback(message)
			}
		})

		return err
	}
}

func (b *BrokerNats) OnGetMessage(topic string, callback func(interface{})) error {
	if b.jsnats {
		return b.OnGetMessageJetStream(topic, callback)
	} else {
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
				if strings.Contains(err.Error(), "CMD100") {
					// Ignorar CMD100 y continuar
					log.Println("BrokerNats: OnMessage: CMD100 detectado, ignorando y continuando.")
				} else {
					// Manejar otros errores
					log.Println("BrokerNats: OnMessage: Error transformando el mensaje:", err)
					return
				}
			} else {
				log.Printf("BrokerNats: OnMessage:  message: %v\n", message)
				// Llamar al callback con el mensaje deserializado
				callback(message)
			}
		})

		return err
	}
}

// /Consumidores con Jetstream . invocan a una funcin de calback cad e que se emite un mensaje al topic indicado:
func (b *BrokerNats) OnMessageJetStream(topic string, callback func(interface{})) error {
	log.Printf("BrokerNats: OnMessageJetStream: Iniciando suscripción al topic: [%s]", topic)

	// Usar la instancia de JetStream existente
	js := b.js
	if js == nil {
		log.Println("Error: No se encontró una instancia válida de JetStream.")
		return fmt.Errorf("no se encontró una instancia válida de JetStream")
	}

	log.Printf("BrokerNats: OnMessageJetStream: Usando instancia de JetStream existente")

	// Suscribirse al topic sin crear un nuevo stream
	_, err := js.Subscribe(topic, func(m *nats.Msg) {
		log.Printf("Recibido mensaje en el topic [%s]: %s\n", m.Subject, string(m.Data))

		// Construir el mensaje NATS personalizado
		natsMsg := &NatsMessage{
			Subject: m.Subject,
			Data:    m.Data,
		}
		log.Printf("natsMsg.Subject: %s\n", natsMsg.Subject)
		log.Printf("natsMsg.Data: %s\n", natsMsg.Data)

		// Transformar el mensaje con el adaptador
		message, err := b.adapter.TransformFromExternal(natsMsg.Data)

		if err != nil {
			if strings.Contains(err.Error(), "CMD100") {
				// Ignorar CMD100 y continuar
				log.Println("BrokerNats: OnMessage: CMD100 detectado, ignorando y continuando.")
			} else {
				// Manejar otros errores
				log.Println("BrokerNats: OnMessage: Error transformando el mensaje:", err)
				return
			}
		} else {
			log.Printf("BrokerNats: OnMessage:  message: %v\n", message)
			// Llamar al callback con el mensaje deserializado
			callback(message)
			// Confirmar el mensaje (opcional)
			m.Ack()
			log.Printf("Mensaje confirmado: %s", string(m.Data))
		}

	}, nats.Durable("durable-consumer"))

	if err != nil {
		log.Printf("Error al suscribir al tópico: %v", err)
		return fmt.Errorf("error al suscribir al tópico %s: %v", topic, err)
	}

	log.Printf("BrokerNats: OnMessageJetStream: Suscripción exitosa al topic: [%s]", topic)
	return nil
}

func (b *BrokerNats) OnGetUsersJetStream(topic string, callback func(interface{})) error {
	log.Printf("BrokerNats: OnGetUsersJetStream: Iniciando suscripción al topic: [%s]", topic)

	// Usar la instancia de JetStream existente
	js := b.js
	if js == nil {
		log.Println("BrokerNats: OnGetUsersJetStream: Error: No se encontró una instancia válida de JetStream.")
		return fmt.Errorf("BrokerNats: OnGetUsersJetStream: no se encontró una instancia válida de JetStream")
	}

	log.Printf("BrokerNats: OnGetUsersJetStream: Usando instancia de JetStream existente")

	// Suscripción al topic sin crear un nuevo stream
	_, err := js.Subscribe(topic, func(m *nats.Msg) {
		log.Printf("BrokerNats: OnGetUsersJetStream: Recibido mensaje en el topic [%s]: %s\n", m.Subject, string(m.Data))

		// Construir el mensaje NATS personalizado
		natsMsg := &NatsMessage{
			Subject: m.Subject,
			Data:    m.Data,
		}
		log.Printf("BrokerNats: OnGetUsersJetStream: natsMsg.Subject: %s\n", natsMsg.Subject)
		log.Printf("BrokerNats: OnGetUsersJetStream: natsMsg.Data: %s\n", natsMsg.Data)

		// Transformar el mensaje con el adaptador
		message, err := b.adapter.TransformFromExternalToGetUsers(natsMsg.Data)
		if err != nil {
			if strings.Contains(err.Error(), "CMD100") {
				// Ignorar CMD100 y continuar
				log.Println("BrokerNats: OnGetUsersJetStream: CMD100 detectado, ignorando y continuando.")
			} else {
				// Manejar otros errores
				log.Println("BrokerNats: OnGetUsersJetStream: Error transformando el mensaje:", err)
				return
			}
		} else {

			log.Printf("BrokerNats: OnGetUsersJetStream: Mensaje transformado con éxito, llamando al callback.")
			// Llamar al callback con el mensaje deserializado
			callback(message)

			// Confirmar el mensaje (opcional)
			m.Ack()
			log.Printf("BrokerNats: OnGetUsersJetStream: Mensaje confirmado: %s", string(m.Data))
		}

	}, nats.Durable("durable-consumer"))

	if err != nil {
		log.Printf("BrokerNats: OnGetUsersJetStream: Error al suscribir al tópico: %v", err)
		return fmt.Errorf("BrokerNats: OnGetUsersJetStream: error al suscribir al tópico %s: %v", topic, err)
	}

	log.Printf("BrokerNats: OnGetUsersJetStream: Suscripción exitosa al topic: [%s]", topic)
	return nil
}

func (b *BrokerNats) OnGetMessageJetStream(topic string, callback func(interface{})) error {
	log.Printf("BrokerNats: OnGetMessageJetStream: Iniciando suscripción al topic: [%s]", topic)

	// Usar la instancia de JetStream existente
	js := b.js
	if js == nil {
		log.Println("BrokerNats: OnGetMessageJetStream: Error: No se encontró una instancia válida de JetStream.")
		return fmt.Errorf("BrokerNats: OnGetMessageJetStream: no se encontró una instancia válida de JetStream")
	}

	log.Printf("BrokerNats: OnGetMessageJetStream: Usando instancia de JetStream existente")

	// Suscripción al topic sin crear un nuevo stream
	_, err := js.Subscribe(topic, func(m *nats.Msg) {
		log.Printf("BrokerNats: OnGetMessageJetStream: Recibido mensaje en el topic [%s]: %s\n", m.Subject, string(m.Data))

		// Construir el mensaje NATS personalizado
		natsMsg := &NatsMessage{
			Subject: m.Subject,
			Data:    m.Data,
		}
		log.Printf("natsMsg.Subject: %s\n", natsMsg.Subject)
		log.Printf("natsMsg.Data: %s\n", natsMsg.Data)

		// Transformar el mensaje con el adaptador
		message, err := b.adapter.TransformFromExternal(natsMsg.Data)
		if err != nil {
			if strings.Contains(err.Error(), "CMD100") {
				// Ignorar CMD100 y continuar
				log.Println("BrokerNats: OnGetMessageJetStream: CMD100 detectado, ignorando y continuando.")
			} else {
				// Manejar otros errores
				log.Println("BrokerNats: OnGetMessageJetStream: Error transformando el mensaje:", err)
				return
			}
		} else {

			log.Printf("BrokerNats: OnGetMessageJetStream: Mensaje transformado con éxito, llamando al callback.")
			// Llamar al callback con el mensaje deserializado
			callback(message)

			// Confirmar el mensaje (opcional)
			m.Ack()
			log.Printf("BrokerNats: OnGetMessageJetStream: Mensaje confirmado: %s", string(m.Data))
		}

	}, nats.Durable("durable-consumer"))

	if err != nil {
		log.Printf("BrokerNats: OnGetMessageJetStream: Error al suscribir al tópico: %v", err)
		return fmt.Errorf("BrokerNats: OnGetMessageJetStream: error al suscribir al tópico %s: %v", topic, err)
	}

	log.Printf("BrokerNats: OnGetMessageJetStream: Suscripción exitosa al topic: [%s]", topic)
	return nil
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
func (b *BrokerNats) PublishGetUsers(topic string, message *ResponseListUser) error {
	log.Printf("BrokerNats: PublishGetUsers: Mensaje ha  ublicar con JetStream. topic: %s, Secuencia: %v\n", topic, message)
	// Transforma el mensaje a su formato externo.
	msgData, err := b.adapter.TransformToExternalUsers(topic, message)
	if err != nil {
		log.Printf("BrokerNats: PublishGetUsers: Error al transformar el mensaje de la app al formato externo: %s\n", err)
		return err
	}
	log.Printf("BrokerNats: PublishGetUsers: Mensaje ha  ublicar con JetStream. topic: %s, msgData: %v\n", topic, msgData)
	// Publica el mensaje usando JetStream.
	ack, err := b.js.Publish(topic, msgData)
	if err != nil {
		log.Printf("BrokerNats: PublishGetUsers: Error al publicar mensaje en JetStream: %s\n", err)
		return err
	}

	log.Printf("BrokerNats: PublishGetUsers: Mensaje publicado en JetStream. Stream: %s, Secuencia: %d\n", ack.Stream, ack.Sequence)
	return nil
}

// Publica un mensaje en un tópico específico.
func (b *BrokerNats) PublishGetMessages(topic string, message *ResponseListMessages) error {
	// Transforma el mensaje a su formato externo.
	msgData, err := b.adapter.TransformToExternalMessages(message)
	if err != nil {
		log.Printf("BrokerNats: Publish: Error al transformar el mensaje de la app al formato externo: %s\n", err)
		return err
	}

	// Publica el mensaje usando JetStream.
	ack, err := b.js.Publish(topic, msgData)
	if err != nil {
		log.Printf("BrokerNats: Publish: Error al publicar mensaje en JetStream: %s\n", err)
		return err
	}

	log.Printf("BrokerNats: Publish: Mensaje publicado en JetStream. Stream: %s, Secuencia: %d\n", ack.Stream, ack.Sequence)
	return nil
}

// Subscribe se suscribe a un tópico y procesa mensajes utilizando JetStream.
func (b *BrokerNats) Subscribe(topic string, handler func(message *Message) error) error {
	sub, err := b.js.PullSubscribe(topic, "durable-consumer")
	if err != nil {
		log.Printf("BrokerNats: Subscribe: error al crear suscripción: %s\n", err)
		return err
	}

	// Procesar los mensajes en un goroutine
	go func() {
		for {
			messages, err := sub.Fetch(10) // Configura la cantidad de mensajes que se obtendrán
			if err != nil {
				log.Printf("BrokerNats: Subscribe: error al recuperar mensajes: %s\n", err)
				continue
			}

			for _, msg := range messages {
				msgApp, err := b.adapter.TransformFromExternal(msg.Data)
				if err != nil {
					if strings.Contains(err.Error(), "CMD100") {
						// Ignorar CMD100 y continuar
						log.Println("BrokerNats: Subscribe: CMD100 detectado, ignorando y continuando.")
					} else {
						// Manejar otros errores
						log.Println("BrokerNats: Subscribe: Error transformando el mensaje:", err)
						return
					}
				} else {
					// Procesar el mensaje
					if err := handler(msgApp); err != nil {
						log.Printf("BrokerNats: Subscribe: error al procesar el mensaje: %s\n", err)
						msg.Nak()
					} else {
						log.Printf("BrokerNats: Subscribe: mensaje procesado con éxito\n")
						msg.Ack()
					}
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
		log.Printf("BrokerNats: Publish_native: Un error al transformar el mensaje de mi app al mensaje de kafka: %s\n", err)
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
			if strings.Contains(err.Error(), "CMD100") {
				// Ignorar CMD100 y continuar
				log.Println("BrokerNats: Publish_native: CMD100 detectado, ignorando y continuando.")
			} else {
				// Manejar otros errores
				log.Println("BrokerNats: Publish_native: Error transformando el mensaje:", err)
				return
			}
		} else {
			// Llamada al handler con los datos originales del mensaje (msg.Data)
			if err := handler(msg.Data); err != nil {
				log.Printf("BrokerNats: Publish_native:  Un error al procesar el mensaje: %s\n", err)
			} else {
				// Si no hay error, mostramos el mensaje de NATS
				log.Printf("BrokerNats: Publish_native:  Mensaje recibido de NATS: %s\n", string(msg.Data))
				log.Printf("BrokerNats: Publish_native: Mensaje procesado: ID: %d, Name: %s\n", msgApp.MessageId, msgApp.Nickname)
			}
		}

	})
	return err
}

// GetUnreadMessages obtiene los mensajes no leídos desde un subject de NATS.
func (b *BrokerNats) GetUnreadMessages(topic string) ([]Message, error) {
	// Establecer la conexión con JetStream
	js, err := b.conn.JetStream()
	if err != nil {
		log.Printf("BrokerNats: GetUnreadMessages: error conectando a JetStream: %v\n", err)
		return nil, fmt.Errorf("error conectando a JetStream: %w", err)
	}
	subject := topic
	// Suscripción durable al subject
	sub, err := js.SubscribeSync(subject, nats.Durable("message-consumer"))
	if err != nil {
		log.Printf("BrokerNats: GetUnreadMessages: error suscribiendo al subject %s: %v\n", subject, err)
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
			log.Printf("BrokerNats: GetUnreadMessages: Error al obtener mensaje: %v\n", err)
			continue
		}

		// Transformar el mensaje de NATS a la entidad Message
		var message Message
		err = json.Unmarshal(msg.Data, &message)
		if err != nil {
			log.Printf("BrokerNats: GetUnreadMessages: Error al desempaquetar el mensaje: %v\n", err)
			// NAK el mensaje para volver a intentarlo
			msg.Nak()
			continue
		}

		// ACK del mensaje después de procesarlo
		err = msg.Ack()
		if err != nil {
			log.Printf("BrokerNats: GetUnreadMessages: Error al enviar ACK: %v\n", err)
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
