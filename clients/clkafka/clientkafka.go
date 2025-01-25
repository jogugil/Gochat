package clkafka

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Estructura para el cuerpo de la solicitud POST al API REST
type LoginRequest struct {
	Nickname string `json:"nickname"`
	X_GoChat string `json:"x_gochat"`
}

// Definimos el tipo MessageType como int
type MessageType int

// Ahora definimos constantes para los valores posibles de MessageType
const (
	// Los diferentes tipos de mensajes
	Text         MessageType = iota // 0
	Image                           // 1
	Video                           // 2
	Notification                    // 3
)

// KafkaMessage represents the structure of a message in Kafka
type KafkaMessage struct {
	Key       string                 `json:"key"`
	Value     string                 `json:"value"`
	Headers   map[string]interface{} `json:"headers"`
	Timestamp time.Time              `json:"timestap"`
}

// Clase Mensaje
type Metadata struct {
	AckStatus    bool   `json:"ackstatus"`    // Estado de confirmación
	Priority     int    `json:"priority"`     // Prioridad del mensaje
	OriginalLang string `json:"originallang"` // Idioma original
}

type Message struct {
	MessageId   uuid.UUID   `json:"messageid"`   // Identificador único
	MessageType MessageType `json:"messagetype"` // Tipo de mensaje
	SendDate    time.Time   `json:"senddate"`    // Fecha de envío
	ServerDate  time.Time   `json:"serverdate"`  // Fecha en el servidor
	Nickname    string      `json:"nickname"`    // Nombre del remitente
	Token       string      `json:"tokenjwt"`    // JWT del remitente
	MessageText string      `json:"messagetext"` // Contenido del mensaje
	RoomID      uuid.UUID   `json:"roomid"`      // Sala destino
	RoomName    string      `json:"roomname"`    // Nombre de la sala
	Metadata    Metadata    `json:"metadata"`    // Metadatos adicionales
}

type LoginResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Token    string `json:"token"`
	Nickname string `json:"nickname"`
	Roomid   string `json:"roomid"`
	Roomname string `json:"roomname"`
}

// Estructura para los usuarios activos
type AliveUsers struct {
	Nickname       string `json:"nickname"`
	LastActionTime string `json:"lastactiontime"`
}

type MessageResponse struct {
	MessageId   uuid.UUID `json:"messageid"`
	Nickname    string    `json:"nickname"`
	MessageText string    `json:"messagetext"`
}

// Estructura para la respuesta
type ResponseListMessages struct {
	Status      string            `json:"status"`
	Message     string            `json:"message"`
	TokenSesion string            `json:"tokenSesion"`
	Nickname    string            `json:"nickname"`
	RoomId      uuid.UUID         `json:"roomid"`
	X_GoChat    string            `json:"x_gochat"`
	ListMessage []MessageResponse `json:"data,omitempty"` // Lista de mensajes si existen
}

// Estructura para la respuesta general
type ResponseListUser struct {
	Status      string       `json:"status"`
	Message     string       `json:"message"`
	TokenSesion string       `json:"tokenSesion"`
	Nickname    string       `json:"nickname"`
	RoomId      string       `json:"roomId"`
	X_GoChat    string       `json:"x_gochat"`
	AliveUsers  []AliveUsers `json:"data,omitempty"`
}
type RequestListUser struct {
	RoomId      uuid.UUID `json:"roomid"`
	TokenSesion string    `json:"tokensesion"`
	Nickname    string    `json:"nickname"`
	Operation   string    `json:"operation"`
	Request     string    `json:"request"`
	Topic       string    `json:"topic"`
	X_GoChat    string    `json:"x_gochat"`
}
type RequestListMessages struct {
	Operation     string    `json:"operation"`
	LastMessageId uuid.UUID `json:"lastmessageid"`
	TokenSesion   string    `json:"tokensesion"`
	Nickname      string    `json:"nickname"`
	RoomId        uuid.UUID `json:"roomid"`
	Topic         string    `json:"topic"`
	X_GoChat      string    `json:"x_gochat"`
}

