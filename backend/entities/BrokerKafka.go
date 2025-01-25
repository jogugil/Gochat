package entities

import (
	"backend/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type BrokerKafka struct {
	producer        sarama.SyncProducer // Para producir mensajes
	consumer        sarama.Consumer     // Para consumir mensajes
	activeConsumers map[string]sarama.PartitionConsumer
	adapter         KafkaTransformer // Transformador para mensajes
	brokers         []string         // Lista de brokers
	config          map[string]interface{}
	topicToSubject  sync.Map // Mapa de topics a subjects topic -> subject.
}

// NewKafkaBroker crea una nueva instancia de KafkaBroker.
func NewKafkaBroker(config map[string]interface{}) (MessageBroker, error) {
	// Acceder a la configuración de Kafka
	kafkaConfigInterface, ok := config["kafka"]
	if !ok {
		return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker: 'kafka' no está presente en la configuración")
	}

	kafkaConfigMap, ok := kafkaConfigInterface.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker: 'kafka' no es un mapa válido")
	}

	// Acceder a la lista de brokers
	brokersInterface, ok := kafkaConfigMap["brokers"]
	if !ok {
		return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker: 'brokers' no está presente en la configuración de Kafka")
	}

	var brokerList []string

	// Manejar cadenas y listas para "brokers"
	switch v := brokersInterface.(type) {
	case string: // Si es una cadena, dividirla en una lista
		brokerList = strings.Split(v, ",")
	case []interface{}: // Si es una lista, procesar cada elemento
		for _, b := range v {
			if broker, ok := b.(string); ok {
				brokerList = append(brokerList, broker)
			} else {
				return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker: 'brokers' contiene un valor no válido")
			}
		}
	default: // Si no es ninguno de los tipos esperados
		return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker: 'brokers' debe ser una cadena o una lista")
	}

	// Configurar Kafka
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Return.Successes = true // Asegura confirmaciones de mensajes enviados
	kafkaConfig.Consumer.Return.Errors = true    // Permite manejar errores en el consumidor
	// Obtener todos los topics
	topics, err := utils.GetTopics(config)
	if err != nil {
		return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker:  error al obtener los topics: %w", err)
	}

	// Crear un topics
	for _, topic := range topics {
		err := utils.CreateTopicIfNotExists(brokerList, topic) // Crear stream único por cada topic
		if err != nil {
			return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker:  error al crear el stream para el topic %s: %w", topic, err)
		}
	}
	// Crear productor Kafka
	producer, err := sarama.NewSyncProducer(brokerList, kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker: error creando productor Kafka: %w", err)
	}

	// Crear consumidor Kafka
	consumer, err := sarama.NewConsumer(brokerList, kafkaConfig)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker: error creando consumidor Kafka: %w", err)
	}
	var activeConsumers = make(map[string]sarama.PartitionConsumer)
	// Retornar instancia del broker
	brock := &BrokerKafka{
		producer:        producer,
		consumer:        consumer,
		brokers:         brokerList,
		activeConsumers: activeConsumers,
		config:          config,
		topicToSubject:  sync.Map{},
	}
	// Crear productores y consumidores según el sufijo del topic
	for _, topic := range topics {
		if strings.HasSuffix(topic, ".client") {
			log.Printf("BrokerKafka: NewKafkaBroker Creando productor para el topic '%s'\n", topic)
			// Enviar un mensaje de ejemplo al topic
			msg := &sarama.ProducerMessage{
				Topic: topic,
				Value: sarama.StringEncoder("Mensaje inicial para topic .client"),
			}
			_, _, err := producer.SendMessage(msg)
			if err != nil {
				log.Printf("BrokerKafka: NewKafkaBroker Error enviando mensaje inicial a '%s': %v\n", topic, err)
			} else {
				log.Printf("BrokerKafka: NewKafkaBroker Mensaje enviado a '%s'\n", topic)
			}
		} else if strings.HasSuffix(topic, ".server") {
			log.Printf("BrokerKafka: NewKafkaBroker Creando consumidor para el topic '%s'\n", topic)
			consum, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
			if err != nil {
				consumer.Close()
				producer.Close()
				return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker error creando consumidor para el topic %s: %w", topic, err)
			}
			activeConsumers[topic] = consum
			brock.AssignSubjectToTopic(topic, topic)
		}
	}

	log.Printf("BrokerKafka: NewKafkaBroker: Salgo de NewNatsBroker [%v]\n", brock)
	return brock, nil

}

