import axios from 'axios';
import { Message, UUID } from '../models/Message'; 
import { LoginResponse, MessageType, Metadata,  NatsMessage } from '../types/typesComm';
import {WebSocketManager} from '../comm/WebSocketManager';
import { NatsManager } from 'comm/WebNatsManager';
 
 
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
export const sendMessage = async (topic: string, natsManager: NatsManager, nickName: string, token: string, roomId: string, roomName: string, message: string) => {
  const metadata: Metadata = {
    ackStatus: true,           // Estado de confirmación
    priority: 1,               // Prioridad del mensaje
    originalLang: 'es',        // Idioma original
  };
  const requestData = new Message(
    nickName,                     // nombre de usuario
    message,                      // el mensaje que deseas enviar
    new Date().toISOString(),     // fecha de envío (sendDate)
    roomId,                       // ID de la sala
    token,                        // token de sesión
    roomName,                     // nombre de la sala
    metadata,                     // metadata (puede ser más detallada según lo necesites)
    MessageType.TEXT              // messageType (tipo de mensaje)
  );

  try {
    console.log(`sendMessage: Enviando mensaje al topic ${topic} con el contenido: ${message}`);

    // Usar NATS para enviar el mensaje
    await natsManager.publishMessage (requestData, topic);  // Usar tu clase NatsStreamManager para enviar el mensaje
    
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
  
// Función para deserializar el mensaje NatsManager desde un string JSON
const deserializeMessage = (jsonStr: string): NatsManager => {
  const parsedMsg = JSON.parse(jsonStr);

  // Si hay un campo 'data', convertirlo de base64 a Buffer
  if (parsedMsg.data) {
    parsedMsg.data = Buffer.from(parsedMsg.data, 'base64');
  }

  return parsedMsg;
};
export const handleNatsMessageWrapper = (
  setMessages: React.Dispatch<
    React.SetStateAction<Map<string, { nickname: string; message: string }>>
  >
) => {
  return (err: Error | null, msg: any, sharedObject: any) => {
    if (err) {
      console.error("Error recibido en el callback:", err);
      return;
    }

    try {
      console.log("Mensaje recibido en el wrapper:", msg);
      handleNatsMessage(setMessages)(msg as NatsMessage);
    } catch (error) {
      console.error("Error al procesar el mensaje en el wrapper:", error);
    }
  };
};

export const handleNatsMessage = (
  setMessages: React.Dispatch<
    React.SetStateAction<Map<string, { nickname: string; message: string }>>
  >
) => {
  return (msg: NatsMessage) => {
    try {
      console.log("Procesando mensaje:", msg);

      // Deserialización y procesamiento del mensaje
      const nickname = msg.headers?.nickname;
      const key = msg.subject ?? "default";
      const value = msg.data ? Buffer.from(msg.data).toString() : "";

      console.log("Detalles del mensaje:");
      console.log("  Subject:", key);
      console.log("  Nickname:", nickname);
      console.log("  Value:", value);

      if (typeof nickname === "string") {
        setMessages((prevMessages) => {
          const newMessages = new Map(prevMessages);
          newMessages.set(nickname, { nickname, message: value });
          return newMessages;
        });
      } else {
        console.error("El campo 'nickname' no es válido:", nickname);
      }
    } catch (error) {
      console.error("Error al procesar el mensaje:", error);
    }
  };
};
export const handleNatsGetAliveUsersWrapper = (
  setAliveUsers: React.Dispatch<React.SetStateAction<string[]>>
) => {
  return (err: Error | null, msg: any, sharedObject: any) => {
    if (err) {
      console.error("Error recibido en el callback de usuarios vivos:", err);
      return;
    }

    try {
      console.log("Mensaje recibido en el wrapper de usuarios vivos:", msg);
      handleNatsGetAliveUsers(setAliveUsers)(msg as NatsMessage);
    } catch (error) {
      console.error("Error al procesar el mensaje en el wrapper de usuarios vivos:", error);
    }
  };
};

export const handleNatsGetAliveUsers = (
  setAliveUsers: React.Dispatch<React.SetStateAction<string[]>>
) => {
  return (msg: NatsMessage) => {
    try {
      console.log("Procesando mensaje para usuarios vivos:", msg);

      // Verificar encabezados necesarios en el mensaje
      const status = msg.headers?.status;
      const aliveUsers = msg.headers?.aliveUsers;

      if (status === "OK" && Array.isArray(aliveUsers)) {
        // Extraer los nicknames de los usuarios vivos
        const aliveUsersList = aliveUsers.map((user: any) => user.Nickname).filter(Boolean);

        // Actualizar el estado con los usuarios vivos
        setAliveUsers(aliveUsersList);

        console.log("Usuarios vivos actualizados:", aliveUsersList);
      } else {
        console.error("El mensaje no contiene información válida sobre usuarios vivos:", msg);
      }
    } catch (error) {
      console.error("Error procesando el mensaje de usuarios vivos:", error);
    }
  };
};
