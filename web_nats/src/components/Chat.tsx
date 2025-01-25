import React, { useState, useEffect, useContext, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { sendMessage, handleNatsMessageWrapper, handleNatsGetAliveUsersWrapper, requestActiveUsers } from '../api/api';
import { useAuth } from './AuthContext';  // Importamos el contexto de autenticación
import { Room } from '../models/Room';
import { Message, UUID } from "../models/Message";
import { User } from "../models/User";
 
import { WErrorMessage, } from "./ErrorMessage"; // Componente para mostrar errores
import { MessageResponse, ResponseUser, AliveUser,Response } from '../types/typesComm'; // Ajusta la ruta según la estructura de tu proyecto

import '../styles/chat.css';
import BannerProgramming from './BannerProgramming';
import BannerCloud from './BannerCloud';
import prohibitedWords from "./prohibitedWords";
import { getClientInformation } from '../utils/ClientData'; // Ajusta la ruta según tu estructura de carpetas
import {NatsManager, VITE_MAINROOM_TOPIC, VITE_GET_USERS_TOPIC} from '../comm/WebNatsManager';
 
const Clock = () => {
  const [currentTime, setCurrentTime] = useState<string>('');

  // Función para actualizar la hora
  const updateTime = () => {
    const now = new Date();
    setCurrentTime(now.toLocaleTimeString());
  };

  useEffect(() => {
    // Llamar a updateTime inmediatamente al montar el componente para mostrar la hora inicial
    updateTime();

    // Establecer el intervalo para actualizar la hora cada segundo
    const intervalId = setInterval(updateTime, 1000);

    // Limpiar el intervalo cuando el componente se desmonte
    return () => clearInterval(intervalId);
  }, []);  // El array vacío asegura que se ejecute solo una vez al montar

  return (
    <p><strong>Fecha:</strong> {currentTime}</p>
  );
};

const Chat = () => {

  const navigate = useNavigate(); //Navegar por la web de Gochat

  const [isMessageSendable, setIsMessageSendable] = useState(false);       // Para habilitar/deshabilitar input y botón
  const [messages, setMessages] = useState<Map<string, { nickname: string; message: string }>>(new Map());

  const [messageText, setMessageText]   = useState<string> ('');           // Estado para el texto del mensaje
  const [aliveUsers, setAliveUsers]     = useState<string[]> ([]);         // Estado para los usuarios activos
 
  //Control scroll zona central
  const [scrollToBottomFlag, setScrollToBottomFlag] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);

  // Función para desplazarse al final del contenedor de mensajes
  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  //Ventana de error /información
  const [errorQueue, setErrorQueue] = useState<string[]>([]);
  const [minimized, setMinimized]       = useState (false);

  const closeErrorMessage = () => {
    setErrorQueue((prevQueue) => {
      // Eliminar el primer mensaje de la cola
      const newQueue = prevQueue.slice(1);
  
      // Actualizar el estado de errorMessage con el siguiente mensaje (si hay alguno)
      setErrorMessage(newQueue[0] || ''); // Si la cola está vacía, ponemos un mensaje por defecto
  
      // Si ya no hay mensajes en la cola, cerramos el componente
      if (newQueue.length === 0) {
        setShowError(false);
      }
  
      return newQueue; // Retornar la nueva cola
    });
  };

  const minimizeErrorMessage = () => {
    setMinimized(true);
  }; 
  const restoreErrorMessage  = () => setMinimized (false);  // Función para restaurar el mensaje de error

  // Función para mostrar el error
  const showErrorModal = (message: string) => {
    console.log("Dentro de showErrorModal con message:",message)
   // Agregar el nuevo mensaje a la cola
   setErrorQueue((prevQueue) => [...prevQueue, message]);

   // Si la ventana no está mostrando un mensaje, mostrar el primero de la cola
   if (!showError) {
     setShowError(true);
     setErrorMessage(message); // Tomamos el primer mensaje de la cola
   }
  };

  //Control coor fondo zona central
  const [isDarkMode, setIsDarkMode] = useState(false);
  const toggleTheme = () => {
    setIsDarkMode(!isDarkMode);
  };
  
  // Timeout definido en el archivo de entorno (.env). Cada cuanto tiempo el polling realia la petición de mensajes/usuarios
  const timeout = parseInt(import.meta.env.VITE_POLLING_TIMEOUT, 10 ) || 2000;
 
  //Logica de usuario y chat
  const { token, nickName, roomId, roomName } = useAuth();  // Obtener el usuario y el token del contexto
  const [userChat, setUserChat]               = useState<User | null>(null);  // Estado para el objeto User
  const [room, setRoom]                       = useState<Room | null>(null);  // Estado para el objeto Room
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [initialized, setInitialized]         = useState<boolean>(false);
  const [showError, setShowError]             = useState(false);
  const [errorMessage, setErrorMessage]       = useState('');
  const [isErrorActive, setIsErrorActive]     = useState(false);
  const intervalIdRef = useRef<number | null>(null);
  const [callbackExecuted, setCallbackExecuted] = useState(false);
  //Conexión servidor mensjaeria (Nats|Kafka)
  const [natsManager, setNatsManager] = useState<NatsManager | null>(null);
  const [connectionError, setConnectionError] = useState<boolean>(false); // Estado para el error de conexión
  const [shouldConnectToNats, setShouldConnectToNats] = useState(false);

  // Controlo input envío de mensjaes
  // Tipo explícito para las claves válidas
  type EscapeChar = "<" | ">" | "&" | "\"" | "'";

  // Mapa de caracteres a escapar
  const escapeMap: Record<EscapeChar, string> = {
    "<": "&lt;",
    ">": "&gt;",
    "&": "&amp;",
    "\"": "&quot;",
    "'": "&#39;",
  };

  // Función para escapar caracteres peligrosos
  const escapeHTML = (text: string): string => {
    return text.replace(/[<>&"']/g, (char) => escapeMap[char as EscapeChar] || char);
  };

  // Eliminamos cualquier elemento que no sea carácter alfanumérico
  const sanitizeMessage = (text: string): string => {
    return text.replace(/[^a-zA-Z0-9\s]/g, "");
  };

  // Verifica si el mensaje contiene palabras prohibidas
  const containsProhibitedWords = (text: string) => {
    return prohibitedWords.some((word) => text.toLowerCase().includes(word));
    };
  
  //envia el mensaje que el usuario pone en el input
  const handleSendMessage = async () => {
    if (!messageText.trim() || !isMessageSendable) return; // No enviar si el mensaje está vacío o el envío está deshabilitado
   
      try { 
        if (userChat && room) {
          const sanitizedMessage = sanitizeMessage(messageText.trim());
          const escapedMessage = escapeHTML(sanitizedMessage);

          if (!escapedMessage) {
            showErrorModal("No puedes enviar un mensaje vacío.");
 
            return;
          }

          if (containsProhibitedWords(escapedMessage)) {
            showErrorModal("El mensaje contiene lenguaje prohibido.");
 
            return;
          }
          if (natsManager) {
            const response = await sendMessage(VITE_MAINROOM_TOPIC, natsManager, userChat.nickname, userChat.token, room.roomId, room.roomName, messageText);
          } else {
            console.error("No se pudo enviar el mensaje. No existe conexión con Nats");
          }
          
          setMessageText(''); // Limpiamos el input
        
        }
      } catch (err) {
        showErrorModal('No se pudo enviar el mensaje. Intente de nuevo.');
        console.error("Error al enviar el mensaje:", err);        
        throw new Error('El servicio de chat no está disponible. Disculpe las molestias. Por favor, intente ingresar nuevamente más tarde.');
 
      }
   
};


const connectToNats = async () => {
  console.log("Dentro de onst connectToNats = async ()")
  if (!isAuthenticated || !userChat) {
    console.error('Usuario no autenticado. No se puede conectar a NATS.');
    return;
  }

  try {
    if (!natsManager) {
      // Verifica que nickname esté disponible y sea válido
      const initializeNatsManager = async () => {
        if (userChat.nickname) {
          try {
            // Crear NatsManager con el nickname
            console.log("Creando NatsManager para el usuario:", userChat.nickname);
            const createdNatsManager = await NatsManager.create(userChat.nickname);
            
            // Actualiza el estado de NatsManager
            setNatsManager(createdNatsManager);
            console.log('Conexión a NATS establecida para el usuario:', userChat.nickname);
            console.log (createdNatsManager)
            // Una vez que todo esté listo, actualizamos el estado para que continúe el flujo
            setShouldConnectToNats(true);
            console.log("Consumidores y productores creados correctamente.");
          } catch (error) {
            console.error('Error al crear NatsManager o al inicializar consumidores/productores:', error);
            setConnectionError(true);
          }
        } else {
          console.error('Nickname no disponible o no válido.');
        }
      };

      // Llamar a la función de inicialización
      await initializeNatsManager();

    } else {
      console.error('No se pudo conectar a NATS. El objeto natsManager ya existe.');
    }
  } catch (error) {
    console.error('Error al conectar al servidor de mensajería NATS');
    setConnectionError(true);
  }
};
const pushCallback = async () => {
  if (natsManager && userChat) {
    console.log("--> Dentro de const pushCallback = async () => { ")
    // Obtener los valores de los topics directamente desde WebNatsManager
    const mainRoomTopic = VITE_MAINROOM_TOPIC  || "principal";     // Default a "principal" si no está definido
    const usersTopic    = VITE_GET_USERS_TOPIC || "roomlistusers"; // Default a "roomlistusers" si no está definido

    // Asignar el callback handleNatsMessageWrapper al consumidor para el tema principal
    natsManager.assignCallbackPullConsumer(mainRoomTopic, handleNatsMessageWrapper(setMessages), true);
    console.log(`--> Callback asignado para el consumidor ${mainRoomTopic}.client`);

    // Asignar el callback handleNatsGetAliveUsersWrapper al consumidor para el tema específico de roomlistusers
    const userRoomTopic = usersTopic; // El tema se crea dinámicamente con el nickname del usuario
    // Asignar el callback de usuarios vivos
    natsManager.assignCallbackPullConsumer(usersTopic, handleNatsGetAliveUsersWrapper(setAliveUsers), false);
    console.log(`--> Callback asignado para el consumidor de usuarios vivos: ${usersTopic}.client`);
  }
}
// Función que realiza la actualización el listado inicial de mensajes y de usuarios activos
const startUpdates = ( ) => {
  console.log("startPeriodicUpdates = ( ) => {"); 
  console.log("userChat:", userChat);
 

  // Verificar que el usuario esté disponible
  if (!userChat) {
    showErrorModal("Usuario no disponible.");
    console.error("Usuario no disponible.");    
    throw new Error('Usuario no disponible. Por favor, intente ingresar nuevamente más tarde.'); 
  }

  
  // Obtner listado de mensajes presentes en el topic:'principal.client'
  // loadMessages();
};

//Función que proicesa el mensaje JSON del servidor GoChat. lista de usaurios activos
// Debe ser la función de callcbak para el listado de usuarios activos. Nada más logarse 
// 1. NAda mñas logarse se envia una petición para obtener todos los mensakes en el topic roomlistuser.server
// 2. Se obtiene la lista de usuarios activos del servidor GoChat del topic <<nickname>>.client
// 3. Se actualiza la pantalla de usuarios activos
// 4. Se crea un consumidor para obtener loa usuarios nuevos y los que se dan de baja (funcionalidad no implementada en esta versión
const loadAliveUsers = async () => {
    try {
      const datosCliente = await getClientInformation();
      console.log('Usuarios activos:datosCliente:', datosCliente);
      if ( userChat &&  natsManager) {
        // --123-- Aqui debe llegar el mensaje JSON del servidor GoChat. lista de usaurios activos Del topic ""userChat.nickname.client"""
        requestActiveUsers(natsManager,userChat.roomId,userChat.token, userChat.nickname, datosCliente)
     
        // Parsear la respuesta JSON
        /*const data: ResponseUser = JSON.parse(response);  // --123-- Asegúrar de que la respuesta es un JSON. Tenmemos que deserializr elk objeto NatsMessage
        console.log('Usuarios activos:data :', data );
        console.log('Usuarios activos:data.data:', data.data);
        if (data.status === 'OK' && data.data) {
          // Extraer los nicknames de los usuarios activos
          const nicknames = data.data.map((userlist: AliveUser) => userlist.nickname);  // Aquí accedemos a 'data' y mapeamos los 'nickname'
          console.log('Usuarios activos:', nicknames);
          setAliveUsers(nicknames);  // Establecer el estado con los nicknames
        } else {
          showErrorModal(`Error al obtener usuarios activos: ${data.message}`);
          console.error('Error al obtener usuarios activos:', data.message);  // 'data.message' contiene el mensaje de error
        }*/
      }     
    } catch (err) {
      showErrorModal('Error GRAVE al obtener los usuarios activos: ' + err); 
      console.error('Error al obtener usuarios activos:', err);
      return;
    }
};
 

  // Función para manejar el cambio en el campo de mensaje
  const handleMessageChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setMessageText(e.target.value);
    console.log('Cambio en el mensaje:', e.target.value);
    setIsMessageSendable(e.target.value.trim() !== ''); // Habilitar el botón solo si hay texto
  };

  //Función para salir del GoChat
  const logoutAndRedirect = (event: React.MouseEvent<HTMLButtonElement>) => {
    // Lógica para destruir los objetos Room, User, Messages
    localStorage.removeItem("Room");
    localStorage.removeItem("User");
    localStorage.removeItem("Messages");
    setIsAuthenticated(false);
    // Redirigir al login
    window.location.href = "/"; // Redirige a la página de login
  };
 
  const authenticateUser = async ()=> {
    console.log("Dentro de authenticateUser:")
    try {
      if (nickName && roomId && roomName && token) {
        const user = new User(nickName, 'Alive', roomId, roomName, token);
        setUserChat(user);
        setIsMessageSendable(true);
        setIsAuthenticated(true);
        console.log('Usuario autenticado:', user);
        console.log('Usuario autenticado: IsAuthenticated:', isAuthenticated);
      } else {
        throw new Error('Datos del usuario no válidos');
      }
    } catch (error) {
      setIsAuthenticated(false);
      showErrorModal(`Error setIsAuthenticated: ${error}`);
      console.error(error);
      throw error;
    }
  };
  
  const initializeRoom = async ()  => {
    console.log(
      "Se llama a initializeRoom. Valores actuales:\n" +
      `userChat: ${JSON.stringify(userChat)},\n` + 
      `natsManager.isConnected: ${natsManager?.isConnected},\n` +
      `initialized: ${initialized}`
    );
    if (userChat && natsManager?.isConnected) {
      const roomU = new Room(userChat.roomId, userChat.roomName);
      setRoom(roomU);
      console.log('Sala creada:', roomU);
    }  
  };
   // Detectar presionar la tecla Enter en el input y enviar el mensaje
   const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter' && isMessageSendable) {
      handleSendMessage();
    }
  };
 // UseEffect para crear userChat
