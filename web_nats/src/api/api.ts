import axios from 'axios';
import { UUID } from '../models/Message'; 
import { LoginResponse } from '../types/typesComm';
import {WebSocketManager} from '../comm/WebSocketManager';
import { NatsStreamManager } from 'comm/WebNatsManager';

 
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
          'x-gochat': apiUrl,
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
private handleNatsMessage = (msg: NatsMessage) => {
  try {
    // Parsear el mensaje recibido
    const messageData = JSON.parse(this.codec.decode(msg.data));
    console.log("Mensaje recibido y procesado:", messageData);

    // Aquí puedes realizar cualquier procesamiento adicional de los datos recibidos
    const processedData = this.processMessage(messageData);

    // Enviar el mensaje procesado al frontend, por ejemplo, utilizando WebSocket
    this.sendToFrontend(processedData);
    
    // Acknowledge the message (manualmente, ya que estamos usando manualAck)
    msg.ack();
  } catch (error) {
    console.error("Error al procesar el mensaje:", error);
  }
};

// Función que realiza el procesamiento adicional del mensaje
private processMessage(messageData: any) {
  // Aquí puedes agregar cualquier lógica para procesar el mensaje
  // Por ejemplo, cambiar los datos, agregar campos adicionales, etc.
  console.log("Procesando mensaje:", messageData);
  return messageData; // Solo ejemplo, aquí deberías modificar el mensaje según tus necesidades
}

// Función para enviar el mensaje procesado al frontend (usando WebSocket, por ejemplo)
private sendToFrontend(processedData: any) {
  // Si tienes un WebSocket o cualquier otro mecanismo de comunicación, lo usas aquí
  // Por ejemplo:
  console.log("Enviando datos procesados al frontend:", processedData);
  // Ejemplo: this.websocket.emit('message', processedData);
}

// Conectar al consumidor y asignarle la función callback
async startConsumerWithCallback(topic: string) {
  try {
    const consumer = this.producersConsumers[topic]?.consumer;
    if (consumer) {
      // Asignamos la función de callback para manejar los mensajes del consumidor
      consumer.on('message', this.handleNatsMessage);
      console.log(`Consumer para el topic ${topic} iniciado con callback.`);
    } else {
      console.error(`No se encontró consumidor para el topic: ${topic}`);
    }
  } catch (error) {
    console.error("Error al iniciar el consumidor con callback:", error);
  }
}
 
 

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
): Promise<string> => {
  
  console.log ("<<<<<< Dentro de getAliveUsers >>>>>")

  // Crear los datos que se enviarán en el WebSocket
  const requestData = {
    roomid: roomId,
    tokensesion: token,
    nickname: nickname,
    operation: "listusers",
    x_gochat: x_gochat
  };

  try {
    // Obtener el WebSocket real desde WebSocketManager
    const ws = socketManager.getSocket();

    if (!ws) {
      throw new Error('WebSocket no disponible.');
    }
    console.log ("Obtengo el websocket con url:",ws?.url)
    socketManager.setOnMessageCallback(handleMessage);
    console.log (" he creado setOnMessageCallback " );
    // Promise que se resuelve cuando la conexión WebSocket se abre
   // Verificar si el WebSocket está abierto antes de enviar el mensaje
   if (ws.readyState === WebSocket.OPEN) {
    console.log("WebSocket ya está abierto, enviando datos:", requestData);
    ws.send(JSON.stringify(requestData));  // Enviar los datos a través del WebSocket
  } else {
    throw new Error('WebSocket no está abierto.');
  }

  // Esperar y procesar la respuesta
  return new Promise<string>((resolve, reject) => {
    ws.onmessage = (event) => {
      try {
        const result = handleMessage(event);  // Usamos el callback `handleMessage`
        console.log("Me llega el resultado response:", result);
        resolve(result);  // Devolvemos el resultado procesado
      } catch (error) {
        reject(error);
      }
    };

    // Manejar errores durante la conexión WebSocket
    ws.onerror = (err) => {
      console.error("Error de WebSocket:", err);
      reject(new Error('Error de WebSocket'));
    };
  });
} catch (error) {
  console.error('Error al conectar con WebSocket:', error);
  throw error;
}
};