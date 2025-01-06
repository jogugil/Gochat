import { getSystemErrorName } from "util";

export class WebSocketManager {
  private socket: WebSocket | null = null;
  private url: string;
  private messageQueue: string[] = [];
  private _isConnected: boolean = false;
  private retryCount: number = 0;
  private maxRetries: number = 5; // Número máximo de reintentos
  private retryDelay: number = 2000; // Retraso entre reintentos en ms
  private retries: number = 0;
  
  private socketId: string;
  private onErrorCallback: ((error: string) => void) | null = null;
  private onMessageCallback?: (event: MessageEvent) => any; //Función para procesar lso mensajes que llegan como contestacion a las peticiones.

   // Agregar una propiedad para el callback
   onOpenCallback?: () => void;

  constructor(url: string, socketId: string, onErrorCallback?: (error: string) => void) {
    this.url = url;
    this.socketId = socketId;
    this.maxRetries = parseInt(import.meta.env.VITE_RECONNECT_WEBSOCKET, 10) || 3;
    if (onErrorCallback) {
      this.onErrorCallback = onErrorCallback;  // Guardamos el callback de error
    }
    this.connect();
  }

  private connect() {
    console.log(`Conectando a WebSocket para ${this.socketId}...`);
    console.log(`Conectando a WebSocket con la url ${this.url}...`);
    this.socket = new WebSocket(this.url);

    this.socket.onopen = () => {
      console.log(`WebSocket conectado para ${this.socketId}`);
      this.isConnected = true;
      this.retryCount = 0;
      console.log('Cola de mensajes en onopen:', this.messageQueue);
      while (this.messageQueue.length > 0) {
        const message = this.messageQueue.shift();
        if (message) this.socket?.send(message);
      }

      // Ejecutar el callback si está definido
      if (this.onOpenCallback) {
        this.onOpenCallback();
      }
    };
 
    this.socket.onmessage = (event) => {
      console.log(`Mensaje recibido para ${this.socketId}:`, event.data);
      if (this.onMessageCallback) {
        this.onMessageCallback(event);
      }
    };

    this.socket.onclose = () => {
      console.log(`WebSocket desconectado para ${this.socketId}, intentando reconectar...`);
      this.isConnected = false;
    
      if (this.retryCount < this.maxRetries) {
        setTimeout(() => this.connect(), 2000);
        this.retryCount += 1;
      } else {
        if (this.onErrorCallback) {
          this.onErrorCallback(`No se pudo conectar al WebSocket después de ${this.maxRetries} intentos.`);
        }
      }
    };

 

    this.socket.onerror = (error) => {
      console.warn(`Error en WebSocket para ${this.socketId}:`, error);
    
      const errorMessage = `Error en WebSocket (${this.socketId}): ` + 
        (error instanceof ErrorEvent ? error.message : "Error desconocido");
    
      const isReconnectError = this.handleReconnect(error); // Devuelve si es un error de reconexión
    
      if (!isReconnectError) {
        // Si no es un error relacionado con reconexión, llama al callback
        this.onErrorCallback?.(errorMessage);
      }
    };
  }
  public setOnMessageCallback(callback: (event: MessageEvent) => any) {
    this.onMessageCallback = callback;
  }
  public sendMessage(message: string) {
    if (this.isConnected) {
      this.socket?.send(message);
    } else {
      this.messageQueue.push(message);
    }
  }

  public close() {
    if (this.socket) {
      this.socket.close();
    }
  }

  public get isConnected(): boolean {
    return this._isConnected;
  }

  public set isConnected(value: boolean) {
    this._isConnected = value;
  }

  public getSocket() {
    return this.socket;
  }

  // Método para asignar el callback onOpen desde fuera de la clase
  setOnOpenCallback(callback: () => void) {
    this.onOpenCallback = callback;
  }

  private handleReconnect(error?: Event): boolean {
    if (this.retries < this.maxRetries) {
      this.retries++;
      console.log(`Intentando reconectar (${this.socketId})... (${this.retries}/${this.maxRetries})`);
      setTimeout(() => this.connect(), this.retryDelay);
      return true; // Es un error de reconexión
    } else {
      const errorMsg = `No se pudo reconectar (${this.socketId}) después de ${this.maxRetries} intentos.`;
      console.error(errorMsg);
      this.onErrorCallback?.(errorMsg); // Callback al agotar los intentos
      return false; // No es un error de reconexión, ya se agotaron los intentos
    }
  }

}