package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

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

// Estructura para la respuesta del API REST
/*
	responseData := gin.H{
		"status":   "ok",
		"message":  "login realizado",
		"token":    usuario.Token,
		"nickname": usuario.Nickname,
		"roomid":   usuario.RoomId,   // Sala por defecto
		"roomname": usuario.RoomName, // Nombre de la sala
	}

*/
// KafkaMessage represents the structure of a message in Kafka
type KafkaMessage struct {
	Key       string                 `json:"key"`
	Value     string                 `json:"value"`
	Headers   map[string]interface{} `json:"headers"`
	Timestamp time.Time              `json:"timestamp"`
}

// NatsMessage represents the structure of a message in NATS
type NatsMessage struct {
	Subject   string                 `json:"subject"`
	Data      []byte                 `json:"data"`
	Headers   map[string]interface{} `json:"headers"`
	Timestamp time.Time              `json:"timestamp"`
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

// Estructura para los usuarios activos
type AliveUsers struct {
	Nickname       string `json:"nickname"`
	LastActionTime string `json:"lastactiontime"`
}

// Estructura para la respuesta general
type ResponseUser struct {
	Status      string       `json:"status"`
	Message     string       `json:"message"`
	TokenSesion string       `json:"tokenSesion"`
	Nickname    string       `json:"nickname"`
	RoomId      string       `json:"roomId"`
	X_GoChat    string       `json:"x_gochat"`
	AliveUsers  []AliveUsers `json:"data,omitempty"`
}
type RequestListuser struct {
	RoomId      uuid.UUID `json:"roomid"`
	TokenSesion string    `json:"tokensesion"`
	Nickname    string    `json:"nickname"`
	Operation   string    `json:"operation"`
	Topic       string    `json:"topic"`
	X_GoChat    string    `json:"x_gochat"`
}

// LoginRequest representa la estructura de la solicitud de login
type LoginRequest struct {
	Nickname string `json:"Nickname"`
	X_GoChat string `json:"X_GoChat"`
}

// LoginResponse representa la estructura de la respuesta del login
type LoginResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Token    string `json:"token"`
	Nickname string `json:"nickname"`
	RoomID   string `json:"roomid"`
	RoomName string `json:"roomname"`
}

// Login realiza el login al API REST y maneja posibles errores
func login(nickname string) LoginResponse {
	apiURL := "http://localhost:8081/login"
	// Crear el objeto LoginRequest
	loginRequest := LoginRequest{
		Nickname: nickname,
		X_GoChat: "http://localhost:8081",
	}

	// Convertir la solicitud a JSON
	jsonData, err := json.Marshal(loginRequest)
	if err != nil {
		fmt.Printf("Error al serializar LoginRequest: %v\n", err)
		return LoginResponse{
			Status:  "nok",
			Message: "Error al preparar la solicitud.",
		}
	}

	// Crear la solicitud HTTP POST
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/login", apiURL), bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error al crear la solicitud HTTP: %v\n", err)
		return LoginResponse{
			Status:  "nok",
			Message: "Error al preparar la solicitud HTTP.",
		}
	}

	// Añadir encabezados
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-gochat", apiURL)

	// Enviar la solicitud
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error al realizar la solicitud HTTP: %v\n", err)
		return LoginResponse{
			Status:  "nok",
			Message: "El servidor GoChat no está disponible. Disculpe las molestias.",
		}
	}
	defer resp.Body.Close()

	// Leer el cuerpo de la respuesta
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error al leer la respuesta: %v\n", err)
		return LoginResponse{
			Status:  "nok",
			Message: "Error al leer la respuesta del servidor.",
		}
	}

	// Parsear la respuesta JSON
	var loginResponse LoginResponse
	err = json.Unmarshal(body, &loginResponse)
	if err != nil {
		fmt.Printf("Error al deserializar la respuesta: %v\n", err)
		return LoginResponse{
			Status:  "nok",
			Message: "Error al procesar la respuesta del servidor.",
		}
	}

	// Retornar la respuesta parseada o con valores por defecto
	if loginResponse.Status == "" {
		loginResponse.Status = "nok"
	}
	if loginResponse.Message == "" {
		loginResponse.Message = "Error desconocido."
	}
	if loginResponse.Nickname == "" {
		loginResponse.Nickname = nickname
	}

	return loginResponse
}

