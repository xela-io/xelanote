export interface PendingTaskEvent {
  noteId: string;
  taskText: string;
  taskIndex: number;
  eventType: 'completed' | 'reopened';
}

export interface TaskEventQueue {
  add: (event: PendingTaskEvent) => void;
  getForNote: (noteId: string) => PendingTaskEvent[];
  clearForNote: (noteId: string) => void;
}

export function createTaskEventQueue(storageKey = 'pendingTaskEvents'): TaskEventQueue {
  let pendingTaskEvents: PendingTaskEvent[] = [];

  if (typeof window !== 'undefined') {
    try {
      const stored = sessionStorage.getItem(storageKey);
      if (stored) {
        const parsed = parsePendingTaskEvents(stored);
        if (parsed) {
          pendingTaskEvents = parsed;
        }
      }
    } catch {
      /* ignore */
    }
  }

  const persist = () => {
    if (typeof window === 'undefined') return;
    try {
      sessionStorage.setItem(storageKey, JSON.stringify(pendingTaskEvents));
    } catch {
      /* ignore */
    }
  };

  return {
    add: (event) => {
      pendingTaskEvents.push(event);
      persist();
    },
    getForNote: (noteId) => pendingTaskEvents.filter((event) => event.noteId === noteId),
    clearForNote: (noteId) => {
      pendingTaskEvents = pendingTaskEvents.filter((event) => event.noteId !== noteId);
      persist();
    },
  };
}

function parsePendingTaskEvents(raw: string): PendingTaskEvent[] | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return null;
  }

  if (!Array.isArray(parsed)) return null;
  if (
    !parsed.every(
      (entry) =>
        !!entry &&
        typeof entry === 'object' &&
        typeof (entry as { noteId?: unknown }).noteId === 'string' &&
        typeof (entry as { taskText?: unknown }).taskText === 'string' &&
        typeof (entry as { taskIndex?: unknown }).taskIndex === 'number' &&
        ((entry as { eventType?: unknown }).eventType === 'completed' ||
          (entry as { eventType?: unknown }).eventType === 'reopened')
    )
  ) {
    return null;
  }

  return parsed as PendingTaskEvent[];
}
