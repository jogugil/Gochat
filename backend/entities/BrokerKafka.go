package entities

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type BrokerKafka struct {
	producer sarama.SyncProducer // Para producir mensajes
	consumer sarama.Consumer     // Para consumir mensajes
	adapter  KafkaTransformer    // Transformador para mensajes
	brokers  []string            // Lista de brokers
	config   map[string]interface{}
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

	// Retornar instancia del broker
	return &BrokerKafka{
		producer: producer,
		consumer: consumer,
		brokers:  brokerList,
		config:   config,
	}, nil
}

// Implementación del método Publish para Kafka.

func (k *BrokerKafka) Publish(topic string, message *Message) error {
	msgK, err := k.adapter.TransformToExternal(message)
	if err != nil {
		log.Printf("BrokerKafka: Publish: Un error al transformar el mensaje de mi app al mensaje de kafka: %s", err)
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msgK),
	}
	_, _, errsnd := k.producer.SendMessage(msg)
	return errsnd
}
func (k *BrokerKafka) OnMessage(topic string, callback func(interface{})) error {
	// Crear un consumidor para la partición del topic
	consumer, err := k.consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		return fmt.Errorf("failed to start consumer for topic %s: %w", topic, err)
	}

	// Procesar mensajes en un goroutine
	go func() {
		for msg := range consumer.Messages() {
			// Convertir los encabezados y crear el mensaje Kafka
			kafkaMsg := &KafkaMessage{
				Key:     string(msg.Key),
				Value:   string(msg.Value),
				Headers: convertHeaders(msg.Headers),
			}
			// Convertir NatsMessage a entities.Message
			rawMsg, err := json.Marshal(kafkaMsg)
			if err != nil {
				// Manejar el error de serialización
				fmt.Println("Error serializando el mensaje NATS:", err)
				return
			}
			message, err := k.adapter.TransformFromExternal(rawMsg)
			if err != nil {
				// Manejar el error si es necesario
				fmt.Println("Error transformando el mensaje:", err)
				return
			}
			// Llamar al callback con el mensaje deserializado
			callback(message)
		}

	}()
	return nil
}

// convertHeaders transforma los RecordHeaders de Sarama a un mapa de strings
func convertHeaders(headers []*sarama.RecordHeader) map[string]string {
	result := make(map[string]string)
	for _, header := range headers {
		if header != nil {
			result[string(header.Key)] = string(header.Value)
		}
	}
	return result
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
					log.Printf("BrokerKafka: Subscribe:error, %v", err)
				}
				// Llamada al handler con el mensaje original (msg.Value).
				if err := handler(msgApp); err != nil {
					log.Println("BrokerKafka: Subscribe: Error al manejar el mensaje:", err)
				} else {
					// Logueo del mensaje recibido de Kafka
					log.Printf("BrokerKafka: Subscribe: Mensaje recibido de Kafka: %s", string(msg.Value))

					// Transformación del mensaje de Kafka a la aplicación
					msgApp, err := k.adapter.TransformFromExternal(msg.Value)
					if err != nil {
						log.Printf("BrokerKafka: Subscribe: Un error al transformar el mensaje de Kafka a mi app: %s", err)
					} else {
						// Logueo del mensaje procesado
						log.Printf("BrokerKafka: Subscribe:Mensaje procesado (app): ID: %d, Name: %s", msgApp.MessageId, msgApp.Nickname)
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
		log.Printf("BrokerKafka: GetUnreadMessages: error obteniendo particiones: %v", err)
		return nil, fmt.Errorf("error obteniendo particiones de Kafka: %w", err)
	}

	var messages []Message
	for _, partition := range partitions {
		// Establecer el offset para leer desde el principio o desde el último mensaje disponible
		offset := sarama.OffsetNewest // o sarama.OffsetOldest si deseas leer desde el principio

		// Consumir los mensajes de la partición
		pc, err := b.consumer.ConsumePartition(topic, partition, offset)
		if err != nil {
			log.Printf("BrokerKafka: GetUnreadMessages: error creando consumidor para partición: %v", err)
			continue
		}
		defer pc.Close()

		// Leer los mensajes de la partición
		for msg := range pc.Messages() {
			var message Message
			err := json.Unmarshal(msg.Value, &message)
			if err != nil {
				log.Printf("BrokerKafka: GetUnreadMessages: Error al desempaquetar mensaje: %v", err)
				continue
			}

			// Agregar el mensaje a la lista
			messages = append(messages, message)
		}
	}

	return messages, nil
}

