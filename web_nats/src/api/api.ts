import axios from 'axios';
import { Message, UUID } from '../models/Message'; 
import { LoginResponse, MessageType, Metadata,  NatsMessage, RequestListuser } from '../types/typesComm';
import { NatsManager, VITE_NATS_PREFIX_STREAMNAME, VITE_GET_USERS_TOPIC } from '../comm/WebNatsManager';
 
 
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
  setMessages: React.Dispatch<React.SetStateAction<Map<string, { nickname: string; message: string }>>>
) => {
  return (msg: NatsMessage) => {
    try {
      console.log("Procesando mensaje:", msg);

      // Verifica si el 'nickname' es un Buffer y conviértelo a string utilizando TextDecoder
      const nickname = msg.headers?.Nickname;

      // Usar TextDecoder si es un Uint8Array
      const nicknameString = nickname instanceof Uint8Array 
        ? new TextDecoder().decode(nickname) 
        : String(nickname); // Convertir a string si no es un Uint8Array

      if (!nicknameString) {
        console.error("El campo 'nickname' no es válido o está vacío:", nickname);
        return; // Si no hay un 'nickname', no procesamos el mensaje
      }

      const key = `${nicknameString}-${msg.timestamp}-${Math.random()}`; // Crear una clave única

      let value = "";
      if (msg.data) {
        if (msg.data instanceof Uint8Array) {
          value = new TextDecoder().decode(msg.data); // Decodificar si es necesario
        } else {
          value = msg.data.toString(); // Si los datos ya son texto
        }
      }

      console.log("Detalles del mensaje recibido:");
      console.log("  Nickname (remitente):", nicknameString);
      console.log("  Message (contenido):", value);

      setMessages((prevMessages) => {
        const newMessages = new Map(prevMessages); // Mantener los mensajes previos
        newMessages.set(key, { nickname: nicknameString, message: value }); // Usar la clave generada
        return newMessages; // Retornar el mapa con todos los mensajes
      });
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
  return (msg: any) => {
    console.log("********** Inicio de handleNatsGetAliveUsers **********");
    console.log("Mensaje recibido en handleNatsGetAliveUsers:", msg);

    try {
      // Verificar si el mensaje existe
      if (!msg) {
        console.warn("Mensaje recibido es nulo o indefinido.");
        return;
      }

      // Verificar la estructura del mensaje (JsMsgImpl)
      console.log("Propiedades del mensaje recibido:", Object.keys(msg));

      // Verificar si tiene `headers` y `data`
      console.log("Headers del mensaje:", msg.headers);
      console.log("Datos crudos del mensaje:", msg.data);

      // El dato relevante está en msg.headers.AliveUsers, que es un JSON en formato string.
      const aliveUsersString = msg.headers?.AliveUsers;

      // Verificar si AliveUsers está presente y es una cadena de texto
      if (typeof aliveUsersString === "string") {
        try {
          // Intentar parsear el JSON de AliveUsers
          const aliveUsers = JSON.parse(aliveUsersString);
          console.log("Lista de usuarios vivos deserializada:", aliveUsers);

          // Filtrar los nicknames de los usuarios activos
          const aliveUserNicknames = aliveUsers.map((user: any) => user?.nickname).filter(Boolean);

          // Actualizar el estado con los nicknames de los usuarios vivos
          setAliveUsers(aliveUserNicknames);
          console.log("Estado actualizado con los usuarios vivos:", aliveUserNicknames);
        } catch (error) {
          console.error("Error al intentar parsear el JSON de AliveUsers:", error);
        }
      } else {
        console.warn("El campo 'AliveUsers' no contiene un JSON válido.");
      }

    } catch (error) {
      console.error("Error procesando el mensaje de usuarios vivos:", error);
    }

    console.log("********** Fin de handleNatsGetAliveUsers **********");
  };
};
// Función para solicitar la lista de usuarios activos
export async function requestActiveUsers(
  natsManager: NatsManager,
  roomId: string, 
  tokenSesion: string, 
  nickname: string, 
  X_GoChat: string
): Promise<string> {
  const topic =  VITE_GET_USERS_TOPIC;
  const subject = `${VITE_NATS_PREFIX_STREAMNAME}_${VITE_GET_USERS_TOPIC}_server`;
  const request = `${VITE_NATS_PREFIX_STREAMNAME}_${nickname}_client`;
  try {
    // Crear el mensaje RequestListuser
    const message: RequestListuser = {
      roomId,
      tokenSesion,
      nickname,
      request:request,
      message: 'Solicitud de usuarios activos',
      operation: 'listUsers',
      topic: subject,
      X_GoChat:X_GoChat,
    };

    // Enviar el mensaje utilizando el método de natsManager
    await natsManager.publisRequestListUsershMessage(message, topic);

    // Si todo va bien, devolvemos 'ok'
    return 'ok';
  } catch (error) {
    // Si ocurre algún error, lo capturamos y lo mostramos
    console.error('Error al enviar la solicitud de lista de usuarios:', error);
    return `Error: ${error instanceof Error ? error.message : String(error)}`;
  }
}

function uuidv4() {
  throw new Error('Function not implemented.');
}
