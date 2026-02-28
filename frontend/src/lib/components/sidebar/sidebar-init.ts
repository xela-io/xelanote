export interface SidebarInitDeps {
  loadExpandedState: () => void;
  loadTrashCount: () => void;
  loadShared: () => void;
  loadJournalFeature: () => void;
  loadRecipeFeature: () => void;
  loadCanvasFeature: () => void;
  loadShoppingFeature: () => void;
  startInterval: (handler: () => void, ms: number) => number;
  clearInterval: (id: number) => void;
}

export function initSidebarOnMount(deps: SidebarInitDeps) {
  deps.loadExpandedState();
  deps.loadTrashCount();
  deps.loadShared();
  deps.loadJournalFeature();
  deps.loadRecipeFeature();
  deps.loadCanvasFeature();
  deps.loadShoppingFeature();

  const interval = deps.startInterval(() => {
    deps.loadTrashCount();
  }, 30000);

  return () => deps.clearInterval(interval);
}
