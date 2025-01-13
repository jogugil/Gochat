import axios from 'axios';
import { Message, UUID } from '../models/Message'; 
import { LoginResponse, MessageType, Metadata } from '../types/typesComm';
import {WebSocketManager} from '../comm/WebSocketManager';
import { KafkaMessage} from '../types/typesComm';
import { KafkaManager } from 'comm/WebkafkaManager';
 
 
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
export const sendMessage = async (kafkaManager: KafkaManager, nickName: string, token: string, roomId: string, roomName: string, message: string) => {
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
    console.log(`sendMessage: Enviando mensaje al topic 'principal.server' con el contenido: ${message}`);

    // Usar NATS para enviar el mensaje
    await kafkaManager.sendKafkaMessage (requestData, 'principal.server');  // Usar tu clase NatsStreamManager para enviar el mensaje
    
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
const serializeMessage = (msg: KafkaMessage): string => {
  // Convertir el Uint8Array a base64 antes de serializar
  const serializedMsg = {
    ...msg,
    data: btoa(String.fromCharCode(...msg.value)), // Convertir Uint8Array a base64
  };

  // Serializar el objeto completo a JSON
  return JSON.stringify(serializedMsg);
};
  
// Función para deserializar el mensaje KafkaMessage desde un string JSON
const deserializeMessage = (jsonStr: string): KafkaMessage => {
  const parsedMsg = JSON.parse(jsonStr);

  // Si hay un campo 'data', convertirlo de base64 a Buffer
  if (parsedMsg.data) {
    parsedMsg.data = Buffer.from(parsedMsg.data, 'base64');
  }

  return parsedMsg;
};

export const handleKafkaMessage = (setMessages: React.Dispatch<React.SetStateAction<Map<string, { nickname: string; message: string }>>>) => {
  return (msg: KafkaMessage) => {
    try {
     

      // Acceder al campo 'nickname' de los headers y comprobar su tipo
      const nickname = msg.headers['nickname'];
      const key = msg.key ?? Buffer.from('');  // Default if key is null or undefined
      const value = msg.value ? Buffer.from(msg.value.toString()) : Buffer.from(''); // Default if value is null or undefined
      
      // Trazas de los atributos del mensaje
      console.log('Nuevo mensaje recibido:');
      console.log('  Key:', key.toString());  // Muestra la clave del mensaje
      console.log('  Nickname:', nickname);  // Muestra el nickname del mensaje
      console.log('  Value:', value.toString());  // Muestra el valor del mensaje

      if (typeof nickname === 'string') {
        // Si 'nickname' es una cadena, procesamos el mensaje
        setMessages((prevMessages) => {
          console.log(`Procesando mensaje para el nickname: ${nickname}`);

          const newMessages = new Map(prevMessages); // Crear una copia del Map de mensajes
          newMessages.set(nickname, { nickname, message: value.toString() }); // Agregar el nuevo mensaje
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

export const handleKafkaGetAliveUsers = (
      setAliveUsers: React.Dispatch<React.SetStateAction<string[]>>
    ) => {
      return (msg: KafkaMessage) => {
        try {
          // Inicializar arreglo de usuarios activos
          const aliveUsersList: string[] = [];
    
          // Obtener estado y lista de usuarios activos desde el mensaje
          const status = msg.headers['status'];
          const aliveUsers = msg.headers['aliveUsers'];
    
          // Validar estado y usuarios activos
          if (status === "OK" && Array.isArray(aliveUsers)) {
            // Extraer los nicknames
            const nicknames = aliveUsers.map((user: any) => user.Nickname);
            aliveUsersList.push(...nicknames);
    
            // Actualizar el estado
            setAliveUsers(aliveUsersList);
    
            console.log("Usuarios activos actualizados:", aliveUsersList);
          } else {
            console.error("El mensaje no contiene usuarios activos válidos:", msg);
          }
        } catch (error) {
          console.error("Error procesando el mensaje Kafka:", error);
        }
      };
};
