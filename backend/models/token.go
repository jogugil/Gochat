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

// Structure representing the JWT claims (payload)
type Claims struct {
	Nickname string `json:"nickname"`
	jwt.RegisteredClaims
}

// CreateSessionToken creates a JWT token
func CreateSessionToken(nickname string) string {
	// Get the secret key from the environment
	secretKey, err := utils.GetEnvVariable("SecretKey")
	if err != nil {
		log.Printf("Token: CreateSessionToken: Error retrieving the secret key: %v", err)
		return ""
	}

	// Define the token's claims
	claims := &Claims{
		Nickname: nickname,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Token valid for 24 hours
			Issuer:    "ChatSphere",                                       // Token issuer (can be a name or domain)
		},
	}

	// Create the token with claims and sign it with the secret key
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		log.Println("Token:  CreateSessionToken: Error generating the token: ", err)
		return ""
	}

	return signedToken
}

// ValidateSessionToken validates a JWT token and returns the nickname if valid
func ValidateSessionToken(tokenStr string) (string, error) {
	// Get the secret key from the environment
	secretKey, err := utils.GetEnvVariable("SecretKey")
	if err != nil {
		return "", fmt.Errorf("Token:  ValidateSessionToken: error retrieving the secret key: %v", err)
	}

	// Parse the token using the secret key
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the token's signing method is as expected
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Token:  ValidateSessionToken: unexpected signing method in token: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	// If there's an error parsing the token, return the error
	if err != nil {
		return "", fmt.Errorf("Token:  ValidateSessionToken: error parsing the token: %v", err)
	}

	// Verify the token is valid
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Check if the token has expired
		if claims.ExpiresAt.Time.Before(time.Now()) {
			return "", fmt.Errorf("Token:  ValidateSessionToken: the token has expired")
		}
		// If everything is correct, return the nickname
		return claims.Nickname, nil
	}

	return "", fmt.Errorf("Token:  ValidateSessionToken: invalid token")
}
