// src/bot/ChatBotClient.ts
import { useEffect, useState } from 'react';
import React from 'react';

// Configuración del WebSocket
const SOCKET_URL = 'ws://localhost:8080/';  // Cambia la URL según tu configuración

const ChatBotClient = () => {
  const [messages, setMessages] = useState<string[]>([]); // Mensajes del chat
  const [socket, setSocket] = useState<WebSocket | null>(null);

  useEffect(() => {
    // Crear conexión WebSocket al cliente de chat
    const ws = new WebSocket(SOCKET_URL);
    setSocket(ws);

    ws.onopen = () => {
      console.log("Conectado al servidor de chat");
    };

    ws.onmessage = (event) => {
      const message = event.data;
      // Procesar los mensajes entrantes
      handleIncomingMessage(message);
    };

    ws.onclose = () => {
      console.log("Conexión cerrada");
    };

    return () => {
      if (ws) {
        ws.close();
      }
    };
  }, []);

  const handleIncomingMessage = (message: string) => {
    // Aquí se puede agregar lógica para procesar el mensaje, como responder si se menciona al bot
    setMessages((prevMessages) => [...prevMessages, message]);

    // Responder si el mensaje menciona al bot
    if (message.includes('@bot')) {
      sendMessage("¡Hola! Soy el bot IA, ¿en qué puedo ayudarte?");
    }
  };

  const sendMessage = (message: string) => {
    if (socket) {
      socket.send(message); // Enviar un mensaje al chat
    }
  };

  // Simula una conversación del bot
  const simulateBotActivity = () => {
    setInterval(() => {
      sendMessage("Bot: ¿Estás interesado en aprender sobre algo?");
    }, 10000); // El bot envía un mensaje cada 10 segundos
  };

  return (
    <div>
      <h2>ChatBot Interactivo</h2>
      <div>
        {messages.map((msg, index) => (
          <div key={index}>{msg}</div>
        ))}
      </div>
      <button onClick={simulateBotActivity}>Iniciar Simulación de Actividad</button>
    </div>
  );
};

export default ChatBotClient;
