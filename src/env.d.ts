/// <reference types="vite/client" />
interface Window {
  mygo?: {
    call: <T = unknown>(method: string, ...args: unknown[]) => Promise<T>;
  };
}
