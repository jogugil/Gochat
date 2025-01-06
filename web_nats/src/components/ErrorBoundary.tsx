import React, { Component, ErrorInfo } from 'react';
import { Navigate } from 'react-router-dom';

type Props = {
  children: React.ReactNode;
};

type State = {
  hasError: boolean;
  errorMessage: string;
};

class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      errorMessage: '',
    };
  }

  static getDerivedStateFromError(error: Error) {
    // Solo actualiza el estado si el error proviene de una parte de la app que no está relacionado con el login
    if (error.message.includes("login")) {
      return { hasError: true, errorMessage: 'Ocurrió un error inesperado. Redirigiendo al login...' };
    }
    // Si no es un error relacionado con el login, no actualices el estado
    return { hasError: false, errorMessage: '' };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // Puedes registrar el error para diagnóstico o monitoreo
    console.warn('ErrorBoundary capturó un error:', error, info);
  }

  render() {
    if (this.state.hasError) {
      // Usa <Navigate> para redirigir al login
      return <Navigate to="/" state={{ errorMessage: this.state.errorMessage }} />;
    }

    return this.props.children; // Renderiza los hijos normalmente si no hay error
  }
}

export default ErrorBoundary;
  