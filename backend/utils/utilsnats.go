package utils

import (
	"log"
	"regexp"
	"strings"
	"time"

	"fmt"

	"github.com/nats-io/nats.go" // Ensure you're using the NATS Go client
)

func CheckTopicConflicts(js nats.JetStreamContext, topics []string) ([]string, error) {
	var conflictingStreams []string

	// Obtener todos los nombres de los streams
	streams := js.StreamNames()
	if streams == nil {
		return nil, fmt.Errorf("error al obtener los nombres de los streams")
	}

	// Iterar sobre los streams para verificar conflictos
	for streamName := range streams {
		// Verificar si el nombre del stream coincide con algún tópico
		for _, currentTopic := range topics {
			if streamName == currentTopic {
				if !contains(conflictingStreams, streamName) {
					conflictingStreams = append(conflictingStreams, streamName)
					fmt.Printf("utilsnats: conflicto detectado - nombre del stream: %s coincide con el tópico: %s\n", streamName, currentTopic)
				}
			}
		}

		// Obtener información del stream
		streamInfo, err := js.StreamInfo(streamName)
		if err != nil {
			fmt.Printf("utilsnats: error al obtener información del stream '%s': %v\n", streamName, err)
			continue
		}

		// Comparar los tópicos del stream con los tópicos actuales
		for _, streamTopic := range streamInfo.Config.Subjects {
			for _, currentTopic := range topics {
				if streamTopic == currentTopic {
					// Agregar el stream conflictivo a la lista si aún no está
					if !contains(conflictingStreams, streamName) {
						conflictingStreams = append(conflictingStreams, streamName)
					}
					fmt.Printf("utilsnats: conflicto detectado - stream: %s, tópico conflictivo: %s\n", streamName, currentTopic)
				}
			}
		}
	}

	return conflictingStreams, nil
}

// Función auxiliar para verificar si un elemento ya está en la lista
func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// Función auxiliar para validar nombres de streams
func isValidStreamName(name string) bool {
	// Verifica que el nombre sea alfanumérico y permita guiones bajos y puntos
	validName := regexp.MustCompile(`^[a-zA-Z0-9._]+$`)
	return validName.MatchString(name)
}

func CreateOrUpdateStream(js nats.JetStreamContext, streamName string, topics []string) error {
	log.Printf("BrokerNats: CreateOrUpdateStream: streamName:[%s]", streamName)

	// Verificar si el stream ya existe
	if _, err := js.StreamInfo(streamName); err == nil {
		log.Printf("BrokerNats: CreateOrUpdateStream: El stream '%s' ya existe, no es necesario crearlo nuevamente.", streamName)
		return nil
	}

	// Verificar conflictos de topics
	conflictingStreams, err := CheckTopicConflicts(js, topics)
	if err != nil {
		return fmt.Errorf("BrokerNats: CreateOrUpdateStream: error al verificar conflictos de topics: %w", err)
	}
	log.Printf("BrokerNats: CreateOrUpdateStream: conflictingStreams:%v", conflictingStreams)

	// Eliminar streams conflictivos
	for _, stream := range conflictingStreams {
		err := js.DeleteStream(stream)
		if err != nil {
			return fmt.Errorf("BrokerNats: CreateOrUpdateStream: error al eliminar stream conflictivo %s: %w", stream, err)
		}
		log.Printf("BrokerNats: CreateOrUpdateStream: Stream conflictivo eliminado: %s", stream)
	}

	// Crear el stream nuevo
	log.Printf("BrokerNats: CreateOrUpdateStream: Creando el stream '%s'", streamName)
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: topics,
		Storage:  nats.FileStorage,
		Replicas: 1,
	})
	if err != nil {
		return fmt.Errorf("BrokerNats: CreateOrUpdateStream: error al crear el stream '%s': %w", streamName, err)
	}

	log.Println("BrokerNats: CreateOrUpdateStream: Stream creado exitosamente")
	return nil
}
func CleanUpNatsResources(js nats.JetStreamContext, streamName string, topics []string) error {
	for _, topic := range topics {
		// Eliminar el consumidor si existe
		consumerName := fmt.Sprintf("%s-user-consumer", strings.TrimSuffix(topic, ".server"))
		if _, err := js.ConsumerInfo(streamName, consumerName); err == nil {
			// Si el consumidor existe, lo eliminamos
			err := js.DeleteConsumer(streamName, consumerName)
			if err != nil {
				return fmt.Errorf("error al eliminar el consumidor '%s': %w", consumerName, err)
			}
			log.Printf("Consumidor '%s' eliminado exitosamente.", consumerName)
		}

		// Verificar si el stream ya existe
		if _, err := js.StreamInfo(streamName); err == nil {
			// Si el stream existe, lo eliminamos
			err := js.DeleteStream(streamName)
			if err != nil {
				return fmt.Errorf("error al eliminar el stream '%s': %w", streamName, err)
			}
			log.Printf("Stream '%s' eliminado exitosamente.", streamName)
		}

		// Espera opcional antes de recrear el stream (para asegurarse de que el stream haya sido eliminado completamente)
		time.Sleep(1 * time.Second) // Ajusta el tiempo si es necesario

		// Volver a crear el stream después de eliminarlo
		_, err := js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{topic}, // Asegúrate de agregar el subject correcto
		})
		if err != nil {
			return fmt.Errorf("error al crear el stream '%s': %w", streamName, err)
		}
		log.Printf("Stream '%s' creado exitosamente.", streamName)
	}

	return nil
}

