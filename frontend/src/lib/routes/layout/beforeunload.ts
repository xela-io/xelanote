export interface BeforeUnloadDeps {
  isDirty: () => boolean;
  isSyncing: () => boolean;
  warningMessage: string;
}

export function handleBeforeUnload(e: BeforeUnloadEvent, deps: BeforeUnloadDeps) {
  if (deps.isDirty() || deps.isSyncing()) {
    e.preventDefault();
    e.returnValue = deps.warningMessage;
    return e.returnValue;
  }
  return undefined;
}
