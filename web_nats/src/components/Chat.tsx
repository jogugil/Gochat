import React, { useState, useEffect, useContext, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { getMessages, sendMessage, getAliveUsers } from '../api/api';
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
import {NatsStreamManager} from '../comm/WebNatsManager';
const [currentTime, setCurrentTime] = useState<string>('');

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

  //WebSocket
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const apiIP          = import.meta.env.VITE_IP_SERVER_GOCHAT;
  const apiPORT        = import.meta.env.VITE_PORT_SERVER_GOCHAT;
  const socketUrl      = `ws://${apiIP}:${apiPORT}/ws`;
  // Referencias para las instancias de WebSocketManager
 
  const [isUserSocketConnected, setIsUserSocketConnected] = useState(false);
  const [isMessageSocketConnected, setIsMessageSocketConnected] = useState(false);

  const [isMessageSendable, setIsMessageSendable] = useState(false); // Para habilitar/deshabilitar input y botón
  const [messages, setMessages]         = useState<Message[]> ([]);        // Estado para los mensajes
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
  const [startPolling, setStartPolling]       = useState<boolean>(false);
  const [showError, setShowError]             = useState(false);
  const [errorMessage, setErrorMessage]       = useState('');
  const [isErrorActive, setIsErrorActive]     = useState(false);
  const intervalIdRef = useRef<number | null>(null);
  
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

          const response = await sendMessage(userChat.nickname, userChat.token, room.roomId, room.roomName, messageText);
          setMessageText(''); // Limpiamos el input
          // Despues de enviar un mensaje actualizamos la lsita de mensajes para mostrar y de usuarios
          updateData();
        }
      } catch (err) {
        showErrorModal('No se pudo enviar el mensaje. Intente de nuevo.');
        console.error("Error al enviar el mensaje:", err);        
        throw new Error('El servicio de chat no está disponible. Disculpe las molestias. Por favor, intente ingresar nuevamente más tarde.');
 
      }
   
};

// Función para realizar ambas tareas: Petición listado de mensajes y listado de usuarios
const updateData = async () => {  // Añadir async si loadMessages y loadAliveUsers son asincrónicas
    try {
      console.log("Cargando mensajes del chat...");
      await loadMessages();  // Asumimos que loadMessages es asincrónica
  
      console.log("Cargando usuarios activos...");
      await loadAliveUsers();  // Asumimos que loadAliveUsers es asincrónica
    } catch (error) {
      showErrorModal(`Error durante la actualización periódica: ${error}`);
      console.error("Error durante la actualización periódica:", error);
    }
  };
  
// Función que realiza la actualización periódica
const startPeriodicUpdates = ( ) => {
  console.log("startPeriodicUpdates = ( ) => {"); 
  console.log("userChat:", userChat);
 

  // Verificar que el usuario esté disponible
  if (!userChat) {
    showErrorModal("Usuario no disponible.");
    console.error("Usuario no disponible.");    
    throw new Error('Usuario no disponible. Por favor, intente ingresar nuevamente más tarde.'); 
  }

  // Ejecutar inmediatamente antes de iniciar el intervalo
  updateData();

  // Configurar la actualización periódica
  if (intervalIdRef.current) {
    clearInterval(intervalIdRef.current);  // Limpiar cualquier intervalo previo
  }
  // Configurar la actualización periódica
  intervalIdRef.current = setInterval(() => {
    console.log("Intervalo ejecutado");
    updateData();
  }, timeout) as unknown as number;  // Forzar tipo como `number`
};

