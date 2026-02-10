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
      if (stored) pendingTaskEvents = JSON.parse(stored);
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
