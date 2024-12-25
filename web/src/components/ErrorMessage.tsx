import React, { useState } from 'react';
import '../styles/errorMessage.css';

interface WErrorMessageProps {
  message: string;
  showError: boolean;
  isDarkMode: boolean;
  closeErrorMessage: () => void;
  minimizeErrorMessage: () => void;
  restoreErrorMessage: () => void;
  minimized: boolean;
  iconType: "error" | "info"; // Mantén el tipo de icono como string literal
}
function getStackTrace() {
  const error = new Error();
  const stack = error.stack || ''; // Accesses the stack trace
  console.log('Stack trace:', stack);  // Prints the stack trace
}
// ErrorMessage.tsx
export const WErrorMessage = ({
  message,
  showError, isDarkMode,
  closeErrorMessage,
  minimizeErrorMessage,
  restoreErrorMessage,
  minimized,
  iconType,  // Asegúrate de que iconType está definido
}: WErrorMessageProps) => {
  
  console.log("Dentro de WErrorMessage con mensaje:",message);
  console.log("Dentro de WErrorMessage con showError:",showError);
  console.log("Dentro de WErrorMessage con isDarkMode:",isDarkMode);
  console.log("Dentro de WErrorMessage con minimized:",minimized);
  return (
    showError && (
      <div
        className={`w-message-window ${
          isDarkMode ? "dark" : "light"
        } ${iconType === "error" ? "error" : "info"}`}
      >
        <div className={`w-header ${isDarkMode ? "dark" : "light"}`}>
          <span className={`w-icon ${iconType}`}>
            {iconType === "error" ? (
              <i className="fas fa-times-circle"></i>
            ) : (
              <i className="fas fa-exclamation-triangle"></i>
            )}
          </span>
          <span className="w-title">
            {iconType === "error" ? "Error" : "Información"}
          </span>
          <div className="w-controls">
            <button
              onClick={minimized ? restoreErrorMessage : minimizeErrorMessage}
              title={minimized ? "Maximizar" : "Minimizar"}
            >
              {minimized ? "⬜" : "_"}
            </button>
            <button onClick={closeErrorMessage} title="Cerrar">
              X
            </button>
          </div>
        </div>
        {!minimized && (
          <div className="w-content">
            <p>{message}</p>
            <button
              className={`w-ok-button ${isDarkMode ? "dark" : "light"}`}
              onClick={closeErrorMessage}
            >
              Ok
            </button>
          </div>
        )}
      </div>
    )
  );
};