// BrokerKafka mantiene el productor, consumidor de Kafka y los consumidores activos por tópico
type BrokerKafka struct {
	producer        sarama.SyncProducer
	consumer        sarama.Consumer
	brokers         []string
	config          map[string]interface{}
	activeConsumers map[string]sarama.PartitionConsumer // Mapa de consumidores activos por tópico
}

// Función para generar un mensaje de ejemplo 1
func GenerateMessage1(nickname, token, roomId, roomName string) Message {
	return Message{
		MessageId:   uuid.New(),
		MessageType: Notification,
		SendDate:    time.Now(),
		ServerDate:  time.Now(),
		Nickname:    nickname,
		Token:       token,
		MessageText: "Este es un mensaje de ejemplo 1.",
		RoomID:      uuid.MustParse(roomId),
		RoomName:    roomName,
		Metadata: Metadata{
			AckStatus:    true,
			Priority:     1,
			OriginalLang: "es",
		},
	}
}

// Función para generar un mensaje de ejemplo 2
func GenerateMessage2(nickname, token, roomId, roomName string) Message {
	return Message{
		MessageId:   uuid.New(),
		MessageType: Notification,
		SendDate:    time.Now(),
		ServerDate:  time.Now(),
		Nickname:    nickname,
		Token:       token,
		MessageText: "Este es un mensaje de ejemplo 2.",
		RoomID:      uuid.MustParse(roomId),
		RoomName:    roomName,
		Metadata: Metadata{
			AckStatus:    false,
			Priority:     1,
			OriginalLang: "es",
		},
	}
}

func TransformToExternal(message *Message) (KafkaMessage, error) {
	// Crear el objeto KafkaMessage y mapear los valores correspondientes
	kafkaMsg := KafkaMessage{
		Key:       message.MessageId.String(),   // El Key es el MessageId (UUID como string)
		Value:     message.MessageText,          // El Value es el MessageText
		Headers:   make(map[string]interface{}), // Iniciar el mapa de headers
		Timestamp: time.Now(),
	}

	// Mapear los valores adicionales a los headers
	headers := map[string]interface{}{
		"MessageId":    message.MessageId.String(),
		"MessageType":  string(rune(Notification)),
		"SendDate":     message.SendDate.Format(time.RFC3339),
		"ServerDate":   message.ServerDate.Format(time.RFC3339),
		"Nickname":     message.Nickname,
		"Token":        message.Token,
		"RoomID":       message.RoomID.String(),
		"RoomName":     message.RoomName,
		"AckStatus":    fmt.Sprintf("%t", message.Metadata.AckStatus),
		"Priority":     fmt.Sprintf("%d", message.Metadata.Priority),
		"OriginalLang": message.Metadata.OriginalLang,
	}

	kafkaMsg.Headers = headers

	// Si tienes más campos en Message que necesitas incluir en el header, agrégales aquí
	// Ejemplo: kafkaMsg.Headers["OtroCampo"] = message.OtroCampo

	// Retornar el mensaje serializado en formato JSON
	return kafkaMsg, nil
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

// Función para realizar el login al API REST
func Login(nickname string, apiURL string) (*LoginResponse, error) {

	// Crear el objeto LoginRequest
	loginRequest := LoginRequest{
		Nickname: nickname,
		X_GoChat: "http://localhost:8081",
	}

	// Convertir la solicitud a JSON
	jsonData, err := json.Marshal(loginRequest)
	if err != nil {
		fmt.Printf("Error al serializar LoginRequest: %v\n", err)
		return &LoginResponse{
			Status:  "nok",
			Message: "Error al preparar la solicitud.",
		}, fmt.Errorf("error al preparar la solicitud")
	}

	// Crear la solicitud HTTP POST
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/login", apiURL), bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error al crear la solicitud HTTP: %v\n", err)
		return &LoginResponse{
			Status:  "nok",
			Message: "Error al preparar la solicitud HTTP.",
		}, fmt.Errorf("error al preparar la solicitud HTTP")
	}

	// Añadir encabezados
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x_GoChat", apiURL)

	// Enviar la solicitud
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error al realizar la solicitud HTTP: %v\n", err)
		return &LoginResponse{
			Status:  "nok",
			Message: "El servidor GoChat no está disponible. Disculpe las molestias.",
		}, fmt.Errorf("el servidor GoChat no está disponible. Disculpe las molestias")
	}
	defer resp.Body.Close()

	// Leer la respuesta
	var loginResp LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		return nil, fmt.Errorf("error al leer la respuesta del API REST: %v", err)
	}
	fmt.Printf("Respuesta del login: %s\n", loginResp)
	return &loginResp, nil
}