// Función para asociar un topic con su subject
func (b *BrokerKafka) AssignSubjectToTopic(topic string, subject string) {
	b.topicToSubject.Store(topic, subject)
}

// Función para obtener el subject asociado con un topic
func (b *BrokerKafka) GetSubjectByTopic(topic string) (string, bool) {
	value, ok := b.topicToSubject.Load(topic)
	if ok {
		return value.(string), true
	}
	return "", false
}

// Implementación del método Publish para Kafka.

func (k *BrokerKafka) Publish(topic string, message *Message) error {
	msgK, err := k.adapter.TransformToExternal(message)
	if err != nil {
		log.Printf("BrokerKafka: Publish: Un error al transformar el mensaje de mi app al mensaje de kafka: %s\n", err)
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msgK),
	}
	_, _, errsnd := k.producer.SendMessage(msg)
	return errsnd
}

// convertHeaders transforma los RecordHeaders de Sarama a un mapa de strings
func (k *BrokerKafka) convertHeaders(headers []*sarama.RecordHeader) map[string]interface{} {
	result := make(map[string]interface{})
	for _, header := range headers {
		if header != nil {
			result[string(header.Key)] = string(header.Value)
		}
	}
	return result
}
func (k *BrokerKafka) OnMessage(topic string, callback func(interface{})) error {
	log.Printf("BrokerKafka: OnMessage: Creando callback para el topic '%s'\n", topic)

	// Verificar si el consumidor ya existe para este topic
	if _, exists := k.activeConsumers[topic]; !exists {
		// Crear un consumidor para la partición del topic si no existe
		consumer, err := k.consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
		if err != nil {
			return fmt.Errorf("BrokerKafka: OnMessage: failed to start consumer for topic %s: %w", topic, err)
		}

		// Almacenar el consumidor en un mapa para reutilizarlo más tarde
		k.activeConsumers[topic] = consumer
		log.Printf("BrokerKafka: OnMessage: Consumer creado para el topic '%s'\n", topic)
	}

	// Procesar mensajes en un goroutine
	go func() {
		consumer := k.activeConsumers[topic] // Recuperar el consumidor del mapa

		for msg := range consumer.Messages() {
			// Convertir los encabezados y crear el mensaje Kafka
			kafkaMsg := &KafkaMessage{
				Key:     string(msg.Key),
				Value:   string(msg.Value),
				Headers: k.convertHeaders(msg.Headers),
			}

			// Convertir kafkaMessage a entities.Message
			rawMsg, err := json.Marshal(kafkaMsg)
			if err != nil {
				// Manejar el error de serialización
				fmt.Println("BrokerKafka: OnMessage: Error serializando el mensaje Kafka:", err)
				return
			}

			message, err := k.adapter.TransformFromExternal(rawMsg)
			// Comprobar si el error contiene "CMD100"
			if err != nil {
				if strings.Contains(err.Error(), "CMD100") {
					// Ignorar CMD100 y continuar
					fmt.Println("BrokerKafka: OnMessage: CMD100 detectado, ignorando y continuando.")
				} else {
					// Manejar otros errores
					fmt.Println("BrokerKafka: OnMessage: Error transformando el mensaje:", err)
					return
				}
			} else {
				// Llamar al callback con el mensaje deserializado
				callback(message)
			}
		}
	}()
	return nil
}

