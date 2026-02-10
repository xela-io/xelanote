export interface LayoutInteractionDeps {
  isAuthenticated: () => boolean;
  toggleQuickSwitcher: () => void;
  toggleMarkdownGuideDropdown: () => void;
  canUndo: () => boolean;
  canRedo: () => boolean;
  undo: () => void;
  redo: () => void;
  saveNote: () => void;
  goto: (path: string) => void;
  graphEnabled: () => boolean;
  recordActivity: () => void;
  setShowUnlockModal: (value: boolean) => void;
  getCurrentNote: () => { content_encrypted?: boolean } | null;
  isEncryptionUnlocked: () => boolean;
}

export function createLayoutInteractions(deps: LayoutInteractionDeps) {
  const handleKeydown = (e: KeyboardEvent) => {
    if (!deps.isAuthenticated()) return;

    if ((e.ctrlKey || e.metaKey) && e.key === 'p') {
      e.preventDefault();
      deps.toggleQuickSwitcher();
    }

    if ((e.ctrlKey || e.metaKey) && e.key === 'g' && deps.graphEnabled()) {
      e.preventDefault();
      deps.goto('/graph');
    }

    if ((e.ctrlKey || e.metaKey) && e.key === '/') {
      e.preventDefault();
      deps.toggleMarkdownGuideDropdown();
    }

    if ((e.ctrlKey || e.metaKey) && e.key === 's') {
      e.preventDefault();
      deps.saveNote();
    }

    if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey) {
      if (deps.canUndo()) {
        e.preventDefault();
        deps.undo();
      }
    }

    if (
      ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'z') ||
      ((e.ctrlKey || e.metaKey) && e.key === 'y')
    ) {
      if (deps.canRedo()) {
        e.preventDefault();
        deps.redo();
      }
    }
  };

  const handleActivity = () => {
    deps.recordActivity();

    const currentNote = deps.getCurrentNote();
    if (currentNote?.content_encrypted && !deps.isEncryptionUnlocked()) {
      deps.setShowUnlockModal(true);
    }
  };

  return { handleKeydown, handleActivity };
}
