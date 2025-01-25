// Uasar REST Proxy de Confluent para acceder al servidor de mensajeria kafka desde el navegador
// Creación de un backend ligero que funcione a modo de proxy entre el cleinte y el servidor kafka
 

import { connect, StringCodec, NatsConnection, JetStreamManager, JetStreamClient, RetentionPolicy, AckPolicy, MsgHdrs, headers ,   DeliverPolicy, ReplayPolicy, JsMsg   } from 'nats.ws';
import { RequestListuser, KafkaMessage, NatsMessage } from '../types/typesComm'; 
import { Message, UUID } from "../models/Message";
import { v4 as uuidv4 } from 'uuid';
import { OrderedConsumerMessages } from 'nats.ws/lib/jetstream/consumer';
import {serializeMessage, deserializeMessage, createNatsMessageFromText} from '../utils/messageutil';

// Accede a las variables de entorno
export const VITE_NATS_URLS = import.meta.env.VITE_NATS_URLS?.split(',') || ['ws://localhost:8080'];
export const VITE_MAINROOM_TOPIC = import.meta.env.VITE_MAINROOM_TOPIC || "principal";
export const VITE_NATS_USER = import.meta.env.VITE_NATS_USER || "user";
export const VITE_NATS_PASSWD = import.meta.env.VITE_NATS_PASSWD || "user123";
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
     
      const options = {
        servers: VITE_NATS_URLS,  // Usar la URL de NATS desde las variables de entorno
        user: VITE_NATS_USER,     // El usuario de NATS desde las variables de entorno
        pass: VITE_NATS_PASSWD,   // La contraseña de NATS desde las variables de entorno
        //protocol: 'ws',            // Asegúrate de indicar que usas WebSocket
        //port:8080,
      };
      console.log(`Iniciando conexión con ${nickname} NATS:`, options);
 
      //console.log("Connectado con nc1")
      // Establecer la conexión
      this.nc = await connect(options);
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
      await this.ensureTopicExists(VITE_MAINROOM_TOPIC, nickname, false);
      await this.ensureTopicExists(VITE_GET_USERS_TOPIC, nickname, true);
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
    const encoder = new TextEncoder();
    const now = new Date();
    return {
      subject: topic,
      data:  btoa(requestData), //Buffer.from(JSON.stringify(requestData))
      timestamp: now.toLocaleTimeString(),
      headers: headers,
    };
  }

  // Verifica si un stream existe, si no, lo crea
  // Verifica si un stream existe, si no, lo crea
  private async ensureTopicExists(topic: string, nickname: string, usernick: boolean): Promise<void> {
    
    console.log(`ensureTopicExists iniciado: topic=${topic}, nickname=${nickname}`);

    const producerTopic = `${topic}.server`;
    const consumerTopic = usernick ? `${nickname}.client` : `${topic}.client`;

    console.log(`Producer Topic: ${producerTopic}, Consumer Topic: ${consumerTopic}`);

    await Promise.all([
      await this.ensureStreamExists(nickname,producerTopic, topic, 'server'),
      await this.ensureStreamExists(nickname,consumerTopic, topic, 'client'),
    ]);
    console.log ( `Hemos creado el consu idor y productor del topic ${topic}`)
    console.log (this.producersConsumers)
  }

  private async ensureStreamExists(nickname:string, topic: string, baseTopic: string, type: 'server' | 'client'): Promise<void> {
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
            console.log(existingEntry);
            // Si el tipo es 'server', creamos el productor, y si es 'client', creamos el consumidor
            if (type === 'server' && !existingEntry.producer) {
                console.log(` Procediendo a crear un productor. Se llama a createProducer`);
                // Lógica para inicializar el productor si no existe
                existingEntry.producer = await this.createProducer(streamName, type);
                existingEntry.producerSubject = streamName;
                console.log(`Productor asociado al stream ${streamName} con subject ${streamName}`);
                existingEntry.consumer = existingEntry.consumer;
                existingEntry.consumerSubject = existingEntry.consumerSubject;
            } 
            if (type === 'client' && !existingEntry.consumer) {
                console.log(` Procediendo a crear un consumidor. Se llama a : createConsumer`);
                // Lógica para inicializar el consumidor si no existe
                existingEntry.consumer = await this.createConsumer(streamName, nickname, type);
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
            await this.createStream(nickname, streamName, baseTopic, type);
            console.log(`Stream creado. Procediendo a asociar productor y consumidor.`);

            // Crear el productor o consumidor dependiendo del tipo
            let producer, consumer;
            let producerSubject, consumerSubject;

            if (type === 'server') {
                console.log(`Stream creado. Procediendo a crear un productor. Se llama a createProducer`);
                producer = await this.createProducer(streamName, type);
                consumer = null;   
                producerSubject = streamName;
                consumerSubject = "";  
            } else if (type === 'client') {
                console.log(`Stream creado. Procediendo a crear un consumidor. Se llama a : createConsumer`);
                producer = null;   
                consumer = await this.createConsumer(streamName, nickname, type);
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
                await this.createStream(nickname, streamName, baseTopic, type);
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
      publish: async (subject: string, data: string) => {
          console.log(`Publicando mensaje al subject: ${subject}`);
          const msg = createNatsMessageFromText (subject, data)
          await this.js?.publish(subject,  serializeMessage (msg));
      },
  };
} 
private async getProducer(streamName: string): Promise<any> {
  console.log(`Obteniendo productor para el stream ${streamName}`);

  // Devuelve un objeto productor para enviar mensajes
  return {
      publish: async (subject: string, data: string) => {
          console.log(`Publicando mensaje al subject: ${subject}`);
          const msg = createNatsMessageFromText (subject, data)
          await this.js?.publish(subject, serializeMessage (msg));
      },
  };
}
private async createConsumer(streamName: string, nickname:string, type: 'server' | 'client'): Promise<any> {
  console.log(`Creando o verificando consumidor para el stream ${streamName} con tipo ${type}`);
  
  // Advertencia: Esta función ahora asume que el consumidor podría no existir,
  // y maneja el caso en el que no lo encuentre sin lanzar una excepción.
  
  try {
    // Intentamos obtener el consumidor. Si no existe, lo crearemos automáticamente.
    const consumer = await this.js?.consumers.get(streamName, `${streamName}-${nickname}-consumer`);
    console.log(`Consumidor encontrado o ya existe: ${streamName}-${nickname}-consumer`);
    return consumer; // Si existe, lo devolvemos directamente

  } catch (error) {
    console.error(`Error al obtener el consumidor para el stream ${streamName}:`, error);
    
      console.log(`El consumidor no existe para el stream ${streamName}, creando uno nuevo...`);
      
      // Intentamos crear el consumidor ya que no existe
      try {
        // Verificamos si el stream existe antes de crear el consumidor
        const stream = await this.jsm?.streams.get(streamName);
        if (!stream) {
          console.error(`El stream ${streamName} no existe. Asegúrate de crearlo antes.`);
          return null; // Error temprano si el stream no existe
        }

        // Creamos el consumidor con la configuración deseada
        await this.jsm?.consumers.add(streamName, {
          durable_name: `${streamName}-${nickname}-consumer`, // Nombre duradero del consumidor
          ack_policy: AckPolicy.Explicit,        // Política de confirmación explícita
          deliver_policy: DeliverPolicy.All,      // Empezar a recibir desde el último mensaje nuevo
          filter_subject: streamName,             // Filtrado de los mensajes (puedes dejarlo como está si quieres filtrar por el nombre del stream)
          //deliver_subject: `${streamName}.push`,  // Aquí defines el "deliver_subject" donde se enviarán los mensajes (esto lo convierte en un consumer push)
          //flow_control: false,                     // Habilita el control de flujo para la entrega de mensajes
          //idle_heartbeat: 200,                 // Define un "heartbeat" durante la inactividad (en milisegundos)
        });
        console.log(`this.jsm?.consumers: ${this.jsm?.consumers}`);
        console.log(`Consumidor creado exitosamente para el stream ${streamName}`);
        
        // Recuperamos el consumidor recién creado
        const newConsumer = await this.js?.consumers.get(streamName, `${streamName}-${nickname}-consumer`);
        console.log(`newConsumer: ${newConsumer}`);
        return newConsumer;
      } catch (creationError) {
        console.error(`Error al crear el consumidor para el stream ${streamName}:`, creationError);
        return null; // Enmascaramos el error de creación y devolvemos null
      }
  }
}
 


private async getConsumer(streamName: string, consumerName: string): Promise<any> {
  try {
    // Usar las variables de entorno para configurar la conexión
    const options = {
      servers: VITE_NATS_URLS,    // Usamos la URL de NATS desde las variables de entorno
      user: VITE_NATS_USER,       // Usamos el usuario de NATS desde las variables de entorno
      pass: VITE_NATS_PASSWD      // Usamos la contraseña de NATS desde las variables de entorno
    };

    // Conectar a NATS usando las credenciales de las variables de entorno
    const nc = await connect(options);
    const js = nc.jetstream(); // Usar JetStream del cliente NATS

    console.log(`Obteniendo consumidor para el stream ${streamName}, consumidor: ${consumerName}`);

    // Obtener el consumidor
    const consu = await js.consumers.get(streamName, consumerName);

    return consu;
  } catch (error) {
    console.error('Error al obtener el consumidor:', error);
    throw error;
  }
}
  // Función de guardia para verificar si el error tiene la propiedad 'message'
 private isErrorWithMessage(error: unknown): error is { message: string } {
      return typeof error === 'object' && error !== null && 'message' in error;
  }

  // Crear un stream si no existe
  private async createStream(nickname: string, streamName: string, topic: string, type: 'server' | 'client') {
    console.log(`createStream iniciado: streamName=${streamName}, topic=${topic}, type=${type}`);
    try {
      let streamWithPrefix = `${streamName}`;
      streamWithPrefix = streamWithPrefix.replace(/\./g, '_');
      const config = {
        name: streamWithPrefix,
        subjects: [streamWithPrefix],
        retention: RetentionPolicy.Limits,
        max_msgs_per_subject: 10000,
        max_consumers: -1, // Ilimitado
        max_age: 0, // Retención infinita hasta que alcances el límite
        //max_age: 60 * 60 * 1000,
      };

      console.log(`Configuración del stream: ${JSON.stringify(config, null, 2)}`);
      await this.jsm?.streams.add(config);
      console.log(`Stream creado con éxito: ${streamWithPrefix}`);

      await this.createProducerConsumer(nickname, streamWithPrefix, topic, type);
    } catch (error) {
      console.error(`Error al crear el stream ${streamName} para el topic ${topic}:`, error);
    }
  }
    
  // Crear un productor y un consumidor para un stream
  private async createProducerConsumer(nickname: string, streamName: string, topic: string, type: 'server' | 'client') {
    console.log(`**** createProducerConsumer iniciado: streamName=${streamName}, topic=${topic}, type=${type}`);
    const subject = streamName; //`${topic}.${type}`;
    console.log(`Subject generado: ${subject}`);

    try {
      const existingEntry = this.producersConsumers.get(topic) || { producer: null, consumer: null,  producerSubject:"", consumerSubject:""};
      if (type === 'server') {
        const encoder = new TextEncoder();
        console.log(`Creando productor para el subject: ${subject}`);
        // Serializar el mensaje utilizando la función serializeMessage
        const msg = createNatsMessageFromText (subject, 'Mensaje de prueba')
        await this.js?.publish(subject, serializeMessage(msg));
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
          name: `${subject}-${nickname}-consumer`, // ID único por cliente
          durable_name: `${subject}-${nickname}-consumer`,
          ack_policy: AckPolicy.Explicit,
          deliver_policy: DeliverPolicy.All, // O DeliverPolicy.Last si solo quieres el último mensaje
          filter_subject: subject,
        });
        /*await this.jsm?.consumers.add(streamName, {
          durable_name: `${subject}-${nickname}-consumer`,
          ack_policy: AckPolicy.Explicit,
          deliver_subject: DeliverPolicy.New, // O DeliverPolicy.Last si solo quieres el último mensaje
          filter_subject: subject,
        });*/
        console.log(`Consumidor registrado en JetStreamManager para el subject: ${subject}`);

        const consumer = await this.js?.consumers.get(streamName, `${subject}-${nickname}-consumer`);
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
    const messageData = btoa(message.messageText);// encoder.encode(message.messageText); // Convierte el texto en un Uint8Array

    // Crear el NatsMessage
     
    const natsMessage: NatsMessage = {
      subject: topic, // El topic es el nombre del subject
      data:  messageData , // El contenido del mensaje como Uint8Array
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
        const producerSubject = this.getProducerSubjectForTopic(topic);
        console.log("Productor encontrado:", producer);
        
        // Verificar que JetStream esté conectado
        if (!this._isConnected) {
            throw new Error("JetStream no está conectado");
        }

        try {
            console.log('Streams en await this.js?.views:', await this.js?.views);
            console.log('Streams en await this.js?.streams:', await this.js?.streams);
        } catch (error) {
            console.error('Error al obtener información de los streams de JetStream:', error);
        }

        // Transformar Message a NatsMessage
        const natsMessage: NatsMessage = this.createNatsMessageFromMessage(message, producerSubject);
        console.log(`Mensaje transformado a NatsMessage: ${JSON.stringify(natsMessage)}`);

        // Serializar el mensaje utilizando la función serializeMessage
        const serializedMessage = serializeMessage(natsMessage);
        console.log(`Mensaje transformado y serializado: ${serializedMessage}`);
        
        // Intentar publicar con reintentos
        const MAX_RETRIES = 3;
        let retries = 0;
        let pubAck = null;
 
        while (retries < MAX_RETRIES) {
            try {
                pubAck = await this.js?.publish(producerSubject, serializedMessage);

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
  private  createNatsMessageFromRequestListUsers (rq: RequestListuser, topic: string): NatsMessage {
    console.log ("createNatsMessageFromRequestListUsers:",rq)
    // Mapeamos los valores del request a los headers del NatsMessage
    const headers: Record<string, string | Buffer> = {
      roomId: rq.roomId,
      tokenSesion: rq.tokenSesion,
      nickname: rq.nickname,
      request:rq.request,
      operation: rq.operation,      
      X_GoChat: rq.X_GoChat,
    };
    const encoder = new TextEncoder();
    // Creamos el mensaje NatsMessage
    const natsMessage: NatsMessage = {
      subject: topic, // El subject será el topic proporcionado
      data: btoa(rq.message) , // Si necesitas incluir datos adicionales, puedes ajustarlo aquí Buffer.from(  request.message  ) 
      timestamp: new Date().toISOString(), // Fecha y hora actual como timestamp
      headers: headers, // Los headers generados a partir del request
    };
    console.log ("createNatsMessageFromRequestListUsers::natsMessage:",natsMessage)
    return natsMessage;
  }
  public async publisRequestListUsershMessage(message: RequestListuser, topic: string): Promise<void> {
    try {
      console.log(`publisRequestListUsershMessage: Obtener el productor asociado al topic-stream: ${topic}`);
      // Obtener el productor asociado al topic-stream

      const producer = this.getProducerForTopic(topic);
      const subject = this.getProducerSubjectForTopic(topic);
      console.log(producer)
      if (!producer) {
        throw new Error(`No se encontró un productor para el topic: ${topic}`);
      }

      // Transformar Message a NatsMessage
      const natsMessage: NatsMessage = this.createNatsMessageFromRequestListUsers (message, topic);
      const serializedMessage =  serializeMessage (natsMessage);  
      console.log(`Mensaje transformado y serializado: ${serializedMessage}`);
       

      const pubAck = await this.js?.publish(subject, serializedMessage);

      console.log(`Mensaje publicado con éxito en ${subject}. Seq: ${pubAck?.seq}`);
    } catch (error) {
      console.error(`Error al publicar el mensaje en el subject del topic : ${topic}:`, error);
    }
  } 
  public async assignCallbackPullConsumer(
    topic: string,
    callback: (err: Error | null, msg: any | null, sharedObject: any) => void,
    sharedObject: any
): Promise<void> {
    console.log(`Asignando callback para el consumidor del tema: "${topic}"`);

    try {
        const consumer = this.getConsumerForTopic(topic)?.consumer;

        if (!consumer) {
            console.error(`No se encontró un consumidor para el tema: "${topic}"`);
            throw new Error(`No se encontró un consumidor para el tema: ${topic}`);
        }

        console.log(`Consumidor encontrado para el tema: "${topic}"`);

        let stopConsuming = false; // Control del ciclo de consumo

        const consumeMessages = async () => {
            while (!stopConsuming) {
                try {
                    // Consultar el estado del consumidor
                    const consumerInfo = await consumer.info();
                    const pendingMessages = consumerInfo.num_pending;

                    //console.log(`Mensajes pendientes en el tema "${topic}": ${pendingMessages}`);

                    // Solo consumir si hay mensajes pendientes
                    if (pendingMessages > 0) {
                        console.log(`Consumiendo mensajes del tema "${topic}"...`);

                        // Consumir un solo mensaje cada vez
                        const result = await consumer.consume({
                            callback: async (m: {
                                seq: any;
                                ack: () => void;
                                _data: Uint8Array;
                                data: any;
                                subject: string;
                                reply: string;
                                headers: any;
                            }) => {
                                try {
                                    console.log("m:", m);
                                    console.log("Sequence:", m.seq);
                                    console.log("Reply:", m.reply);
                                    console.log("Subject:", m.subject);

                                    // Verificación de datos antes de deserializar
                                    if (!m.data || m.data.length === 0) {
                                        console.warn(`Mensaje vacío recibido en el tema "${topic}"`);
                                        return; // No procesar mensajes vacíos
                                    }

                                    const decodedString = new TextDecoder().decode(m.data);
                                    
                                    // Verificación de deserialización segura
                                    let msgdecode;
                                    try {
                                        msgdecode = deserializeMessage(decodedString);
                                        console.log(`Mensaje deserializado con éxito:`, msgdecode);
                                    } catch (error) {
                                        console.error(`Error al deserializar el mensaje:`, error);
                                        return; // Si falla la deserialización, no procesar el mensaje
                                    }

                                    const message = {
                                        seq: m.seq,
                                        subject: m.subject,
                                        data: m.data,
                                        decodedData: msgdecode,
                                        headers: m.headers,
                                    };

                                    console.log("Full message:", message);
                                    callback(null, msgdecode, sharedObject);
                                    m.ack(); // Confirmar el procesamiento del mensaje

                                } catch (error) {
                                    console.error("Error al procesar el mensaje:", error);
                                    // Puedes manejar el error como prefieras
                                }
                            },
                        });

                        // Verificar si hay más mensajes pendientes para seguir consumiendo
                        if (result?.statusIterator?.closed) {
                            //console.log("El iterador se ha cerrado, esperando nuevo ciclo.");
                            await new Promise(resolve => setTimeout(resolve, 1000)); // Esperar antes de intentar nuevamente
                        }
                    } else {
                        //console.log(`No hay mensajes pendientes. Esperando...`);
                        await new Promise(resolve => setTimeout(resolve, 1000)); // Espera corta entre verificaciones
                    }
                } catch (error) {
                    console.error("Error al consumir mensajes:", error);
                    stopConsuming = true; // Detener el ciclo en caso de error crítico
                }
            }
        };

        // Iniciar el consumo de mensajes
        consumeMessages()
            .catch((err) => {
                console.error(`Error inesperado al consumir mensajes: ${err}`);
            })
            .finally(() => {
                console.log(`Proceso de consumo terminado para el tema: "${topic}"`);
            });

        console.log(`Callback asignado correctamente para el tema: "${topic}"`);

    } catch (error) {
        console.error(`Error al asignar callback para el tema "${topic}":`, error);
        throw error instanceof Error ? error : new Error(String(error));
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
    
        console.log(`Verificando si el consumidor para el tema "${topic}" existe...`);
        const consumer = this.getConsumerForTopic(topic)?.consumer;

        if (!consumer) {
            console.error(`No se encontró un consumidor para el tema: "${topic}"`);
            throw new Error(`No se encontró un consumidor para el tema: ${topic}`);
        }

        console.log(`Consumidor encontrado para el tema: "${topic}"`);
        const consumerSubject = this.getConsumerSubjectForTopic(topic);
        console.log('Consumer desde el contexto:', consumer);
        await consumer.consume({
          callback: (m: { 
            seq: any; 
            ack: () => void; 
            _data: Uint8Array; 
            data: any; 
            subject: string; 
            reply: string; 
            headers: any; 
        }) => {
            try {
              console.log("m:", m );   
              // 1. Acceder a los campos generales
              console.log("Sequence:", m.seq);  // Número de secuencia del mensaje
              console.log("Reply:", m.reply);    // Respuesta asociada al mensaje
              console.log("Subject:", m.subject); // Tema o asunto del mensaje
              
              // 2. Acceder a los datos binarios (_data) y decodificarlos si es necesario
              const decodedString = new TextDecoder().decode(m.data);
              const msgdecode = deserializeMessage(decodedString);
              console.log(`Mensaje deserializado con éxito:`, msgdecode);
          
              // 3. Acceder a los datos adicionales (si los hay)
              console.log("Data:", m.data); // Datos adicionales del mensaje, puede ser objeto o cadena
          
              // 4. Acceder a los encabezados (si los hay)
              console.log("Headers:", m.headers); // Encabezados del mensaje
          
              // 5. Crear el mensaje con los atributos que deseas
              const message = {
                  seq: m.seq,
                  subject: m.subject,
                  data: m.data,
                  decodedData: msgdecode,  // Aquí guardamos los datos decodificados
                  headers: m.headers
              };
       
              // Mostrar el objeto message completo
              console.log("Full message:", message);
              callback(null, msgdecode, sharedObject);
              // 6. Confirmar la recepción del mensaje (ack)
              m.ack();
            } catch (error) {
              console.error("Error al procesar el mensaje:", error);
          }
          },
        });
        
    
        console.log(`Callback asignado correctamente para el tema: "${topic}"`);
    } catch (error) {
        console.error(`Error al asignar callback para el tema "${topic}":`, error);
        throw error instanceof Error ? error : new Error(String(error));
    }
}
}