<script lang="ts">
  import type { ComponentType } from 'svelte';

  import type { TreeNode } from '$lib/stores/tree.svelte';

  interface Props {
    node: TreeNode;
    // Rename folder dialog
    showRenameDialog: boolean;
    RenameFolderDialogComponent: ComponentType | null;
    onCloseRenameDialog: () => void;
    // Delete folder dialog
    showDeleteDialog: boolean;
    DeleteFolderDialogComponent: ComponentType | null;
    onCloseDeleteDialog: () => void;
    // Color picker dialog
    showColorPicker: boolean;
    ColorPickerDialogComponent: ComponentType | null;
    onCloseColorPicker: () => void;
    onColorSelect: (color: string | null) => void;
    // Share dialog
    showShareDialog: boolean;
    ShareDialogComponent: ComponentType | null;
    onCloseShareDialog: () => void;
    // Rename note dialog
    showRenameNoteDialog: boolean;
    RenameNoteDialogComponent: ComponentType | null;
    onCloseRenameNoteDialog: () => void;
  }

  const {
    node,
    showRenameDialog,
    RenameFolderDialogComponent,
    onCloseRenameDialog,
    showDeleteDialog,
    DeleteFolderDialogComponent,
    onCloseDeleteDialog,
    showColorPicker,
    ColorPickerDialogComponent,
    onCloseColorPicker,
    onColorSelect,
    showShareDialog,
    ShareDialogComponent,
    onCloseShareDialog,
    showRenameNoteDialog,
    RenameNoteDialogComponent,
    onCloseRenameNoteDialog,
  }: Props = $props();
</script>

<!-- Rename dialog -->
{#if node.type === 'folder' && showRenameDialog}
  {#if RenameFolderDialogComponent}
    <RenameFolderDialogComponent
      open={true}
      folderId={node.id}
      currentName={node.name}
      onClose={onCloseRenameDialog}
    />
  {/if}
{/if}

<!-- Delete dialog -->
{#if node.type === 'folder' && showDeleteDialog}
  {#if DeleteFolderDialogComponent}
    <DeleteFolderDialogComponent
      open={true}
      folderId={node.id}
      folderName={node.name}
      folderPath={node.path}
      noteCount={node.noteCount}
      onClose={onCloseDeleteDialog}
    />
  {/if}
{/if}

<!-- Color picker dialog -->
{#if showColorPicker}
  {#if ColorPickerDialogComponent}
    <ColorPickerDialogComponent
      currentColor={node.color}
      onClose={onCloseColorPicker}
      onSelect={onColorSelect}
    />
  {/if}
{/if}

<!-- Share dialog (note or folder) -->
{#if showShareDialog && ShareDialogComponent}
  <ShareDialogComponent
    resourceType={node.type === 'folder' ? 'folder' : 'note'}
    resourceId={node.id}
    onClose={onCloseShareDialog}
  />
{/if}

<!-- Rename note dialog -->
{#if node.type === 'note' && showRenameNoteDialog}
  {#if RenameNoteDialogComponent}
    <RenameNoteDialogComponent
      open={true}
      noteId={node.id}
      currentTitle={node.title}
      onClose={onCloseRenameNoteDialog}
    />
  {/if}
{/if}
