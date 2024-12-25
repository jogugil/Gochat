import { validate as validateUUID } from 'uuid';
import { RoomChat } from '../types/types'; // Ajusta la ruta según la estructura de tu proyecto
import { MessageResponse, ResponseUser, AliveUser } from '../types/typesComm'; // Ajusta la ruta según la estructura de tu proyecto
import { Message } from '../models/Message'; // Ajusta la ruta según la estructura de tu proyecto

export type UUID = string;

// Clase Room que implementa la interfaz RoomChat y valida el idRoom
export class Room implements RoomChat {
  _roomid: UUID = "00000000-0000-0000-0000-000000000000"; 
  nombre: string;
  usuarios: string[];
  messages: Message[] = [];
  private lastIdMessage: UUID = "00000000-0000-0000-0000-000000000000";

  constructor(roomId: string, nombre: string) {
    this.roomId = roomId; // Valida el ID de la sala al inicializar
    this.nombre = nombre;
    this.usuarios = []; // Inicializa la lista de usuarios vacía
  }

  // Getter y Setter para idRoom
  get roomId(): UUID {
    return this._roomid;
  }

  set roomId(value: string) {
    // Utilizando validateUUID del paquete uuid
    if (!validateUUID(value)) {
      throw new Error('El ID de la sala no es un UUID válido');
    }
    this._roomid = value; // Si la validación pasa, asigna el valor
  }

  // Getter y Setter para nombre
  get roomName(): string {
    return this.nombre;
  }

  set roomName(value: string) {
    this.nombre = value;
  }

  // Getter y Setter para usuarios
  get userList(): string[] {
    return this.usuarios;
  }

  set userList(usuarios: string[]) {
    this.usuarios = usuarios;
  }

  // Getter y Setter para messages
  get messageList(): Message[] {
    return this.messages;
  }

  set messageList(messages: Message[]) {
    this.messages = messages;
  }

  // Getter y Setter para lastIdMessage
  get lastIDMessageId(): UUID | string {
    return this.lastIdMessage;
  }

  set lastIDMessageId(id: UUID) {
    this.lastIdMessage = id;
  }

  // Método para añadir un usuario a la sala
  addUser(nickname: string): void {
    if (!this.usuarios.includes(nickname)) {
      this.usuarios.push(nickname);
    } else {
      console.log('El usuario ya está en la sala');
    }
  }

  // Método para eliminar un usuario de la sala
  removeUser(nickname: string): void {
    const index = this.usuarios.indexOf(nickname);
    if (index > -1) {
      this.usuarios.splice(index, 1);
    } else {
      console.log('El usuario no está en la sala');
    }
  }
  // Nueva función para procesar el JSON y añadir usuarios
  addUsersFromJson(json: string): void {
    try {
      const response: ResponseUser = JSON.parse(json);

      // Validamos que la sala del JSON coincida con la sala de la instancia
      if (response.roomId !== this.roomId) {
        console.log("La sala en el JSON no coincide con la sala de la instancia.");
        return;
      }

      // Añadir los usuarios activos a la lista de usuarios de la sala
      response.data.forEach(usuario => {
        this.addUser(usuario.nickname); // Añade cada usuario a la sala
      });

      console.log(`${response.data.length} usuarios añadidos a la sala.`);
    } catch (error) {
      console.warn("Error al procesar el JSON:", error);
    }
  }
  // Método para actualizar los mensajes de la sala con los datos de la respuesta
  updateMessages(responseMessages: MessageResponse[]) {
      // Si hay mensajes, actualizamos la lista de mensajes y el último id de mensaje
      if (responseMessages.length > 0) {
          // Iteramos sobre cada mensaje que llega en el array
          responseMessages.forEach((msg) => {
              // Creamos un nuevo objeto Message con el id, nickname, mensaje, etc.
              const message = new Message(
                  msg.nickname,               // nickname extraído
                  msg.messagetext,            // messageText extraído
                  new Date().toISOString(),   // Asignamos la fecha actual como timestamp
                  msg.messageid,              // id del mensaje
                  this.roomId                 // roomId correspondiente
              );

              // Añadimos el mensaje a la lista de mensajes
              this.messages.push(message);
          });

          // Si es necesario, actualizamos el último ID de mensaje
          const lastMessage = responseMessages[responseMessages.length - 1];
          this.lastIDMessageId = lastMessage.messageid; // Actualizamos el último ID de mensaje
      }
  }

  // Método para obtener el último mensaje
  getLastMessage() {
    if (this.lastIdMessage) {
      return this.messages.find(msg => msg.idMessage === this.lastIdMessage);
    }
    return null;
  }

  // Método para obtener todos los usuarios de la sala
  getUsers(): string[] {
    return this.usuarios;
  }
}
