package models

import (
	"backend/entities"
	"errors"
	"reflect"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	entities.Message // Embedding de entities.Message
}

// Implementación del método BuildMessage para cumplir con la interfaz IMessage
func (m *Message) BuildMessage(nickname, messageText, roomName string, roomID uuid.UUID, priority int, originalLang string) entities.Message {
	return entities.Message{
		MessageId:   uuid.New(),
		MessageType: entities.Text, // O puedes asignar el tipo que necesites
		SendDate:    time.Now(),
		ServerDate:  time.Now(),
		Nickname:    nickname,
		Token:       "sample-token", // Aquí deberías asignar un token real
		MessageText: messageText,
		RoomID:      roomID,
		RoomName:    roomName,
		Metadata: entities.Metadata{
			AckStatus:    false,
			Priority:     priority,
			OriginalLang: originalLang,
		},
	}
}

// Implementación del método Get para la interfaz IMessage
func (m *Message) Get(attribute string) interface{} {
	v := reflect.ValueOf(&m.Message)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(attribute)
	if field.IsValid() && field.CanInterface() {
		return field.Interface() // Devuelve el valor del campo
	}
	return nil // Si el campo no es válido o no existe
}

// Implementación del método Keys para la interfaz IMessage
func (m *Message) Keys() []string {
	// Listado de los campos de la estructura Message como claves
	return []string{
		"MessageId",
		"MessageType",
		"SendDate",
		"ServerDate",
		"Nickname",
		"Token",
		"MessageText",
		"RoomID",
		"RoomName",
		"Metadata",
	}
}

// Implementación del método Set
func (m *Message) Set(attribute string, value interface{}) {
	v := reflect.ValueOf(&m.Message)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName(attribute)
	if field.IsValid() && field.CanSet() {
		field.Set(reflect.ValueOf(value))
	}
}

// Implementación del método TransformFromExternal para la interfaz IMessage
// Este es solo un ejemplo, puedes implementarlo según tu lógica.
func (m *Message) TransformFromExternal(rawMsg []byte) error {
	// Aquí puedes convertir los datos rawMsg en los valores adecuados para Message
	// Este es un ejemplo básico, deberías implementar la conversión real según tus necesidades.
	if len(rawMsg) == 0 {
		return errors.New("rawMsg is empty")
	}
	// Aquí realizarías la lógica para mapear rawMsg en los campos de Message
	return nil
}

// Implementación del método TransformToExternal para la interfaz IMessage
// Este es solo un ejemplo, puedes implementarlo según tu lógica.
func (m *Message) TransformToExternal() ([]byte, error) {
	// Aquí puedes convertir los datos de Message a una representación externa (como JSON, XML, etc.)
	// Este es un ejemplo básico, deberías implementar la conversión real según tus necesidades.
	return []byte{}, nil
}
