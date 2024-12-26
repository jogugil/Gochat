package utils

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"time"
)

// Log the error to the log file
func LogCriticalError(errorMessage string) {
	// Get the log file path from the environment variable
	logFilePath, err := GetEnvVariable("LogFile")
	if err != nil {
		log.Fatal("Failed to retrieve the environment variable for the log file: ", err)
	}

	// Open the log file in "append" mode without deleting the existing content
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Error opening the log file: ", err)
	}
	defer file.Close()

	// Write the error to the log file with a timestamp
	timestamp := time.Now().Format(time.RFC3339)
	logMessage := fmt.Sprintf("[%s] ERROR: %s\n", timestamp, errorMessage)
	_, err = file.WriteString(logMessage)
	if err != nil {
		log.Fatal("Error writing to the log file: ", err)
	}

	// Send an email to the administrator for critical errors
	SendAdminEmail(errorMessage)
}

// Send an email to the administrator
func SendAdminEmail(errorMessage string) {
	// Get the administrator's email address from the environment variable
	adminEmail, err := GetEnvVariable("EmailAdmin")
	if err != nil {
		log.Fatal("Failed to retrieve the administrator's email: ", err)
	}

	// Configure the SMTP server
	// Retrieve email credentials from environment variables
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	smtpUser := os.Getenv("GMAIL_USER")     // Gmail account email
	smtpPassword := os.Getenv("GMAIL_PASS") // Gmail app password or account password

	// Configure the recipients
	from := smtpUser
	to := []string{adminEmail}

	// Create the email message
	subject := "Critical Server Error"
	body := fmt.Sprintf("A critical error occurred on the server: %s", errorMessage)
	msg := []byte("To: " + adminEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	// Connect to the SMTP server
	auth := smtp.PlainAuth("", smtpUser, smtpPassword, smtpHost)
	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, msg)
	if err != nil {
		log.Fatal("Error sending the email to the administrator: ", err)
	}
	fmt.Println("Email sent to the administrator with the error.")
}