useEffect(() => {
  if (nickName && roomId && roomName && token && !isAuthenticated) {
    console.log("Autenticación no realizada, llamando a authenticateUser.");
    authenticateUser();
  } else {
    console.log("Condiciones de autenticación no cumplidas o ya autenticado.");
  }
}, [nickName, roomId, roomName, token, isAuthenticated]);

useEffect(() => {
  console.log("Dentro de useEffect: connectToNats()");
  console.log("isAuthenticated:", isAuthenticated);
  console.log("shouldConnectToNats:", shouldConnectToNats);
  console.log("userChat (nickname):", userChat?.nickname);

  // Verifica si el usuario está autenticado, tiene un nickname válido y el flag para conectar está a true
  if (isAuthenticated && userChat?.nickname && !shouldConnectToNats) {
    console.log("Usuario autenticado y nickname no vacío. Conectando a Nats...");
    connectToNats();
  } else {
    if (!shouldConnectToNats) {
      console.log("No se conecta a Nats, no se cumplen las condiciones.");
    } else {
      console.log("Conexión Nats ya está establecida")
    }
  }
}, [isAuthenticated, userChat, shouldConnectToNats]);


// UseEffect para la lógica de la conexión a Nats, activada cuando 'shouldConnectToNats' cambia
useEffect(() => {
  console.log("Dentro de useEffect: pushCallback.");
  console.log("isAuthenticated:", isAuthenticated);
  console.log("shouldConnectToNats:", shouldConnectToNats);
  console.log("natsManager.isConnected:", natsManager?.isConnected);
  console.log("callbackExecuted:", callbackExecuted);
  
  if (shouldConnectToNats && natsManager && natsManager.isConnected && !callbackExecuted) {
    console.log("Llamando a pushCallback para la conexión.");
    pushCallback();
    setCallbackExecuted(true);  // Marca que la función ya fue ejecutada
    loadAliveUsers ();  // Carga los usuarios conectados
  } else {
    console.log("No se  llama a pushCallback porque no se cumplen las condiciones.");
  }
}, [shouldConnectToNats, callbackExecuted]);  

