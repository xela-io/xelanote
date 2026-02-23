export interface ViewportDeps {
  getIsMobile: () => boolean;
  setIsMobile: (value: boolean) => void;
  getEditorMode: () => 'split' | 'edit' | 'preview' | 'live';
  setEditorMode: (mode: 'split' | 'edit' | 'preview' | 'live') => void;
  getSidebarOpen: () => boolean;
  setSidebarOpen: (open: boolean) => void;
  setIsKeyboardOpen: (open: boolean) => void;
}

export interface ViewportHandlers {
  handleResize: () => void;
  debouncedHandleResize: () => void;
  handleVisualViewportResize: () => void;
  handleFocusIn: (e: FocusEvent) => void;
  handleFocusOut: () => void;
  handleTouchStart: (e: TouchEvent) => void;
  cleanup: () => void;
}

export function createViewportHandlers(
  deps: ViewportDeps,
  options: {
    getResizeTimeout: () => ReturnType<typeof setTimeout> | null;
    setResizeTimeout: (handle: ReturnType<typeof setTimeout> | null) => void;
    isInputElement: (el: Element | null) => boolean;
    windowObj: Window;
    documentObj: Document;
  }
): ViewportHandlers {
  const { windowObj, documentObj } = options;
  let inputFocused = false;
  let lastNonKeyboardViewportHeight = 0;

  const setViewportHeightCssVar = () => {
    const style = documentObj.documentElement.style;
    const windowHeight = windowObj.innerHeight;
    const visualViewportHeight = windowObj.visualViewport?.height ?? windowHeight;
    const keyboardOpen = windowHeight - visualViewportHeight > 150;

    // Keep a stable "full app" height while the keyboard is open to avoid
    // shrinking the root layout and causing large reflows/jumps in editors.
    if (!keyboardOpen) {
      lastNonKeyboardViewportHeight = Math.round(visualViewportHeight);
    }

    const targetHeight = Math.max(
      1,
      Math.round(
        keyboardOpen
          ? lastNonKeyboardViewportHeight || windowHeight
          : lastNonKeyboardViewportHeight || visualViewportHeight || windowHeight
      )
    );

    style.setProperty('--app-viewport-height', `${targetHeight}px`);
  };

  const updateKeyboardState = () => {
    let viewportKeyboard = false;
    if (windowObj.visualViewport) {
      const viewportHeight = windowObj.visualViewport.height;
      const windowHeight = windowObj.innerHeight;
      viewportKeyboard = windowHeight - viewportHeight > 150;
    }

    const keyboardOpen = viewportKeyboard || (deps.getIsMobile() && inputFocused);
    deps.setIsKeyboardOpen(keyboardOpen);
  };

  const handleResize = () => {
    setViewportHeightCssVar();

    const mobile = windowObj.innerWidth < 768;
    const wasMobile = deps.getIsMobile();
    deps.setIsMobile(mobile);

    if (mobile && !wasMobile) {
      deps.setSidebarOpen(false);
      if (deps.getEditorMode() === 'split') {
        deps.setEditorMode('edit');
      }
    }
    if (!mobile && wasMobile) {
      deps.setSidebarOpen(true);
    }
  };

  const debouncedHandleResize = () => {
    const existing = options.getResizeTimeout();
    if (existing) clearTimeout(existing);
    options.setResizeTimeout(setTimeout(handleResize, 150));
  };

  const handleVisualViewportResize = () => {
    setViewportHeightCssVar();
    updateKeyboardState();
  };

  const handleFocusIn = (e: FocusEvent) => {
    const target = e.target as Element | null;
    if (target && options.isInputElement(target)) {
      inputFocused = true;
      updateKeyboardState();
    }
  };

  const handleFocusOut = () => {
    setTimeout(() => {
      if (!options.isInputElement(documentObj.activeElement)) {
        inputFocused = false;
        updateKeyboardState();
      }
    }, 100);
  };

  const handleTouchStart = (e: TouchEvent) => {
    if (!deps.getIsMobile()) return;
    const target = e.target as Element | null;
    if (target && options.isInputElement(target)) {
      setTimeout(() => {
        inputFocused = true;
        updateKeyboardState();
      }, 50);
    }
  };

  const cleanup = () => {
    if (windowObj.visualViewport) {
      windowObj.visualViewport.removeEventListener('resize', handleVisualViewportResize);
    }
    documentObj.removeEventListener('focusin', handleFocusIn);
    documentObj.removeEventListener('focusout', handleFocusOut);
    documentObj.removeEventListener('touchstart', handleTouchStart);
    const existing = options.getResizeTimeout();
    if (existing) {
      clearTimeout(existing);
      options.setResizeTimeout(null);
    }
  };

  return {
    handleResize,
    debouncedHandleResize,
    handleVisualViewportResize,
    handleFocusIn,
    handleFocusOut,
    handleTouchStart,
    cleanup,
  };
}
