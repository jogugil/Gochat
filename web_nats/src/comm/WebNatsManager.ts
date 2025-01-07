import { connect,NatsConnection, StringCodec, JetStreamManager,  ConsumerOpts   } from 'nats.ws';

 
import { v4 as uuidv4 } from 'uuid';

// Accede a las variables de entorno
export const VITE_NATS_URLS = import.meta.env.VITE_NATS_URLS?.split(',') || ['localhost:4222'];
export const VITE_NATS_PREFIX_STREAMNAME = import.meta.env.VITE_NATS_PREFIX_STREAMNAME || "MYGOCHAT_STREAM";
export const VITE_MAINROOM_CLIENT_TOPIC = import.meta.env.VITE_MAINROOM_CLIENT_TOPIC || "principal.client";
export const VITE_MAINROOM_SERVER_TOPIC = import.meta.env.VITE_MAINROOM_SERVER_TOPIC || "principal.server";
export const VITE_GET_USERS_TOPIC = import.meta.env.VITE_GET_USERS_TOPIC || "roomlistusers.server";

// Estructura para almacenar productores y consumidores
interface NatsProducerConsumer {
  producer: any;  // El productor para el stream
  consumer: any;  // El consumidor para el stream
}
export  interface NatsMessage {
  subject: string;
  data: Uint8Array; // Ajustado para el entorno de navegador . Antes data: Buffer;
  timestamp: string;
  headers: Record<string, unknown>;  // Acepta claves de tipo string, pero los valores son 'unknown'
}
export  class NatsStreamManager {
  private natsClient: NatsConnection | null = null;
  private jsm: any = null; 
  private producersConsumers: { [key: string]: { producer: any; consumer: any; callback?: Function } } = {};
  private codec = StringCodec();
  private _isConnected: boolean = false;

  // Conectar a NATS y JetStream
  async connect() {
    try {
      console.log(`Conectando a NATS con URLs: ${VITE_NATS_URLS.join(", ")}`);
      this.natsClient = await connect({ servers:  "ws://localhost:4222" });
      console.log("Conexión a NATS exitosa.");

      if (this.natsClient) {
        this.jsm = this.natsClient.jetstream(); // Aquí es donde obtenemos JetStream
      }

      // Conectar y crear streams según el topic
      await this.ensureStreamExists(VITE_MAINROOM_SERVER_TOPIC, 'server');
      await this.ensureStreamExists(VITE_MAINROOM_CLIENT_TOPIC, 'client');
      await this.ensureStreamExists(VITE_GET_USERS_TOPIC, 'server');
      this._isConnected = true;
    } catch (error) {
      console.error("Error de conexión con NATS:", error);
    }
  }
  
