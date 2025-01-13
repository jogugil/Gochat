// Uasar REST Proxy de Confluent para acceder al servidor de mensajeria kafka desde el navegador
// Creación de un backend ligero que funcione a modo de proxy entre el cleinte y el servidor kafka

import { Kafka, Admin, Producer, Consumer, EachMessagePayload,ConsumerSubscribeTopics } from 'kafkajs';
import { RequestListuser, KafkaMessage } from '../types/typesComm'; 
import { Message, UUID } from "../models/Message";
import { v4 as uuidv4 } from 'uuid';

// Accede a las variables de entorno
export const VITE_KAFKA_BROKERS = import.meta.env.VITE_KAFKA_BROKERS?.split(',') || ['localhost:9092'];
export const VITE_MAINROOM_TOPIC = import.meta.env.VITE_MAINROOM_TOPIC || "principal";
export const VITE_GET_USERS_TOPIC = import.meta.env.VITE_GET_USERS_TOPIC || "roomlistusers.server";

// Definición de las interfaces
interface KafkaProducerConsumer {
  producer: Producer | null;
  consumer: Consumer | null;
}

export class KafkaManager {
  private kafka: Kafka;
  private admin: Admin | null;
  private producersConsumers: Map<string, { producer: Producer; consumer: Consumer }> = new Map();
  private _isConnected: boolean = false;
  private subscribedTopics: Set<string> = new Set();
  
  public get isConnected(): boolean {
    return this._isConnected;
  }

  static async create(nickname: string | null): Promise<KafkaManager> {
    const brokers: string[] = VITE_KAFKA_BROKERS.split(',');
    const manager = new KafkaManager(brokers);
    await manager.connect(nickname);
    return manager;
  }

  constructor(private brokers: string[]) {
    this.kafka = new Kafka({
      clientId: 'KafkaManagerClient',
      brokers: this.brokers,
    });
    this.producersConsumers = new Map();
    this.admin = null;
    this._isConnected = false;
  }

 
  // Conectar a Kafka
  async connect(nickname: string | null): Promise<void> {
    try {
      console.log(`KafkaManager: Conectando a Kafka con brokers: ${VITE_KAFKA_BROKERS.join(", ")}`);
      
      // Conectar administrador
      await this.connectAdmin();

      this._isConnected = true;

      // Asegura la existencia de topics
      await Promise.all([
        this.ensureTopicExists(VITE_MAINROOM_TOPIC, nickname),
        this.ensureTopicExists(VITE_GET_USERS_TOPIC, null),
      ]);

      console.log("KafkaManager: Conexión a Kafka exitosa.");
    } catch (error) {
      console.error("KafkaManager: Error de conexión con Kafka:", error);
    }
  }
  // Conectar el administrador
  async connectAdmin(): Promise<void> {
    try {
      this.admin = this.kafka.admin();
      await this.admin.connect();
      console.log('KafkaManager: Admin conectado con éxito.');
    } catch (error) {
      console.error('KafkaManager: Error al conectar el Admin:', error);
    }
  }
  // Verificar si un topic existe y crear un productor y consumidor
  // Función para asegurarse de que el topic existe y crear productor y consumidor
  private async ensureTopicExists(topic: string, nickname: string | null): Promise<void> {
    // Determinar productor y consumidor
    const producerTopic = `${topic}.server`; // El productor siempre se conecta al topic.server
    const consumerTopic = nickname ? `${nickname}.client` : `${topic}.client`; // El consumidor depende del nickname

    // Crear productor y consumidor
    const producer: Producer = this.kafka.producer(); // El productor es para topic.server
    const consumer: Consumer = this.kafka.consumer({ groupId: uuidv4() }); // El consumidor usa un groupId único

    try {
      // Conectar productor y consumidor
      await producer.connect();
      await consumer.connect();

      // El consumidor se suscribe al topic adecuado
      await consumer.subscribe({ topic: consumerTopic });
      this.subscribedTopics.add(consumerTopic);
      console.log(`KafkaStreamManager: Productor para ${producerTopic} y consumidor para ${consumerTopic} creados.`);

      // Guardar productor y consumidor en el mapa
      this.producersConsumers.set(topic, { producer, consumer });
    } catch (error) {
      console.error(`KafkaStreamManager: Error al crear productor o consumidor para el topic ${topic}:`, error);
    }
  }
  // Función para obtener el consumidor de un topic
  private getConsumerForTopic(topic: string): Consumer | null {
    const consumer = this.producersConsumers.get(topic)?.consumer ?? null; 
    if (!consumer) {
      console.error('No se encontró consumidor para el topic:', topic);
    }
    return consumer;
  }
    // Función para obtener el productor de un topic
    private getProducerForTopic(topic: string): Producer | null {
      const producer = this.producersConsumers.get(topic)?.producer ?? null;
      if (!producer) {
        console.error('No se encontró productor para el topic:', topic);
      }
      return producer;
    }
 
