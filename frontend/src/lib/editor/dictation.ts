/**
 * Voice dictation engine — Web Speech API (browser-native) + MediaRecorder (server/Whisper).
 */

// ── Web Speech API type declarations (not in standard TypeScript lib) ─

interface SpeechRecognitionEventResult {
  readonly isFinal: boolean;
  readonly length: number;
  [index: number]: { transcript: string; confidence: number };
}

interface SpeechRecognitionEventMap {
  readonly resultIndex: number;
  readonly results: SpeechRecognitionEventResult[];
}

interface SpeechRecognitionErrorMap {
  readonly error: string;
  readonly message: string;
}

interface SpeechRecognitionInstance extends EventTarget {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  onresult: ((event: SpeechRecognitionEventMap) => void) | null;
  onend: (() => void) | null;
  onerror: ((event: SpeechRecognitionErrorMap) => void) | null;
  start(): void;
  stop(): void;
  abort(): void;
}

// ── Types ────────────────────────────────────────────────────────────

export type DictationState = 'idle' | 'listening' | 'recording' | 'processing';

export interface BrowserDictation {
  start(): void;
  stop(): void;
  destroy(): void;
}

export interface ServerDictation {
  start(): Promise<void>;
  stop(): Promise<Blob>;
  destroy(): void;
}

// ── Feature Detection ────────────────────────────────────────────────

export function isBrowserSpeechSupported(): boolean {
  return !!(
    (window as unknown as Record<string, unknown>).SpeechRecognition ||
    (window as unknown as Record<string, unknown>).webkitSpeechRecognition
  );
}

export function isMediaRecorderSupported(): boolean {
  return typeof MediaRecorder !== 'undefined';
}

/**
 * Pick the best supported audio MIME for MediaRecorder.
 * Safari/iOS doesn't support WebM — fall back to mp4.
 */
export function getBestAudioMime(): string | null {
  if (!isMediaRecorderSupported()) return null;
  const candidates = ['audio/webm;codecs=opus', 'audio/webm', 'audio/mp4', 'audio/ogg;codecs=opus'];
  for (const mime of candidates) {
    if (MediaRecorder.isTypeSupported(mime)) return mime;
  }
  return null;
}

// ── Browser Speech API ───────────────────────────────────────────────

const SpeechRecognitionCtor =
  (window as unknown as Record<string, unknown>).SpeechRecognition ||
  (window as unknown as Record<string, unknown>).webkitSpeechRecognition;

export function createBrowserDictation(
  onResult: (text: string, isFinal: boolean) => void,
  onStateChange: (state: DictationState) => void,
  onError: (message: string) => void
): BrowserDictation {
  if (!SpeechRecognitionCtor) {
    throw new Error('SpeechRecognition not supported');
  }

  const recognition = new (SpeechRecognitionCtor as new () => SpeechRecognitionInstance)();
  recognition.continuous = true;
  recognition.interimResults = true;
  recognition.lang = ''; // auto-detect

  let intentionalStop = false;

  recognition.onresult = (event: SpeechRecognitionEventMap) => {
    for (let i = event.resultIndex; i < event.results.length; i++) {
      const result = event.results[i];
      onResult(result[0].transcript, result.isFinal);
    }
  };

  recognition.onend = () => {
    if (!intentionalStop) {
      // Web Speech API stops after silence — auto-restart
      try {
        recognition.start();
      } catch {
        onStateChange('idle');
      }
    } else {
      onStateChange('idle');
    }
  };

  recognition.onerror = (event: SpeechRecognitionErrorMap) => {
    if (event.error === 'not-allowed') {
      onError('microphone_denied');
      intentionalStop = true;
      onStateChange('idle');
    } else if (event.error === 'no-speech') {
      // Ignore — auto-restart will handle it
    } else if (event.error === 'aborted') {
      // Intentional abort — ignore
    } else {
      onError(event.error);
    }
  };

  return {
    start() {
      intentionalStop = false;
      recognition.start();
      onStateChange('listening');
    },
    stop() {
      intentionalStop = true;
      recognition.stop();
    },
    destroy() {
      intentionalStop = true;
      recognition.abort();
    },
  };
}

// ── Server Dictation (MediaRecorder → Whisper) ───────────────────────

export function createServerDictation(
  onStateChange: (state: DictationState) => void,
  onError: (message: string) => void
): ServerDictation {
  let mediaRecorder: MediaRecorder | null = null;
  let stream: MediaStream | null = null;
  let chunks: Blob[] = [];
  const mimeType = getBestAudioMime();

  return {
    async start() {
      if (!mimeType) {
        onError('mediarecorder_unsupported');
        return;
      }
      try {
        stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      } catch {
        onError('microphone_denied');
        return;
      }
      chunks = [];
      mediaRecorder = new MediaRecorder(stream, { mimeType });
      mediaRecorder.ondataavailable = (e) => {
        if (e.data.size > 0) chunks.push(e.data);
      };
      mediaRecorder.start(250); // collect every 250ms
      onStateChange('recording');
    },

    async stop(): Promise<Blob> {
      return new Promise((resolve) => {
        if (!mediaRecorder || mediaRecorder.state === 'inactive') {
          resolve(new Blob([], { type: mimeType ?? 'audio/webm' }));
          return;
        }
        mediaRecorder.onstop = () => {
          // Release microphone
          stream?.getTracks().forEach((t) => t.stop());
          stream = null;
          const blob = new Blob(chunks, { type: mimeType ?? 'audio/webm' });
          chunks = [];
          resolve(blob);
        };
        mediaRecorder.stop();
      });
    },

    destroy() {
      if (mediaRecorder && mediaRecorder.state !== 'inactive') {
        mediaRecorder.stop();
      }
      stream?.getTracks().forEach((t) => t.stop());
      stream = null;
      mediaRecorder = null;
      chunks = [];
    },
  };
}