func (k *BrokerKafka) OnGetUsers(topic string, callback func(interface{})) error {
	log.Printf("BrokerKafka: OnGetUsers: Creando callback para el topic '%s'\n", topic)

	// Verificar si el consumidor ya existe para el topic
	consum, exists := k.activeConsumers[topic]
	if !exists {
		log.Printf("BrokerKafka: OnGetUsers: No existe consumidor para el topic '%s', creando uno nuevo.\n", topic)

		// Crear un consumidor para el topic si no existe
		var err error
		consum, err = k.consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
		if err != nil {
			return fmt.Errorf("BrokerKafka: OnGetUsers: Error creando consumidor para el topic %s: %w", topic, err)
		}

		// Almacenar el consumidor en el mapa
		k.activeConsumers[topic] = consum
		log.Printf("BrokerKafka: OnGetUsers: Consumidor creado y agregado para el topic '%s'\n", topic)
	} else {
		log.Printf("BrokerKafka: OnGetUsers: Consumidor ya existe para el topic '%s'.\n", topic)
	}

	// Procesar mensajes en un goroutine
	go func() {
		for msg := range consum.Messages() { // Corregido: "rane" a "range"
			// Convertir los encabezados y crear el mensaje Kafka
			kafkaMsg := &KafkaMessage{
				Key:     string(msg.Key),
				Value:   string(msg.Value),
				Headers: k.convertHeaders(msg.Headers),
			}
			// Convertir NatsMessage a entities.Message
			rawMsg, err := json.Marshal(kafkaMsg)
			if err != nil {
				// Manejar el error de serialización
				log.Println("BrokerKafka: OnGetUsers: Error serializando el mensaje Kafka:", err)
				return
			}
			message, err := k.adapter.TransformFromExternalToGetUsers(rawMsg)
			if err != nil {
				if strings.Contains(err.Error(), "CMD100") {
					// Ignorar CMD100 y continuar
					fmt.Println("BrokerKafka: OnGetUsers: CMD100 detectado, ignorando y continuando.")
				} else {
					// Manejar otros errores
					fmt.Println("BrokerKafka: OnGetUsers: Error transformando el mensaje:", err)
					return
				}
			} else {
				// Llamar al callback con el mensaje deserializado
				callback(message)
			}
		}
	}()
	return nil
}

func (k *BrokerKafka) OnGetMessage(topic string, callback func(interface{})) error {
	log.Printf("BrokerKafka: OnGetMessage Creando callback para el topic '%s'\n", topic)

	// Verificar si el consumidor ya existe para el topic
	consum, exists := k.activeConsumers[topic]
	if !exists {
		log.Printf("BrokerKafka: No existe consumidor para el topic '%s', creando uno nuevo.\n", topic)

		// Crear un consumidor para el topic si no existe
		var err error
		consum, err = k.consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
		if err != nil {
			return fmt.Errorf("BrokerKafka: OnGetMessage: failed to start consumer for topic %s: %w", topic, err)
		}
		// Almacenar el consumidor en el mapa
		k.activeConsumers[topic] = consum
		log.Printf("BrokerKafka: OnGetMessage: Consumidor creado y agregado para el topic '%s'\n", topic)
	} else {
		log.Printf("BrokerKafka: OnGetMessage: Consumidor ya existe para el topic '%s'.\n", topic)
	}

	// Procesar mensajes en un goroutine
	go func() {
		for msg := range consum.Messages() {
			// Convertir los encabezados y crear el mensaje Kafka
			kafkaMsg := &KafkaMessage{
				Key:     string(msg.Key),
				Value:   string(msg.Value),
				Headers: k.convertHeaders(msg.Headers),
			}
			// Convertir NatsMessage a entities.Message
			rawMsg, err := json.Marshal(kafkaMsg)
			if err != nil {
				// Manejar el error de serialización
				log.Println("BrokerKafka: OnGetMessage: Error serializando el mensaje Kafka:", err)
				return
			}
			message, err := k.adapter.TransformFromExternalToGetMessage(rawMsg)
			if err != nil {
				if strings.Contains(err.Error(), "CMD100") {
					// Ignorar CMD100 y continuar
					log.Println("BrokerKafka: Subscribe: CMD100 detectado, ignorando y continuando.")
				} else {
					// Manejar otros errores
					log.Println("BrokerKafka: Subscribe: Error transformando el mensaje:", err)
					return
				}
			} else {
				// Llamar al callback con el mensaje deserializado
				callback(message)
			}
		}
	}()
	return nil
}

