package models

import (
	"backend/entities"
	"encoding/json"
	"log"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// GenericMessageProxy es un proxy genérico para manejar mensajes de diferentes tipos (NATS, Kafka)
type GenericMessageProxy struct {
	message interface{} // El mensaje que se maneja (puede ser cualquier tipo de mensaje)
}

// Constructor para crear un nuevo proxy
func NewGenericMessageProxy(msg interface{}) *GenericMessageProxy {
	return &GenericMessageProxy{message: msg}
}

// Implementación de la interfaz MessageProxy para acceder a los atributos de los mensajes
func (p *GenericMessageProxy) Get(attribute string) interface{} {
	v := reflect.ValueOf(p.message)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(attribute)
	if field.IsValid() && field.CanInterface() {
		return field.Interface() // Devuelve el valor del campo como interface{}
	}
	return nil // Si el campo no es válido o no existe
}

func (p *GenericMessageProxy) Set(attribute string, value interface{}) {
	v := reflect.ValueOf(p.message)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(attribute)
	if field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(value)) // Establece el valor del campo
	}
}

func (p *GenericMessageProxy) Keys() []string {
	v := reflect.ValueOf(p.message)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	var keys []string
	for i := 0; i < v.NumField(); i++ {
		keys = append(keys, v.Type().Field(i).Name) // Devuelve los nombres de todos los campos
	}
	return keys
}

// Función para transformar un mensaje de NATS o Kafka a la estructura Message interna
func (p *GenericMessageProxy) TransformFromExternal(rawMsg []byte) error {
	// Intentamos transformar el mensaje en una estructura Message
	message := entities.Message{}
	err := json.Unmarshal(rawMsg, &message)
	if err != nil {
		log.Printf("Error al transformar el mensaje: %v", err)
		return err
	}
	p.message = message
	return nil
}

// Función para transformar un mensaje interno Message a un formato compatible con NATS o Kafka
func (p *GenericMessageProxy) TransformToExternal() ([]byte, error) {
	message, ok := p.message.(entities.Message)
	if !ok {
		log.Println("El mensaje no es del tipo esperado: Message")
		return nil, nil
	}

	// Convertimos el mensaje a JSON para enviarlo
	rawMsg, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error al transformar el mensaje a JSON: %v", err)
		return nil, err
	}
	return rawMsg, nil
}

// Función para construir un nuevo mensaje con los atributos necesarios
func (p *GenericMessageProxy) BuildMessage(nickname string, messageText string, roomID uuid.UUID, roomName string, priority int, originalLang string) entities.Message {
	// Crear un nuevo mensaje con los valores proporcionados
	message := entities.Message{
		MessageId:   uuid.New(),
		MessageType: entities.Text,
		SendDate:    time.Now(),
		ServerDate:  time.Now(),
		Nickname:    nickname,
		Token:       "sample-token", // Aquí deberías obtener el token real del remitente
		MessageText: messageText,
		RoomID:      roomID,
		RoomName:    roomName,
		Metadata: entities.Metadata{
			AckStatus:    false,
			Priority:     priority,
			OriginalLang: originalLang,
		},
	}

	return message
}

// Función para enviar un mensaje a través de NATS
func (p *GenericMessageProxy) SendMessageToNATS(nc *nats.Conn, message entities.Message) error {
	// Transformar el mensaje a formato adecuado (JSON) para NATS
	messageBytes, err := p.TransformToExternal()
	if err != nil {
		return err
	}

	// Publicar el mensaje en el canal NATS (usamos RoomID como el subject para organizar los mensajes por sala)
	err = nc.Publish(message.RoomID.String(), messageBytes)
	if err != nil {
		log.Printf("Error al enviar el mensaje a NATS: %v", err)
		return err
	}

	log.Println("Mensaje enviado exitosamente a NATS")
	return nil
}

// Función para leer un mensaje desde NATS
func (p *GenericMessageProxy) ReadMessageFromNATS(nc *nats.Conn, roomID uuid.UUID) ([]entities.Message, error) {
	// Suscribirse al canal NATS
	subscription, err := nc.SubscribeSync(roomID.String())
	if err != nil {
		log.Printf("Error al suscribirse a NATS: %v", err)
		return nil, err
	}

	var messages []entities.Message

	// Leer los mensajes desde NATS
	for {
		// Esperar hasta que se reciba un mensaje
		msg, err := subscription.NextMsg(1000) // 1000ms de espera
		if err != nil {
			log.Printf("Error al recibir mensaje de NATS: %v", err)
			break
		}

		// Transformar el mensaje recibido en la estructura Message
		err = p.TransformFromExternal(msg.Data)
		if err != nil {
			log.Printf("Error al transformar el mensaje recibido: %v", err)
			continue
		}

		// Agregar el mensaje a la lista
		messages = append(messages, p.message.(entities.Message))
	}

	return messages, nil
}
