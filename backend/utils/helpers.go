package utils

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"time"
)

// Registrar el error en el archivo de log
func LogCriticalError(errorMessage string) {
	// Obtener la ruta del archivo de log desde la variable de entorno
	logFilePath, err := ObtenerVariableDeEntorno("LogFile")
	if err != nil {
		log.Fatal("No se pudo obtener la variable de entorno para el archivo de log: ", err)
	}

	// Abrir el archivo de log en modo "append" (añadir) sin borrar el contenido existente
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Error al abrir el archivo de log: ", err)
	}
	defer file.Close()

	// Escribir el error en el archivo de log con una marca de tiempo
	timestamp := time.Now().Format(time.RFC3339)
	logMessage := fmt.Sprintf("[%s] ERROR: %s\n", timestamp, errorMessage)
	_, err = file.WriteString(logMessage)
	if err != nil {
		log.Fatal("Error al escribir en el archivo de log: ", err)
	}

	// Enviar un correo al administrador en caso de error crítico
	EnviarCorreoAdministrador(errorMessage)
}

// Enviar un correo electrónico al administrador
func EnviarCorreoAdministrador(errorMessage string) {
	// Obtener la dirección de correo del administrador desde la variable de entorno
	adminEmail, err := ObtenerVariableDeEntorno("EmailAdmin")
	if err != nil {
		log.Fatal("No se pudo obtener el correo del administrador: ", err)
	}

	// Configurar el servidor SMTP
	// Obtener las credenciales de correo desde las variables de entorno o definirlas
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	smtpUser := os.Getenv("GMAIL_USER")     // Email de la cuenta de Gmail
	smtpPassword := os.Getenv("GMAIL_PASS") // Contraseña o clave de aplicación de Gmail

	// Formato de los destinatarios
	from := smtpUser
	to := []string{adminEmail}

	// Configurar el mensaje
	subject := "Error crítico en el servidor"
	body := fmt.Sprintf("Se ha producido un error crítico en el servidor: %s", errorMessage)
	msg := []byte("To: " + adminEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	// Conectar con el servidor SMTP
	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)
	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, msg)
	if err != nil {
		log.Fatal("Error al enviar el correo al administrador: ", err)
	}
	fmt.Println("Correo enviado al administrador con el error.")
}