func createStreamWithoutConflict(js nats.JetStreamContext, streamName string, subjects []string) (string, error) {
	// Obtener los nombres de los streams existentes
	streamNames := js.StreamNames()

	// Verificar si ya existe un stream con el nombre dado
	for existingStreamName := range streamNames {
		fmt.Printf(" st:[%s] - str:[%s] \n", existingStreamName, streamName)
		if existingStreamName == streamName {
			fmt.Printf(" equal --- st:[%s] - str:[%s] \n", existingStreamName, streamName)
			// Obtener la información del stream existente
			streamInfo, err := js.StreamInfo(existingStreamName)
			if err != nil {
				return "", fmt.Errorf("error al obtener información del stream '%s': %v", existingStreamName, err)
			}

			// Verificar si los subjects de este stream coinciden con los nuevos
			if equalSubjects(streamInfo.Config.Subjects, subjects) {
				fmt.Printf("El stream '%s' ya existe con los mismos subjects. Reutilizándolo.\n", existingStreamName)
				return existingStreamName, nil // Devolver el stream existente
			} else {
				fmt.Printf(" El stream '%s' no tiene los mismos subjects. Reutilizándolo.\n", existingStreamName)
				// Si los subjects no coinciden, agregar los nuevos subjects al stream existente
				updatedSubjects := append(streamInfo.Config.Subjects, subjects...)
				_, err := js.UpdateStream(&nats.StreamConfig{
					Name:     existingStreamName,
					Subjects: updatedSubjects,
				})
				if err != nil {
					return "", fmt.Errorf("error al actualizar el stream '%s' con nuevos subjects: %v", existingStreamName, err)
				}

				fmt.Printf("El stream '%s' actualizado con los nuevos subjects: %v\n", existingStreamName, updatedSubjects)
				// Devolver el JetStreamContext actualizado
				return existingStreamName, nil
			}
		} else {
			fmt.Println("Si no tienen el mismo nombrem comprueba si tienen los mismos topics. So los tiene se usa..")
			// Verificar si los subjects de este stream coinciden con los nuevos
			// Obtener la información del stream existente
			streamInfo, err := js.StreamInfo(existingStreamName)
			if err != nil {
				return "", fmt.Errorf("error al obtener información del stream '%s': %v", existingStreamName, err)
			}
			if equalSubjects(streamInfo.Config.Subjects, subjects) {
				fmt.Printf("El stream '%s' ya existe con los mismos subjects. Reutilizándolo.\n", existingStreamName)
				return existingStreamName, nil // Devolver el stream existente
			} else {
				continue
			}
		}
	}

	// Si no existe el stream con ese nombre ni con los mismos subjects, crearlo
	_, err := js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: subjects,
	})
	if err != nil {
		return "", fmt.Errorf("error al crear el stream '%s': %v", streamName, err)
	}

	fmt.Printf("Stream '%s' creado correctamente con subjects: %v\n", streamName, subjects)
	return streamName, nil
}

// Función para comparar si dos listas de subjects son iguales
func equalSubjects(subjects1, subjects2 []string) bool {
	fmt.Printf("Comparando subjects:\n")
	fmt.Printf("subjects1: %v\n", subjects1)
	fmt.Printf("subjects2: %v\n", subjects2)

	// Convertir los slices a sets (mapas) para evitar duplicados y simplificar la comparación
	subjectsMap1 := make(map[string]struct{})
	subjectsMap2 := make(map[string]struct{})

	// Llenar los mapas con los subjects de cada stream
	for _, subject := range subjects1 {
		subjectsMap1[subject] = struct{}{}
	}
	for _, subject := range subjects2 {
		subjectsMap2[subject] = struct{}{}
	}

	fmt.Printf("subjectsMap1: %v\n", subjectsMap1)
	fmt.Printf("subjectsMap2: %v\n", subjectsMap2)

	// Verificar si todos los subjects de subjects2 están en subjects1
	var iscontain = false
	for subject := range subjectsMap2 {
		if _, found := subjectsMap1[subject]; !found {
			fmt.Printf("El subject '%s' no está en subjects1.\n", subject)

		} else {
			fmt.Printf("El subject '%s'  está en subjects1.\n", subject)
			iscontain = true
			break
		}
	}

	return iscontain
}

