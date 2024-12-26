package kafka

import (
	"log"

	"github.com/IBM/sarama"
)

type KafkaBroker struct {
	producer sarama.SyncProducer
	consumer sarama.Consumer
}

// NewKafkaBroker crea una nueva instancia de KafkaBroker.
func NewKafkaBroker(brokerList []string) (*KafkaBroker, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true // Asegura confirmaciones de mensajes enviados
	config.Consumer.Return.Errors = true    // Permite manejar errores en el consumidor

	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		return nil, err
	}

	consumer, err := sarama.NewConsumer(brokerList, config)
	if err != nil {
		return nil, err
	}

	return &KafkaBroker{
		producer: producer,
		consumer: consumer,
	}, nil
}

// Implementación del método Publish para Kafka.
func (k *KafkaBroker) Publish(topic string, message []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(message),
	}
	_, _, err := k.producer.SendMessage(msg)
	return err
}

// Implementación del método Subscribe para Kafka.
func (k *KafkaBroker) Subscribe(topic string, handler func(message []byte) error) error {
	partitionList, err := k.consumer.Partitions(topic)
	if err != nil {
		return err
	}

	for _, partition := range partitionList {
		pc, err := k.consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
		if err != nil {
			return err
		}
		go func(pc sarama.PartitionConsumer) {
			defer pc.Close()
			for msg := range pc.Messages() {
				if err := handler(msg.Value); err != nil {
					log.Println("Error handling message:", err)
				}
			}
		}(pc)
	}
	return nil
}

// SubscribeWithPattern implementa la suscripción basada en patrones. Sarama no tiene soporte directo para patrones,
// pero puedes usar herramientas adicionales como librerías externas o crear una abstracción.
func (k *KafkaBroker) SubscribeWithPattern(pattern string, handler func(message []byte) error) error {
	// Suscripciones basadas en patrones requieren un ConsumerGroup o lógica adicional.
	return nil // Devolver error o implementación personalizada
}

// Acknowledge confirma un mensaje. Kafka no soporta confirmaciones manuales, ya que las confirmaciones son implícitas.
func (k *KafkaBroker) Acknowledge(messageID string) error {
	return nil // Kafka maneja ACKs implícitamente, así que esta función no es aplicable.
}

// Retry intenta reenviar un mensaje fallido. No es nativo en Kafka, pero puedes simularlo.
func (k *KafkaBroker) Retry(messageID string) error {
	return nil // Puedes personalizar esta función según el caso de uso.
}

// GetMetrics obtiene métricas relacionadas con el broker.
func (k *KafkaBroker) GetMetrics() (map[string]interface{}, error) {
	// Aquí podrías incluir métricas como producción, consumo, errores, etc.
	metrics := map[string]interface{}{
		"producer_state": "active",
		"consumer_state": "active",
		// Agrega más métricas si es necesario
	}
	return metrics, nil
}

// Close cierra las conexiones del productor y consumidor.
func (k *KafkaBroker) Close() error {
	if err := k.producer.Close(); err != nil {
		return err
	}
	if err := k.consumer.Close(); err != nil {
		return err
	}
	return nil
}
