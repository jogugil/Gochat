import { UUID } from "../models/User";

export const TOKEN_NULO = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiIwIiwidXNlcm5hbWUiOiJkZWZhdWx0IiwiZXhwIjoxNjk4Mzg2Mjc5fQ.MGkJd5EYHQffQ9jrUzX7Djgmd4mOuH3aPRvcOP61TnM";
 


export interface LoginResponse {
    status: string;
    message: string;
    token: string;
    nickname: string;
    roomid: UUID;
    roomname: string;
  }

export interface JwtPayload {
    userid: string;
    username: string;
    exp?: number;  // La expiración es opcional
  }



    /*
  type ResponseUser struct {
    Status      string `json:"status"`
    Message     string `json:"message"`
    TokenSesion string `json:"tokenSesion"`
    Nickname    string `json:"nickname"`
    RoomId      string `json:"roomId"`
    AliveUsers  []struct {
      Nickname       string `json:"nickname"`
      LastActionTime string `json:"lastactiontime"`
    } `json:"data,omitempty"`
  }
  */
    // Estructuras que define los usuarios  activos que vienen del Servidor Gochat
  // Para el lsitado de mensajes
  // Interfaz para un mensaje
  export interface MessageResponse {
    messageid: string;    // UUID representado como string
    nickname: string;
    messagetext: string;
  }
  
  // Interfaz para la respuesta completa de la API
  export interface Response {
    status: string;        // El estado de la respuesta ("OK", "NOK", etc.)
    message: string;       // El mensaje de la respuesta (detalles del estado)
    tokensesion: string;   // El token de sesión
    nickname: string;      // El nombre de usuario asociado a la sesión
    roomid: string;        // El ID de la sala de chat
    x_gochat: string;      // El endpoint de WebSocket o información relacionada
    data: MessageResponse[];  // La lista de mensajes
  }
  
  // Definición de la interfaz para los usuarios activos
  export interface AliveUser {
    nickname: string;
    lastactiontime: string;
  }
  
  export interface ResponseUser {
    status: string;
    message: string;
    tokenSesion: string;
    nickname: string;
    roomId: string;
    X_GoChat: string;
    data: AliveUser[];  // Aquí cambiamos 'AliveUsers' por 'Data' y es un arreglo de objetos AliveUser
  }