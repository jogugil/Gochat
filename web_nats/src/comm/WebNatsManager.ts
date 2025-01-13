// Uasar REST Proxy de Confluent para acceder al servidor de mensajeria kafka desde el navegador
// Creación de un backend ligero que funcione a modo de proxy entre el cleinte y el servidor kafka

import { connect, StringCodec, NatsConnection, JetStreamManager, JetStreamClient, RetentionPolicy, AckPolicy, MsgHdrs, headers ,  ConsumerOptsBuilder } from 'nats.ws';
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
interface NatsProducerConsumer {
  producer: any;  // El productor para el stream
  consumer: any;  // El consumidor para el stream
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
    const urls: string[] = VITE_NATS_URLS.split(',');
    const manager = new NatsManager();
    nickname = nickname || 'user-default';
    // Conectar automáticamente al inicializar la instancia
    manager.connect(nickname, urls).catch((error) => {
          console.error('Error al conectar en el constructor:', error);
    })
    
    return manager;
  }

  constructor() {
    console.log (" Constructor NatsManagement")
  }

  // Método para establecer la conexión a NATS y JetStream
  private async connect(nickname: string, urls: string[]): Promise<void> {
    try {
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
      await this.ensureTopicExists(VITE_MAINROOM_TOPIC, nickname);
      await this.ensureTopicExists(VITE_GET_USERS_TOPIC, null);
      this._isConnected = true;
    } catch (error) {
      console.error('Error al conectar con NATS:', error);
      throw error;
    }
  }

  ProducerConsumer(topic: string, producer: any, consumer: any): void {
    this.producersConsumers.set(topic, { producer, consumer });
  }
  public getProducerForTopic(topic: string): NatsProducerConsumer | undefined {
    return this.producersConsumers.get(topic)?.producer; // Usar `get` en lugar de corchetes
  }
  public getConsumerForTopic(topic: string): NatsProducerConsumer | undefined {
    return this.producersConsumers.get(topic)?.consumer; // Usar `get` en lugar de corchetes
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
  private async ensureTopicExists(topic: string, nickname: string | null): Promise<void> {
    console.log(`ensureTopicExists: topic=${topic}, nickname=${nickname}`);

    // Determinar productor y consumidor
    const producerTopic = `${topic}.server`; // El productor siempre se conecta al topic.server
    const consumerTopic = nickname ? `${nickname}.client` : `${topic}.client`; // El consumidor depende del nickname

    console.log(`Producer Topic: ${producerTopic}, Consumer Topic: ${consumerTopic}`);

    // Crear o verificar ambos streams
    await Promise.all([
      this.ensureStreamExists(producerTopic, topic, 'server'),
      this.ensureStreamExists(consumerTopic, topic, 'client'),
    ]);
  }

  private async ensureStreamExists(topic: string, baseTopic: string, type: 'server' | 'client'): Promise<void> {
    console.log(`ensureStreamExists: topic=${topic}, baseTopic=${baseTopic}, type=${type}`);
    const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic}`;

    try {
      const stream = await this.jsm?.streams.info(streamName);
      if (stream) {
        console.log(`Stream para ${topic} ya existe: ${JSON.stringify(stream, null, 2)}`);
      } else {
        await this.createStream(streamName, baseTopic, type);
      }
    } catch (error) {
      console.error(`Error al verificar o crear el stream para ${topic}:`, error);
    }
  }

  // Crear un stream si no existe
  private async createStream(streamName: string, topic: string, type: 'server' | 'client') {
    console.log(`createStream: streamName=${streamName}, topic=${topic}, type=${type}`);
    try {
      const streamWithPrefix = `${streamName}_CL`;

      const config = {
        name: streamWithPrefix,
        subjects: [`${topic}.*`],
        retention: RetentionPolicy.Limits, // Usa el enum en lugar del string literal
        max_msgs_per_subject: 1000,
        max_age: 60 * 60 * 1000, // Tiempo en milisegundos (1 hora)
      };

      console.log(`Stream Config: ${JSON.stringify(config, null, 2)}`);
      await this.jsm?.streams.add(config);
      console.log(`Stream para ${topic} creado con éxito: ${streamWithPrefix}`);

      // Crear productor y consumidor
      await this.createProducerConsumer(streamWithPrefix, topic, type);
    } catch (error) {
      console.error(`Error al crear el stream para ${topic}:`, error);
    }
  }

  // Crear un productor y un consumidor para un stream
  private async createProducerConsumer(streamName: string, topic: string, type: 'server' | 'client') {
    console.log(`createProducerConsumer: streamName=${streamName}, topic=${topic}, type=${type}`);
    const subject = `${topic}.${type}`;
    console.log(`Subject generado: ${subject}`);

    try {
      if (type === 'server') {
        // Crear productor publicando un mensaje de prueba
        await this.js?.publish(subject, Buffer.from('Mensaje de prueba'));
        console.log(`Productor creado para el stream ${streamName}: ${subject}`);

        // Guardar el productor en el mapa
        const existingEntry = this.producersConsumers.get(topic) || { producer: null, consumer: null };
        if (existingEntry) {
          this.producersConsumers.set(topic, {
            producer: subject, // Mantener el productor existente
            consumer:existingEntry.consumer, // Asignar el consumidor creado
          });
          console.log(`Productor creado y almacenado para el topic: ${topic}`);
        } else {
          this.producersConsumers.set(topic, {
            producer: subject,  
            consumer: null,  
          });
          console.log(`producersConsumers creado y almacenado para el topic: ${topic}`);
        }
      } else {
        // Crear consumidor con JetStreamManager primero
        await this.jsm?.consumers.add(streamName, {
          durable_name: `${subject}-consumer`,
          ack_policy: AckPolicy.Explicit, // Usar el valor de la enumeración
          filter_subject: subject, // Filtro para este consumidor
        });

        console.log(`Consumidor registrado en JetStreamManager para el stream: ${streamName}, subject: ${subject}`);

        // Asociar consumidor existente usando `bind`
        const consumer = await this.js?.pullSubscribe(subject, {
          config: {
            durable_name: `${subject}-consumer`,
          },
        });

        console.log(`Consumidor creado con pullSubscribe para el stream ${streamName}: ${subject}`);

        if (consumer) {
          const existingEntry = this.producersConsumers.get(topic) || { producer: null, consumer: null };
          this.producersConsumers.set(topic, {
            producer: existingEntry.producer, // Mantener el productor existente
            consumer, // Asignar el consumidor creado
          });
          console.log(`Consumidor creado y almacenado para el topic: ${topic}`);
        }

        // Iniciar pull del consumidor
        consumer?.pull();
      }
    } catch (error) {
      console.error(`Error al crear productor o consumidor para el stream ${streamName} y topic ${topic}:`, error);
    }
  }
  // Crear consumidor para un stream específico
  async createConsumer(streamName: string, topic: string) {
      console.log(`Creando consumidor para stream: ${streamName}, subject: ${topic}`);
      const durableName = `durableName.${topic}`;
      try {
        console.log(`Creando consumidor con nats.ws y jsm (JsetStreamManager) : Stream=${streamName}, Subject=${topic}, Durable=${durableName}`);
    
        // Configurar el consumidor
        await this.jsm?.consumers.add(streamName, {
          durable_name: durableName,
          ack_policy: AckPolicy.Explicit,
          filter_subject: topic,
        });
    
        console.log(`Consumidor creado exitosamente en nats.ws`);
      } catch (error) {
        console.error(`Error al crear el consumidor con nats.ws:`, error);
      }
      console.log(`Consumidor registrado en JetStreamManager para el stream ${streamName}, subject: ${topic}`);

      

      console.log(`Consumidor creado con pullSubscribe para el stream ${streamName}, subject: ${topic}`);

      try {
        console.log(`Vinculando consumidor existente con nats.ws: Durable=${durableName}`);
    
        const opts = ConsumerOptsBuilder.bind({ durable_name: durableName });
        const consumer = await js.pullSubscribe(topic, opts);
    
        console.log(`Consumidor vinculado exitosamente con nats.ws`);
        return consumer;
      } catch (error) {
        console.error(`Error al vincular el consumidor con nats.ws:`, error);
        return undefined;
      }

      // Iniciar el proceso de extracción de mensajes (pull loop)
      //this.startPullingMessages(topic);

    } catch (error) {
      console.error(`Error al crear el consumidor para stream: ${streamName}, subject: ${subject}`, error);
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

  // Crear el NatsMessage
  const natsMessage: NatsMessage = {
    subject: topic, // El topic es el nombre del subject
    data: Buffer.from(message.messageText), // El contenido del mensaje
    timestamp: new Date().toISOString(), // Fecha y hora actuales como timestamp
    headers: headers, // Los headers definidos
  };

  return natsMessage;
}
public async publishMessage(message: Message, topic: string): Promise<void> {
  try {
    // Asegurarse de que el stream existe para el topic
    await this.ensureTopicExists(topic, null);

    // Obtener el productor asociado al topic-stream
    const producer = this.getProducerForTopic(topic);
    if (!producer) {
      throw new Error(`No se encontró un productor para el topic: ${topic}`);
    }

    // Transformar Message a NatsMessage
    const natsMessage: NatsMessage = this.createNatsMessageFromMessage (message, topic);

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
 
 
// Función para iniciar un consumidor y asignar un callback
 
async startConsumerCallback(topic: string, callback: Function): Promise<void> {
  try {
    // Verificar si ya existe una suscripción para este tópico
    const existingEntry = this.producersConsumers.get(topic);

    if (existingEntry?.consumer) {
      console.warn(`Ya existe un consumidor activo para el topic ${topic}.`);
      return;
    }

    // Crear una nueva suscripción
    const subscription = await this.nc?.subscribe(topic, {
      callback: (err, msg) => {
        if (err) {
          console.error(`Error en el consumidor del topic ${topic}:`, err);
          return;
        }

        // Procesar el mensaje con el callback proporcionado
        callback(msg);
      },
    });

    if (subscription) {
      // Almacenar la suscripción y el callback
      this.producersConsumers.set(topic, {
        subscription,
        callback,
      });
      console.log(`Callback asignado para el topic ${topic}.`);
    }
  } catch (error) {
    console.error(`Error al iniciar el consumidor para el topic ${topic}:`, error);
  }
}

// Función para detener un consumidor y su callback
async stopConsumerCallback(topic: string) {
  const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split('.')[0]}`;
  const consumer = this.producersConsumers[streamName]?.consumer;

  if (consumer) {
    // Detenemos el consumidor y eliminamos el callback
    await consumer.unsubscribe();
    this.producersConsumers[streamName].callback = undefined;
    console.log(`El consumidor para el topic ${topic} ha sido detenido y el callback ha sido eliminado.`);
  } else {
    console.error(`No se encontró consumidor para el topic ${topic}`);
  }
}