// Conectar al cliente Kafka
func ConnectTokafka(config map[string]interface{}) (*BrokerKafka, error) {
	log.Printf("ConnectTokafka: Vamos a conectar y enviar mensajes a kafka\n")
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
				return nil, fmt.Errorf("ConnectTokafka: 'brokers' contiene un valor no válido")
			}
		}
	default: // Si no es ninguno de los tipos esperados
		return nil, fmt.Errorf("ConnectTokafka: 'brokers' debe ser una cadena o una lista")
	}
	log.Printf("ConnectTokafka: Vamos a conectar y enviar mensajes a kafka\n")
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Return.Successes = true // Asegura confirmaciones de mensajes enviados
	kafkaConfig.Consumer.Return.Errors = true    // Permite manejar errores en el consumidor
	log.Printf("ConnectTokafka: kafkaConfig :[%v]", kafkaConfig)
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
	topics, err := GetTopics(config)
	if err != nil {
		return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker: error  cliente Kafka: %w", err)
	}
	var activeConsumers = make(map[string]sarama.PartitionConsumer)
	// Crear productores y consumidores según el sufijo del topic
	for _, topic := range topics {
		if strings.HasSuffix(topic, ".server") {
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
		} else if strings.HasSuffix(topic, ".client") {
			log.Printf("BrokerKafka: NewKafkaBroker Creando consumidor para el topic '%s'\n", topic)
			consum, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
			if err != nil {
				consumer.Close()
				producer.Close()
				return nil, fmt.Errorf("BrokerKafka: NewKafkaBroker error creando consumidor para el topic %s: %w", topic, err)
			}
			activeConsumers[topic] = consum
		}
	}
	// Retornar instancia del broker
	return &BrokerKafka{
		producer:        producer,
		consumer:        consumer,
		brokers:         brokerList,
		config:          config,
		activeConsumers: activeConsumers,
	}, nil
}
func GetTopicMessageCount(bs *BrokerKafka, topic string) (map[int32]int64, error) {
	// Obtener las particiones del topic
	partitions, err := bs.consumer.Partitions(topic)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo particiones: %v", err)
	}

	// Crear un mapa para almacenar el número de mensajes por partición
	partitionMessageCount := make(map[int32]int64)

	for _, partition := range partitions {
		// Obtener el cliente de Kafka
		client, err := sarama.NewClient(bs.brokers, nil)
		if err != nil {
			return nil, fmt.Errorf("error creando cliente de Kafka: %v", err)
		}
		defer client.Close()

		// Obtener el offset más bajo (más antiguo) y más alto (más reciente) de la partición
		lowOffset, err := client.GetOffset(topic, partition, sarama.OffsetOldest)
		if err != nil {
			return nil, fmt.Errorf("error obteniendo offset más bajo para partición %d: %v", partition, err)
		}

		highOffset, err := client.GetOffset(topic, partition, sarama.OffsetNewest)
		if err != nil {
			return nil, fmt.Errorf("error obteniendo offset más alto para partición %d: %v", partition, err)
		}

		// Calcular el número de mensajes en la partición
		numMessages := highOffset - lowOffset
		partitionMessageCount[partition] = numMessages
	}

	return partitionMessageCount, nil
}

