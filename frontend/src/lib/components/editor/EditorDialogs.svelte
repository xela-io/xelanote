<script lang="ts">
  import { _ } from 'svelte-i18n';

  import type { AIAction, Note } from '$lib/api';
  import { FEATURE_FLAGS } from '$lib/config';
  import type { AITransformState } from '$lib/editor/ai-actions';
  import type { DialogLoaderState } from '$lib/editor/dialog-loaders';

  import AITransformDialog from '../AITransformDialog.svelte';
  import ColorPickerPopover from '../ColorPickerPopover.svelte';
  import EditorMoreMenu from '../EditorMoreMenu.svelte';
  import TableInsertDialog from '../TableInsertDialog.svelte';
  import EditorInsertMenu from './EditorInsertMenu.svelte';

  type EditorMode = 'edit' | 'split' | 'preview' | 'live';

  interface Props {
    note: Note | null;
    noteId: string;
    // Move dialog
    showMoveDialog: boolean;
    dialogLoaders: DialogLoaderState;
    onCloseMoveDialog: () => void;
    // Version history
    showVersionHistory: boolean;
    onCloseVersionHistory: () => void;
    onVersionRestored: () => void;
    // Markdown guide
    markdownGuideOpen: boolean;
    markdownGuideDropdownOpen: boolean;
    onCloseMarkdownGuide: () => void;
    onCloseMarkdownGuideDropdown: () => void;
    // Color picker
    showColorPicker: boolean;
    onColorSelect: (color: string) => void;
    onCloseColorPicker: () => void;
    // Insert menu
    showInsertMenu: boolean;
    insertMenuTriggerRect: DOMRect | null;
    onInsertTask: () => void;
    onInsertTable: () => void;
    onUpload: () => void;
    onIndent: () => void;
    onOutdent: () => void;
    onCloseInsertMenu: () => void;
    // More menu
    showMoreMenu: boolean;
    moreMenuTriggerRect: DOMRect | null;
    editorMode: EditorMode;
    isMobile: boolean;
    aiEnabled: boolean;
    isEncrypted: boolean;
    onDelete: () => void;
    onMove: () => void;
    onExport: () => void;
    onColorPicker: () => void;
    onHelp: () => void;
    onAIToggle: () => void;
    onShare: () => void;
    onEncryptionToggle: () => void;
    onSetEditorMode: (mode: EditorMode) => void;
    onCloseMoreMenu: () => void;
    // AI Actions dropdown
    showAIActionsDropdown: boolean;
    lazyDialogs: DialogLoaderState;
    aiActionsTriggerRect: DOMRect | null;
    onAIActionSelect: (action: AIAction, customPrompt?: string) => void;
    onCloseAIActionsDropdown: () => void;
    // Dictation panel
    showDictationPanel: boolean;
    dictationTriggerRect: DOMRect | null;
    onDictationInsert: (text: string, withAICleanup: boolean) => void;
    onCloseDictationPanel: () => void;
    // Table insert dialog
    showTableInsertDialog: boolean;
    onCloseTableInsertDialog: () => void;
    onTableInsert: (rows: number, cols: number) => void;
    // AI Transform dialog
    showAITransformDialog: boolean;
    aiTransformState: AITransformState | null;
    getCurrentContent: () => string;
    onApplyAITransform: (transformedText: string) => void;
    onCloseAITransformDialog: () => void;
    // Share dialog
    showShareDialog: boolean;
    onCloseShareDialog: () => void;
  }

  const {
    note,
    noteId: _noteId,
    showMoveDialog,
    dialogLoaders,
    onCloseMoveDialog,
    showVersionHistory,
    onCloseVersionHistory,
    onVersionRestored,
    markdownGuideOpen,
    markdownGuideDropdownOpen,
    onCloseMarkdownGuide,
    onCloseMarkdownGuideDropdown,
    showColorPicker,
    onColorSelect,
    onCloseColorPicker,
    showInsertMenu,
    insertMenuTriggerRect,
    onInsertTask,
    onInsertTable,
    onUpload,
    onCloseInsertMenu,
    showMoreMenu,
    moreMenuTriggerRect,
    editorMode,
    isMobile,
    aiEnabled,
    isEncrypted,
    onDelete,
    onMove,
    onExport,
    onColorPicker,
    onHelp,
    onIndent,
    onOutdent,
    onAIToggle,
    onShare,
    onEncryptionToggle,
    onSetEditorMode,
    onCloseMoreMenu,
    showAIActionsDropdown,
    lazyDialogs,
    aiActionsTriggerRect,
    onAIActionSelect,
    onCloseAIActionsDropdown,
    showDictationPanel,
    dictationTriggerRect,
    onDictationInsert,
    onCloseDictationPanel,
    showTableInsertDialog,
    onCloseTableInsertDialog,
    onTableInsert,
    showAITransformDialog,
    aiTransformState,
    getCurrentContent,
    onApplyAITransform,
    onCloseAITransformDialog,
    showShareDialog,
    onCloseShareDialog,
  }: Props = $props();
