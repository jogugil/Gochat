// user.ts

import { jwtDecode } from "jwt-decode"; 
import { UserChat } from '../types/types';      
import { validate as validateUUID } from 'uuid';
import { TOKEN_NULO } from '../types/typesComm';
import {JwtPayload} from "../types/typesComm"
 
export type UUID = string;
export class User {
  nickname: string;
  status: string;
  private _token: string    = TOKEN_NULO;
  private _roomId: UUID     = "00000000-0000-0000-0000-000000000000"; 
  private _roomName: string = "Sala Principal";

  constructor(nickname: string,  status: string, roomId: string, roomName: string, token: string) {
    this.nickname = nickname;
    this.status   = status;
    this.roomId   = roomId; // Esto llamará al setter que valida el UUID
    this.roomName = roomName; // Esto llamará al setter que valida el UUID
    
    console.log(`Creando el objeto user: nick: ${nickname}, status: ${status}, roomid: ${roomId}, roomname: ${roomName}, token: ${token}`);
    this.token = token || TOKEN_NULO;
  }
  // Getter para _token
  get token(): string {
    return this._token;
  }

  // Setter para _token con validación
  set token(value: string) {
    if (this.isValidToken(value)) {
      this._token = value; // Solo asigna si es un token válido
    } else {
      throw new Error('El token no es válido');
    }
  }
  // Método para validar si el token es válido
  private isValidToken(token: string): boolean {
    try {
      const decoded = <JwtPayload>jwtDecode(token);
      return decoded != null && this.tokenIsValid(decoded.exp);
    } catch (error) {
      console.warn('Error al validar el token:', error);
      return false;
    }
  }

    // Verifica si el token ha expirado
    private tokenIsValid(expiration: number | undefined): boolean {
        if (expiration === undefined) {
          return false; // Si no hay fecha de expiración, no es válido
        }

        const now = Math.floor(Date.now() / 1000); // Obtenemos la fecha y hora actual en segundos
        const expirationDatePlusOneDay = expiration + 60 * 60 * 24; // Añadir 24 horas (un día)

        return now < expirationDatePlusOneDay; // Verificar si la fecha actual es antes de la fecha de expiración + 1 día
    }

  // Método para obtener el ID del usuario decodificado desde el token
  getUserIdFromToken(): string | null {
    try {
      const decoded = <JwtPayload>jwtDecode(this._token);
      console.log ("decode token",decoded);
      return decoded ? decoded.userid : null;
    } catch (error) {
      console.warn('Error al obtener el ID del token:', error);
      return null;
    }
  } 
    // Getter y Setter para roomId
    get roomName(): UUID {
      return this._roomName;
    }
    set roomName(value: string) {
      this._roomName = value; // Si la validación pasa, asigna el valor
    }
    // Getter y Setter para roomId
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