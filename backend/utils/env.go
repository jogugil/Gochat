// backend/utilities/env.go
package utils

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

// Estructura para almacenar las variables de entorno cargadas
var envVars map[string]string
var once sync.Once

// RecargarVariablesDeEntorno recarga las variables de entorno desde el archivo .env en caliente
func RecargarVariablesDeEntorno() {
	// Volver a cargar las variables de entorno desde el archivo .env
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error recargando el archivo .env: %w", err)
	}

	// Limpiar el mapa de variables anteriores
	envVars = make(map[string]string)

	// Cargar nuevamente las variables de entorno
	//for _, e := range os.Environ() {
	//parts := splitEnv(e)
	//if len(parts) == 2 {
	//envVars[parts[0]] = parts[1]
	//}
	//}
	//Cargamos las variables entorno una a una u le ponermos el nombre que quiera
	envVars["SecretKey"] = os.Getenv("SECRET_KEY")
	envVars["PortServer"] = os.Getenv("EXP_PORT")
	envVars["NameServer"] = os.Getenv("NAME_SERVER")
	envVars["URIMongo"] = os.Getenv("MONGODB_URI")
	envVars["NameMongo"] = os.Getenv("NAME_MONGO")
	envVars["SizeQueue"] = os.Getenv("EXP_SIZE_QMESSAGE")
	envVars["LogFile"] = os.Getenv("LOGFILE_GOCHAT")
	envVars["EmailAdmin"] = os.Getenv("EMAIL_ADMIN")
	envVars["GMAIL_USER"] = os.Getenv("GMAIL_USER")
	envVars["GMAIL_PASS"] = os.Getenv("GMAIL_PASS")
	envVars["GIN_MODE"] = os.Getenv("GIN_MODE")
}

// CargarVariablesDeEntorno carga las variables de entorno desde un archivo .env
func CargarVariablesDeEntorno() {
	once.Do(func() {
		// Cargar el archivo .env
		if err := godotenv.Load(); err != nil {
			log.Fatal("CargarVariablesDeEntorno: Error cargando el archivo .env")
		}

		// Inicializar el mapa para guardar las variables de entorno
		envVars = make(map[string]string)

		// Cargar todas las variables del archivo .env en memoria
		//for _, e := range os.Environ() {
		//parts := splitEnv(e)
		//if len(parts) == 2 {
		//envVars[parts[0]] = parts[1]
		//}
		//}
		//Cargamos las variables entorno una a una u le ponermos el nombre que quiera
		RecargarVariablesDeEntorno()
	})
}

// ObtenerVariableDeEntorno devuelve el valor de una variable de entorno cargada en memoria
func ObtenerVariableDeEntorno(nombreVariable string) (string, error) {
	valor, existe := envVars[nombreVariable]
	if !existe {
		return "", fmt.Errorf("la variable de entorno %s no está configurada", nombreVariable)
	}
	return valor, nil
}

// splitEnv divide una variable de entorno en su nombre y valor
func splitEnv(env string) []string {
	parts := make([]string, 2)
	equalsIndex := -1
	for i, c := range env {
		if c == '=' {
			equalsIndex = i
			break
		}
	}
	if equalsIndex != -1 {
		parts[0] = env[:equalsIndex]
		parts[1] = env[equalsIndex+1:]
	}
	return parts
}