func consumeHistoricalMessages(js nats.JetStreamContext, topic, streamName, consumerName string) ([]*nats.Msg, error) {
	// Obtener la información del stream
	streamInfo, errjs := js.StreamInfo(streamName)
	if errjs != nil {
		log.Fatal(errjs)
	}

	// Mostrar estadísticas del stream
	fmt.Printf("Stream: %s\n", streamInfo.Config.Name)
	fmt.Printf("Mensajes totales en el stream: %d\n", streamInfo.State.Msgs)
	fmt.Printf("Bytes totales en el stream: %d\n", streamInfo.State.Bytes)
	fmt.Printf("Mensajes no entregados: %d\n", streamInfo.State.Msgs)

	// Crear el consumidor como pull para recibir mensajes históricos
	fmt.Println("Creando el consumidor pull para mensajes históricos...")
	_, err := js.AddConsumer(streamName, &nats.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: topic,
		AckPolicy:     nats.AckExplicitPolicy,
		DeliverPolicy: nats.DeliverAllPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("error al crear el consumidor pull: %v", err)
	}
	fmt.Println("Consumidor pull creado correctamente")

	// Suscribirse como pull para recibir mensajes históricos
	sub, err := js.PullSubscribe(topic, consumerName)
	if err != nil {
		return nil, fmt.Errorf("error al suscribirse como pull: %v", err)
	}

	var allMessages []*nats.Msg
	totalMessages := streamInfo.State.Msgs // Número total de mensajes en el stream
	batchSize := 10                        // Cantidad de mensajes a obtener por vez
	totalFetched := uint64(0)              // Convertir totalFetched a uint64

	// Procesar mensajes históricos
	fmt.Println("Recibiendo mensajes históricos...")
	for totalFetched < totalMessages {
		msgs, err := sub.Fetch(batchSize, nats.MaxWait(2*time.Second))
		if err != nil {
			log.Printf("Error al recibir mensajes históricos: %v", err)
			break
		}
		if len(msgs) == 0 {
			// No hay más mensajes disponibles
			break
		}

		for _, msg := range msgs {
			if len(msg.Data) == 0 {
				// Ignorar mensajes vacíos
				continue
			}
			// Solo agregar mensajes no vacíos
			fmt.Printf("---> Mensaje histórico recibido: [%s]\n", string(msg.Data))
			msg.Ack()
			allMessages = append(allMessages, msg) // Guardar el mensaje en el slice
		}

		totalFetched += uint64(len(msgs)) // Convertir len(msgs) a uint64
		fmt.Printf("Mensajes procesados: %d/%d\n", totalFetched, totalMessages)
	}

	// Retornar los mensajes recibidos
	return allMessages, nil
}
func SetupConsumer(js nats.JetStreamContext, topic string, streamName string, consumerName string) {
	// Verificar si el consumidor 'principal-consumer' ya existe
	consumerInfo, err := js.ConsumerInfo(streamName, consumerName)
	if err == nil && consumerInfo != nil {
		fmt.Println("El consumidor ya existe, eliminando el existente...")
		// Si el consumidor existe, eliminarlo
		err := js.DeleteConsumer(streamName, consumerName)
		if err != nil {
			log.Fatalf("Error al eliminar el consumidor: %v", err)
		}
		fmt.Println("Consumidor eliminado exitosamente")
	}
	// Procesar mensajes históricos
	fmt.Println("Recibiendo mensajes históricos...")
	// Crear el consumidor como pull para recibir mensajes históricos
	msgs, err := consumeHistoricalMessages(js, topic, streamName, consumerName)
	if err != nil {
		log.Fatalf("Error al eliminar el consumidor: %v", err)
	}
	count := 0
	for _, msg := range msgs {
		fmt.Printf("---> mns(%d): Mensaje histórico recibido: [%s]\n", count, string(msg.Data))
		msg.Ack()
		count++
	}

	// Cambiar el consumidor a push
	fmt.Println("Cambiando el consumidor a push...")
	err = js.DeleteConsumer(streamName, consumerName)
	if err != nil {
		log.Fatalf("Error al eliminar el consumidor pull: %v", err)
	}

	_, err = js.AddConsumer(streamName, &nats.ConsumerConfig{
		Durable:        consumerName,
		FilterSubject:  "principal.client",
		AckPolicy:      nats.AckExplicitPolicy,
		DeliverPolicy:  nats.DeliverNewPolicy,
		DeliverSubject: fmt.Sprintf("%s-deliver", consumerName),
	})
	if err != nil {
		log.Fatalf("Error al crear el consumidor push: %v", err)
	}
	fmt.Println("Consumidor push creado correctamente")

	// Suscribirse como push para recibir nuevos mensajes
	_, err = js.Subscribe("principal.client", func(msg *nats.Msg) {
		fmt.Printf("-Sale [%s] -->>> Nuevo mensaje recibido : [%s]\n", streamName, string(msg.Data))
		msg.Ack()
	}, nats.Durable(consumerName))
	if err != nil {
		log.Fatalf("Error al suscribirse como push: %v", err)
	}
	fmt.Println("Suscripción push configurada correctamente")
}

