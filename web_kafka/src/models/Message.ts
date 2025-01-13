import { randomUUID } from 'crypto';
import { MessageData, Metadata, MessageType} from '../types/typesComm'; // Ajusta la ruta según la estructura de tu proyecto
import { validate as validateUUID } from 'uuid';
import { v4 as uuidv4 } from 'uuid'; 

export type UUID = string;
export class Message implements MessageData {
    messageId: string;      // Identificador único (UUID)
    messageType: MessageType; // Tipo de mensaje
    sendDate: string;       // Fecha de envío (ISO string)
    serverDate: string;     // Fecha en el servidor (ISO string)
    nickname: string;       // Nombre del remitente
    token: string;          // JWT del remitente
    messageText: string;    // Contenido del mensaje
    roomName: string;       // Nombre de la sala
    metadata: Metadata;     // Metadatos adicionales
  
    private _roomId: UUID    = "00000000-0000-0000-0000-000000000000"; 
    private _idMessage: UUID = "00000000-0000-0000-0000-000000000000"; 

    constructor(
        nickname: string,
        messageText: string,
        sendDate: string,
        roomId: string,
        token: string,
        roomName: string,
        metadata: Metadata,
        messageType: MessageType
    ) {
        this.nickname = nickname;
        this.messageText = messageText;
        this.sendDate = sendDate;
        this.serverDate = new Date().toISOString();  // Se asigna la fecha actual como fecha del servidor
        this.token = token;
        this.roomName = roomName;
        this.metadata = metadata;
        this.messageType = messageType;
        this.messageId =   uuidv4(); // Generación del UUID para el messageId
        this.roomId = roomId;        // Llamará al setter para validar el UUID
    }
 
    // Getter y Setter para idSala
    get idMessage(): UUID {
        return this.idMessage;
    }
    set idMessage(value: string) {
        if (!validateUUID(value)) {
            throw new Error('El ID de la sala no es un UUID válido');
        }
        this._idMessage = value; // Si la validación pasa, asigna el valor
    }
    // Getter y Setter para idSala
    get roomId(): UUID {
        return this._roomId;
    }
    set roomId(value: string) {
        if (!validateUUID(value)) {
            throw new Error('El ID de la sala no es un UUID válido');
        }
        this._roomId = value; // Si la validación pasa, asigna el valor
    }
    
  }
  // const msg = new Message('usuario1', 'Hola, ¿cómo estás?', new Date().toISOString(), '123e4567-e89b-12d3-a456-426614174000');
  // const msg = new Message('usuario2', '¡Hola!', new Date().toISOString(), 'invalid-uuid');
 