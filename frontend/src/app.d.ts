/// <reference types="@sveltejs/kit" />
/// <reference types="vite-plugin-pwa/client" />

declare global {
  namespace App {
    // interface Error {}
    // interface Locals {}
    // interface PageData {}
    // interface PageState {}
    // interface Platform {}
  }

  interface Window {
    turnstile?: {
      render: (
        container: HTMLElement,
        options: {
          sitekey: string;
          callback: (token: string) => void;
          'expired-callback'?: () => void;
          'error-callback'?: () => void;
          theme?: 'light' | 'dark' | 'auto';
        }
      ) => string;
      reset: (widgetId: string) => void;
      remove: (widgetId: string) => void;
    };
    onTurnstileLoaded?: () => void;
  }

  interface DocumentEventMap {
    'spell-check-replace': CustomEvent<{ from: number; to: number; replacement: string }>;
  }
}

declare module '*.css' {
  const content: string;
  export default content;
}

declare module 'katex/dist/katex.min.css';

export {};