// Función para iniciar un consumidor sin callback
async startConsumer(topic: string) {
  const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split('.')[0]}`;
  const consumer = this.producersConsumers[streamName]?.consumer;

  if (consumer) {
    await consumer.pull();
    console.log(`Consumidor iniciado para el topic ${topic}`);
  } else {
    console.error(`No se encontró consumidor para el topic ${topic}`);
  }
}

// Función para detener un consumidor sin eliminar el callback
async stopConsumer(topic: string) {
  const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split('.')[0]}`;
  const consumer = this.producersConsumers[streamName]?.consumer;

  if (consumer) {
    await consumer.unsubscribe();
    console.log(`Consumidor detenido para el topic ${topic}`);
  } else {
    console.error(`No se encontró consumidor para el topic ${topic}`);
  }
}


// Obtenemos histórico de menajes del tópic
/*async getLastMessagesFromTopic(topic: string, limit: number = 100) {
  const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split('.')[0]}`;
  const consumer = this.producersConsumers[streamName]?.consumer;

  if (consumer) {
      try {
          // Recuperar los últimos `limit` mensajes desde el stream
          const result = await consumer.fetch({ maxMessages: limit });

          console.log(result); // Ver la estructura del objeto

          // Suponiendo que los mensajes están en un array llamado `messages`
          const messages = result.messages || [];  // Asegúrate de acceder al array correctamente

          console.log(`Recuperados ${messages.length} mensajes de ${streamName}`);
          return messages;
      } catch (error) {
          console.error(`Error al obtener los mensajes del stream ${streamName}:`, error);
          return [];
      }
  } else {
      console.error(`No se encontró un consumidor para el topic ${topic}`);
      return [];
  }
}
// Función para enviar la solicitud a NATS para obtener los usuarios
async sendGetUsers(request: any, topic: string) {
  // Crear el objeto NATSMessage
  const natsMessage: NatsMessage = {
    subject: topic,  // El topic es el 'subject' del mensaje
    data: Buffer.from(JSON.stringify(request)),  // Los datos de la solicitud van en 'data'
    timestamp: new Date().toISOString(),  // Timestamp actual
    headers: {  // Los atributos de la solicitud van en los 'headers'
      RoomId: request.RoomId,
      TokenSesion: request.TokenSesion,
      Nickname: request.Nickname,
      Operation: request.Operation,
      X_GoChat: request.X_GoChat
    }
  };

  try {
    // Usar el productor de JetStream para enviar el mensaje
    await this.producersConsumers[topic]?.producer?.publish(natsMessage);
    console.log(`Mensaje enviado al topic ${topic}`);
  } catch (error) {
    console.error('Error al enviar el mensaje:', error);
  }
}*/
  async getLastMessagesFromTopic(topic: string, limit: number = 10) {
    if (!this.jsm) {
      console.error("JetStreamManager no inicializado");
      return [];
    }

    try {
      const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split(".")[0]}`;
      const consumerInfo = await this.jsm.consumers.info(streamName, topic);

      if (consumerInfo) {
        console.log(`Mensajes recuperados para ${topic}:`, consumerInfo);
        return consumerInfo;
      } else {
        console.error(`No se encontró información del consumidor para ${topic}`);
        return [];
      }
    } catch (error) {
      console.error(`Error al obtener mensajes históricos de ${topic}:`, error);
      return [];
    }
  }

// Función para recibir un mensaje y ejecutar el callback asociado
//sólo recibe un único menasaje, no es uns subscripción continua
/*
 async receiveMessageFromStream(topic: string) {
  const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split('.')[0]}`;
  const consumer = this.producersConsumers[streamName]?.consumer;
  
  if (consumer) {
      // Esperar el mensaje del consumidor
      const msg = await consumer.fetch(); // O pull(), dependiendo de tu configuración

      console.log(`Mensaje recibido de ${topic}:`, msg);

      // Ejecutar el callback si está asignado
      const callback = this.producersConsumers[streamName]?.callback;
      if (callback) {
          console.log(`Ejecutando callback para el topic ${topic}`);
          callback(msg); // Llamar al callback con el mensaje recibido
      } else {
          console.error(`No se ha asignado un callback para el topic ${topic}`);
      }
  } else {
      console.error(`No se encontró un consumidor para el stream ${streamName}`);
  }
}*/
async receiveMessageFromStream(topic: string) {
  const consumer = this.getConsumerForTopic(topic);
  if (consumer) {
    const msg = await consumer.fetch();  // Método de `fetch` en nats.ws
    console.log(`Mensaje recibido de ${topic}:`, msg);

    const callback = this.producersConsumers[topic]?.callback;
    if (callback) {
      callback(msg);
    } else {
      console.error(`No se ha asignado un callback para el topic ${topic}`);
    }
  } else {
    console.error(`No se encontró un consumidor para el topic ${topic}`);
  }
}
// Función para asignar un callback a un consumidor específico
/*assignCallbackToConsumer(topic: string, callback: Function) {
  const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split('.')[0]}`;
  const consumer = this.producersConsumers[streamName]?.consumer;

  if (consumer) {
      // Asignamos el callback al consumidor
      this.producersConsumers[streamName].callback = callback;
      console.log(`Callback asignado al consumidor para el topic ${topic}`);
  } else {
      console.error(`No se encontró consumidor para el topic ${topic}`);
  }
}*/
// Suscribirse a un topic con callback
  async subscribe(topic: string, callback: (msg: NatsMessage) => void) {
    if (!this.nc) {
      console.error("No conectado a NATS");
      return;
    }

    try {
      const subscription = this.nc.subscribe(topic);
      console.log(`Suscripción iniciada para el topic: ${topic}`);

      for await (const msg of subscription) {
        const data = JSON.parse(this.codec.decode(msg.data));
        const natsMessage: NatsMessage = {
          subject: msg.subject,
          data: msg.data,
          timestamp: new Date().toISOString(),
          headers: {}, // Agregar headers si es necesario
        };
        callback(natsMessage);
      }
    } catch (error) {
      console.error(`Error al suscribirse a ${topic}:`, error);
    }
  }

}