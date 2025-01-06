package utils

import (
	"fmt"

	"github.com/IBM/sarama"
)

func CreateTopicIfNotExists(brokerList []string, topicName string) error {
	// Configurar Kafka
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Version = sarama.V2_3_0_0 // Asegúrate de usar la versión correcta de Kafka

	// Crear AdminClient
	adminClient, err := sarama.NewClusterAdmin(brokerList, kafkaConfig)
	if err != nil {
		return fmt.Errorf("error al crear AdminClient para Kafka: %w", err)
	}
	defer adminClient.Close()

	// Verificar si el topic ya existe
	topics, err := adminClient.ListTopics()
	if err != nil {
		return fmt.Errorf("error al obtener la lista de topics: %w", err)
	}

	// Si el topic no existe, crearlo
	if _, exists := topics[topicName]; !exists {
		topicDetail := &sarama.TopicDetail{
			NumPartitions:     1, // Puedes cambiar este valor según tus necesidades
			ReplicationFactor: 1, // Puedes cambiar este valor según tus necesidades
		}
		err := adminClient.CreateTopic(topicName, topicDetail, false) // `false` para no crear de manera automática
		if err != nil {
			return fmt.Errorf("error al crear el topic '%s': %w", topicName, err)
		}
		fmt.Printf("Topic '%s' creado exitosamente\n", topicName)
	}

	return nil
}


