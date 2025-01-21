// Uasar REST Proxy de Confluent para acceder al servidor de mensajeria kafka desde el navegador
// Creación de un backend ligero que funcione a modo de proxy entre el cleinte y el servidor kafka
 

import { connect, StringCodec, NatsConnection, JetStreamManager, JetStreamClient, RetentionPolicy, AckPolicy, MsgHdrs, headers ,   DeliverPolicy, ReplayPolicy   } from 'nats.ws';
import { RequestListuser, KafkaMessage, NatsMessage } from '../types/typesComm'; 
import { Message, UUID } from "../models/Message";
import { v4 as uuidv4 } from 'uuid';
import { OrderedConsumerMessages } from 'nats.ws/lib/jetstream/consumer';
 

// Accede a las variables de entorno
export const VITE_NATS_URLS = import.meta.env.VITE_NATS_URLS?.split(',') || ['ws://localhost:8080'];
export const VITE_MAINROOM_TOPIC = import.meta.env.VITE_MAINROOM_TOPIC || "principal";
export const VITE_GET_USERS_TOPIC = import.meta.env.VITE_GET_USERS_TOPIC || "roomlistusers";
export const VITE_NATS_PREFIX_STREAMNAME = import.meta.env.VITE_NATS_PREFIX_STREAMNAME || "MYGOCHAT_STREAM";

// Estructura para almacenar productores y consumidores
// Estructura para almacenar productores y consumidores con sus respectivos subjects
interface NatsProducerConsumer {
  producer: any;       // El productor para el stream
  consumer: any;       // El consumidor para el stream
  producerSubject: string; // El subject asociado al productor
  consumerSubject: string; // El subject asociado al consumidor
}

export class NatsManager {
  private nc: NatsConnection | null = null;            // Conexión a nats
  private jsm: JetStreamManager | null = null;         // Gestor de streams de nats
  private js: JetStreamClient | null = null;           // Cliente de streams de nats
  private codec = StringCodec();                       // Codificador de mensajes
  private producersConsumers: Map<string, NatsProducerConsumer> = new Map();
  private _isConnected: boolean = false;               // Estado de la conexión
  private subscribedTopics: Set<string> = new Set();   // Conjunto de temas a los que se ha suscrito
  
  public get isConnected(): boolean {
    return this._isConnected;
  }

  static async create(nickname: string | null): Promise<NatsManager> {
    const urls: string[] = VITE_NATS_URLS;
    const manager = new NatsManager();
    nickname = nickname || 'user-default';
    // Conectar automáticamente al inicializar la instancia
    await manager.connect(nickname, urls).catch((error) => {
          console.error('Error al conectar en el constructor:', error);
    })
    console.log ( `En final de create: Hemos creado los productores y consumidore `)
    console.log (manager.producersConsumers)
    return manager;
  }

  constructor() {
    console.log (" Constructor NatsManagement")
  }

  // Método para establecer la conexión a NATS y JetStream
  private async connect(nickname: string, urls: string[]): Promise<void> {
    try {
      console.log('Iniciando conexión con NATS:', urls);
  
      // Establecer la conexión
      this.nc = await connect({ servers: urls });
      console.log('Conexión establecida con NATS:', urls);
  
      // Inicializar JetStream Manager y Cliente
      this.jsm = await this.nc.jetstreamManager();
      this.js = this.nc.jetstream();
  
      if (this.jsm && this.js) {
        console.log('JetStream Manager y Client inicializados correctamente.');
      } else {
        throw new Error('Fallo al inicializar JetStream Manager o Client.');
      }
  
      // Conectar y crear streams según el topic
      console.log('Iniciando configuración de streams para los topics...');
      await this.ensureTopicExists(VITE_MAINROOM_TOPIC, null);
      await this.ensureTopicExists(VITE_GET_USERS_TOPIC, nickname);
      console.log('Todos los streams y consumidores configurados.');
      this._isConnected = true;
    } catch (error) {
      console.error('Error al conectar con NATS:', error);
      throw error;
    }
  }