// Función para conectar a NATS, enviar mensajes y recibir respuestas de forma concurrente
func connectToNATS(token, roomId, roomName, nickname string) *nats.Conn {
	// Conectarse al servidor NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Error al conectar a NATS: %v", err)
	}

	// Acceder a JetStream
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Error al acceder a JetStream: %v", err)
	}

	// Verificar si el stream 'principal' ya existe
	// Intentar crear el stream sin conflictos
	streamName, err := createStreamWithoutConflict(js, "principal", []string{"principal.client"})
	if err != nil {
		log.Fatalf("Error al manejar el stream: %v", err)
	}
	topic := "principal.client"
	consumerName := "principal-consumer"
	SetupConsumer(js, topic, streamName, consumerName)
	// Enviar mensajes al servidor después de la suscripción
	msg1 := GenerateMessage1(nickname, token, roomId, roomName)
	msg2 := GenerateMessage2(nickname, token, roomId, roomName)

	// Convertir los mensajes a NatsMessage
	natsMsg1, err := ConvertToNatsMessage("principal.server", msg1)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}
	natsMsg2, err := ConvertToNatsMessage("principal.server", msg2)
	if err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	// Convertir los mensajes a JSON
	msgBytes1, err := json.Marshal(natsMsg1)
	if err != nil {
		log.Fatalf("Error al convertir el mensaje a JSON: %v", err)
	}

	// Enviar el mensaje al servidor
	err = nc.Publish("principal.server", msgBytes1)
	if err != nil {
		log.Fatalf("Error al enviar el mensaje al servidor: %v", err)
	}
	fmt.Println("Mensaje enviado a principal.server:", string(msgBytes1))

	// Convertir el mensaje 2 a JSON
	msgBytes2, err := json.Marshal(natsMsg2)
	if err != nil {
		log.Fatalf("Error al convertir el mensaje a JSON: %v", err)
	}

	// Enviar el mensaje al servidor
	err = nc.Publish("principal.server", msgBytes2)
	if err != nil {
		log.Fatalf("Error al enviar el mensaje al servidor: %v", err)
	}
	fmt.Println("Mensaje enviado a principal.server:", string(msgBytes2))

	// Esperar un tiempo para que el goroutine reciba las respuestas antes de finalizar
	time.Sleep(10 * time.Second) // Ajustar el tiempo de espera según sea necesario

	return nc
}