// GetRoomMessagesByRoomId obtiene los mensajes asociados a un `roomId` específico.
func (k *BrokerKafka) GetRoomMessagesByRoomId(roomId string) ([]Message, error) {
	// Configurar el consumidor de Kafka.
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	consumer, err := sarama.NewConsumer([]string{"localhost:9092"}, config)
	if err != nil {
		return nil, fmt.Errorf("BrokerKafka: GetRoomMessagesByRoomId:Error creando el consumidor Kafka: %v", err)
	}
	defer consumer.Close()

	// Obtener las particiones de un topic basado en roomId.
	topic := fmt.Sprintf("room_%s_messages", roomId)
	partitionList, err := consumer.Partitions(topic)
	if err != nil {
		return nil, fmt.Errorf("BrokerKafka: GetRoomMessagesByRoomId:Error obteniendo particiones: %v", err)
	}

	var messages []Message
	// Iteramos sobre las particiones.
	for _, partition := range partitionList {
		pc, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
		if err != nil {
			return nil, fmt.Errorf("BrokerKafka: GetRoomMessagesByRoomId:Error consumiendo partición: %v", err)
		}
		defer pc.Close()

		// Leemos los mensajes de la partición.
		for msg := range pc.Messages() {
			var message Message
			if err := json.Unmarshal(msg.Value, &message); err != nil {
				log.Printf("BrokerKafka: GetRoomMessagesByRoomId:Error unmarshaling Kafka message: %v", err)
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
			return nil, fmt.Errorf("error leyendo mensaje de Kafka: %w", err)
		}

		// Verificar si el MessageId coincide
		if string(msg.Key) == messageId.String() {
			found = true
		}

		// Si ya se encontró, procesar el mensaje
		if found {
			var message Message
			if err := json.Unmarshal(msg.Value, &message); err != nil {
				log.Printf("error desempaquetando mensaje: %v", err)
				continue
			}
			messages = append(messages, message)
		}
	}

	// Retornar error si no se encontró el ID
	if !found {
		return nil, fmt.Errorf("mensaje con ID %s no encontrado", messageId.String())
	}

	return messages, nil
}

// GetMessagesWithLimit obtiene los mensajes a partir de un `messageId` específico y limita la cantidad de mensajes recuperados.
func (k *BrokerKafka) GetMessagesWithLimit(roomId string, messageId uuid.UUID, count int) ([]Message, error) {
	// Configurar el consumidor de Kafka.
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	consumer, err := sarama.NewConsumer(k.brokers, config)
	if err != nil {
		return nil, fmt.Errorf("BrokerKafka: GetMessagesWithLimit: error creando el consumidor Kafka: %v", err)
	}
	defer consumer.Close()

	// Crear el tópico basado en el roomId.
	topic := fmt.Sprintf("room_%s_messages", roomId)

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
				log.Printf("BrokerKafka: GetMessagesWithLimit: error deserializando mensaje: %v", err)
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
func (k *BrokerKafka) Acknowledge(messageID string) error {
	// Kafka maneja ACKs implícitamente, así que esta función no es aplicable.
	return nil
}

// Implementación de Retry (Kafka no soporta reintentos de mensajes, pero se puede personalizar).
func (k *BrokerKafka) Retry(messageID string) error {
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
		return fmt.Errorf("BrokerKafka: CreateTopic:  error creating Kafka admin client: %v", err)
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
		log.Printf("BrokerKafka: CreateTopic: Error creating topic %s: %v", topic, err)
		return err
	}

	log.Printf("BrokerKafka: CreateTopic: Topic %s created successfully", topic)
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
