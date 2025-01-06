/// <reference types="vite/client" />

interface ImportMetaEnv {
  [key: string]: any;
  readonly VITE_API_URL: string;
  readonly VITE_IP_SERVER_GOCHAT: string;
  readonly VITE_PORT_SERVER_GOCHAT: string;
  readonly VITE_POLLING_TIMEOUT: string;
  readonly VITE_RECONNECT_WEBSOCKET: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}