    async isSubscribed(consumer: Consumer, topicName: string): Promise<boolean> {
      return this.subscribedTopics.has(topicName);
    }

  // Función para enviar mensajes a Kafka
  async sendKafkaMessage(requestData: Message, topic: string): Promise<void> {
    try {
      const kafkaMessage = this.createKafkaMessage(requestData, topic);
      const producer = this.producersConsumers.get(topic)?.producer;

      if (producer) {
        const now = new Date();
        await producer.send({
          topic: kafkaMessage.key?.toString() ?? "principal.server",
          messages: [{ key: Buffer.from(kafkaMessage.key?.toString() ?? "principal.server"), value: kafkaMessage.value, timestamp: now.toISOString(), headers: kafkaMessage.headers }],
        });
        console.log(`KafkaManager: Mensaje enviado a Kafka: ${topic}`);
      } else {
        console.error('KafkaManager: No se encontró productor para el topic', topic);
      }
    } catch (error) {
      console.error('KafkaManager: Error al crear o enviar el KafkaMessage:', error);
    }
  }

  // Función para crear el mensaje Kafka con datos
  private createKafkaMessage(requestData: Message, topic: string): KafkaMessage {
    const headers = {
      messageId: requestData.messageId, // Identificador único
      messageType: requestData.messageType, // Tipo de mensaje
      sendDate: requestData.sendDate, // Fecha de envío
      serverDate: requestData.serverDate, // Fecha en el servidor
      nickname: requestData.nickname, // Nombre del remitente
      token: requestData.token, // JWT del remitente
      messageText: requestData.messageText, // Contenido del mensaje
      roomId: requestData.roomId, // Sala destino
      roomName: requestData.roomName, // Nombre de la sala
      metadata: JSON.stringify(requestData.metadata), // Metadatos adicionales
    };
  
    const now = new Date();
  
    return {
      key: Buffer.from(topic), // El key es el topic
      value: Buffer.from(JSON.stringify(requestData.messageText)), // El mensaje es el objeto RequestData en formato JSON
    

      timestamp: now.toISOString(), // Marca de tiempo actual
      headers, // Los headers contienen todos los atributos del mensaje
    };
  }
  // Función para enviar un mensaje de tipo RequestListuser a Kafka
  async sendKafkaMessageForListUser(requestData: RequestListuser, topic: string): Promise<void> {
    try {
      // Crear el mensaje Kafka con los datos de RequestListuser
      const kafkaMessage = this.createKafkaMessageForListUser(requestData, topic);

      // Obtener el productor del topic
      const producer = this.producersConsumers.get(topic)?.producer;

      if (producer) {
        // Enviar el mensaje con Kafka
        await producer.send({
          topic: kafkaMessage.key?.toString() ?? "roomlistusers.server",
          messages: [
            {
              key: kafkaMessage.key, // Establecer el key como el topic
              value: kafkaMessage.value, // El valor es el mensaje JSON de RequestListuser
              timestamp:kafkaMessage.timestamp,
              headers: kafkaMessage.headers, // Los headers con los atributos
            },
          ],
        });
        console.log(`KafkaManager: Mensaje de RequestListuser enviado a Kafka: ${topic}`);
      } else {
        console.error('KafkaManager: No se encontró productor para el topic', topic);
      }
    } catch (error) {
      console.error('KafkaManager: Error al crear o enviar el KafkaMessage:', error);
    }
  }