// Publica un mensaje en un tópico específico.
func (k *BrokerKafka) PublishGetUsers(topic string, message *ResponseListUser) error {
	log.Printf("BrokerKafka: PublishGetUsers: [%s]\n", topic)

	// Verificar si el topic existe.
	exists, err := k.topicExists(topic)
	if err != nil {
		log.Printf("BrokerKafka: PublishGetUsers: Error al verificar la existencia del topic '%s': %s\n", topic, err)
		return err
	}

	// Si el topic no existe, intentar crearlo.
	if !exists {
		log.Printf("BrokerKafka: PublishGetUsers: El topic '%s' no existe. Intentando crearlo...\n", topic)
		err := k.createTopic(topic)
		if err != nil {
			// Manejar el error si el topic ya existe
			if strings.Contains(err.Error(), "Topic with this name already exists") {
				log.Printf("BrokerKafka: PublishGetUsers: Topic '%s' ya existe, continuando con la publicación...\n", topic)
			} else {
				log.Printf("BrokerKafka: PublishGetUsers: Error al crear el topic '%s': %s\n", topic, err)
				return err
			}
		} else {
			log.Printf("BrokerKafka: PublishGetUsers: Topic '%s' creado exitosamente.\n", topic)
		}
	}

	// Transformar el mensaje al formato externo.
	msgData, err := k.adapter.TransformToExternalUsers(topic, message)
	if err != nil {
		log.Printf("BrokerKafka: PublishGetUsers: Error al transformar el mensaje de la app al formato externo: %s\n", err)
		return err
	}

	// Publicar el mensaje en el topic.
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msgData),
	}
	_, _, errsnd := k.producer.SendMessage(msg)
	if errsnd != nil {
		log.Printf("BrokerKafka: PublishGetUsers: Error al enviar el mensaje al topic '%s': %s\n", topic, errsnd)
		return errsnd
	}

	log.Printf("BrokerKafka: PublishGetUsers: Mensaje publicado exitosamente en '%s'\n", topic)
	return nil
}

// Publica un mensaje en un tópico específico.
func (k *BrokerKafka) PublishGetMessages(topic string, message *ResponseListMessages) error {
	log.Printf("BrokerKafka: PublishGetMessages: [%s]\n", topic)

	// Verificar si el topic existe.
	exists, err := k.topicExists(topic)
	if err != nil {
		log.Printf("BrokerKafka: PublishGetMessages: Error al verificar la existencia del topic '%s': %s\n", topic, err)
		return err
	}

	// Si el topic no existe, crearlo.
	if !exists {
		log.Printf("BrokerKafka: PublishGetMessages: El topic '%s' no existe. Intentando crearlo...\n", topic)
		err := k.createTopic(topic)
		if err != nil {
			log.Printf("BrokerKafka: PublishGetMessages: Error al crear el topic '%s': %s\n", topic, err)
			return err
		}
		log.Printf("BrokerKafka: PublishGetMessages: Topic '%s' creado exitosamente.\n", topic)
	}

	// Transformar el mensaje al formato externo.
	msgData, err := k.adapter.TransformToExternalMessages(message)
	if err != nil {
		log.Printf("BrokerKafka: PublishGetMessages: Error al transformar el mensaje de la app al formato externo: %s\n", err)
		return err
	}

	// Publicar el mensaje en el topic.
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msgData),
	}
	_, _, errsnd := k.producer.SendMessage(msg)
	if errsnd != nil {
		log.Printf("BrokerKafka: PublishGetMessages: Error al enviar el mensaje al topic '%s': %s\n", topic, errsnd)
	}
	return errsnd
}

// Implementación del método Subscribe para Kafka.

