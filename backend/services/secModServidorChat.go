package services

import (
	"backend/entities"
	"backend/models"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
)

type SecModServidorChat struct {
	GestionSalas    *GestionSalas
	GestionUsuarios *GestionUsuarios
}

var onceServerChat sync.Once
var instanciaSecMod *SecModServidorChat

func CrearSecModServidorChat(persistence *entities.Persistencia, configRoomFile string) *SecModServidorChat {
	onceServerChat.Do(func() {
		instanciaSecMod = &SecModServidorChat{
			GestionSalas:    NuevaGestionSalas(persistence, configRoomFile),
			GestionUsuarios: NuevaGestionUsuarios(),
		}
	})
	if instanciaSecMod.GestionSalas == nil {
		log.Fatalf("SecModServidorChat: CrearSecModServidorChat: GestionSalas no está configurado correctamente")
	}
	return instanciaSecMod
}

// Función para obtener la instancia existente, sin parámetros
func GetSecModServidorChat() (*SecModServidorChat, error) {
	if instanciaSecMod == nil {
		return nil, errors.New("SecModServidorChat: la instancia no ha sido creada. Debes llamar a CrearSecModServidorChat primero")
	}

	return instanciaSecMod, nil
}

// ValidarTokenAccion valida si un token es correcto y si el nickname está asociado a ese token
func (secMod *SecModServidorChat) ValidarTokenAccion(token, nickname, opt string) bool {
	log.Printf("SecModServidorChat: ValidarTokenAccion: Validando token: %s, nickname: %s, acción: %s\n", token, nickname, opt)
	usuario, err := secMod.GestionUsuarios.BuscarUsuarioPorToken(token)
	if err != nil || usuario.Nickname != nickname {
		log.Println("SecModServidorChat: ValidarTokenAccion: Token o nickname inválido")
		return false
	}

	// Validación de la acción (opt) se puede agregar aquí según el tipo de acción
	switch opt {
	case "enviarMensaje":
		log.Println("SecModServidorChat: ValidarTokenAccion: Acción válida: enviarMensaje")
		return true
	case "verSala":
		log.Println("SecModServidorChat: ValidarTokenAccion: Acción válida: verSala")
		return true
	default:
		log.Println("SecModServidorChat: ValidarTokenAccion: Acción no válida")
		return false
	}
}

// CrearTokenSesion genera un nuevo token para un usuario al registrarse
func (secMod *SecModServidorChat) CrearTokenSesion(nickname string) string {
	log.Printf("SecModServidorChat: CrearTokenSesion: Creando token de sesión para el usuario: %s\n", nickname)
	return models.CrearTokenSesion(nickname)
}

// EjecutarLogin maneja el proceso de login
func (secMod *SecModServidorChat) EjecutarLogin(nickname string) (*entities.User, error) {
	log.Printf("SecModServidorChat: EjecutarLogin:Ejecutando login para el usuario: %s\n", nickname)

	// Llamar a RegistrarUsuario para asegurarnos de que el usuario esté registrado
	if !secMod.GestionUsuarios.VerificarUsuarioExistente(nickname) {
		log.Println("SecModServidorChat: EjecutarLogin: CODL00:El nickname ya está en uso")
		return nil, fmt.Errorf("EjecutarLogin: CODL00:el nickname ya está en uso")
	}

	token := secMod.CrearTokenSesion(nickname)
	if token == "" {
		log.Println("SecModServidorChat: EjecutarLogin: CODL01:No se pudo crear el token")
		return nil, fmt.Errorf("EjecutarLogin: CODL01:no se pudo crear el token")
	}

	newUser, err := secMod.GestionUsuarios.RegistrarUsuario(nickname, token, secMod.GestionSalas.SalaPrincipal)
	if err != nil {
		log.Printf("SecModServidorChat: EjecutarLogin:CODL02:  Error al registrar el usuario: %v\n", err)
		return nil, fmt.Errorf("EjecutarLogin: CODL02: error al registrar el usuario %v", err)
	}

	secMod.GestionSalas.SalaPrincipal.Users = append(secMod.GestionSalas.SalaPrincipal.Users, *newUser)
	newUser.RoomId = secMod.GestionSalas.SalaPrincipal.RoomId
	newUser.RoomName = secMod.GestionSalas.SalaPrincipal.RoomName

	// Mostrar el mensaje de éxito
	log.Printf("SecModServidorChat: EjecutarLogin: Usuario %s logueado con el token %s\n", nickname, token)
	log.Printf("SecModServidorChat: EjecutarLogin: Sala principal: %s (ID: %v)\n", secMod.GestionSalas.SalaPrincipal.RoomName, secMod.GestionSalas.SalaPrincipal.RoomId)

	// Devolver el usuario con los detalles de la sala
	return newUser, nil
}