  // Función para registrar productor y consumidor con sus respectivos subjects
  public ProducerConsumer(topic: string, producer: any, consumer: any, producerSubject: string, consumerSubject: string): void {
    this.producersConsumers.set(topic, { 
      producer, 
      consumer, 
      producerSubject,   
      consumerSubject    
    });
  }
  public getProducerForTopic(topic: string): NatsProducerConsumer | undefined {
    return this.producersConsumers.get(topic)?.producer; // Usar `get` en lugar de corchetes
  }
  public getProducerSubjectForTopic(topic: string): string {
    return this.producersConsumers.get(topic)?.producerSubject ?? "defaultSubject";
  }
  public getConsumerForTopic(topic: string): NatsProducerConsumer | undefined {
    return this.producersConsumers.get(topic); // Usar `get` en lugar de corchetes
  }
  public getConsumerSubjectForTopic(topic: string): string {
    return this.producersConsumers.get(topic)?.consumerSubject ?? "defaultConsumerSubject";
  }
  private createNatsMessage(requestData: any, topic: string): NatsMessage {
    const headers = {
      roomId: requestData.roomid,
      nickname: requestData.nickname,
      tokensession: requestData.tokensession,
      roomname: requestData.roomname,
    };

    const now = new Date();
    return {
      subject: topic,
      data: Buffer.from(JSON.stringify(requestData)),
      timestamp: now.toLocaleTimeString(),
      headers: headers,
    };
  }

  // Verifica si un stream existe, si no, lo crea
  // Verifica si un stream existe, si no, lo crea
  private async ensureTopicExists(topic: string, nickname: string | null): Promise<void> {
    console.log(`ensureTopicExists iniciado: topic=${topic}, nickname=${nickname}`);

    const producerTopic = `${topic}.server`;
    const consumerTopic = nickname ? `${nickname}.client` : `${topic}.client`;

    console.log(`Producer Topic: ${producerTopic}, Consumer Topic: ${consumerTopic}`);

    await Promise.all([
      await this.ensureStreamExists(producerTopic, topic, 'server'),
      await this.ensureStreamExists(consumerTopic, topic, 'client'),
    ]);
    console.log ( `Hemos creado el consu idor y productor del topic ${topic}`)
    console.log (this.producersConsumers)
  }

