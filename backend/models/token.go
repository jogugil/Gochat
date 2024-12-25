package models

import (
	"backend/utils"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Token struct {
	Token string `json:"token"`
}

// Estructura que representa los claims del JWT (el payload)
type Claims struct {
	Nickname string `json:"nickname"`
	jwt.RegisteredClaims
}

// CrearTokenSesion genera un token JWT
func CrearTokenSesion(nickname string) string {
	// Obtener la clave secreta desde el entorno
	secretKey, err := utils.ObtenerVariableDeEntorno("SecretKey")
	if err != nil {
		log.Printf("Error al obtener la clave secreta: %v", err)
		return ""
	}

	// Definir los datos (claims) del token
	claims := &Claims{
		Nickname: nickname,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Token válido por 24 horas
			Issuer:    "ChatSphere",                                       // Emisor del token (puede ser un nombre o dominio)
		},
	}

	// Crear el token con los claims y firmarlo con la clave secreta
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Firmar el token
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		log.Println("Error al generar el token: ", err)
		return ""
	}

	return signedToken
}

// ValidarTokenSesion valida un token JWT y devuelve el nickname si es válido
func ValidarTokenSesion(tokenStr string) (string, error) {
	// Obtener la clave secreta desde el entorno
	secretKey, err := utils.ObtenerVariableDeEntorno("SecretKey")
	if err != nil {
		return "", fmt.Errorf("error al obtener la clave secreta: %v", err)
	}

	// Parsear el token usando la clave secreta
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verificar que el método de firma del token sea el esperado
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("el token tiene un método de firma inesperado: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	// Si hay error al parsear el token, devolver el error
	if err != nil {
		return "", fmt.Errorf("error al parsear el token: %v", err)
	}

	// Verificar que el token sea válido
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Comprobar si el token ha expirado
		if claims.ExpiresAt.Time.Before(time.Now()) {
			return "", fmt.Errorf("el token ha expirado")
		}
		// Si todo es correcto, devolver el nickname
		return claims.Nickname, nil
	}

	return "", fmt.Errorf("token no válido")
}