</script>

<!-- Move to folder dialog -->
{#if showMoveDialog && note}
  {#if dialogLoaders.moveToFolderDialog}
    {@const MoveToFolderDialog = dialogLoaders.moveToFolderDialog}
    <MoveToFolderDialog
      noteId={note.id}
      currentFolder={note.folder_path}
      onClose={onCloseMoveDialog}
    />
  {/if}
{/if}

<!-- Markdown Guide Dropdown -->
{#if markdownGuideDropdownOpen && dialogLoaders.markdownGuideDropdown}
  {@const MarkdownGuideDropdown = dialogLoaders.markdownGuideDropdown}
  <MarkdownGuideDropdown onClose={onCloseMarkdownGuideDropdown} />
{/if}

<!-- Markdown Guide Dialog -->
{#if markdownGuideOpen && dialogLoaders.markdownGuideDialog}
  {@const MarkdownGuideDialog = dialogLoaders.markdownGuideDialog}
  <MarkdownGuideDialog onClose={onCloseMarkdownGuide} />
{/if}

<!-- Version History Dialog -->
{#if showVersionHistory && note && dialogLoaders.versionHistoryDialog}
  {@const VersionHistoryDialog = dialogLoaders.versionHistoryDialog}
  <VersionHistoryDialog
    noteId={note.id}
    noteTitle={note.title}
    currentVersion={note.version}
    currentContent={note.content}
    aiEnabled={note.ai_enabled ?? false}
    onClose={onCloseVersionHistory}
    onRestored={onVersionRestored}
  />
{/if}

<!-- Color Picker Popover -->
{#if showColorPicker && FEATURE_FLAGS.colorSyntax}
  <ColorPickerPopover onSelect={onColorSelect} onClose={onCloseColorPicker} />
{/if}

<!-- Insert Menu (rendered outside toolbar to avoid overflow clipping) -->
{#if showInsertMenu}
  <EditorInsertMenu
    {onInsertTask}
    {onInsertTable}
    {onUpload}
    {onIndent}
    {onOutdent}
    {isMobile}
    onClose={onCloseInsertMenu}
    triggerRect={insertMenuTriggerRect}
  />
{/if}

<!-- More Menu (rendered outside toolbar to avoid overflow clipping) -->
{#if showMoreMenu}
  <EditorMoreMenu
    {onDelete}
    {onMove}
    {onExport}
    {onColorPicker}
    {onHelp}
    {onAIToggle}
    {onShare}
    {onEncryptionToggle}
    {onSetEditorMode}
    {editorMode}
    {isMobile}
    {aiEnabled}
    {isEncrypted}
    onClose={onCloseMoreMenu}
    triggerRect={moreMenuTriggerRect}
  />
{/if}

<!-- AI Actions Dropdown (lazy-loaded, rendered outside toolbar to avoid overflow clipping) -->
{#if showAIActionsDropdown && lazyDialogs.aiActionsDropdown}
  <lazyDialogs.aiActionsDropdown
    onSelectAction={onAIActionSelect}
    onClose={onCloseAIActionsDropdown}
    triggerRect={aiActionsTriggerRect}
  />
{/if}

<!-- Dictation Panel (lazy-loaded) -->
{#if showDictationPanel && lazyDialogs.dictationPanel}
  <lazyDialogs.dictationPanel
    noteId={note?.id ?? ''}
    {aiEnabled}
    hasOpenAIKey={false}
    onInsert={onDictationInsert}
    onClose={onCloseDictationPanel}
    triggerRect={dictationTriggerRect}
  />
{/if}

<!-- Table Insert Dialog -->
<TableInsertDialog
  open={showTableInsertDialog}
  onClose={onCloseTableInsertDialog}
  onInsert={onTableInsert}
/>

<!-- AI Transform Dialog -->
{#if showAITransformDialog && note && aiTransformState}
  <AITransformDialog
    noteId={note.id}
    action={aiTransformState.action}
    customPrompt={aiTransformState.customPrompt}
    originalText={aiTransformState.originalText}
    selectionFrom={aiTransformState.selectionFrom}
    selectionTo={aiTransformState.selectionTo}
    isFullContent={aiTransformState.isFullContent}
    initialContentHash={aiTransformState.initialContentHash}
    {getCurrentContent}
    onApply={onApplyAITransform}
    onClose={onCloseAITransformDialog}
  />
{/if}

<!-- Share Note Dialog (lazy-loaded) -->
{#if showShareDialog && note && lazyDialogs.shareDialog}
  <lazyDialogs.shareDialog
    resourceType="note"
    resourceId={note.id}
    isEncrypted={note.content_encrypted ?? false}
    onClose={onCloseShareDialog}
  />
{/if}