func (k *BrokerKafka) Subscribe(topic string, handler func(message *Message) error) error {
	// Obtener la lista de particiones para el tópico.
	partitionList, err := k.consumer.Partitions(topic)
	if err != nil {
		return err
	}

	// Consumir los mensajes de cada partición.
	for _, partition := range partitionList {
		pc, err := k.consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
		if err != nil {
			return err
		}

		// Consumir mensajes en paralelo para cada partición.
		go func(pc sarama.PartitionConsumer) {
			defer pc.Close()

			// Iterar sobre los mensajes de la partición.
			for msg := range pc.Messages() {
				msgApp, err := k.adapter.TransformFromExternal(msg.Value)
				if err != nil {
					if strings.Contains(err.Error(), "CMD100") {
						// Ignorar CMD100 y continuar
						fmt.Println("BrokerKafka: Subscribe: CMD100 detectado, ignorando y continuando.")
					} else {
						// Manejar otros errores
						fmt.Println("BrokerKafka: Subscribe: Error transformando el mensaje:", err)
						return
					}
				} else {
					// Llamada al handler con el mensaje original (msg.Value).
					if err := handler(msgApp); err != nil {
						log.Println("BrokerKafka: Subscribe: Error al manejar el mensaje:", err)
					} else {
						// Logueo del mensaje recibido de Kafka
						log.Printf("BrokerKafka: Subscribe: Mensaje recibido de Kafka: %s\n", string(msg.Value))

						// Transformación del mensaje de Kafka a la aplicación
						msgApp, err := k.adapter.TransformFromExternal(msg.Value)

						if err != nil {
							if strings.Contains(err.Error(), "CMD100") {
								// Ignorar CMD100 y continuar
								log.Println("BrokerKafka: Subscribe: CMD100 detectado, ignorando y continuando.")
							} else {
								// Manejar otros errores
								log.Println("BrokerKafka: Subscribe: Error transformando el mensaje:", err)
								return
							}
						} else {
							// Logueo del mensaje procesado
							log.Printf("BrokerKafka: Subscribe: Mensaje procesado (app): ID: %d, Name: %s\n", msgApp.MessageId, msgApp.Nickname)
						}
					}
				}
			}
		}(pc)
	}
	return nil
}

// GetUnreadMessages obtiene los mensajes no leídos desde un topic de Kafka
func (b *BrokerKafka) GetUnreadMessages(topic string) ([]Message, error) {
	// Obtener las particiones del topic
	partitions, err := b.consumer.Partitions(topic)
	if err != nil {
		log.Printf("BrokerKafka: GetUnreadMessages: error obteniendo particiones: %v\n", err)
		return nil, fmt.Errorf("error obteniendo particiones de Kafka: %w", err)
	}

	var messages []Message
	for _, partition := range partitions {
		// Establecer el offset para leer desde el principio o desde el último mensaje disponible
		offset := sarama.OffsetNewest // o sarama.OffsetOldest si deseas leer desde el principio

		// Consumir los mensajes de la partición
		pc, err := b.consumer.ConsumePartition(topic, partition, offset)
		if err != nil {
			log.Printf("BrokerKafka: GetUnreadMessages: error creando consumidor para partición: %v\n", err)
			continue
		}
		defer pc.Close()

		// Leer los mensajes de la partición
		for msg := range pc.Messages() {
			var message Message
			err := json.Unmarshal(msg.Value, &message)
			if err != nil {
				log.Printf("BrokerKafka: GetUnreadMessages: Error al desempaquetar mensaje: %v\n", err)
				continue
			}

			// Agregar el mensaje a la lista
			messages = append(messages, message)
		}
	}

	return messages, nil
}

