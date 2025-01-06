import React, { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { login } from '../api/api';
import { useAuth } from './AuthContext';
import { LoginResponse } from '../types/typesComm';
import '../styles/login.css';

const Login: React.FC = () => {
  const [nicknamelogin, setNickname] = useState<string>('');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [showError, setShowError] = useState<boolean>(false);
  const [minimized, setMinimized] = useState<boolean>(false);

  const navigate = useNavigate();
  const { setNickName: setAuthNickname, setToken, setRoomId, setRoomName } = useAuth();
  const location = useLocation();
 // Capturar el mensaje de error desde la URL (si existe)
 useEffect(() => {
  const queryParams = new URLSearchParams(location.search);
  const errorMessageFromUrl = queryParams.get('errorMessage');

  // Si hay un errorMessage en la URL, mostrarlo
  if (errorMessageFromUrl) {
    setErrorMessage(errorMessageFromUrl);
    setShowError(true);
  } else {
    // Si no hay errorMessage en la URL, no mostrar el error
    setShowError(false);
    setErrorMessage(null);
  }
}, [location]);
  // Obtener el mensaje de error desde el estado de la navegación
  useEffect(() => {
    if (location.state?.errorMessage) {
      setErrorMessage(location.state.errorMessage);
      setShowError(true);
    }
  }, [location])

  const handleLogin = async () => {
    console.log("Intentando iniciar sesión...");
    setShowError(false); // Ocultar el error al intentar de nuevo
    setNickname(''); // Limpiar el nickname al intentar iniciar sesión nuevamente
    try {
      const response: LoginResponse = await login(nicknamelogin);
      console.log("Login --> response :",response);

      if (response.status === 'nickname_taken') {
        setErrorMessage('El nickname ya está en uso. Intenta con otro.');
        setShowError(true);
        console.log("Mostrar error: Nickname ya en uso");
        return; // No hacer redirect aún
      }

      if (response.status === 'nok') {
        setErrorMessage(response.message || 'Error desconocido al iniciar sesión.');
        setShowError(true);
        console.log("Mostrar error: Error desconocido");
        return; // No hacer redirect aún
      }

      if (response.status === 'ok') {
        setAuthNickname(nicknamelogin);
        setToken(response.token);
        setRoomId(response.roomid); // Establece roomid
        setRoomName(response.roomname); // Establece roomname
        navigate('/chat');  // Redirigir solo si el login es exitoso
      }
    } catch (error) {
      setErrorMessage('Hubo un error al intentar iniciar sesión. Intenta de nuevo.');
      setShowError(true);
      console.log("Mostrar error: Error al iniciar sesión");
    }
  };

  const closeErrorMessage = () => setShowError(false);
  const minimizeErrorMessage = () => setMinimized(true);
  const restoreErrorMessage = () => setMinimized(false);

  return (
    <div className="login-container">
      <div className="left-column">
        <div className="logo">ChatSphere</div>
        <div className="subtitle">GoChat ZeroMQ</div>
        <div className="footer">
          <p>&copy; 2024 SmartIAService's</p>
          <p>{new Date().toLocaleDateString()}</p>
        </div>
      </div>

      <div className="right-column">
        <div className="login-box">
          <h2>Iniciar sesión</h2>
          <input
            type="text"
            placeholder="Introduce tu nickname"
            value={nicknamelogin}
            onChange={(e) => setNickname(e.target.value)}
            className="nickname-input"
            disabled={showError}  // Deshabilita el formulario si hay un error
          />
          <button onClick={handleLogin} disabled={showError}>Iniciar sesión</button>  {/* Deshabilita el botón si hay un error */}
        </div>

        {showError && (
          <div className="message-window">
            <div className="header">
              <span className="title">Error</span>
              <div className="controls">
                {minimized ? (
                  <button onClick={restoreErrorMessage} title="Maximizar">
                    ⬜
                  </button>
                ) : (
                  <button onClick={minimizeErrorMessage} title="Minimizar">
                    _
                  </button>
                )}
                <button onClick={closeErrorMessage} title="Cerrar">
                  X
                </button>
              </div>
            </div>
            {!minimized && (
              <div className="content">
                <p>{errorMessage}</p>
                <button className="ok-button" onClick={closeErrorMessage}>Ok</button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default Login;
