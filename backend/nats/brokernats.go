package nats

import (
	"backend/interfaces"
	"errors"
	"sync"

	"github.com/nats-io/nats.go"
)

// BrokerNats es la estructura que representa la conexión al broker NATS.
type BrokerNats struct {
	conn    *nats.Conn
	metrics sync.Map // Para almacenar métricas simuladas (o reales, si aplicable)
}

// CreateTopic implements interfaces.MessageBroker.
func (b *BrokerNats) CreateTopic(topic string) error {
	panic("unimplemented")
}

var (
	instance *BrokerNats
	once     sync.Once
)

// GetNatsBroker devuelve la instancia única del broker NATS, creándola si es necesario.
func GetNatsBroker(url string) (interfaces.MessageBroker, error) {
	var err error
	once.Do(func() {
		// Solo creamos la instancia una vez.
		conn, err := nats.Connect(url)
		if err != nil {
			// Si hay un error al conectar, lo retornamos.
			return
		}
		instance = &BrokerNats{conn: conn}
	})
	if err != nil {
		return nil, err
	}
	return instance, nil
}

// Publish publica un mensaje en un tópico.
func (b *BrokerNats) Publish(topic string, message []byte) error {
	return b.conn.Publish(topic, message)
}

// Subscribe se suscribe a un tópico con un manejador para procesar los mensajes.
func (b *BrokerNats) Subscribe(topic string, handler func(message []byte) error) error {
	_, err := b.conn.Subscribe(topic, func(msg *nats.Msg) {
		if err := handler(msg.Data); err != nil {
			// Aquí puedes implementar lógica adicional para manejar errores del handler.
		}
	})
	return err
}

// SubscribeWithPattern implementa la suscripción por patrón. NATS no soporta patrones directamente,
// por lo que esto podría emularse usando wildcards en los tópicos.
func (b *BrokerNats) SubscribeWithPattern(pattern string, handler func(message []byte) error) error {
	_, err := b.conn.Subscribe(pattern, func(msg *nats.Msg) {
		if err := handler(msg.Data); err != nil {
			// Aquí puedes manejar errores del handler.
		}
	})
	return err
}

// Acknowledge confirma un mensaje recibido. En NATS, las confirmaciones no son manuales por defecto,
// por lo que esta función podría devolver un error o estar vacía.
func (b *BrokerNats) Acknowledge(messageID string) error {
	return errors.New("NATS no soporta confirmaciones manuales")
}

// Retry intenta reentregar un mensaje fallido. En NATS esto no se gestiona nativamente,
// así que puedes simular la lógica o devolver un error.
func (b *BrokerNats) Retry(messageID string) error {
	return errors.New("NATS no soporta reintentos manuales")
}

// GetMetrics obtiene métricas simuladas del broker.
func (b *BrokerNats) GetMetrics() (map[string]interface{}, error) {
	metrics := make(map[string]interface{})
	b.metrics.Range(func(key, value interface{}) bool {
		metrics[key.(string)] = value
		return true
	})
	return metrics, nil
}

// Close cierra la conexión con el broker.
func (b *BrokerNats) Close() error {
	b.conn.Close()
	return nil
}
