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
  closeCurrentTab: () => void;
  nextTab: () => string | null;
  prevTab: () => string | null;
  isDesktop: () => boolean;
  getPathname: () => string;
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

    // Tab navigation: Ctrl+W to close (desktop only)
    if ((e.ctrlKey || e.metaKey) && e.key === 'w' && deps.isDesktop()) {
      if (deps.getPathname().startsWith('/note/')) {
        e.preventDefault();
        deps.closeCurrentTab();
      }
    }

    // Tab navigation: Ctrl+PageDown = next tab, Ctrl+PageUp = prev tab
    if ((e.ctrlKey || e.metaKey) && e.key === 'PageDown') {
      if (deps.getPathname().startsWith('/note/')) {
        e.preventDefault();
        const noteId = deps.nextTab();
        if (noteId) deps.goto(`/note/${noteId}`);
      }
    }

    if ((e.ctrlKey || e.metaKey) && e.key === 'PageUp') {
      if (deps.getPathname().startsWith('/note/')) {
        e.preventDefault();
        const noteId = deps.prevTab();
        if (noteId) deps.goto(`/note/${noteId}`);
      }
    }
  };

  const handleActivity = () => {
    deps.recordActivity();

    // Note: encryption lock detection and silent KEK restore are handled
    // by the layout $effect (attemptSilentRestoreOrShowModal). The effect
    // reacts to isEncryptionUnlocked() changes and tries to restore KEK
    // from IndexedDB before showing the modal for balanced/convenient modes.
  };

  return { handleKeydown, handleActivity };
}
