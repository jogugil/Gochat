package persistence

import (
	"log"
	"time"

	"backend/entities"
	"backend/types"

	"github.com/google/uuid"
)

func TestTuFuncionMongoDB() {
	// Simulando una instancia de MongoPersistencia
	uri := "mongodb://localhost:27017" // Cambia esto según tu URI de MongoDB
	dbName := "chatDB"
	persistencia, err := NuevaMongoPersistencia(uri, dbName)
	if err != nil {
		log.Fatalf("Error al crear la instancia de persistencia: %v", err)
	}
	idsala, err := uuid.Parse("10000000-0000-0000-0000-000000000010") //ponemos  un id de sala
	if err != nil {
		log.Printf("error a poner nil una sla del usuario: %v", err)
	}
	// Crear un usuario de prueba
	usuario := &entities.User{
		UserId:         uuid.New().String(),
		Nickname:       "usuario_test",
		Token:          "token_123",
		LastActionTime: time.Now(),
		State:          types.Activo,
		Type:           "regular",
		RoomId:         idsala, // Suponiendo que puede ser nil
		RoomName:       "SalaId",
	}

	// 1. Test de GuardarUsuario
	log.Println("Iniciando prueba de GuardarUsuario...")
	startTime := time.Now()
	err = (*persistencia).GuardarUsuario(usuario)
	duration := time.Since(startTime)
	if err != nil {
		log.Printf("Error en GuardarUsuario: %v\n", err)
	} else {
		log.Printf("GuardarUsuario completado en %v\n", duration)
	}

	// 2. Test de GuardarSala
	sala := entities.Room{
		RoomId:   uuid.New(),
		RoomName: "SalaTest",
	}
	log.Println("Iniciando prueba de GuardarSala...")
	startTime = time.Now()
	err = (*persistencia).GuardarSala(sala)
	duration = time.Since(startTime)
	if err != nil {
		log.Printf("Error en GuardarSala: %v\n", err)
	} else {
		log.Printf("GuardarSala completado en %v\n", duration)
	}

	// 3. Test de ObtenerSala
	log.Println("Iniciando prueba de ObtenerSala...")
	startTime = time.Now()
	_, err = (*persistencia).ObtenerSala(sala.RoomId)
	duration = time.Since(startTime)
	if err != nil {
		log.Printf("Error en ObtenerSala: %v\n", err)
	} else {
		log.Printf("ObtenerSala completado en %v\n", duration)
	}

	// 4. Test de GuardarMensaje
	mensaje := &entities.Message{
		MessageId:   uuid.New(),
		Nickname:    "usuario_test",
		SendDate:    time.Now(),
		RoomID:      sala.RoomId,
		RoomName:    sala.RoomName,
		Token:       "token_123",
		MessageText: "Este es un mensaje de prueba",
	}
	log.Println("Iniciando prueba de GuardarMensaje...")
	startTime = time.Now()
	err = (*persistencia).GuardarMensaje(mensaje)
	duration = time.Since(startTime)
	if err != nil {
		log.Printf("Error en GuardarMensaje: %v\n", err)
	} else {
		log.Printf("GuardarMensaje completado en %v\n", duration)
	}

	// 5. Test de ObtenerMensajesDesdeSala
	log.Println("Iniciando prueba de ObtenerMensajesDesdeSala...")
	startTime = time.Now()
	_, err = (*persistencia).ObtenerMensajesDesdeSala(sala.RoomId)
	duration = time.Since(startTime)
	if err != nil {
		log.Printf("Error en ObtenerMensajesDesdeSala: %v\n", err)
	} else {
		log.Printf("ObtenerMensajesDesdeSala completado en %v\n", duration)
	}

	// 6. Test de ObtenerMensajesDesdeId
	log.Println("Iniciando prueba de ObtenerMensajesDesdeId...")
	startTime = time.Now()
	_, err = (*persistencia).ObtenerMensajesDesdeSalaPorId(sala.RoomId, mensaje.MessageId)
	duration = time.Since(startTime)
	if err != nil {
		log.Printf("Error en ObtenerMensajesDesdeId: %v\n", err)
	} else {
		log.Printf("ObtenerMensajesDesdeId completado en %v\n", duration)
	}
}
