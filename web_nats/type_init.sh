#!/bin/bash

# Limpiar la cache de npm (opcional, solo si es necesario)
npm cache clean --force

# Borrar la carpeta node_modules y el archivo package-lock.json si existen
rm -rf node_modules package-lock.json

# Inicializar el proyecto con npm (si no lo has hecho ya)
npm init -y

# Instalar las dependencias principales necesarias para el proyecto
npm install react react-dom react-router-dom axios jsonwebtoken uuid @fortawesome/fontawesome-free
install puppeteer 
# Instalar dependencias de desarrollo (con versiones específicas)
npm install --save-dev \
  typescript@latest --save-dev\
  vite@latest --save-dev \
  @vitejs/plugin-react@latest --save-dev \
  @types/node@^22.10.2 \
  @types/uuid@^10.0.0 \
  @types/react@^18.0.0 \
  @types/react-dom@^18.0.0 \
  @types/react-router-dom@^5.3.3 \
  eslint@^8.57.1 \
  eslint-plugin-prettier@^5.2.1 \
  eslint-config-prettier@^8.10.0 \
  eslint-plugin-react@^7.32.0 \
  eslint-plugin-react-hooks@^4.0.0 \
  prettier@^3.0.0 \
  dotenv@^16.4.7
npm install nats
# Actualizar todos los paquetes obsoletos (de forma controlada)
npm outdated
npm update

# Corregir vulnerabilidades encontradas en las dependencias
npm audit fix --force

# Crear o actualizar tsconfig.json
echo '{
  "compilerOptions": {
    "allowJs": true,
    "checkJs": false,
    "strict": true,
    "lib": ["dom", "esnext"],
    "module": "ESNext",
    "moduleResolution": "Node",
    "jsx": "react-jsx",
    "baseUrl": "./src",
    "outDir": "./dist",
    "target": "esnext",
    "esModuleInterop": true,
    "skipLibCheck": true,
    "typeRoots": ["node_modules/@types"],
    "types": ["vite/client", "node"]
  },
  "include": ["src/**/*", "vite.config.ts"],
  "exclude": ["node_modules"]
}' > tsconfig.json

# Crear el archivo .env
echo 'VITE_API_URL=http:/localhost:8081
VITE_API=localhost
VITE_PORT=8081' > .env

# Instalar dependencias para asegurarse de que todo está actualizado
npm install