// GetRoomMessagesByRoomId obtiene los mensajes asociados a un `roomId` específico.
func (k *BrokerKafka) GetRoomMessagesByRoomId(topic string) ([]Message, error) {
	// Configurar el consumidor de Kafka.
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	consumer, err := sarama.NewConsumer([]string{"localhost:9092"}, config)
	if err != nil {
		return nil, fmt.Errorf("BrokerKafka: GetRoomMessagesByRoomId: Error creando el consumidor Kafka: %v\n", err)
	}
	defer consumer.Close()

	partitionList, err := consumer.Partitions(topic)
	if err != nil {
		return nil, fmt.Errorf("BrokerKafka: GetRoomMessagesByRoomId: Error obteniendo particiones: %v\n", err)
	}

	var messages []Message
	// Iteramos sobre las particiones.
	for _, partition := range partitionList {
		pc, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
		if err != nil {
			return nil, fmt.Errorf("BrokerKafka: GetRoomMessagesByRoomId: Error consumiendo partición: %v", err)
		}
		defer pc.Close()

		// Leemos los mensajes de la partición.
		for msg := range pc.Messages() {
			var message Message
			if err := json.Unmarshal(msg.Value, &message); err != nil {
				log.Printf("BrokerKafka: GetRoomMessagesByRoomId: Error unmarshaling Kafka message: %v\n", err)
				continue
			}
			messages = append(messages, message)
		}
	}

	return messages, nil
}

// GetMessagesFromId obtiene mensajes desde un MessageId específico en Kafka.
func (k *BrokerKafka) GetMessagesFromId(topic string, messageId uuid.UUID) ([]Message, error) {
	// Configuración del lector de Kafka
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: k.brokers,         // Brokers definidos en BrokerKafka
		Topic:   topic,             // Tópico al que conectarse
		GroupID: "message-fetcher", // Grupo de consumidores
	})

	defer reader.Close()

	var messages []Message
	found := false

	// Leer mensajes del tópico
	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			// Finaliza si se encuentra un error, como EOF
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("BrokerKafka: GetMessagesFromId:  error leyendo mensaje de Kafka: %w", err)
		}

		// Verificar si el MessageId coincide
		if string(msg.Key) == messageId.String() {
			found = true
		}

		// Si ya se encontró, procesar el mensaje
		if found {
			var message Message
			if err := json.Unmarshal(msg.Value, &message); err != nil {
				log.Printf("BrokerKafka: GetMessagesFromId: error desempaquetando mensaje: %v\n", err)
				continue
			}
			messages = append(messages, message)
		}
	}

	// Retornar error si no se encontró el ID
	if !found {
		return nil, fmt.Errorf("BrokerKafka: GetMessagesFromId: mensaje con ID %s no encontrado", messageId.String())
	}

	return messages, nil
}

// GetMessagesWithLimit obtiene los mensajes a partir de un `messageId` específico y limita la cantidad de mensajes recuperados.
func (k *BrokerKafka) GetMessagesWithLimit(topic string, messageId uuid.UUID, count int) ([]Message, error) {
	// Configurar el consumidor de Kafka.
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	consumer, err := sarama.NewConsumer(k.brokers, config)
	if err != nil {
		return nil, fmt.Errorf("BrokerKafka: GetMessagesWithLimit: error creando el consumidor Kafka: %v", err)
	}
	defer consumer.Close()

	// Obtener las particiones del tópico.
	partitions, err := consumer.Partitions(topic)
	if err != nil {
		return nil, fmt.Errorf("BrokerKafka: GetMessagesWithLimit: error obteniendo particiones del tópico %s: %v", topic, err)
	}

	var messages []Message
	var found bool
	for _, partition := range partitions {
		// Consumir desde el principio o el último mensaje.
		pc, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
		if err != nil {
			return nil, fmt.Errorf("BrokerKafka: GetMessagesWithLimit: error consumiendo partición %d: %v", partition, err)
		}
		defer pc.Close()

		// Contador para limitar la cantidad de mensajes recuperados.
		messageCount := 0

		// Leer los mensajes de la partición.
		for msg := range pc.Messages() {
			// Deserializar el mensaje a una estructura Message.
			var message Message
			if err := json.Unmarshal(msg.Value, &message); err != nil {
				log.Printf("BrokerKafka: GetMessagesWithLimit: error deserializando mensaje: %v\n", err)
				continue
			}

			// Verificar si el mensaje es después del messageId especificado.
			if message.MessageId.String() == messageId.String() || found {
				// Agregar el mensaje a la lista
				messages = append(messages, message)
				messageCount++

				// Verificar si ya hemos alcanzado el límite de mensajes.
				if messageCount >= count {
					return messages, nil
				}

				// Si encontramos el messageId, seguimos recibiendo mensajes.
				found = true
			}
		}
	}

	// Retornar los mensajes recuperados
	return messages, nil
}

