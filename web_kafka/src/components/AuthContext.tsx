import React, { createContext, useContext, useState, ReactNode, useMemo, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';  // Usar useNavigate en lugar de useHistory

interface AuthContextType {
  token: string;
  nickName: string;
  roomId: string;
  roomName: string;
  setToken: (token: string) => void;
  setNickName: (nickName: string) => void;
  setRoomId: (roomId: string) => void;
  setRoomName: (roomName: string) => void;
  error: string | null;
  setError: (error: string | null) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [token, setToken] = useState('');
  const [nickName, setNickName] = useState('');
  const [roomId, setRoomId] = useState('');
  const [roomName, setRoomName] = useState('');
  const [error, setError] = useState<string | null>(null);  // Estado para manejar errores
  const navigate = useNavigate();

// Añadir console.log para inspeccionar los valores del estado
useEffect(() => {
  const storedToken = localStorage.getItem('token');
  if (storedToken) {
    setToken(storedToken);
    setNickName(localStorage.getItem('nickName') || '');
    setRoomId(localStorage.getItem('roomId') || '');
    setRoomName(localStorage.getItem('roomName') || '');
  }
}, []);
 
useEffect(() => {
  console.log("Verificando error:", error);
  if (error) {
    // Si hay un error, redirigir al login con el mensaje de error.
    console.log("Redirigiendo por error:", error);
    navigate(`/?errorMessage=${encodeURIComponent(error)}`);
  } else {
    console.log("No existe error alguno")
  }
}, [token, error, navigate]);

 

// Memorizamos el valor del contexto para evitar re-renderizados innecesarios
  
const value = useMemo(() => ({
    token,
    nickName,
    roomId,
    roomName,
    setToken,
    setNickName,
    setRoomId,
    setRoomName,
    error,
    setError,  // Incluir la función setError
  }), [token, nickName, roomId, roomName, error]);

  // Agregar console.log aquí para ver los valores en la consola
  console.log('AuthContext State:', { token, nickName, roomId, roomName, error });

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  console.log('useAuth State context:', { context });

  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }

  return context;
};