  // Función para crear el KafkaMessage para RequestListuser
  private createKafkaMessageForListUser(requestData: RequestListuser, topic: string): KafkaMessage {
    const headers = {
      roomId: requestData.roomId,
      tokenSesion: requestData.tokenSesion,
      nickname: requestData.nickname,
      operation: requestData.operation,
      topic: requestData.topic,
      xGoChat: requestData.xGoChat,
    };

    const now = new Date();
    return {
      key:Buffer.from(topic), // El topic es el que se pasa como parámetro
      value: Buffer.from(JSON.stringify({ mensaje: "Petición de listado de usuarios activos." })),// El mensaje es el objeto RequestListuser convertido a JSON
      timestamp: now.toISOString(), // Timestamp actual
      headers, // Los headers contienen los atributos de RequestListuser
    };
  }
  // Función para iniciar el consumidor de Kafka con un callback
  async startConsumerCallback(topic: string, callback: (message: KafkaMessage) => void): Promise<void> {
    const consumer = this.producersConsumers.get(topic)?.consumer;

    if (consumer) {
      await consumer.run({
        eachMessage: async ({ topic, partition, message }: EachMessagePayload) => {
          const kafkaMessage: KafkaMessage = {
            key: Buffer.from(topic),
            value:  message.value ? Buffer.from(message.value.toString()) : Buffer.alloc(0),  
            timestamp: new Date().toISOString(),
            headers: message.headers || {},
          };
          callback(kafkaMessage);
        },
      });
      console.log(`KafkaManager: Callback asignado para el topic ${topic}`);
    } else {
      console.error(`KafkaManager: No se encontró consumidor para el topic ${topic}`);
    }
  }

  // Función para detener el consumidor de Kafka
  async stopConsumer(topic: string): Promise<void> {
    const consumer = this.producersConsumers.get(topic)?.consumer;

    if (consumer) {
      await consumer.disconnect();
      console.log(`KafkaManager: El consumidor para el topic ${topic} ha sido detenido.`);
    } else {
      console.error(`KafkaManager: No se encontró consumidor para el topic ${topic}`);
    }
  }