func GetPartitionOffsets(bs *BrokerKafka, topic string, partition int32) (struct {
	lowOffset  int64
	highOffset int64
}, error) {
	// Crear el cliente de Kafka para obtener los offsets
	client, err := sarama.NewClient(bs.brokers, nil)
	if err != nil {
		return struct {
			lowOffset  int64
			highOffset int64
		}{}, fmt.Errorf("error creando cliente de Kafka: %v", err)
	}
	defer client.Close()

	// Obtener el offset más bajo (más antiguo) y más alto (más reciente) de la partición
	lowOffset, err := client.GetOffset(topic, partition, sarama.OffsetOldest)
	if err != nil {
		return struct {
			lowOffset  int64
			highOffset int64
		}{}, fmt.Errorf("error obteniendo offset más bajo para partición %d: %v", partition, err)
	}

	highOffset, err := client.GetOffset(topic, partition, sarama.OffsetNewest)
	if err != nil {
		return struct {
			lowOffset  int64
			highOffset int64
		}{}, fmt.Errorf("error obteniendo offset más alto para partición %d: %v", partition, err)
	}

	// Devolver los offsets como una estructura
	return struct {
		lowOffset  int64
		highOffset int64
	}{lowOffset, highOffset}, nil
}
func CalculateMessagesPerPartition(partitionMessageCount map[int32]int64, numMessages int) map[int32]int {
	messagesNeeded := numMessages
	partitionMessagesNeeded := make(map[int32]int)

	for partition, count := range partitionMessageCount {
		if messagesNeeded <= 0 {
			break
		}

		if int64(messagesNeeded) <= count {
			partitionMessagesNeeded[partition] = messagesNeeded
			messagesNeeded = 0
		} else {
			partitionMessagesNeeded[partition] = int(count)
			messagesNeeded -= int(count)
		}
	}

	return partitionMessagesNeeded
}
func ReadMessagesFromPartitions(bs *BrokerKafka, topic string, partitionMessagesNeeded map[int32]int) ([]KafkaMessage, error) {
	var messages []KafkaMessage
	var mu sync.Mutex // Protege la lista de mensajes
	var wg sync.WaitGroup
	errCh := make(chan error, len(partitionMessagesNeeded))

	for partition, needed := range partitionMessagesNeeded {
		wg.Add(1)
		go func(partition int32, needed int) {
			defer wg.Done()

			// Obtener los offsets de la partición
			offsets, err := GetPartitionOffsets(bs, topic, partition)
			if err != nil {
				errCh <- fmt.Errorf("error obteniendo offsets para partición %d: %v", partition, err)
				return
			}

			// Calcular el offset de inicio
			startOffset := offsets.highOffset - int64(needed)
			if startOffset < offsets.lowOffset {
				startOffset = offsets.lowOffset
			}

			// Crear un Reader para la partición
			r := kafka.NewReader(kafka.ReaderConfig{
				Brokers:     bs.brokers,
				Topic:       topic,
				Partition:   int(partition),
				StartOffset: startOffset,
				MinBytes:    10e3,
				MaxBytes:    10e6,
			})
			defer r.Close()

			// Leer los mensajes desde el offset inicial
			var partitionMessages []KafkaMessage
			for i := 0; i < needed; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
				msg, err := r.ReadMessage(ctx)
				cancel()
				if err != nil {
					errCh <- fmt.Errorf("error leyendo mensaje de partición %d: %v", partition, err)
					return
				}

				partitionMessages = append(partitionMessages, KafkaMessage{
					Key:       string(msg.Key),
					Value:     string(msg.Value),
					Headers:   ConvertKafkaHeadersToMap(msg.Headers),
					Timestamp: msg.Time,
				})
			}

			// Agregar mensajes de esta partición a la lista principal
			mu.Lock()
			messages = append(messages, partitionMessages...)
			mu.Unlock()
		}(partition, needed)
	}

	wg.Wait()
	close(errCh)

	// Si hubo errores, devolver el primero
	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}

	return messages, nil
}

