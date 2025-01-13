import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';
import { AuthProvider } from './components/AuthContext';

 
// Configuración global de capturadores de errores
window.onerror = (message, source, lineno, colno, error) => {
  console.error('Error global capturado:', { message, source, lineno, colno, error });
  // Redirigir a la página de login con el mensaje de error
  //const errorm  = typeof message === 'string' ? message : 'Error inesperado';
  //window.location.href = `/?errorMessage=${'Error desconocido. Intentelo denuevo pasado unos instanes. Avise al administrador' }`;
};

window.onunhandledrejection = (event) => {
  console.error('Promesa no manejada:', event.reason);
  // Redirigir a la página de login con el mensaje de error
  //window.location.href = `/?errorMessage=${ 'Error inesperado en promesa. Intentelo denuevo pasado unos instanes. Avise al administrador'}`;
};
const root = ReactDOM.createRoot(document.getElementById('root') as HTMLElement);
root.render(
  <BrowserRouter>
  <AuthProvider>  
    <App />
  </AuthProvider>
</BrowserRouter>
);