  // Getter para obtener el estado de conexión
  public get isConnected(): boolean {
    return this._isConnected;
  }
  private getProducerForTopic(topic: string): any | null {
        const producer = this.producersConsumers[topic]?.producer;
        if (!producer) {
            console.error('No se encontró productor para el topic:', topic);
        }
        return producer;
  }
  private getConsumerForTopic(topic: string): any | null {
    const producer = this.producersConsumers[topic]?.consumer;
    if (!producer) {
        console.error('No se encontró productor para el topic:', topic);
    }
    return producer;
}
  /*private createNatsMessage(requestData: any, topic: string): any {
    // Crear los headers que contienen información adicional (como token, roomId, nickname)
    const headers = {
      roomId: requestData.roomid,
      nickname: requestData.nickname,
      tokensession: requestData.tokensession,
      roomname: requestData.roomname,
    };

    // Crear el objeto NatsMessage (como un objeto estándar de JavaScript)
    const now = new Date();
    const natsMessage = {
      subject: topic,                      // El topic es el nombre de la sala o el tipo de mensaje
      data: Buffer.from(JSON.stringify(requestData)),  // Convertir el requestData a un Buffer
      timestamp: now.toLocaleTimeString(),   // Obtener el timestamp actual
      headers: headers,                    // Agregar los headers
    };

    return natsMessage;
  }*/
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
  /*private async ensureStreamExists(topic: string, type: 'server' | 'client') {
    const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split('.')[0]}`;

    try {
      // Verificar si el stream ya existe
      const stream = await this.jsm?.streams.info(streamName);
      if (stream) {
        console.log(`Stream para ${topic} ya existe.`);
      } else {
        // Si no existe, lo creamos
        await this.createStream(streamName, topic, type);
      }
    } catch (error) {
      console.error(`Error al verificar o crear el stream para ${topic}:`, error);
    }
  }*/
    private async ensureStreamExists(topic: string, type: 'server' | 'client') {
      const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split('.')[0]}`;
  
      try {
        const stream = await this.jsm?.streams.info(streamName);
        if (stream) {
          console.log(`Stream para ${topic} ya existe.`);
        } else {
          await this.createStream(streamName, topic, type);
        }
      } catch (error) {
        console.error(`Error al verificar o crear el stream para ${topic}:`, error);
      }
    }

  // Crear un stream si no existe
  /*private async createStream(streamName: string, topic: string, type: 'server' | 'client') {
    try {
      // Obtener el prefijo desde la variable de entorno
      const prefix = process.env.VITE_NATS_PREFIX_STREAMNAME || '';  // Prefijo por defecto vacío si no está definido
      const streamWithPrefix = `${prefix}_${streamName}_CL`;  // Agregar sufijo "_CL" al nombre del stream
  
      // Definir la configuración del stream
      const config = {
        name: streamWithPrefix,  // Usamos el nombre del stream con el prefijo y sufijo
        subjects: [`${topic}.*`], // El stream recibe todos los mensajes de los subtopics del topic
        retention: 'limits', // Retención de mensajes configurada
        max_msgs_per_subject: 1000, // Límite de mensajes por subtopic
        max_age: 60 * 60 * 1000, // Mensajes caducan después de una hora
      };
  
      // Crear el stream en JetStream
      await this.jsm?.streams.add(config);  // Usar el objeto jsm para crear el stream
      console.log(`Stream para ${topic} creado con éxito: ${streamWithPrefix}`);
      
      // Crear productor y consumidor
      this.createProducerConsumer(streamWithPrefix, topic, type);
    } catch (error) {
      console.error(`Error al crear el stream para ${topic}:`, error);
    }
  }*/
    private async createStream(streamName: string, topic: string, type: 'server' | 'client') {
      try {
        const prefix = process.env.VITE_NATS_PREFIX_STREAMNAME || '';
        const streamWithPrefix = `${prefix}_${streamName}_CL`;
  
        const config = {
          name: streamWithPrefix,
          subjects: [`${topic}.*`],
          retention: 'limits',
          max_msgs_per_subject: 1000,
          max_age: 60 * 60 * 1000,
        };
  
        await this.jsm?.streams.add(config);
        console.log(`Stream para ${topic} creado con éxito: ${streamWithPrefix}`);
  
        // Crear productor y consumidor
        this.createProducerConsumer(streamWithPrefix, topic, type);
      } catch (error) {
        console.error(`Error al crear el stream para ${topic}:`, error);
      }
    }
  // Crear un productor y un consumidor para un stream
  /*private async createProducerConsumer(streamName: string, topic: string, type: 'server' | 'client') {
    const subject = `${topic}.${type}`;
    
    if (type === 'server') {
      // Crear un productor
      const producer = await this.jsm.publish(subject, Buffer.from('Mensaje de prueba'));
      console.log(`Productor creado para el stream ${streamName}: ${subject}`);
      if (producer) {
        // Guardamos el productor y el consumidor en el objeto producersConsumers
        this.producersConsumers[topic] = {
          producer: producer,
          consumer: null, // Inicializamos el consumidor si es necesario
        };
        console.log(`Productor creado y almacenado para el topic: ${topic}`);
      } else {
        console.error('Error al crear el productor para el topic:', topic);
      }
    } else {
      // Crear un consumidor
      const consumer = await this.jsm.pullSubscribe(subject, { 
        manualAck: true, 
        deliverTo: `${subject}.worker` 
      });
      console.log(`Consumidor creado para el stream ${streamName}: ${subject}`);
      if (consumer) {
        // Guardamos el productor y el consumidor en el objeto producersConsumers
        this.producersConsumers[topic] = {
          producer: null,
          consumer: consumer, // Inicializamos el consumidor si es necesario
        };
        console.log(`Productor creado y almacenado para el topic: ${topic}`);
      } else {
        console.error('Error al crear el productor para el topic:', topic);
      }
      // Agregamos el consumidor a un manejador
      consumer?.pull(); // Empieza a recibir mensajes
    }

    this.producersConsumers[streamName] = { producer: null, consumer: null };
  }*/
// Crear un productor y un consumidor para un stream
  private async createProducerConsumer(streamName: string, topic: string, type: 'server' | 'client') {
    const subject = `${topic}.${type}`;

    if (type === 'server') {
      const producer = await this.jsm?.publish(subject, Buffer.from('Mensaje de prueba'));
      console.log(`Productor creado para el stream ${streamName}: ${subject}`);

      if (producer) {
        this.producersConsumers[topic] = { producer, consumer: null };
        console.log(`Productor creado y almacenado para el topic: ${topic}`);
      } else {
        console.error('Error al crear el productor para el topic:', topic);
      }
    } else {
      const consumer = await this.jsm?.pullSubscribe(subject, { manualAck: true, deliverTo: `${subject}.worker` });
      console.log(`Consumidor creado para el stream ${streamName}: ${subject}`);

      if (consumer) {
        this.producersConsumers[topic] = { producer: null, consumer };
        console.log(`Consumidor creado y almacenado para el topic: ${topic}`);
      } else {
        console.error('Error al crear el consumidor para el topic:', topic);
      }

      consumer?.pull();
    }
  }

  /*async sendNatsMessage(requestData: any, topic: string) {
    try {
        // Crear el mensaje NATS usando el método de la instancia
        const natsMessage = this.createNatsMessage(requestData, topic);

        // Publicar el mensaje en el stream correspondiente
        const producer = this.getProducerForTopic(topic);
        if (producer) {
            await producer.publish(natsMessage.subject, natsMessage.data); // Enviar el mensaje sin esperar respuesta
            console.log(`Mensaje enviado a NATS: ${topic}`);
        } else {
            console.error('No se encontró productor para el topic', topic);
        }

    } catch (error) {
        console.error('Error al crear o enviar el NatsMessage:', error);
    }
}*/
async sendNatsMessage(requestData: any, topic: string) {
  try {
    const natsMessage = this.createNatsMessage(requestData, topic);
    const producer = this.getProducerForTopic(topic);

    if (producer) {
      await producer.publish(natsMessage.subject, natsMessage.data);
      console.log(`Mensaje enviado a NATS: ${topic}`);
    } else {
      console.error('No se encontró productor para el topic', topic);
    }
  } catch (error) {
    console.error('Error al crear o enviar el NatsMessage:', error);
  }
}
 
  // Enviar un mensaje al stream correspondiente
  /*sendMessageToStream(topic: string, message: string) {
    const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split('.')[0]}`;
    const producer = this.producersConsumers[streamName]?.producer;
    