func ConsumeLastMessages(bs *BrokerKafka, topic string, numMessages int) ([]KafkaMessage, error) {
	log.Printf("ConsumeLastMessages: topic: %s, numMessages: %d\n", topic, numMessages)

	// Obtener las particiones del topic
	partitions, err := bs.consumer.Partitions(topic)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo particiones: %v", err)
	}
	log.Printf("Número de particiones: [%d]\n", partitions)

	// Obtener el conteo total de mensajes por partición
	partitionMessageCount, err := GetTopicMessageCount(bs, topic)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo número de mensajes del topic: %v", err)
	}

	// Calcular el número total de mensajes en el tópico
	totalMessages := int64(0)
	for _, count := range partitionMessageCount {
		totalMessages += count
	}

	// Si hay menos mensajes que los requeridos, ajustar el límite
	if totalMessages < int64(numMessages) {
		numMessages = int(totalMessages)
	}

	// Calcular los mensajes necesarios por partición
	partitionMessagesNeeded := CalculateMessagesPerPartition(partitionMessageCount, numMessages)
	log.Printf("Mensajes necesarios por partición: %v\n", partitionMessagesNeeded)

	// Leer los mensajes de las particiones
	messages, err := ReadMessagesFromPartitions(bs, topic, partitionMessagesNeeded)
	if err != nil {
		return nil, fmt.Errorf("error leyendo mensajes de las particiones: %v", err)
	}

	// Ordenar los mensajes por Timestamp
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})

	// Retornar solo los últimos `numMessages` mensajes
	if len(messages) > numMessages {
		messages = messages[len(messages)-numMessages:]
	}

	// Asegurarse de cerrar el consumidor después de que hayamos terminado de consumir
	if bs.consumer != nil {
		// Si se utilizó un consumidor en esta función, lo cerramos explícitamente
		defer bs.consumer.Close()
	}

	log.Printf("ConsumeLastMessages: Se obtuvieron %d mensajes de topic: %s\n", len(messages), topic)
	return messages, nil
}
func ConvertKafkaHeadersToMap(kafkaHeaders []kafka.Header) map[string]interface{} {
	headersMap := make(map[string]interface{})
	for _, h := range kafkaHeaders {
		// Asumimos que el 'Key' de cada Header es único
		headersMap[string(h.Key)] = h.Value
	}
	return headersMap
}

// Solicitar lista de usuarios
func RequestUserList(bs *BrokerKafka, nickname string, token string, roomId uuid.UUID) error {
	log.Printf("* RequestUserList: nickname:[%s] -- token:[%s]\n", nickname, token)
	msg := RequestListUser{
		Operation:   "listusers",
		Nickname:    nickname,
		TokenSesion: token,
		RoomId:      roomId,
		Request:     nickname + ".client",
		Topic:       nickname + ".client",
		X_GoChat:    "http://localhost:8081",
	}
	fmt.Printf(" msg: [%v]\n", msg)
	kafkaMsg, err := ConvertToListuserMessage("roomlistusers.server", msg)
	fmt.Printf(" kafkaMsg: [%v]\n", kafkaMsg)
	if err != nil {
		return fmt.Errorf("error converting to KafkaMessage: %v", err)
	}
	errs := SendMessage(bs, "roomlistusers.server", kafkaMsg)
	if errs != nil {
		fmt.Printf(" Error al enviar la peticion de usuarios: %v\n", errs)
	}
	ConsumeClientTopic(bs, nickname+".client", true)

	return errs
}

func ConsumeClientTopic(bs *BrokerKafka, topic string, stopAfterFirstMessage bool) error {
	// Crear un Kafka Reader para el topic específico
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: bs.brokers,
		Topic:   topic,
	})
	defer func() {
		log.Printf("ConsumeClientTopic:: Cerrando el Kafka Reader para el topic [%s].\n", topic)
		reader.Close()
	}()

	log.Printf("ConsumeClientTopic:: Iniciando consumo de mensajes en el topic [%s].\n", topic)

	// Bucle para consumir mensajes
	for {
		log.Printf("ConsumeClientTopic:: Esperando mensaje en el topic [%s]...\n", topic)

		// Leer mensaje
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("ConsumeClientTopic:: Error al leer mensaje del topic [%s]: %v\n", topic, err)
			return err
		}

		// Mostrar información del mensaje
		log.Printf("ConsumeClientTopic:: Mensaje recibido del topic [%s]:\n", topic)
		log.Printf("  Offset: %d\n", msg.Offset)
		log.Printf("  Partition: %d\n", msg.Partition)
		log.Printf("  Key: %s\n", string(msg.Key))
		log.Printf("  Value: %s\n", string(msg.Value))

		// Mostrar headers del mensaje
		log.Printf("ConsumeClientTopic:: Headers del mensaje:\n")
		for _, header := range msg.Headers {
			log.Printf("  Header Key: %s, Header Value: %s\n", string(header.Key), string(header.Value))
		}

		// Si se desea detener después del primer mensaje
		if stopAfterFirstMessage {
			log.Printf("ConsumeClientTopic:: Deteniendo consumo tras recibir el primer mensaje en el topic [%s].\n", topic)
			break
		}

		log.Printf("ConsumeClientTopic:: Listo para recibir el siguiente mensaje en el topic [%s].\n", topic)
	}

	log.Printf("ConsumeClientTopic:: Finalizando consumo de mensajes en el topic [%s].\n", topic)
	return nil
}

