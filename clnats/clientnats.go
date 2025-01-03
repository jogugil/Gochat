package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

const apiURL = "http://localhost:8081/login" // Cambia esta URL a la de tu servidor REST

// Estructura para el cuerpo de la solicitud POST al API REST
type LoginRequest struct {
	Nickname string `json:"nickname"`
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
	Key     string            `json:"key"`
	Value   string            `json:"value"`
	Headers map[string]string `json:"headers"`
}

// NatsMessage represents the structure of a message in NATS
type NatsMessage struct {
	Subject string            `json:"subject"`
	Data    []byte            `json:"data"`
	Headers map[string]string `json:"headers"`
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
	RoomId      string `json:"roomid"`
	TokenSesion string `json:"tokensesion"`
	Nickname    string `json:"nickname"`
	Operation   string `json:"operation"`
	Topic       string `json:"topic"`
	X_GoChat    string `json:"x_gochat"`
}

// Función para realizar el login al API REST
func login(nickname string) (*LoginResponse, error) {
	loginReq := LoginRequest{
		Nickname: nickname,
	}

	// Convertir el loginReq a JSON
	reqBody, err := json.Marshal(loginReq)
	if err != nil {
		return nil, fmt.Errorf("error al convertir la solicitud de login a JSON: %v", err)
	}

	// Hacer la solicitud POST al API REST
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error al realizar solicitud POST: %v", err)
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
func SetupConsumer(js nats.JetStreamContext, streamName string, consumerName string) {
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

	// Crear el consumidor como pull para recibir mensajes históricos
	fmt.Println("Creando el consumidor como pull para mensajes históricos...")
	_, err = js.AddConsumer(streamName, &nats.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: "principal.client",
		AckPolicy:     nats.AckExplicitPolicy,
		DeliverPolicy: nats.DeliverAllPolicy,
	})
	if err != nil {
		log.Fatalf("Error al crear el consumidor pull: %v", err)
	}
	fmt.Println("Consumidor pull creado correctamente")

	// Suscribirse como pull para recibir mensajes históricos
	sub, err := js.PullSubscribe("principal.client", consumerName)
	if err != nil {
		log.Fatalf("Error al suscribirse como pull: %v", err)
	}

	// Procesar mensajes históricos
	fmt.Println("Recibiendo mensajes históricos...")
	for {
		msgs, err := sub.Fetch(10, nats.MaxWait(2*time.Second))
		if err != nil {
			log.Printf("Error al recibir mensajes históricos: %v", err)
			break
		}
		if len(msgs) == 0 {
			// No hay más mensajes históricos
			break
		}
		count := 0
		for _, msg := range msgs {
			fmt.Printf("---> mns(%d): Mensaje histórico recibido: [%s]\n", count, string(msg.Data))
			msg.Ack()
			count++
		}
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
	defer nc.Close()

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
	consumerName := "principal-consumer"
	SetupConsumer(js, streamName, consumerName)
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
	headers := map[string]string{
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
func CreateNatsMessage(msg RequestListuser) (NatsMessage, error) {

	// Crear el mapa de headers
	headers := map[string]string{
		"Nickname":    msg.Nickname,
		"RoomId":      msg.RoomId,
		"Operation":   msg.Operation,
		"Topic":       msg.Topic,
		"TokenSesion": msg.TokenSesion,
		"x_gochat":    msg.X_GoChat,
	}

	// Crear la estructura del mensaje
	natsMsg := NatsMessage{
		Subject: msg.Topic,
		Data:    []byte("Petición de lista de usuarios"), // Mensaje descriptivo en bytes
		Headers: headers,                                 // Headers contienen todos los atributos
	}

	return natsMsg, nil
}
func iniciarGestionUsuarios(js nats.JetStreamContext, nickname, idsala, token string) (<-chan ResponseUser, chan error) {
	// Canales para manejar las respuestas y errores
	responseChannel := make(chan ResponseUser)
	errorChannel := make(chan error)

	go func() {
		defer close(responseChannel)
		defer close(errorChannel)

		requestData := RequestListuser{
			RoomId:      idsala,
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
		msgs, err := subscription.Fetch(1, nats.MaxWait(5*time.Second)) // Timeout de 5 segundos
		if err != nil {
			errorChannel <- fmt.Errorf("error al recibir la respuesta o timeout: %v", err)
			return
		}

		if len(msgs) == 0 {
			errorChannel <- fmt.Errorf("no se recibieron mensajes en el tiempo esperado")
			return
		}

		// Parsear la respuesta
		var response ResponseUser
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
	streamInfo, err := js.StreamInfo(streamName)
	if err != nil && !errors.Is(err, nats.ErrStreamNotFound) {
		return fmt.Errorf("error al obtener información del stream: %v", err)
	}

	if streamInfo == nil {
		// Crear el stream si no existe
		_, err := js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: subjects,
		})
		if err != nil {
			return fmt.Errorf("error al crear el stream: %v", err)
		}
	} else {
		// Actualizar el stream si existe
		_, err := js.UpdateStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: subjects,
		})
		if err != nil {
			return fmt.Errorf("error al actualizar el stream: %v", err)
		}
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
	loginResp, err := login(nickname)
	if err != nil {
		log.Fatalf("Error durante el login: %v", err)
	}

	if loginResp.Status == "ok" {
		// Mostrar la información del login
		fmt.Printf("Login exitoso: NickName: %s Token: %s, RoomID: %s, RoomName: %s\n", loginResp.Nickname, loginResp.Token, loginResp.Roomid, loginResp.Roomname)

		// Conectar a NATS y enviar/recibir mensajes
		nc := connectToNATS(loginResp.Token, loginResp.Roomid, loginResp.Roomname, loginResp.Nickname)
		if nc != nil {
			// Crear contexto de JetStream
			js, err := nc.JetStream()
			if err != nil {
				log.Fatalf("Error al obtener el contexto JetStream: %v", err)
			}

			fmt.Println("Voy a obtener a los usuarios activos")

			// Llamar a iniciarGestionUsuarios
			responseChannel, errorChannel := iniciarGestionUsuarios(js, loginResp.Nickname, loginResp.Roomid, loginResp.Token)

			// Manejar respuestas y errores
			select {
			case response := <-responseChannel:
				fmt.Printf("Usuarios presentes en la sala: %v\n", response)
			case err := <-errorChannel:
				fmt.Printf("Error: %v\n", err)
			}

			fmt.Println("Proceso terminado.")
		}

	} else {
		fmt.Println("Error durante el login. Verifique su nickname y contraseña. Si el problema persiste")
		fmt.Println("E:", loginResp.Message)
	}
}