    if (producer) {
      producer.publish(this.codec.encode(message));
      console.log(`Mensaje enviado a ${topic}: ${message}`);
    } else {
      console.error(`No se encontró un productor para el stream ${streamName}`);
    }
  }*/
    sendMessageToStream(topic: string, message: string) {
      const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split('.')[0]}`;
      const producer = this.producersConsumers[streamName]?.producer;
  
      if (producer) {
        producer.publish(this.codec.encode(message));
        console.log(`Mensaje enviado a ${topic}: ${message}`);
      } else {
        console.error(`No se encontró un productor para el stream ${streamName}`);
      }
    }

   
  // Cerrar la conexión con NATS
  close() {
    this.natsClient?.close();
    console.log("Conexión NATS cerrada.");
  }

  // Función para iniciar un consumidor y asignar un callback
  /*async startConsumerCallback(topic: string, callback: Function) {
    const streamName = `${VITE_NATS_PREFIX_STREAMNAME}.${topic.split('.')[0]}`;
    const consumer = this.producersConsumers[streamName]?.consumer;

    if (consumer) {
      // Asignamos el callback al consumidor
      this.producersConsumers[streamName].callback = callback;
      console.log(`Callback asignado para el topic ${topic}`);
      await consumer.pull();
    } else {
      console.error(`No se encontró consumidor para el topic ${topic}`);
    }
  }*/
    async startConsumerCallback(topic: string, callback: Function) {
      const consumer = this.getConsumerForTopic(topic);
      if (consumer) {
        await consumer.pull();
        console.log(`Callback asignado para el topic ${topic}`);
      } else {
        console.error(`No se encontró consumidor para el topic ${topic}`);
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
  if (!this.natsClient) {
    console.error("No conectado a NATS");
    return;
  }

  try {
    const subscription = this.natsClient.subscribe(topic);
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
 