func ConvertToListuserMessage(topic string, msgListUser RequestListUser) (KafkaMessage, error) {
	// Crear el mapa de headers
	headers := map[string]interface{}{
		"Nickname":    msgListUser.Nickname,
		"RoomId":      msgListUser.RoomId.String(),
		"Operation":   msgListUser.Operation,
		"Topic":       msgListUser.Topic,
		"Request":     msgListUser.Request,
		"TokenSesion": msgListUser.TokenSesion,
		"x_gochat":    msgListUser.X_GoChat,
	}

	// Crear la estructura del mensaje
	kfkMsg := KafkaMessage{
		Key:       msgListUser.Topic,
		Value:     "Petición de lista de usuarios", // Mensaje descriptivo en bytes
		Headers:   headers,
		Timestamp: time.Now(),
	}

	return kfkMsg, nil
}

// Convertir mensaje genérico a KafkaMessage
func ConvertToKafkaMessage(topic string, msg Message) (KafkaMessage, error) {
	return KafkaMessage{
		Key:   topic,
		Value: fmt.Sprintf("%+v", msg),
		Headers: map[string]interface{}{
			"MessageType": msg.MessageType,
			"Nickname":    msg.Nickname,
			"RoomName":    msg.RoomName,
		},
	}, nil
}

func SendMessageWithResponse(bs *BrokerKafka, generateMessageFunc func(string, string, string, string) Message, nickname, token, roomId, roomName string) error {
	// Generar el mensaje con la función proporcionada
	message := generateMessageFunc(nickname, token, roomId, roomName)

	// Transformar el mensaje a KafkaMessage
	kafkaMessage, err := TransformToExternal(&message)
	if err != nil {
		return fmt.Errorf("error transformando a KafkaMessage: %v", err)
	}
	topic := "principal.server"
	// Publicar el mensaje en el topic principal.server
	err = SendMessage(bs, topic, kafkaMessage)
	if err != nil {
		return fmt.Errorf("error enviando el mensaje a 'principal.server': %v", err)
	}
	log.Printf("Mensaje enviado al topic 'principal.server': %v", message)

	// Crear un canal para manejar la respuesta
	responseChannel := make(chan *sarama.ConsumerMessage)
	timeoutChannel := time.After(1 * time.Minute)

	// Crear un hilo para consumir respuestas del topic principal.client
	go func() {
		err := ConsumeTopic(bs, "principal.client", responseChannel)
		if err != nil {
			log.Printf("Error consumiendo del topic 'principal.client': %v\n", err)
		}
	}()

	// Esperar respuesta o timeout
	select {
	case msg := <-responseChannel:
		log.Printf("Respuesta recibida: Key=%s, Value=%s\n", string(msg.Key), string(msg.Value))
		// Procesar la respuesta
		response, err := ProcessResponse(msg.Value)
		if err != nil {
			log.Printf("Error procesando la respuesta: %v\n", err)
			return err
		}
		log.Printf("Respuesta procesada: %v", response)
	case <-timeoutChannel:
		log.Printf("Timeout de 1 minuto alcanzado sin recibir respuesta.\n")
	}

	return nil
}
func (bs *BrokerKafka) ConsumeMessages(topic string, responseChannel chan *sarama.ConsumerMessage) error {
	partitionConsumer, err := bs.consumer.ConsumePartition(topic, 0, sarama.OffsetOldest)
	if err != nil {
		return fmt.Errorf("error creando consumer para %s: %v", topic, err)
	}
	defer partitionConsumer.Close()

	for msg := range partitionConsumer.Messages() {
		responseChannel <- msg
		return nil // Salir después del primer mensaje
	}
	return nil
}

