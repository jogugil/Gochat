export const setToken = (token: string) => {
  localStorage.setItem('token', token); // Guarda el token en localStorage
};

export const getToken = () => {
  return localStorage.getItem('token'); // Obtiene el token del localStorage
};

export const clearToken = () => {
  localStorage.removeItem('token'); // Elimina el token
};

  