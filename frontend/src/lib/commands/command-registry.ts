export interface CommandItem {
  id: string;
  label: string;
  shortcut?: string;
  icon?: string;
  action: () => void | Promise<void>;
  /** i18n key for the label (used if label is empty) */
  i18nKey?: string;
}

let commands: CommandItem[] = [];

/**
 * Register the full set of palette commands.
 * Called once from the QuickSwitcher when command mode is activated.
 */
export function registerCommands(items: CommandItem[]): void {
  commands = items;
}

/**
 * Get all registered commands, optionally filtered by query.
 */
export function getCommands(query?: string): CommandItem[] {
  if (!query) return commands;
  const lower = query.toLowerCase();
  return commands.filter(
    (cmd) =>
      cmd.label.toLowerCase().includes(lower) ||
      (cmd.shortcut && cmd.shortcut.toLowerCase().includes(lower))
  );
}

/**
 * Execute a command by id.
 */
export function executeCommand(id: string): void {
  const cmd = commands.find((c) => c.id === id);
  if (cmd) {
    cmd.action();
  }
}
