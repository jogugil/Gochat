import { MessageData} from '../types/types'; // Ajusta la ruta según la estructura de tu proyecto
import { validate as validateUUID } from 'uuid';

export type UUID = string;
export class Message implements MessageData {
    nickname: string;
    message: string;
    fecha: string;
  
    private _roomId: UUID    = "00000000-0000-0000-0000-000000000000"; 
    private _idMessage: UUID = "00000000-0000-0000-0000-000000000000"; 

    constructor(nickname: string, message: string, fecha: string, idMessage: string, roomId: string) {
      this.nickname  = nickname;
      this.message   = message;
      this.fecha     = fecha;
      this.roomId    = roomId;    // Esto llamará al setter que valida el UUID
      this.idMessage = idMessage; // Esto llamará al setter que valida el UUID
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
 