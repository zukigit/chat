// Runtime environment configuration from Docker
interface RuntimeEnv {
  API_URL: string;
}

declare global {
  interface Window {
    ENV?: RuntimeEnv;
  }
}

export {};