// EjecutarEnvioMensaje permite a un usuario enviar un mensaje
func (secMod *SecModServidorChat) EjecutarEnvioMensaje(nickname, token, mensaje string, idSala uuid.UUID) error {
	log.Printf("SecModServidorChat: EjecutarEnvioMensaje: Ejecutando envío de mensaje: %s para el usuario: %s en la sala: %v\n", mensaje, nickname, idSala)

	// Validar si el usuario está autorizado a enviar el mensaje

	token_usuario, err := secMod.GestionUsuarios.ObtenerTokenDeUsuario(nickname)
	if err != nil {
		log.Println("SecModServidorChat: EjecutarEnvioMensaje: Token inválido o acción no permitida")
		return errors.New("SecModServidorChat: EjecutarEnvioMensaje: CODM00:token inválido o acción no permitida")
	}

	if !secMod.ValidarTokenAccion(token_usuario, nickname, "enviarMensaje") {
		log.Println("SecModServidorChat: EjecutarEnvioMensaje: Token inválido o acción no permitida")
		return errors.New("SecModServidorChat: EjecutarEnvioMensaje: CODM01:token inválido o acción no permitida")
	}

	if token_usuario != token {
		log.Println("SecModServidorChat: EjecutarEnvioMensaje: Token no coincide")
		return errors.New("SecModServidorChat: EjecutarEnvioMensaje: CODM02:token inválido o acción no permitida")
	}
	usuario, err := secMod.GestionUsuarios.BuscarUsuarioPorToken(token_usuario)
	if err != nil {
		log.Printf("SecModServidorChat: EjecutarEnvioMensaje: CODM03:Error al BuscarUsuarioPorToken  : %v\n", err)
		return err
	}
	// Llamar a la lógica para enviar el mensaje
	err = secMod.GestionSalas.EnviarMensaje(idSala, nickname, mensaje, usuario)
	if err != nil {
		log.Printf("SecModServidorChat: EjecutarEnvioMensaje: CODM04: Error al enviar el mensaje: %v\n", err)
		return err
	}

	log.Printf("SecModServidorChat: EjecutarEnvioMensaje: CODM05:Mensaje enviado por %s: %s\n", nickname, mensaje)
	return nil
}

// EjecutarVerSala permite a un usuario ver la lista de mensajes en una sala
func (secMod *SecModServidorChat) EjecutarVerSala(nickname string, idSala uuid.UUID) error {
	log.Printf("SecModServidorChat: EjecutarVerSala: Ejecutando ver sala: %v para el usuario: %s\n", idSala, nickname)

	// Validar si el usuario está autorizado a ver la sala
	token, err := secMod.GestionUsuarios.ObtenerTokenDeUsuario(nickname)
	if err != nil {
		log.Println("SecModServidorChat: EjecutarVerSala: Token inválido o acción no permitida")
		return errors.New("SecModServidorChat: EjecutarVerSala: token inválido o acción no permitida")
	}

	if !secMod.ValidarTokenAccion(token, nickname, "verSala") {
		log.Println("SecModServidorChat: EjecutarVerSala: Token inválido o acción no permitida")
		return errors.New("SecModServidorChat: EjecutarVerSala: token inválido o acción no permitida")
	}

	// Llamar a la lógica para ver los mensajes de la sala
	mensajes, err := secMod.GestionSalas.ObtenerMensajesDesdeCola(idSala)
	if err != nil {
		log.Printf("SecModServidorChat: EjecutarVerSala: Error al obtener los mensajes de la sala: %v\n", err)
		return fmt.Errorf("SecModServidorChat: EjecutarVerSala: token inválido o acción no permitida %v", err)
	}

	// Mostrar los mensajes
	log.Println("SecModServidorChat: EjecutarVerSala: Mensajes en la sala:")
	for _, msg := range mensajes {
		log.Printf("[%s]: %s\n", msg.Nickname, msg.MessageText)
	}

	return nil
}