// Implementación del método SubscribeWithPattern para Kafka (una implementación básica).

func (k *BrokerKafka) SubscribeWithPattern(pattern string, handler func(message *Message) error) error {
	// Nota: Kafka no tiene soporte directo para patrones. Este es un ejemplo básico usando un Consumer Group.
	return nil // Devolver error o implementación personalizada
}

// Implementación de Acknowledge (sin impacto en Kafka).
func (k *BrokerKafka) Acknowledge(messageID uuid.UUID) error {
	// Kafka maneja ACKs implícitamente, así que esta función no es aplicable.
	return nil
}

// Implementación de Retry (Kafka no soporta reintentos de mensajes, pero se puede personalizar).
func (k *BrokerKafka) Retry(messageID uuid.UUID) error {
	// Lógica para intentar reenviar un mensaje, si es necesario.
	return nil
}

// Implementación de GetMetrics para Kafka.
func (k *BrokerKafka) GetMetrics() (map[string]interface{}, error) {
	metrics := map[string]interface{}{
		"producer_state": "active",
		"consumer_state": "active",
		// Agregar más métricas según sea necesario
	}
	return metrics, nil
}

// CreateTopic crea un tema en Kafka.
func (b *BrokerKafka) CreateTopic(topic string) error {
	// Kafka no requiere crear un tema explícitamente antes de usarlo, pero puedes usar
	// la función `AdminClient` para crear el tema si es necesario.
	adminClient, err := sarama.NewClusterAdmin([]string{"localhost:9092"}, nil)
	if err != nil {
		return fmt.Errorf("BrokerKafka: CreateTopic:  error creating Kafka admin client: %v\n", err)
	}
	defer adminClient.Close()

	// Crear el tema
	details := sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
	}

	// Cambiar esto para pasar un puntero a details
	err = adminClient.CreateTopic(topic, &details, false)
	if err != nil {
		log.Printf("BrokerKafka: CreateTopic: Error creating topic %s: %v\n", topic, err)
		return err
	}

	log.Printf("BrokerKafka: CreateTopic: Topic %s created successfully\n", topic)
	return nil
}

// Close cierra las conexiones del productor y consumidor.
func (k *BrokerKafka) Close() error {
	if err := k.producer.Close(); err != nil {
		return err
	}
	if err := k.consumer.Close(); err != nil {
		return err
	}
	return nil
}

// Verifica si un topic existe en Kafka.
func (k *BrokerKafka) topicExists(topic string) (bool, error) {
	admin, err := sarama.NewClusterAdmin(k.brokers, nil)
	if err != nil {
		return false, fmt.Errorf("BrokerKafka: topicExists: error al crear ClusterAdmin: %w", err)
	}
	defer admin.Close()

	topics, err := admin.ListTopics()
	if err != nil {
		return false, fmt.Errorf("BrokerKafka: topicExists: error al listar topics: %w", err)
	}

	_, exists := topics[topic]
	return exists, nil
}

// Crea un topic en Kafka si no existe.
func (k *BrokerKafka) createTopic(topic string) error {
	admin, err := sarama.NewClusterAdmin(k.brokers, nil)
	if err != nil {
		return fmt.Errorf("BrokerKafka: CreateTopic: error al crear ClusterAdmin: %w", err)
	}
	defer admin.Close()

	topicDetail := &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
	}
	err = admin.CreateTopic(topic, topicDetail, false)
	if err != nil {
		if err == sarama.ErrTopicAlreadyExists {
			log.Printf("BrokerKafka: CreateTopic: El topic '%s' ya existe \n", topic)
			return nil
		}
		return fmt.Errorf("BrokerKafka: CreateTopic: error al crear el topic '%s': %w", topic, err)
	}
	return nil
}