//Función que proicesa el mensaje JSON del servidor GoChat. lista de usaurios activos
const loadAliveUsers = async () => {
    try {
      const datosCliente = await getClientInformation();
      console.log('Usuarios activos:datosCliente:', datosCliente);
      if ( userChat &&  userSocketRef  &&  userSocketRef.current?.isConnected) {
        const response = await getAliveUsers ( userSocketRef.current   , userChat.nickname, userChat.token, userChat.roomId, datosCliente );
      
        // Parsear la respuesta JSON
        const data: ResponseUser = JSON.parse(response);  // Asegúrate de que la respuesta es un JSON
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
        }
      }     
    } catch (err) {
      showErrorModal('Error GRAVE al obtener los usuarios activos: ' + err); 
      console.error('Error al obtener usuarios activos:', err);
      return;
    }
};
 
  // Obtener mensajes históricos al cargar la página
  const loadMessages = async () => {
    // Generar un UUID vacío para el primer request
    const emptyUUID: UUID = "00000000-0000-0000-0000-000000000000";
  
    try {
      console.log('loadMessages: userChat:', userChat);
      console.log('loadMessages: room:', room);
      
      if (!room) {
        // Mostrar una ventana emergente de error       
        console.error('Error, no se creó el objeto room');
        throw new Error('El servicio de chat no está disponible. Disculpe las molestias. Por favor, intente ingresar nuevamente más tarde.');
      }
  
      const messageID = room.lastIDMessageId || emptyUUID;
      console.log('loadMessages:messageID:', messageID);
  
      // Hacer la llamada a la API para obtener los mensajes
      const datosCliente = await getClientInformation();
      console.log('loadMessages:datosCliente:', datosCliente);
  
      if (userChat && messageSocketRef.current && messageSocketRef.current.isConnected) {
        console.log('loadMessages: datosCliente:', datosCliente);
     
        // Obtener los mensajes pasando el WebSocketManager real
        const messageList = await getMessages(
          messageSocketRef.current,  // Accede a la instancia de WebSocketManager
          userChat.nickname,
          userChat.token,
          userChat.roomId,
          userChat.roomName,
          messageID,
          datosCliente
        );
        
        // Deserializar el JSON en un objeto Response
        const response: Response = JSON.parse(messageList);
  
        // Acceder a los mensajes
        console.log('loadMessages:messageList:', response);
  
        // Verificar si el status es 'NOK' y manejar el error adecuadamente
        if (response.status === 'NOK') {
          // Mostrar mensaje de error si el estado es 'NOK'
          showErrorModal(response.message || 'Error desconocido al cargar los mensajes');
          console.error('Error al cargar los mensajes:', response.message);
          return;  // Salir de la función sin actualizar los mensajes
        } else {
          // Si 'response.data' está indefinido, es que no hay mensajes
          if (!response.data) {
            console.log('No se encontraron mensajes');
           // setMessages([]);  // Establecer mensajes como un arreglo vacío
            return;  // Salir de la función
          }
  
          // Si la respuesta contiene mensajes, crear la lista de mensajes
          const msgList: MessageResponse[] = response.data.map((msg: MessageResponse) => ({
            messageid: msg.messageid,      // ID del mensaje
            nickname: msg.nickname,        // Nombre del usuario
            messagetext: msg.messagetext   // Contenido del mensaje
          }));
  
          console.log('Mensajes activos:', msgList);
  
          // Ahora pasamos la lista de objetos al método updateMessages
          room.updateMessages(msgList);
          // Actualizar el estado de los mensajes
          setMessages(room.messageList);
          setScrollToBottomFlag(true);
          console.log('Mensajes cargados:', room.messageList);
        }
      } else {
        showErrorModal('No se pudo conectar al servicio de chat.');  
        console.error('Socket no está conectado');
      }
    } catch (err) {
      showErrorModal('El Servicio de Chat está temporalmente cerrado. Intente logarse más tarde');
      console.error('Error al cargar los mensajes:', err);
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
    try {
      if (nickName && roomId && roomName && token) {
        const user = new User(nickName, 'Alive', roomId, roomName, token);
        setUserChat(user);
        setIsMessageSendable(true);
        setIsAuthenticated(true);
        console.log('Usuario autenticado:', user);
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
      `messageSocket.isConnected: ${messageSocketRef.current?.isConnected},\n` +
      `userSocket.isConnected: ${userSocketRef.current?.isConnected},\n` +
      `initialized: ${initialized}`
    );
    if (userChat && messageSocketRef.current?.isConnected && userSocketRef.current?.isConnected) {
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
      authenticateUser();
    }
  }, [nickName, roomId, roomName, token, isAuthenticated]);

  useEffect(() => {
    initializeRoom();
  }, [initialized]); // Este useEffect depende de initialized

  useEffect(() => {
    const connectWebSocket = async () => {
      try {
        // Instancia WebSocketManager con un callback para manejar el error
        messageSocketRef.current = new WebSocketManager(socketUrl, 'messagesSocket', (error: string) => {
          setConnectionError(error);  // Guardamos el error de conexión en el estado
        });
        userSocketRef.current = new WebSocketManager(socketUrl, 'usersSocket', (error: string) => {
          setConnectionError(error);
        });

        // Pasar el callback de onOpen desde React
        userSocketRef.current.setOnOpenCallback(() => {
          console.log('User socket connected');
          setIsUserSocketConnected(true); // Marca como conectado
        });

        messageSocketRef.current.setOnOpenCallback(() => {
          console.log('Message socket connected');
          setIsMessageSocketConnected(true); // Marca como conectado
        });

      } catch (error: unknown) {
        if (error instanceof Error) {
          setConnectionError(error.message);
        } else {
          setConnectionError('Error desconocido al conectar al WebSocket');
        }
      }
    };
  
    connectWebSocket();
  
    return () => {
      if (userSocketRef.current && typeof userSocketRef.current.close === 'function') {
        userSocketRef.current.close();
      }
      if (messageSocketRef.current && typeof messageSocketRef.current.close === 'function') {
        messageSocketRef.current.close();
      }
    };
  }, [socketUrl])

  // Este useEffect comienza los updates periódicos una vez que userChat, room y WebSocket estén listos
  useEffect(() => {
    // Solo ejecutar startPeriodicUpdates cuando userChat, room, y el WebSocket estén listos
    if (userChat && userSocketRef.current?.isConnected && messageSocketRef.current?.isConnected && !initialized) {
      console.log("Se llaman setInitialized para poder crear el objeto room");
 
      if (!initialized) setInitialized(true);  // Marca como inicializado para evitar ejecuciones futuras     

    }  else {
      console.log("No se llamo a setInitialized porque initialized:",initialized);
    }
  }, [userChat, userSocketRef.current?.isConnected, messageSocketRef.current?.isConnected])

  useEffect(() => {
        if (userChat && userSocketRef.current?.isConnected && messageSocketRef.current?.isConnected && initialized) {
           if (!startPolling) {
              console.log("Se llaman los updates periódicos del chat");
              startPeriodicUpdates( ); // Activa los mensajes periódicos
              setStartPolling (true); 
          }  else {
            console.log(" No pudo llamarse a startPolling...");
          }
          
        }
          
        // Limpiar el intervalo cuando el componente se desmonte
         // Limpiar intervalo cuando el componente se desmonta
        return () => {
          if (intervalIdRef.current) {
              clearInterval(intervalIdRef.current);
          }
        };
  }, [room?._roomid]);

  useEffect(() => {
    if (connectionError) {
      showErrorModal ("Error en la conexión con el servidor. Espere unos isntantes.");
    }
  }, [connectionError]);
  
  // Desplázate al final cada vez que cambien los mensajes
  useEffect(() => {
    scrollToBottom();
    setScrollToBottomFlag(false); // Resetear el flag
  }, [messages,scrollToBottomFlag]);


  // Asegúrate de que tanto el usuario como el WebSocket y la sala estén listos antes de mostrar el chat
  if (!userChat || !userSocketRef.current?.isConnected || !messageSocketRef.current?.isConnected || !room) {
    return <div>Cargando...</div>; // O cualquier otro indicador de que el chat no está listo
  }
  else{
    return (
      <div className="chat-container">
        <div className="chat-left-column">
          <div className="chat-title">ChatSphere</div>
          <div className="chat-subtitle">GoChat ZeroMQ</div>
           
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
                {messages.map((msg, index) => (
                  <li key={index}>
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
            <p><strong>Mensajes enviados:</strong> {messages.length}</p>
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



 