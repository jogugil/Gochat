import axios from 'axios';
import { UUID } from '../models/Message'; 
import { LoginResponse } from '../types/typesComm';
import {WebSocketManager} from '../comm/WebSocketManager';
import { NatsStreamManager, NatsMessage} from 'comm/WebNatsManager';
 
 
const apiUrl = import.meta.env.VITE_API_URL; //'http://localhost:8081';

interface MessagesResponse {
  messages: string[];
}

interface UsersResponse {
  users: string[];
}
export const login = async (nickname: string): Promise<LoginResponse> => {
  
  console.log('login _ API URL:', apiUrl);

  try {
    const response = await axios.post(
      `${apiUrl}/login`,
      JSON.stringify({ nickname }),
      {
        headers: {
          'Content-Type': 'application/json',
          'x_gochat': apiUrl,
        },
      }
    );

    const data = response.data;

    // Asegurar un LoginResponse consistente
    return {
      status: data.status || 'nok',
      message: data.message || 'Error desconocido',
      token: data.token || '',
      nickname: data.nickname || nickname,
      roomid: data.roomid || '',
      roomname: data.roomname || '',
    } as LoginResponse;
  } catch (error: any) {
    console.warn('Error during login:', error);

    // Detectar errores específicos
    if (error.code === 'ERR_NETWORK') {
      return {
        status: 'nok',
        message: 'El servidor GoChat no está disponible. Disculpe las molestias.',
        token: '',
        nickname: nickname,
        roomid: '',
        roomname: '',
      } as LoginResponse;
    }

    // Manejar otros errores genéricos
    return {
      status: 'nok',
      message: 'Error durante el login. Inténtelo de nuevo más tarde.',
      token: '',
      nickname: nickname,
      roomid: '',
      roomname: '',
    } as LoginResponse;
  }
};
export const login_new = async (nickname: string): Promise<LoginResponse> => { 
    console.log('login API URL:', apiUrl);
    
    try {
      const loginRequest = {
      Nickname: nickname,              
      X_GoChat: 'http://localhost:8081' 
      };
    
      const response = axios.post(
        `${apiUrl}/login`,
        {
          Nickname: nickname,
          X_GoChat: 'http://localhost:8081'
        },
        { headers: { 'Content-Type': 'application/json', 'x_gochat': apiUrl } }
      );
      const data = (await response).data;
    
      // Asegurar un LoginResponse consistente
      return {
      status: data.status || 'nok',
      message: data.message || 'Error desconocido',
      token: data.token || '',
      nickname: data.nickname || nickname,
      roomid: data.roomid || '',
      roomname: data.roomname || '',
      } as LoginResponse;
    } catch (error: any) {
      console.warn('Error during login:', error);
      if (error.response && error.response.status === 401) {
        return { status: 'nok', nickname: nickname, message: 'Credenciales incorrectas.',
            token: '', roomid: '',  roomname: ''} as LoginResponse;
      }
      // Detectar errores específicos
      if (error.code === 'ERR_NETWORK') {
      return {
        status: 'nok',
        message: 'El servidor GoChat no está disponible. Disculpe las molestias.',
        token: '',
        nickname: nickname,
        roomid: '',
        roomname: '',
      } as LoginResponse;
      }
    
      // Manejar otros errores genéricos
      return {
      status: 'nok',
      message: 'Error durante el login. Inténtelo de nuevo más tarde.',
      token: '',
      nickname: nickname,
      roomid: '',
      roomname: '',
      } as LoginResponse;
    }
  };
 