// UseEffect para inicializar la sala
useEffect(() => {
  console.log("Inicializando la sala...");
  initializeRoom();
}, [initialized]); // Este useEffect depende de 'initialized'

// UseEffect para comenzar updates una vez que userChat, room y nats.connect estén listos
useEffect(() => {
 
  if (userChat && natsManager?.isConnected && !initialized) {
    console.log("Se ejecuta setInitialized para crear el objeto room.");
    if (!initialized) {
      setInitialized(true); // Marca como inicializado para evitar ejecuciones futuras
    }
  } else {
    console.log("No se ejecuta setInitialized porque initialized ya es:", initialized);
  }
}, [userChat, natsManager?.isConnected]);

 

// UseEffect para mostrar el error de conexión si se detecta un problema
useEffect(() => {
  if (connectionError) {
    console.log("Error en la conexión detectado.");
    showErrorModal("Error en la conexión con el servidor. Espere unos instantes y vuelva a logarse.");
  }
}, [connectionError]);

// Desplazar hacia abajo cada vez que cambien los mensajes
useEffect(() => {
  console.log("Desplazándose al final de los mensajes.");
  scrollToBottom();
  setScrollToBottomFlag(false); // Resetear el flag
}, [messages, scrollToBottomFlag]);

  // Asegúrate de que tanto el usuario como el WebSocket y la sala estén listos antes de mostrar el chat
  if (!userChat || !natsManager?.isConnected || !room) {
    return <div>Cargando...</div>; // O cualquier otro indicador de que el chat no está listo
  }
  else{
    return (
      <div className="chat-container">
        <div className="chat-left-column">
          <div className="chat-title">Gochat</div>
          <div className="chat-subtitle">GoChat ZeroMQ//Nats</div>   
           
          <div className="chat-logo-container">
            <div className="chat-logo">
              <img src="/images/logo.webp" alt="Logo" />
            </div>
          </div>
  
           
            <div className="chat-banners">
              <div className="chat-banner-programming">
                <BannerProgramming  
                  titleSlogan="Desarrollos ágiles para tus aplicaciones"
                  subtitleSlogan="Escalabilidad y elasticidad eficientes"
                  imageUrl="/images/pattern.png"
                />
              </div>
            </div>
       
  
          <div className="footer">
            <p>&copy; 2024 José Javier Gutiérrez Gil</p>
            <p className="email-style">&copy; jogugil@gmail.com // jogugi@posgrado.upv.es</p>
          </div>
        </div>
        <div className={`chat-room ${isDarkMode ? 'chat-room-dark' : 'chat-room-light'}`}>
          <div className="chat-content">
            <div className="messages-display">
              <div className="theme-toggle-btn" onClick={toggleTheme}>
                Cambiar Tema
              </div>
              <h3>Mensajes</h3>
              <ul>
                {Array.from(messages.entries()).map(([id, msg], index) => (
                  <li key={id}>
                    <strong>{msg.nickname}</strong>: {msg.message}
                  </li>
                ))}
              </ul>
              <div ref={messagesEndRef} />
            </div>
  
            <div className="input-section">
              {/* Sección para mostrar el nickname */}
              <div className="user-nickname">
                <span className="nickname-value">
                  {userChat && userChat.nickname ? userChat.nickname : "Anónimo"}
                </span>
              </div>
              
              {/* Campo de entrada de mensaje */}
              <input
                type="text"
                value={messageText}
                onChange={handleMessageChange}
                disabled={!isMessageSendable} // Deshabilitar si no es enviable
                placeholder="Escribe tu mensaje..."
                onKeyDown={handleKeyDown} // Detectar presionar Enter
              />
              
              {/* Botón de enviar */}
              <button className="send-btn"
                onClick={handleSendMessage}
                disabled={!isMessageSendable} // Deshabilitar el botón si no es enviable
              >
                Enviar
              </button>
            </div>
          </div>
        </div>
  
        <div className="chat-right-column">
          <div className="chat-metricsbox">
            <Clock />
            <p><strong>Mensajes enviados:</strong> {messages.size}</p>
            <p><strong>Usuarios activos:</strong> {aliveUsers.length}</p>
          </div>
          <div className="chat-active-users-box">
            <div className="chat-active-users-header">
              <h3>Usuarios Activos</h3>
            </div>
            <ul>
              {aliveUsers.map((user, index) => (
                <li key={index}>{user}</li>  
              ))}
            </ul>
          </div>
  
          <div className="chat-banner-cloud">
            <BannerCloud 
              imageUrl="../../public/images/cloudcomm.png"
              titleSlogan="Inteligencia Aplicada en la Nube"
              subtitleSlogan="Soluciones avanzadas en Cloud, Clusters y Serverless para un futuro más eficiente"
            />
          </div>
  
          {showError && (
              <div className="chat-error-message">
                <WErrorMessage
                  message={errorMessage || 'Ocurrió un error desconocido'}
                  showError={showError}
                  isDarkMode={isDarkMode}
                  closeErrorMessage={closeErrorMessage}
                  minimizeErrorMessage={minimizeErrorMessage}
                  restoreErrorMessage={restoreErrorMessage}
                  minimized={minimized}
                  iconType="error"
                />
              </div>
            )}
          <button id="logoutButton" className="logout-btn" onClick={logoutAndRedirect}>Salir</button>
        </div>
      </div>
    );
  }
};
export default Chat;



 