  private async ensureStreamExists(topic: string, baseTopic: string, type: 'server' | 'client'): Promise<void> {
    console.log(`ensureStreamExists iniciado: topic=${topic}, baseTopic=${baseTopic}, type=${type}`);
    let streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic}`;
   

    try {
        console.log(`1. Verificando existencia del stream: ${streamName}`);
        streamName = streamName.replace(/\./g, '_');  // Reemplazar puntos por guiones bajos
        console.log(`2. Verificando existencia del stream: ${streamName}`);

        // Intentar obtener el stream y verificar si existe
        const stream = await this.jsm?.streams.info(streamName);

        // Si el stream ya existe, asociamos el productor y consumidor
        if (stream) {
            console.log(`Stream ya existe: ${JSON.stringify(stream, null, 2)}`);
            
            // Obtener los productores y consumidores existentes del map
            const existingEntry = this.producersConsumers.get(baseTopic) || { producer: null, consumer: null, producerSubject: "", consumerSubject: "" };

            // Si el tipo es 'server', creamos el productor, y si es 'client', creamos el consumidor
            if (type === 'server' && !existingEntry.producer) {
                // Lógica para inicializar el productor si no existe
                existingEntry.producer = await this.createProducer(streamName, type);
                existingEntry.producerSubject = streamName;
                console.log(`Productor asociado al stream ${streamName} con subject ${streamName}`);
                existingEntry.consumer = existingEntry.consumer;
                existingEntry.consumerSubject = existingEntry.consumerSubject;
            } 
            if (type === 'client' && !existingEntry.consumer) {
                // Lógica para inicializar el consumidor si no existe
                existingEntry.consumer = await this.createConsumer(streamName, type);
                existingEntry.consumerSubject = streamName;
                console.log(`Consumidor asociado al stream ${streamName} con subject ${streamName}`);
                existingEntry.producer = existingEntry.producer;
                existingEntry.producerSubject = existingEntry.producerSubject;
            } 

            // Asignar el productor o consumidor al baseTopic
            console.log(`Asignamos el productor y/o consumidor al topic ${baseTopic} -- ${existingEntry}`);
            this.producersConsumers.set(baseTopic, existingEntry);

            return;  // Salir si el stream y los productores/consumidores están correctamente configurados
        } else {
            // Si el stream no existe, lo creamos y luego asociamos productor y consumidor
            console.log(`Stream no encontrado. Procediendo a crear el stream: ${streamName}`);
            await this.createStream(streamName, baseTopic, type);
            console.log(`Stream creado. Procediendo a asociar productor y consumidor.`);

            // Crear el productor o consumidor dependiendo del tipo
            let producer, consumer;
            let producerSubject, consumerSubject;

            if (type === 'server') {
                producer = await this.createProducer(streamName, type);
                consumer = null;   
                producerSubject = streamName;
                consumerSubject = "";  
            } else if (type === 'client') {
                producer = null;   
                consumer = await this.createConsumer(streamName, type);
                producerSubject =  "";   
                consumerSubject = streamName;
            }

            // Asociamos el productor o consumidor correspondiente
            if (producer) {
                this.producersConsumers.set(baseTopic, { producer, consumer: null, producerSubject: streamName, consumerSubject: "" });
                console.log(`Productor creado y asociado al baseTopic ${baseTopic}`);
            }
            if (consumer) {
                this.producersConsumers.set(baseTopic, { producer: null, consumer, producerSubject: "", consumerSubject: streamName});
                console.log(`Consumidor creado y asociado al baseTopic ${baseTopic}`);
            }
        }
    } catch (error: unknown) {
        // Comprobamos si 'error' es un objeto y tiene la propiedad 'message'
        if (this.isErrorWithMessage(error)) {
            console.warn(`Error al verificar o crear el stream ${streamName} para el topic ${topic}:`, error.message);

            // Si el error es específico de stream no encontrado, lo manejamos adecuadamente
            if (error.message.includes('stream not found')) {
                console.log(`El stream ${streamName} no fue encontrado, creando uno nuevo.`);
                await this.createStream(streamName, baseTopic, type);
            } else {
                console.error("Error desconocido al verificar o crear el stream:", error.message);
            }
        } else {
            console.error("Error desconocido al verificar o crear el stream:", error);
        }
    }
}

private async createProducer(streamName: string, type: 'server' | 'client'): Promise<any> {
    return {
      publish: async (subject: string, data: Uint8Array) => {
          console.log(`Publicando mensaje al subject: ${subject}`);
          await this.js?.publish(subject, data);
      },
  };
} 
private async getProducer(streamName: string): Promise<any> {
  console.log(`Obteniendo productor para el stream ${streamName}`);

  // Devuelve un objeto productor para enviar mensajes
  return {
      publish: async (subject: string, data: Uint8Array) => {
          console.log(`Publicando mensaje al subject: ${subject}`);
          await this.js?.publish(subject, data);
      },
  };
}
private async createConsumer(streamName: string, type: 'server' | 'client'): Promise<any> {
   
  console.log(`Creando consumidor para el stream ${streamName} con tipo ${type}`);
 
  await this.jsm?.consumers.add(streamName, {
    durable_name: `${streamName}-consumer`,
    ack_policy: AckPolicy.Explicit,
    filter_subject: streamName,
  });
  console.log(`Consumidor registrado en JetStreamManager para el subject: ${streamName}`);

  const consu = await this.js?.consumers.get(streamName, `${streamName}-consumer`);
 
  return consu;
}
private async getConsumer(streamName: string, consumerName: string): Promise<any> {
  const nc = await connect({ servers: "nats://localhost:4222" });
  const js = nc.jetstream();

  console.log(`Obteniendo consumidor para el stream ${streamName}, consumidor: ${consumerName}`);

  const consu = await this.js?.consumers.get(streamName, `${streamName}-consumer`);
  return consu;
}

  // Función de guardia para verificar si el error tiene la propiedad 'message'
 private isErrorWithMessage(error: unknown): error is { message: string } {
      return typeof error === 'object' && error !== null && 'message' in error;
  }

  // Crear un stream si no existe
  private async createStream(streamName: string, topic: string, type: 'server' | 'client') {
    console.log(`createStream iniciado: streamName=${streamName}, topic=${topic}, type=${type}`);
    try {
      let streamWithPrefix = `${streamName}`;
      streamWithPrefix = streamWithPrefix.replace(/\./g, '_');
      const config = {
        name: streamWithPrefix,
        subjects: [streamWithPrefix],
        retention: RetentionPolicy.Limits,
        max_msgs_per_subject: 1000,
        //max_age: 60 * 60 * 1000,
      };

      console.log(`Configuración del stream: ${JSON.stringify(config, null, 2)}`);
      await this.jsm?.streams.add(config);
      console.log(`Stream creado con éxito: ${streamWithPrefix}`);

      await this.createProducerConsumer(streamWithPrefix, topic, type);
    } catch (error) {
      console.error(`Error al crear el stream ${streamName} para el topic ${topic}:`, error);
    }
  }
    
  // Crear un productor y un consumidor para un stream
  private async createProducerConsumer(streamName: string, topic: string, type: 'server' | 'client') {
    console.log(`**** createProducerConsumer iniciado: streamName=${streamName}, topic=${topic}, type=${type}`);
    const subject = streamName; //`${topic}.${type}`;
    console.log(`Subject generado: ${subject}`);

    try {
      const existingEntry = this.producersConsumers.get(topic) || { producer: null, consumer: null,  producerSubject:"", consumerSubject:""};
      if (type === 'server') {
        console.log(`Creando productor para el subject: ${subject}`);
        await this.js?.publish(subject, Buffer.from('Mensaje de prueba'));
        console.log(`Productor creado para el subject: ${subject}`);

       
        this.producersConsumers.set(topic, {
          producer: subject,
          consumer: existingEntry.consumer,
          producerSubject:streamName,
          consumerSubject:existingEntry.consumerSubject,
        });
        console.log(`Productor almacenado para el topic: ${topic}`);
      } else {
        console.log(`Creando consumidor para el subject: ${subject}`);
        await this.jsm?.consumers.add(streamName, {
          durable_name: `${subject}-consumer`,
          ack_policy: AckPolicy.Explicit,
          filter_subject: subject,
        });
        console.log(`Consumidor registrado en JetStreamManager para el subject: ${subject}`);

        const consumer = await this.js?.consumers.get(streamName, `${subject}-consumer`);
        if (consumer) {
          
          this.producersConsumers.set(topic, {
            producer: existingEntry.producer,
            consumer,
            producerSubject:existingEntry.producerSubject,
            consumerSubject:streamName ,
          });
          console.log(`Consumidor almacenado para el topic: ${topic}`);
        }
      }
    } catch (error) {
      console.error(`Error al crear productor o consumidor para el stream ${streamName} y topic ${topic}:`, error);
    }
  }

  
  private createNatsMessageFromMessage(message: Message, topic: string): NatsMessage {
    // Crear los headers del NatsMessage a partir del objeto Message
    const headers: Record<string, string | Buffer> = {
      messageId: message.messageId,
      messageType: message.messageType,
      sendDate: message.sendDate,
      serverDate: message.serverDate,
      nickname: message.nickname,
      token: message.token,
      roomId: message.roomId,
      roomName: message.roomName,
      metadata: JSON.stringify(message.metadata), // Convertir Metadata a string JSON
    };

    // Utilizar TextEncoder para convertir el texto a binario
    const encoder = new TextEncoder();
    const messageData = encoder.encode(message.messageText); // Convierte el texto en un Uint8Array

    // Crear el NatsMessage
    const natsMessage: NatsMessage = {
      subject: topic, // El topic es el nombre del subject
      data: messageData, // El contenido del mensaje como Uint8Array
      timestamp: new Date().toISOString(), // Fecha y hora actuales como timestamp
      headers: headers, // Los headers definidos
    };

    return natsMessage;
  }

  public async publishMessage(message: Message, topic: string): Promise<void> {
    try {
      console.log(`Entrando en publishMessage para el topic: ${topic}`);
  
      // Verificar que el productor esté disponible
      const producer = this.getProducerForTopic(topic);
      if (!producer) {
        throw new Error(`No se encontró un productor para el topic: ${topic}`);
      }
      const producerSubject = this.getProducerSubjectForTopic (topic);
      console.log("Productor encontrado:", producer);
  
      // Verificar que JetStream esté conectado
      if (!this._isConnected) {
        throw new Error("JetStream no está conectado");
      }
      try {
         
        console.log('Streams en await this.js?.views:', await this.js?.views);
        console.log('Streams en await this.js?.views:', await this.js?.streams);
      } catch (error) {
        console.error('Error al obtener información de los streams de JetStream:', error);
      }
      // Transformar Message a NatsMessage
      const natsMessage: NatsMessage = this.createNatsMessageFromMessage(message, topic);
      console.log(`Mensaje transformado a NatsMessage: ${JSON.stringify(natsMessage)}`);
  
      // Crear los headers
      const natsHeaders = headers();
      for (const [key, value] of Object.entries(natsMessage.headers)) {
        if (value !== undefined) {
          if (Array.isArray(value)) {
            value.forEach(v => {
              if (typeof v === "string" || Buffer.isBuffer(v)) {
                natsHeaders.append(key, v.toString());
              }
            });
          } else if (typeof value === "string" || Buffer.isBuffer(value)) {
            natsHeaders.set(key, value.toString());
          }
        }
      }
      
      // Intentar publicar con reintentos
      const MAX_RETRIES = 3;
      let retries = 0;
      let pubAck = null;
  
      while (retries < MAX_RETRIES) {
        try {
          pubAck = await this.js?.publish(producerSubject, natsMessage.data, {
            headers: natsHeaders,
          });
  
          if (pubAck?.seq) {
            console.log(`Mensaje publicado con éxito en ${topic}. Seq: ${pubAck.seq}`);
            return;
          } else {
            throw new Error("No se recibió un ack de la publicación.");
          }
        } catch (error) {
          retries++;
          if (retries >= MAX_RETRIES) {
            console.error(`Error al publicar el mensaje en ${topic} después de ${MAX_RETRIES} intentos`, error);
            break;
          } else {
            console.log(`Reintentando publicación en ${topic}... Intento ${retries}`);
          }
        }
      }
    } catch (error) {
      console.error(`Error al publicar el mensaje en ${topic}:`, error);
    }
  }
  
  

  // Enviar un mensaje de petición inicial de lsitado de usaurios 
  private  createNatsMessageFromRequestListUsers (request: RequestListuser, topic: string): NatsMessage {
    // Mapeamos los valores del request a los headers del NatsMessage
    const headers: Record<string, string | Buffer> = {
      roomId: request.roomId,
      tokenSesion: request.tokenSesion,
      nickname: request.nickname,
      operation: request.operation,
      xGoChat: request.xGoChat,
    };

    // Creamos el mensaje NatsMessage
    const natsMessage: NatsMessage = {
      subject: topic, // El subject será el topic proporcionado
      data: Buffer.from(JSON.stringify({ message: request.message })), // Si necesitas incluir datos adicionales, puedes ajustarlo aquí
      timestamp: new Date().toISOString(), // Fecha y hora actual como timestamp
      headers: headers, // Los headers generados a partir del request
    };

    return natsMessage;
  }
  public async publisRequestListUsershMessage(message: RequestListuser, topic: string): Promise<void> {
    try {
      // Asegurarse de que el stream existe para el topic
      await this.ensureTopicExists(topic, null);

      // Obtener el productor asociado al topic-stream
      const producer = this.getProducerForTopic(topic);
      if (!producer) {
        throw new Error(`No se encontró un productor para el topic: ${topic}`);
      }

      // Transformar Message a NatsMessage
      const natsMessage: NatsMessage = this.createNatsMessageFromRequestListUsers (message, topic);

      // Crear el objeto de headers estándar (sin MsgHdrs)
      

      const natsHeaders = headers();

      for (const [key, value] of Object.entries(natsMessage.headers)) {
        if (value !== undefined) {
          if (Array.isArray(value)) {
            value.forEach(v => {
              if (typeof v === "string" || Buffer.isBuffer(v)) {
                natsHeaders.append(key, v.toString()); // Agregar valores múltiples a una misma clave
              }
            });
          } else if (typeof value === "string" || Buffer.isBuffer(value)) {
            natsHeaders.set(key, value.toString()); // Asignar un valor único a la clave
          }
        }
      }

      const pubAck = await this.js?.publish(natsMessage.subject, natsMessage.data, {
        headers: natsHeaders, // Usa MsgHdrs aquí
      });

      console.log(`Mensaje publicado con éxito en ${topic}. Seq: ${pubAck?.seq}`);
    } catch (error) {
      console.error(`Error al publicar el mensaje en ${topic}:`, error);
    }
  } 
  
  public async assignCallback(
    topic: string,
    callback: (err: Error | null, msg: any | null, sharedObject: any) => void,
    sharedObject: any,
    persist: boolean = false
  ): Promise<void> {
    console.log(`Entrando en assignCallback con el tema: "${topic}"`);
  
    try {
      // Obtener el consumidor configurado para el tema
      const consumer = this.getConsumerForTopic(topic)?.consumer;
  
      if (!consumer) {
        console.error(`No se encontró un consumidor para el tema: "${topic}"`);
        throw new Error(`No se encontró un consumidor para el tema: ${topic}`);
      }
      const consumerSubject = this.getConsumerSubjectForTopic (topic);
      console.log(`Consumidor encontrado para el tema: "${topic} - subject: ${consumer.subject}- consumerSubject: ${consumerSubject} "`);
  
      // Consumir mensajes del consumidor
      console.log(`Esperando mensajes para el tema: - subject: ${consumerSubject}`);
  
      await consumer.consume({
        callback: (err: Error | null, msg: any) => {
          if (err) {
            console.error(
              `Error al procesar mensaje en el tema "${consumerSubject}}":`,
              err
            );
            callback(err, null, sharedObject);
            return;
          }
  
          if (msg) {
            console.log(`Mensaje recibido para el tema "${consumerSubject}":`, msg);
            callback(null, msg, sharedObject);
  
            // Si persist es false, cancelar la recepción de mensajes
            if (!persist) {
              console.log(
                `Persistencia desactivada, deteniendo recepción para el tema: "${consumerSubject}"`
              );
              msg?.consumer?.close(); // Detener el consumidor si es posible
            }
          }
        },
      });
  
      console.log(`Callback asignado correctamente para el tema: "${topic}"`);
    } catch (error) {
      console.error(
        `Error al asignar callback para el tema "${topic}":`,
        error
      );
      throw error instanceof Error ? error : new Error(String(error));
    }
  }
  
  
  
  
 
}