// Enviar mensajes al servidor GoChat utilizando NATS
export const sendMessage = async (natsMessage: NatsStreamManager, nickName: string, token: string, roomId: string, roomName: string, message: string) => {
  const requestData = {
    nickname: nickName,           // Reemplaza con el nombre del usuario
    roomid: roomId,               // ID de la sala
    roomname: roomName,           // Nombre de la sala
    tokensession: token,          // El token de sesión
    message: message              // El mensaje que deseas enviar
  };

  try {
    console.log(`sendMessage: Enviando mensaje al topic 'principal.server' con el contenido: ${message}`);

    // Usar NATS para enviar el mensaje
    await natsMessage.sendNatsMessage(requestData, 'principal.server');  // Usar tu clase NatsStreamManager para enviar el mensaje
    
    console.log('Mensaje enviado correctamente a través de NATS.');

    // En este caso, no necesitamos una respuesta de servidor, solo confirmamos el envío
    return JSON.stringify({
      status: 'success',
      message: 'El mensaje fue enviado correctamente.'
    });

  } catch (error) {
    // Si hay un error al enviar el mensaje a través de NATS
    console.warn('Error al enviar el mensaje a NATS:', error);
    return JSON.stringify({
      status: 'error',
      message: 'El servidor está temporalmente fuera de servicio. Por favor, intenta nuevamente en un instante.'
    });
  }
};
/*
requestData := map[string]string{
  "operation":     "listmenssage",
  "nickname":      nickname,
  "roomid":        idsala,
  "tokensesion":   token,
  "lastmessageid": ultimoIdMensaje,
  "x_gochat":      "http://localhost:8081",
}
*/
// Obtener mensajes. Función que realiza la petición para obtener los mensajes
// Función para manejar la respuesta del WebSocket
// Definir la función callback para procesar la respuesta
const serializeMessage = (msg: NatsMessage): string => {
  // Convertir el Uint8Array a base64 antes de serializar
  const serializedMsg = {
    ...msg,
    data: btoa(String.fromCharCode(...msg.data)), // Convertir Uint8Array a base64
  };

  // Serializar el objeto completo a JSON
  return JSON.stringify(serializedMsg);
};
  
// Función para deserializar el mensaje NatsMessage desde un string JSON
const deserializeMessage = (jsonStr: string): NatsMessage => {
  const parsedMsg = JSON.parse(jsonStr);

  // Si hay un campo 'data', convertirlo de base64 a Buffer
  if (parsedMsg.data) {
    parsedMsg.data = Buffer.from(parsedMsg.data, 'base64');
  }

  return parsedMsg;
};

export const handleNatsMessage = (setMessages: React.Dispatch<React.SetStateAction<Map<string, { nickname: string; message: string }>>>) => {
  return (msg: NatsMessage) => {
    try {
      // Deserializar el mensaje NATS
      const messageData = JSON.parse(msg.data.toString());

      // Acceder al campo 'nickname' de los headers y comprobar su tipo
      const nickname = msg.headers['nickname'];

      if (typeof nickname === 'string') {
        // Si 'nickname' es una cadena, procesamos el mensaje
        setMessages((prevMessages) => {
          const newMessages = new Map(prevMessages); // Crear una copia del Map de mensajes
          newMessages.set(nickname, { nickname, message: messageData }); // Agregar el nuevo mensaje
          return newMessages;
        });
      } else {
        console.error('El campo "nickname" no es un string o no está presente en los headers');
      }
    } catch (error) {
      console.error("Error al procesar el mensaje:", error);
    }
  };
};


  /*
	requestData := struct {
		RoomId      string `json:"roomid"`
		TokenSesion string `json:"tokensesion"`
		Nickname    string `json:"nickname"`
		Operation   string `json:"operation"`
		X_GoChat    string `json:"x_gochat"`
	}{
		RoomId:      idsala,
		TokenSesion: token,
		Nickname:    nickname,
		Operation:   "listusers",
		X_GoChat:    "http://localhost:8081",


	}

  		var aliveUsers []AliveUsers
		for _, usuario := range usuarios {
			aliveUsers = append(aliveUsers, AliveUsers{
				Nickname:       usuario.Nickname,
				LastActionTime: usuario.LastActionTime.Format("2006-01-02 15:04:05"), // Formato estándar de fecha y hora
			})
		}

		// Preparar la respuesta
		respChan <- ResponseUser{
			Status:      "OK",
			Message:     "Usuarios activos obtenidos correctamente.",
			TokenSesion: requestData.TokenSesion,
			Nickname:    requestData.Nickname,
			RoomId:      requestData.RoomId,
			AliveUsers:  aliveUsers,
			X_GoChat:    urlClient,
		}
  */
export const getAliveUsers = async (
  socketManager: WebSocketManager, 
  nickname: string, 
  token: string, 
  roomId: string, 
  x_gochat: string
) => {
  
  
};