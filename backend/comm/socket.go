package comm

import (
	"log"
	"net/http"
)

func StartWebSocketServer(address string) {
	// Crear la conexión HTTP en la dirección especificada
	http.HandleFunc("/ws", HandleConnections) // Llamar a la función que maneja las conexiones
	go func() {
		// Iniciar el servidor WebSocket
		if err := http.ListenAndServe(address, nil); err != nil {
			log.Println("Error al iniciar el servidor WebSocket:", err)
		}
	}()
}