// Crear un consumidor para un tópico específico q
func CreateConsumer(js nats.JetStreamContext, streamName, consumerName, subject string) error {
	// Crear el consumidor con los parámetros proporcionados
	log.Printf("BrokerNats: CreateConsumer: streamName:[%s], consumerName:[%s], subject:[%s]", streamName, consumerName, subject)

	// Crear el consumidor con el mismo subject y un deliverSubject diferente para evitar ciclos
	deliverSubject := fmt.Sprintf("%s-deliver", subject)
	_, err := js.AddConsumer(streamName, &nats.ConsumerConfig{
		Durable:        consumerName,   // Nombre duradero del consumidor
		DeliverSubject: deliverSubject, // Canal de entrega
		FilterSubject:  subject,        // El 'subject' en el cual el consumidor se suscribe (es necesario)
		AckPolicy:      nats.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("BrokerNats: CreateConsumer: error al crear el consumidor: %w", err)
	}

	log.Printf("BrokerNats: CreateConsumer: Consumidor '%s' creado exitosamente para el stream '%s' con subject '%s'", consumerName, streamName, subject)
	return nil
}

// Crear un productor para un tópico específico
func CreateProducer(js nats.JetStreamContext, topic string, message []byte) error {
	// Publicar un mensaje en el tópico
	_, err := js.Publish(topic, message)
	if err != nil {
		return fmt.Errorf("BrokerNats: createProducer: error al publicar en el tópico '%s': %w", topic, err)
	} else {
		log.Printf("BrokerNats: createProducer: Productor creado en el tópico '%s'", topic)
	}
	return nil
}

// Crear el stream con todos los topics. si existe, lo actualizamos
// Crear el stream con todos los topics
func CreateStream(js nats.JetStreamContext, nc *nats.Conn, topics []string) error {
	if nc.Status() != nats.CONNECTED {
		return fmt.Errorf("BrokerNats: NewNatsBroker: la conexión a NATS no está establecida")
	}
	// Verificar conflictos de topics
	conflictingStreams, err := CheckTopicConflicts(js, topics)
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

// Eliminar streams conflictivos
func DeleteConflictingStreams(js nats.JetStreamContext, topics []string) error {
	streams := js.StreamNames()
	if streams == nil {
		return fmt.Errorf("DeleteConflictingStreams: error al obtener los nombres de los streams")
	}

	for stream := range streams {
		info, err := js.StreamInfo(stream)
		if err != nil {
			log.Printf("DeleteConflictingStreams: No se pudo obtener información del stream %s: %v", stream, err)
			continue
		}

		for _, subject := range info.Config.Subjects {
			for _, topic := range topics {
				if subject == topic {
					log.Printf("DeleteConflictingStreams: Eliminando stream conflictivo: %s (Subjects: %v)", stream, info.Config.Subjects)
					if err := js.DeleteStream(stream); err != nil {
						return fmt.Errorf("DeleteConflictingStreams: error al eliminar el stream %s: %w", stream, err)
					}
					break
				}
			}
		}
	}
	return nil
}
func ListStreams(js nats.JetStreamContext) {
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

// Función para obtener todos los topics a partir de la configuración -
func GetTopics(config map[string]interface{}) ([]string, error) {
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