// Función para convertir Message a NatsMessage
func ConvertToNatsMessage(topic string, msg Message) (NatsMessage, error) {
	// Crear el mapa de headers
	headers := map[string]interface{}{
		"MessageId":    msg.MessageId.String(),
		"MessageType":  string(rune(Notification)),
		"SendDate":     msg.SendDate.Format(time.RFC3339),
		"ServerDate":   msg.ServerDate.Format(time.RFC3339),
		"Nickname":     msg.Nickname,
		"Token":        msg.Token,
		"RoomID":       msg.RoomID.String(),
		"RoomName":     msg.RoomName,
		"AckStatus":    fmt.Sprintf("%t", msg.Metadata.AckStatus),
		"Priority":     fmt.Sprintf("%d", msg.Metadata.Priority),
		"OriginalLang": msg.Metadata.OriginalLang,
	}

	// Convertir el texto del mensaje a bytes
	data := []byte(msg.MessageText)

	// Crear el objeto NatsMessage
	natsMsg := NatsMessage{
		Subject: topic,
		Data:    data,
		Headers: headers,
	}
	return natsMsg, nil
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

func TransformToExternal(message *Message) ([]byte, error) {
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

	// Convertimos KafkaMessage a []byte
	rawMsg, err := json.Marshal(kafkaMsg)
	if err != nil {
		return nil, fmt.Errorf("error al serializar el mensaje de Kafka: %v", err)
	}

	// Retornar el mensaje serializado en formato JSON
	return rawMsg, nil
}
func getHeaderValue(headers map[string]interface{}, key, defaultValue string) string {
	if value, exists := headers[key]; exists {
		log.Printf("KafkaTransformer: getHeaderValue: value :%s", value)
		return value.(string)
	}
	log.Printf("KafkaTransformer: getHeaderValue: defaultValue :%s", defaultValue)
	return defaultValue
}
func CreateNatsMessage(msg RequestListuser) (NatsMessage, error) {

	// Crear el mapa de headers
	headers := map[string]interface{}{
		"Nickname":    msg.Nickname,
		"RoomId":      msg.RoomId.String(),
		"Operation":   msg.Operation,
		"Topic":       msg.Topic,
		"TokenSesion": msg.TokenSesion,
		"x_gochat":    msg.X_GoChat,
	}

	// Crear la estructura del mensaje
	natsMsg := NatsMessage{
		Subject: msg.Topic,
		Data:    []byte("Petición de lista de usuarios 22"), // Mensaje descriptivo en bytes
		Headers: headers,                                    // Headers contienen todos los atributos
	}

	return natsMsg, nil
}
func iniciarGestionUsuarios(js nats.JetStreamContext, nickname, idsala, token string) (<-chan NatsMessage, chan error) {
	// Canales para manejar las respuestas y errores
	responseChannel := make(chan NatsMessage)
	errorChannel := make(chan error)

	go func() {
		defer close(responseChannel)
		defer close(errorChannel)
		roomid, err := uuid.Parse(idsala)
		if err != nil {
			fmt.Printf("errorid sala id format: %v\n", err)
		}
		requestData := RequestListuser{
			RoomId:      roomid,
			TokenSesion: token,
			Nickname:    nickname,
			Operation:   "listusers",
			Topic:       nickname + ".client",
			X_GoChat:    "http://localhost:8081",
		}

		// Convertir a JSON
		req, err := CreateNatsMessage(requestData)
		if err != nil {
			fmt.Printf("Error al convertir el mensaje al formato Nats: %v", err)
		}
		jsonData, err := json.Marshal(req)
		if err != nil {
			errorChannel <- fmt.Errorf("error al convertir a JSON: %v", err)
			return
		}

		fmt.Printf(" [+] iniciarGestionUsuarios: requestData : [%v]\n", req)

		// Verificar o crear el stream para el cliente
		clientTopic := nickname + ".client"
		err = CreateOrUpdateStream(js, "ClientStream", []string{clientTopic})
		if err != nil {
			errorChannel <- fmt.Errorf("error al verificar/crear el stream para '%s': %v", clientTopic, err)
			return
		}

		// Verificar o crear el stream para el servidor
		serverTopic := "roomlistusers.server"
		err = CreateOrUpdateStream(js, "ServerStream", []string{serverTopic})
		if err != nil {
			errorChannel <- fmt.Errorf("error al verificar/crear el stream para '%s': %v", serverTopic, err)
			return
		}
		// Suscribirse al tópico del cliente
		subscription, err := js.PullSubscribe(clientTopic, "userlist")
		if err != nil {
			errorChannel <- fmt.Errorf("error al suscribirse al topic '%s': %v", clientTopic, err)
			return
		}
		defer subscription.Unsubscribe()

		// Publicar el mensaje al tópico del servidor
		_, err = js.Publish(serverTopic, jsonData)
		if err != nil {
			errorChannel <- fmt.Errorf("error al publicar en '%s': %v", serverTopic, err)
			return
		}

		// Recibir la respuesta
		msgs, err := subscription.Fetch(1, nats.MaxWait(35*time.Second)) // Timeout de 5 segundos
		if err != nil {
			errorChannel <- fmt.Errorf("error al recibir la respuesta o timeout: %v", err)
			return
		}

		if len(msgs) == 0 {
			errorChannel <- fmt.Errorf("no se recibieron mensajes en el tiempo esperado")
			return
		}

		// Parsear la respuesta
		var response NatsMessage
		err = json.Unmarshal(msgs[0].Data, &response)
		if err != nil {
			errorChannel <- fmt.Errorf("error al parsear la respuesta: %v", err)
			return
		}

		// Enviar la respuesta al canal
		responseChannel <- response
	}()

	return responseChannel, errorChannel
}

// Función para verificar o crear un stream
func CreateOrUpdateStream(js nats.JetStreamContext, streamName string, subjects []string) error {
	// Verificar si ya existe un stream con el mismo nombre
	streamInfo, err := js.StreamInfo(streamName)
	if err != nil && !errors.Is(err, nats.ErrStreamNotFound) {
		return fmt.Errorf("error al obtener información del stream: %v", err)
	}

	// Si el stream no existe, buscamos si hay otro stream con el mismo topic
	if streamInfo == nil {
		// Buscar si existe otro stream con los mismos subjects
		existingStreams := js.StreamNames()
		if existingStreams == nil {
			return fmt.Errorf("error al obtener los nombres de los streams")
		}

		var existingStream *nats.StreamInfo
		for existingStreamName := range existingStreams {
			info, err := js.StreamInfo(existingStreamName)
			if err != nil {
				continue // Si no se puede obtener la info del stream, lo ignoramos
			}
			// Si algún stream tiene el mismo topic (subject), lo usamos
			for _, subject := range info.Config.Subjects {
				for _, newSubject := range subjects {
					if subject == newSubject {
						existingStream = info
						break
					}
				}
				if existingStream != nil {
					break
				}
			}
			if existingStream != nil {
				break
			}
		}

		// Si encontramos un stream con el mismo subject, lo usamos
		if existingStream != nil {
			log.Printf("BrokerNats: El stream %s ya existe y contiene el topic, utilizaremos ese stream.\n", existingStream.Config.Name)
			return nil // Ya estamos usando el stream existente
		}

		// Si no existe un stream con el mismo topic, creamos uno nuevo
		_, err := js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: subjects,
		})
		if err != nil {
			return fmt.Errorf("error al crear el stream: %v", err)
		}
		log.Printf("BrokerNats: Stream %s creado con éxito.\n", streamName)
	} else {
		// Si el stream ya existe, solo actualizamos
		_, err := js.UpdateStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: subjects,
		})
		if err != nil {
			return fmt.Errorf("error al actualizar el stream: %v", err)
		}
		log.Printf("BrokerNats: Stream %s actualizado con éxito.\n", streamName)
	}

	return nil
}
func main() {
	// Pedir el nickname por consola
	var nickname string
	fmt.Print("Ingresa tu nickname: ")
	_, err := fmt.Scanln(&nickname)
	if err != nil {
		log.Fatalf("Error al leer el nickname: %v", err)
	}

	// Realizar login y obtener token, roomId y roomName
	loginResp := login(nickname)

	if loginResp.Status == "ok" {
		// Mostrar la información del login
		fmt.Printf("Login exitoso: NickName: %s Token: %s, RoomID: %s, RoomName: %s\n", loginResp.Nickname, loginResp.Token, loginResp.RoomID, loginResp.RoomName)

		// Conectar a NATS y enviar/recibir mensajes
		nc := connectToNATS(loginResp.Token, loginResp.RoomID, loginResp.RoomName, loginResp.Nickname)
		if nc != nil {
			// Crear contexto de JetStream
			js, err := nc.JetStream()
			if err != nil {
				log.Fatalf("Error al obtener el contexto JetStream: %v", err)
			}

			fmt.Println("Voy a obtener a los usuarios activos")

			// Llamar a iniciarGestionUsuarios
			responseChannel, errorChannel := iniciarGestionUsuarios(js, loginResp.Nickname, loginResp.RoomID, loginResp.Token)

			// Manejar respuestas y errores
			select {
			case response := <-responseChannel:
				fmt.Printf("Usuarios presentes en la sala: %v\n", response)
			case err := <-errorChannel:
				fmt.Printf("Error: %v\n", err)
			}

			fmt.Println("Proceso terminado.")
		}
		defer nc.Close()
	} else {
		fmt.Println("Error durante el login. Verifique su nickname y contraseña. Si el problema persiste")
		fmt.Println("E:", loginResp.Message)
	}

}
