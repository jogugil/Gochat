// src/types/types.ts
import { Message } from '../models/Message'; // Ajusta la ruta según la estructura de tu proyecto

export interface UserChat {
    nickname: string;
    id: string;     // Asumiendo que cada usuario tiene un identificador único
    status: string; // Puedes agregar más campos, como 'activo', 'ocupado', etc.
}
export interface RoomChat {
  usuarios: string[]; // Lista de nicknames de los usuarios
  messages: Message[];  // Lista de mensajes en la sala, de tipo Message
  nombre: string; // Nombre de la sala
}
export interface MessageData {
  nickname: string;
  message: string;
  fecha: string; // La fecha cuando el mensaje fue enviado (puede ser tipo string en formato ISO)
}
interface JwtPayload {
  userId: string;
  username: string;
  exp: number;
}
  