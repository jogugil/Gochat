package main

import (
	"clkafka/clkafka"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

const apiURL = "http://localhost:8081" // Cambia esta URL a la de tu servidor REST

// CargarConfig lee un archivo JSON y devuelve el mapa de configuración
func LoadConfig(filePath string) (map[string]interface{}, error) {
	// Abrir el archivo
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error abriendo el archivo: %v", err)
	}
	defer file.Close()

	log.Println("Archivo abierto correctamente")

	// Leer el contenido del archivo
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error leyendo el archivo: %v", err)
	}

	// Verificar que el archivo no esté vacío
	if len(content) == 0 {
		return nil, fmt.Errorf("el archivo está vacío")
	}

	//log.Printf("Contenido del archivo leído correctamente: %s\n", content)

	// Decodificar el JSON
	var config map[string]interface{}
	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("error decodificando JSON: %v", err)
	}

	log.Println("JSON decodificado correctamente")
	return config, nil
}

// Acceder a un valor seguro dentro de un map[string]interface{}
func LoadValue(config map[string]interface{}, clave string) (interface{}, error) {
	log.Printf(" * config :[%s]", config)
	log.Printf(" * clave :[%s]", clave)
	val, exists := config[clave]
	if !exists {
		return nil, fmt.Errorf("la clave '%s' no existe en la configuración", clave)
	}
	return val, nil
}

// Main
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Pedir el nickname por consola
	var nickname string
	fmt.Print("Ingresa tu nickname: ")
	_, err := fmt.Scanln(&nickname)
	if err != nil {
		log.Fatalf("Error al leer el nickname: %v", err)
	}

	// Realizar login y obtener token, roomId y roomName
	loginResp, err := clkafka.Login(nickname, apiURL)
	if err != nil {
		log.Fatalf("Error durante el login: %v", err)
	}

	if loginResp.Status == "ok" {
		// Mostrar la información del login
		fmt.Printf("Login exitoso: NickName: %s Token: %s, RoomID: %s, RoomName: %s\n", loginResp.Nickname, loginResp.Token, loginResp.Roomid, loginResp.Roomname)
		// Conexión al cliente Kafka
		configFile := "./gochat.json"

		log.Println("File opened successfully")

		var config map[string]interface{}
		config, err := LoadConfig(configFile)
		if err != nil {
			log.Printf("error al leer el fichero de configuración : %v\n", err)
			os.Exit(1)
		}
		log.Printf("config:[%s]", config)
		// Intentar acceder a un valor específico de la configuración
		kafkacfg, ok := config["kafka"].(map[string]interface{})
		if !ok {
			log.Fatalf("Error: la configuración de Kafka no es válida")
		}
		log.Printf("kafkacfg:[%s]", kafkacfg)

		bs, err := clkafka.ConnectTokafka(config)
		if err != nil {
			log.Fatalf("Error conectando a Kafka: %v\n", err)
		}
		log.Printf("Main: consumdores y productores creados [%v]\n", bs)
		// Consumir mensajes históricos
		historicalMessages, err := clkafka.ConsumeLastMessages(bs, "principal.client", 100)
		if err != nil {
			log.Fatalf("Error consumiendo mensajes históricos: %v\n", err)
		}
		log.Printf("main: Mensajes históricos\n")
		// Recorrer los mensajes uno por uno
		for i, msg := range historicalMessages {
			log.Printf("Mensaje %d: %+v\n", i+1, msg)
		}
		roomIdLogin, err := uuid.Parse(loginResp.Roomid)
		if err != nil {
			log.Printf("Error al parear el id Room: %v", err)
		}
		// Solicitar lista de usuarios
		err = clkafka.RequestUserList(bs, loginResp.Nickname, loginResp.Token, roomIdLogin)
		if err != nil {
			log.Fatalf("Error solicitando lista de usuarios: %v", err)
		}
		// Iniciar el canal de respuestas y el canal de parada
		responseChannel := make(chan *sarama.ConsumerMessage, 10) // Canal con buffer para varios mensajes
		stopChannel := make(chan struct{})                        // Canal para detener el consumidor

		// Llamar a la función de consumidor en un goroutine
		clkafka.StartConsumingMessages(bs, "principal.client", responseChannel, stopChannel)

		// Consumir mensajes personalizados
		err = clkafka.SendMessageToServer(bs, clkafka.GenerateMessage1, loginResp.Nickname, loginResp.Token, loginResp.Roomid, loginResp.Roomname)
		if err != nil {
			log.Fatalf("Error enviando mensaje con respuesta: %v", err)
		}
		// Consumir mensajes personalizados
		err = clkafka.SendMessageToServer(bs, clkafka.GenerateMessage2, loginResp.Nickname, loginResp.Token, loginResp.Roomid, loginResp.Roomname)
		if err != nil {
			log.Fatalf("Error enviando mensaje con respuesta: %v", err)
		}
		//Esperamos un tiempo y terminamos el proceso
		time.Sleep(5 * time.Second)
		close(stopChannel)
		log.Println("Consumidor detenido.")
	} else {
		fmt.Println("Error durante el login. Verifique su nickname y contraseña. Si el problema persiste")
		fmt.Println("E:", loginResp.Message)
	}
}
