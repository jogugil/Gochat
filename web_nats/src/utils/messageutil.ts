// Obtener mensajes. Función que realiza la petición para obtener los mensajes
// Función para manejar la respuesta del WebSocket

import { NatsManager } from "comm/WebNatsManager";
import { debug } from "console";
import { MsgHdrs } from "nats.ws";
import { NatsMessage } from "types/typesComm";

// Definir la función callback para procesar la respuesta
export const serializeMessage = (msg: NatsMessage): string => {
    // Verificar si msg.data es un Uint8Array antes de convertirlo
    let dataBase64 = msg.data;
  
    if (msg.data instanceof Uint8Array) {
      // Convertir el Uint8Array a base64
      dataBase64 = btoa(String.fromCharCode(...msg.data));
    }
  
    // Crear el objeto serializado
    const serializedMsg = {
      ...msg,
      data: dataBase64, // Asignar la versión en base64
    };
  
    // Serializar el objeto completo a JSON
    return JSON.stringify(serializedMsg);
  };
  
  export const deserializeMessage = (jsonStr: string): NatsManager => {
    console.log("deserializeMessage:", jsonStr);
  
    const parsedMsg = JSON.parse(jsonStr);
  
    // Si hay un campo 'data', convertirlo de base64 a string
    if (parsedMsg.data) {
      const decodedStr = atob(parsedMsg.data);  // Decodifica de base64 a string
      parsedMsg.data = decodedStr;  // Asigna la cadena decodificada
    }
  
    return parsedMsg;
  };
export const createNatsMessageFromText = (topic: string, message: string): NatsMessage => {
    return {
      subject: topic,                   // El nombre del topic
      data: new TextEncoder().encode(message), // Convertir el mensaje de texto a Uint8Array
      timestamp: new Date().toISOString(),     // Timestamp actual
      headers: {},                        // Si necesitas agregar encabezados, puedes hacerlo aquí
    };
};
 