// Función para enviar mensajes al servidor
func SendMessageToServer(bs *BrokerKafka, generateMessageFunc func(string, string, string, string) Message, nickname, token, roomId, roomName string) error {
	// Generar el mensaje con la función proporcionada
	message := generateMessageFunc(nickname, token, roomId, roomName)

	// Transformar el mensaje a KafkaMessage
	kafkaMessage, err := TransformToExternal(&message)
	if err != nil {
		return fmt.Errorf("error transformando a KafkaMessage: %v", err)
	}
	topic := "principal.server"

	// Publicar el mensaje en el topic principal.server
	err = SendMessage(bs, topic, kafkaMessage)
	if err != nil {
		return fmt.Errorf("error enviando el mensaje a 'principal.server': %v", err)
	}
	log.Printf("Mensaje enviado al topic '%s': %v\n", topic, message)

	return nil
}

// Función para enviar mensajes sin esperar la respuesta directamente
func SendMessageChat(bs *BrokerKafka, topic string, kafkaMessage KafkaMessage) error {
	_, _, err := bs.producer.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(kafkaMessage.Key),
		Value: sarama.StringEncoder(kafkaMessage.Value),
	})
	if err != nil {
		return fmt.Errorf("error enviando mensaje al topic %s: %v", topic, err)
	}
	log.Printf("Mensaje enviado al topic '%s': %v\n", topic, kafkaMessage)
	return nil
}

// Función para iniciar el consumidor en un hilo separado
func StartConsumingMessages(bs *BrokerKafka, topic string, responseChannel chan *sarama.ConsumerMessage, stopChannel chan struct{}) {
	go func() {
		// Verificar si el consumidor ya existe y no está cerrado
		if bs.consumer == nil {
			log.Println("Error: No hay un consumidor disponible.")
			close(responseChannel)
			return
		}

		// Intentar crear un consumidor para la partición
		partitionConsumer, err := bs.consumer.ConsumePartition(topic, 0, sarama.OffsetOldest)
		if err != nil {
			log.Printf("Error creando consumer para %s: %v\n", topic, err)
			close(responseChannel)
			return
		}
		defer partitionConsumer.Close()

		// Bucle para recibir los mensajes del topic
		for {
			select {
			case msg := <-partitionConsumer.Messages():
				// Enviar el mensaje recibido al canal de respuesta
				responseChannel <- msg
			case <-stopChannel:
				// Si recibimos la señal para detener, salimos
				log.Printf("Parando consumidor para el topic %s\n", topic)
				return
			}
		}
	}()
}

// Función para consumir mensajes de un topic específico
func ConsumeTopic(bs *BrokerKafka, topic string, responseChannel chan *sarama.ConsumerMessage) error {
	partitionConsumer, err := bs.consumer.ConsumePartition(topic, 0, sarama.OffsetOldest)
	if err != nil {
		return fmt.Errorf("error creando consumer para %s: %v", topic, err)
	}
	defer partitionConsumer.Close()

	for msg := range partitionConsumer.Messages() {
		responseChannel <- msg
		break // Salir después de recibir el primer mensaje
	}

	return nil
}

// Función para procesar la respuesta
func ProcessResponse(value []byte) (*ResponseListUser, error) {
	var response ResponseListUser
	err := json.Unmarshal(value, &response)
	if err != nil {
		return nil, fmt.Errorf("error deserializando la respuesta: %v", err)
	}
	return &response, nil
}

// Enviar mensajes
func SendMessage(bs *BrokerKafka, topic string, kafkaMsg KafkaMessage) error {
	log.Printf("SendMessage(bs *BrokerKafka, kafkaMsg KafkaMessage)\n")

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(kafkaMsg.Value),
	}
	for key, value := range kafkaMsg.Headers {
		msg.Headers = append(msg.Headers, sarama.RecordHeader{
			Key:   []byte(key),
			Value: []byte(fmt.Sprintf("%v", value)),
		})
	}

	_, _, err := bs.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("error enviando mensaje: %v", err)
	}
	return nil
}
