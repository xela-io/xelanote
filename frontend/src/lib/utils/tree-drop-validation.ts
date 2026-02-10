/**
 * Shared validation logic for tree drag & drop operations.
 * Used by both desktop (UnifiedTree.svelte) and touch (Sidebar.svelte) DnD.
 *
 * Pure logic - no UI, no side effects. Caller is responsible for showing dialogs.
 */

export interface DragInfo {
  type: 'folder' | 'note' | 'root-dropzone';
  id: string | number;
  path?: string; // folder path (for folders)
  folder_path?: string; // folder path where note lives (for notes)
}

export interface TargetInfo {
  type: 'folder' | 'note' | 'root-dropzone';
  id: string | number;
  path?: string;
  folder_path?: string;
}

export type DropPosition = 'before' | 'after' | 'into';

export interface ValidationResult {
  valid: boolean;
  errorKey?: string;
}

/**
 * Validate whether a drop operation is allowed.
 */
export function validateDrop(
  drag: DragInfo,
  target: TargetInfo,
  position: DropPosition
): ValidationResult {
  // Self-move: dropping an item onto itself
  if (String(drag.id) === String(target.id) && drag.type === target.type) {
    return { valid: false }; // silent no-op
  }

  // Root dropzone: always valid (move to top level)
  if (target.type === 'root-dropzone') {
    // Note already at root
    if (drag.type === 'note' && drag.folder_path === '/') {
      return { valid: false }; // silent no-op
    }
    return { valid: true };
  }

  // Journal protection: cannot move items into /Journal.
  // For 'into': show error (user clearly intended to drop there).
  // For 'before'/'after': silent no-op (imprecise touch or finger passing through;
  // these positions also fall through to move-into behavior in the handler).
  const targetPath = target.type === 'folder' ? target.path : target.folder_path;
  if (target.type === 'folder' && targetPath === '/Journal') {
    if (position === 'into') {
      return { valid: false, errorKey: 'component.tree.cannot_move_to_journal' };
    }
    return { valid: false };
  }

  // Cross-folder note drop (note dropped before/after a note in /Journal)
  if (
    drag.type === 'note' &&
    target.type === 'note' &&
    (position === 'before' || position === 'after') &&
    drag.folder_path !== target.folder_path &&
    target.folder_path === '/Journal'
  ) {
    return { valid: false, errorKey: 'component.tree.cannot_move_to_journal' };
  }

  // Folder self-containment: cannot move a folder into its own subtree
  if (
    drag.type === 'folder' &&
    target.type === 'folder' &&
    drag.path &&
    target.path &&
    position === 'into' &&
    target.path.startsWith(drag.path + '/')
  ) {
    return { valid: false, errorKey: 'component.tree.cannot_move_into_self' };
  }

  // Same-position no-op: note already in this folder
  if (
    drag.type === 'note' &&
    target.type === 'folder' &&
    position === 'into' &&
    drag.folder_path === target.path
  ) {
    return { valid: false }; // silent no-op
  }

  // Same-position no-op: folder dropped "into" its current parent
  // (handled at the caller level via sibling check)

  return { valid: true };
}
