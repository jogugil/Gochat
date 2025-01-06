import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from './AuthContext';  // Asegúrate de importar el contexto correctamente

const AuthRedirect = () => {
  const { token, error } = useAuth(); // Obtenemos el token y el error desde el contexto
  const navigate = useNavigate();

  useEffect(() => {
    console.log(" AuthRedirect Redirigiendo por error:", error);
    if (error) {
      // Si hay un error y no hay token, redirigir al login con el mensaje de error
      console.log("Redirigiendo por error:", error);
      navigate(`/?errorMessage=${encodeURIComponent(error)}`);
    }  
  }, [token, error, navigate]); // Dependencias para que se ejecute cuando cambien

  return <div>Verificando autenticación...</div>; // Puedes mostrar un mensaje mientras verificas
};

export default AuthRedirect;