  async  getAllMessagesFromTopic(topicName: string, limit: number = -1): Promise<Map<string, string>> {
    const messagesMap = new Map<string, string>(); // Mapa para almacenar los mensajes procesados
  
    try {
      // Obtener el consumidor para el topic, o crear uno nuevo si no existe
      const consumer = this.getConsumerForTopic(topicName) || this.kafka.consumer({ groupId: uuidv4() });
  
      // Si no existe un consumidor para el topic en el mapa y tampoco se ha creado uno nuevo, lanzar error
      if (!consumer) {
        console.error(`No se pudo obtener o crear un consumidor para el topic: ${topicName}`);
        return messagesMap; // Retornar el mapa vacío si no se puede obtener un consumidor
      }
  
      // Suscribirse al topic solo si no se ha suscrito previamente
      const topic: ConsumerSubscribeTopics = {
        topics: [topicName],
        fromBeginning: true, // Leer desde el principio
      };
  
      // Verificar si el consumidor ya está suscrito a este topic
      // Si no está suscrito, procedemos a suscribirlo
      const isSubscribed = await this.isSubscribed(consumer, topicName);
      if (!isSubscribed) {
        await consumer.subscribe(topic);
      }
  
      let processedMessages = 0;
      let totalProcessed = 0;
 
      // Procesar los mensajes
      await consumer.run({
        eachMessage: async ({ message }: EachMessagePayload) => {
          const kafkaMessage: KafkaMessage = {
            key: message.key ?? Buffer.from(''),  // Default if key is null or undefined
            value: message.value ? Buffer.from(message.value.toString()) : Buffer.from(''),  // Default if value is null or undefined
            timestamp: message.timestamp ?? '',  // Default if timestamp is null or undefined
            headers: message.headers ?? {},  // Default if headers is null or undefined
          };
  
          // Extraer nickname de los headers
          const nickname = kafkaMessage.headers?.["nickname"];
          const nicknameKey = nickname ? nickname.toString() : "unknown"; // Convertir Buffer a string o usar "unknown"
  
          // Extraer el valor del mensaje
          const value = kafkaMessage.value.toString();
  
          console.log(`Procesando mensaje: nickname=${nicknameKey}, value=${value}`);
  
          // Agregar al mapa
          messagesMap.set(nicknameKey, value);
  
          processedMessages++;
          totalProcessed++;
  
          // Verificar si se han procesado los mensajes que necesitamos
          if (limit > 0 && processedMessages >= limit) {
            console.log(`Se alcanzó el límite de mensajes: ${limit}`);
            // No desconectar aquí. Lo haremos después de terminar todo el procesamiento.
          }
  
          // Si no hay límite o ya se procesaron todos los mensajes, desconectar
          if (limit === -1 && totalProcessed >= totalProcessed) {
            console.log(`Todos los mensajes procesados (${totalProcessed}). Saliendo...`);
          }
        },
      });
  
      // Esperar hasta que se haya procesado el límite de mensajes o todos los mensajes
      if (limit > 0 && processedMessages >= limit) {
        console.log(`Se alcanzó el límite de mensajes: ${limit}`);
      }
  
      // Desconectar al terminar
      await consumer.disconnect();
  
    } catch (error) {
      console.error('Error procesando mensajes:', error);
    }
  
    // Devolver el mapa con los mensajes procesados
    return messagesMap;
  }

  // Función para esperar un solo mensaje de un topic
  async waitForSingleMessage(topic: string): Promise<string | null> {
    const consumer = this.producersConsumers.get(topic)?.consumer;

    if (consumer) {
      try {
        // Suscribirse al topic
        await consumer.subscribe({ topic });

        // Esperar el primer mensaje
        const message = await new Promise<string | null>((resolve, reject) => {
          consumer.run({
            eachMessage: async ({ message }) => {
              resolve(message.value?.toString() || null);
              // Desconectar después de recibir el primer mensaje
              await consumer.disconnect();
            },
          });

          // Timeout por si no se recibe ningún mensaje
          setTimeout(() => reject('No se recibió mensaje en el tiempo esperado'), 5000);
        });

        console.log(`KafkaManager: Mensaje recibido para el topic ${topic}: ${message}`);
        return message;

      } catch (error) {
        console.error(`KafkaManager: Error al esperar el mensaje para el topic ${topic}:`, error);
        return null;
      }
    } else {
      console.error(`KafkaManager: No se encontró consumidor para el topic ${topic}`);
      return null;
    }
  }
  /////////////////
  ///7 Admin Funcitons
  ///////////////
  async listTopics(): Promise<string[]> {
    try {
      // Llamamos al método listTopics del admin y lo devolvemos.
      if (this.admin) {
        const topics = await this.admin.listTopics();
        return topics;
      } else {
        console.warn("Admin instance is null. Returning an empty list.");
        return []; // Devuelve un array vacío si admin es nulo
      }

    } catch (error) {
      console.error("Error listing topics:", error);
      throw error; // Relanzar el error para manejarlo en otro lugar si es necesario.
    }
  }
  async  close() {
    await this.admin?.disconnect()
  }
}
 