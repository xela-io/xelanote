<script lang="ts">
  import type { EditorView } from '@codemirror/view';
  import { tick } from 'svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import type { AIAction } from '$lib/api';
  import { DeleteCommand } from '$lib/commands/DeleteCommand';
  import { FEATURE_FLAGS } from '$lib/config';
  import type { AITransformState } from '$lib/editor/ai-actions';
  import { setLivePreviewMode, updateEditorContent, updateFocusMode } from '$lib/editor/codemirror';
  import {
    type DialogLoaderState,
    loadAIActionsDropdown,
    loadMarkdownGuideDialog,
    loadMarkdownGuideDropdown,
    loadMoveToFolderDialog,
    loadShareDialog,
    loadVersionHistoryDialog,
    maybeLoadDialog,
  } from '$lib/editor/dialog-loaders';
  import {
    handleAIToggle as handleAIToggleAction,
    handleAutoSaveToggle as handleAutoSaveToggleAction,
    handleEncryptionToggle as handleEncryptionToggleAction,
    handleSaveNote as handleSaveNoteAction,
    handleTitleInput as handleTitleInputAction,
  } from '$lib/editor/editor-actions';
  import {
    applyAITransformAction,
    handleAIActionSelectAction,
    openAIActionsMenuAction,
  } from '$lib/editor/editor-ai-menu';
  import {
    handleImageResizeAction,
    handleTaskReorderAction,
  } from '$lib/editor/editor-content-actions';
  import { initEditorAction } from '$lib/editor/editor-init';
  import { ensureEditorReady } from '$lib/editor/editor-mode';
  import {
    handleInsertLinkAction,
    updateCurrentNoteSummary,
  } from '$lib/editor/editor-note-content';
  import { registerSpellCheckReplaceListener } from '$lib/editor/editor-spellcheck';
  import {
    extractFilesFromInputChangeEvent,
    handleColorSelectAction,
    openColorPickerAction,
    openMarkdownHelpAction,
    openMoreMenuAction,
    resetInputEventValue,
  } from '$lib/editor/editor-ui-actions';
  import { uploadImagesFromEditorAction } from '$lib/editor/editor-upload-actions';
  import { readFindReplaceState, writeFindReplaceState } from '$lib/editor/find-replace-state';
  import {
    closeFindReplace as closeFindReplaceUI,
    type FindReplaceState,
    handleExtensionsReady,
    handleNoteChange,
    handleUrlHighlight,
    openFindReplace as openFindReplaceUI,
  } from '$lib/editor/find-replace-ui';
  import { imageResize } from '$lib/editor/image-resize';
  import {
    handleEditorDragOver,
    handleEditorDrop,
    handleEditorPaste,
  } from '$lib/editor/image-upload';
  import { indentSelection, outdentSelection } from '$lib/editor/indentation';
  import { extractHeadings, renderMarkdown } from '$lib/editor/markdown';
  import {
    exportNoteMarkdown,
    handleDeleteNote,
    handleWikilinkClick as handleWikilinkAction,
  } from '$lib/editor/note-actions';
  import { highlightSearchTerms } from '$lib/editor/preview-highlight';
  import { handlePreviewClick, handleTocClick } from '$lib/editor/preview-interactions';
  import { createSplitResizeController } from '$lib/editor/split-resize';
  import { insertTable } from '$lib/editor/table-insert';
  import { taskCollapse, type TaskCollapseOptions } from '$lib/editor/task-collapse';
  import { insertTask } from '$lib/editor/task-insert';
  import { taskSortable, type TaskSortableOptions } from '$lib/editor/task-sortable';
  import { toggleTaskByIndex, toggleTaskByLine } from '$lib/editor/task-toggle';
  import { getIsSyncing, getPendingCount, getSyncProgress } from '$lib/offline/sync-manager.svelte';
  import * as auth from '$lib/stores/auth.svelte';
  import * as autosave from '$lib/stores/autosave.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';
  import * as focusMode from '$lib/stores/focus-mode.svelte';
  import * as folders from '$lib/stores/folders.svelte';
  import * as history from '$lib/stores/history.svelte';
  import * as network from '$lib/stores/network.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import * as toast from '$lib/stores/toast.svelte';
  import * as trash from '$lib/stores/trash.svelte';
  import * as tree from '$lib/stores/tree.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { getTasksInDocument } from '$lib/utils/task-reorder';

  import EditorDialogs from './editor/EditorDialogs.svelte';
  import EditorStatusBar from './editor/EditorStatusBar.svelte';
  import EditorToolbar from './editor/EditorToolbar.svelte';
  import FindReplaceBar from './FindReplaceBar.svelte';
  // ShareDialog is lazy-loaded via dialog-loaders.ts
  import TableOfContents from './TableOfContents.svelte';

  interface Props {
    noteId: string;
  }

  const { noteId }: Props = $props();

  let editorView = $state<EditorView | undefined>(undefined);
  let renderedContent = $state('');
  const completedLabel = (count: number) =>
    $_('component.editor.completed_count', { values: { count } });
  const completedAriaLabel = (count: number) =>
    $_('component.editor.completed_toggle', { values: { count } });
  const taskCollapseOptions = $state<TaskCollapseOptions>({
    completedLabel,
    completedAriaLabel,
    noteId: '',
    revision: '',
  });
  const taskSortableOptions = $state<TaskSortableOptions>({
    onReorder: (fromIndex, toIndex) => handleTaskReorder(fromIndex, toIndex),
    mode: 'preview',
    enabled: true,
    onReorderByLine: undefined,
    revision: '',
  });
  const liveTaskSortableOptions = $state<TaskSortableOptions>({
    onReorder: (fromIndex, toIndex) => handleTaskReorder(fromIndex, toIndex),
    onReorderByLine: (fromLine, toLine) => handleTaskReorderByLine(fromLine, toLine),
    mode: 'live',
    enabled: false,
    revision: '',
  });
  let lastTaskClickTime = 0; // Timestamp-based debounce for task checkbox clicks
  let showMoveDialog = $state(false);
  let showVersionHistory = $state(false);
  let showColorPicker = $state(false);
  let showMoreMenu = $state(false);
  let showShareDialog = $state(false);
  let moreMenuTriggerRect = $state<DOMRect | null>(null);
  let showAIActionsDropdown = $state(false);
  let aiActionsTriggerRect = $state<DOMRect | null>(null);
  let showTableInsertDialog = $state(false);
  let showAITransformDialog = $state(false);
  let aiTransformState = $state<AITransformState | null>(null);
  // Find & Replace state
  let showFindReplace = $state(false);
  let findReplaceQuery = $state('');
  let findReplaceShowReplace = $state(false);
  let findReplaceCaseSensitive = $state(false);
  let pendingHighlightQuery = $state<string | null>(null);
  let editorExtensionsReady = $state(false);
  // Lazy-loaded dialog state
  let lazyDialogs = $state<DialogLoaderState>({});
  const setLazyDialogs = (s: DialogLoaderState) => {
    lazyDialogs = s;
  };

  // Trigger lazy loading when dialogs are requested
  $effect(() => {
    maybeLoadDialog(showShareDialog, lazyDialogs, loadShareDialog, setLazyDialogs);
  });
  $effect(() => {
    maybeLoadDialog(showAIActionsDropdown, lazyDialogs, loadAIActionsDropdown, setLazyDialogs);
  });
  let prevNoteId: string | null = $state(null);

  let dialogLoaders = $state<DialogLoaderState>({
    moveToFolderDialog: null,
    markdownGuideDialog: null,
    markdownGuideDropdown: null,
    versionHistoryDialog: null,
  });

  // Split resize state
  let isSplitResizing = $state(false);
  let splitContainerRef: HTMLDivElement | null = $state(null);
  let previewScrollRef: HTMLDivElement | null = $state(null);
  const splitResizeController = createSplitResizeController(
    {
      getContainerRect: () => splitContainerRef?.getBoundingClientRect() ?? null,
      setSplitPosition: (pos) => ui.setSplitPosition(pos),
      setActive: (active) => {
        isSplitResizing = active;
      },
    },
    () => isSplitResizing
  );

  // Title to ID mapping for wikilink URLs
  const titleToIdMap = $derived.by(() => {
    const allNotes = notes.getNotes();
    const map = new SvelteMap<string, string>();
    for (const note of allNotes) {
      map.set(note.title.toLowerCase().trim(), note.id);
    }
    return map;
  });

  // Extract headings for TOC — debounced so it doesn't re-parse on every keystroke
  let headings = $state<ReturnType<typeof extractHeadings>>([]);
  $effect(() => {
    const note = notes.getCurrentNote();
    if (!note) {
      headings = [];
      return;
    }
    const content = note.content;
    const timer = setTimeout(() => {
      headings = extractHeadings(content);
    }, 300);
    return () => clearTimeout(timer);
  });

  // Reactive: update rendered content when note changes.
  // In split mode, debounce to avoid expensive DOM recreation on every keystroke.
  $effect(() => {
    const note = notes.getCurrentNote();
    if (!note) return;
    // Capture reactive deps synchronously for Svelte tracking
    const content = note.content;
    const map = titleToIdMap;
    const mode = ui.getEditorMode();

    if (mode === 'split') {
      const timer = setTimeout(() => {
        renderedContent = renderMarkdown(content, { titleToIdMap: map });
        taskCollapseOptions.revision = renderedContent;
        taskSortableOptions.revision = renderedContent;
      }, 150);
      return () => clearTimeout(timer);
    } else {
      renderedContent = renderMarkdown(content, { titleToIdMap: map });
      taskCollapseOptions.revision = renderedContent;
      taskSortableOptions.revision = renderedContent;
    }
  });

  $effect(() => {
    taskCollapseOptions.noteId = noteId;
  });

  $effect(() => {
    const note = notes.getCurrentNote();
    liveTaskSortableOptions.enabled =
      FEATURE_FLAGS.taskLists && ui.getEditorMode() === 'live' && Boolean(note);
    liveTaskSortableOptions.revision = note?.content ?? '';
    liveTaskSortableOptions.editorView = editorView;
  });

  // Reactive: load note when ID changes
  $effect(() => {
    // On hard refresh, auth restore is async. Wait until authenticated before
    // loading note to avoid one-time failed loads that leave currentNote null.
    if (!noteId || !auth.isAuthenticated()) return;

    const encryptionUnlocked = encryption.isEncryptionUnlocked();
    const noteError = notes.getError();
    const currentNote = notes.getCurrentNote();
    const noteLoaded = currentNote?.id === noteId;

    if (noteLoaded) return;

    // Avoid retry loops while vault is locked. After unlock, this effect reruns
    // and loads the note automatically.
    if (!encryptionUnlocked && noteError === 'ENCRYPTION_LOCKED') return;
    if (noteError === 'NOT_FOUND') return;

    notes.loadNote(noteId);
  });

  const initEditor = initEditorAction({
    getDoc: () => notes.getCurrentNote()?.content ?? '',
    onChange: (content) => {
      notes.updateCurrentNoteContent(content);
      // Don't render markdown here — the $effect on getCurrentNote() handles it,
      // avoiding duplicate rendering on every keystroke.
      notes.scheduleAutoSave();
    },
    onSave: handleSave,
    onWikilinkClick: handleWikilinkClick,
    onToggleTaskByLine: (lineNumber, checked) => {
      toggleTask(-1, checked, lineNumber);
    },
    onColorPicker: () => {
      openColorPicker();
    },
    onInsertTable: () => {
      handleInsertTable();
    },
    onBeforeNewline: (view) => handleNewlineWithTaskReorder(view),
    onFindReplace: (options) => {
      openFindReplace(undefined, options);
    },
    onExtensionsReady: () => {
      const nextState = handleExtensionsReady(
        getFindReplaceState({ editorExtensionsReadyOverride: true }),
        findReplaceHandlers
      );
      setFindReplaceState(nextState);
    },
    setEditorView: (view) => {
      editorView = view;
    },
    setExtensionsReady: (ready) => {
      editorExtensionsReady = ready;
    },
  });

  $effect(() => {
    if (!FEATURE_FLAGS.livePreview && ui.getEditorMode() === 'live') {
      ui.setEditorMode('edit');
    }
  });

  $effect(() => {
    const view = editorView;
    if (!view) return;
    setLivePreviewMode(view, FEATURE_FLAGS.livePreview && ui.getEditorMode() === 'live', {
      noteId,
    });
  });

  // Update editor when note content changes externally (note switch, load).
  // Skip when dirty to avoid overwriting unsaved user edits during auto-save.
  // No isLoading guard — updateEditorContent already no-ops when content is identical.
  $effect(() => {
    const note = notes.getCurrentNote();
    if (editorView && note && !notes.getIsDirty()) {
      updateEditorContent(editorView, note.content);
    }
  });

  // Update editor focus mode when settings change
  $effect(() => {
    if (editorView) {
      updateFocusMode(editorView, {
        typewriter: focusMode.isTypewriterMode(),
        dimLines: focusMode.shouldDimInactiveLines(),
      });
    }
  });

  // Handle spell-check replacement events
  $effect(() => {
    return registerSpellCheckReplaceListener({
      getEditorView: () => editorView,
      scheduleAutoSave: () => notes.scheduleAutoSave(),
    });
  });

  async function handleSave() {
    await handleSaveNoteAction({
      editorView,
      saveNote: notes.saveNote,
      getCurrentNoteId: () => notes.getCurrentNote()?.id,
      reloadNote: notes.loadNote,
      toast,
      strings: {
        conflictWarning: (version) =>
          $_('component.editor.conflict_warning', { values: { version } }),
        conflictLoadRemote: $_('component.editor.conflict_load_remote'),
        errorRemote: $_('component.editor.status.error_remote'),
        errorSave: $_('component.editor.status.error'),
      },
    });
  }

  function handleTitleInput(e: Event) {
    handleTitleInputAction(e, {
      updateTitle: notes.updateCurrentNoteTitle,
      scheduleAutoSave: notes.scheduleAutoSave,
    });
  }

  function handleAutoSaveToggle() {
    handleAutoSaveToggleAction({
      getAutoSaveEnabled: autosave.getAutoSaveEnabled,
      setAutoSaveEnabled: autosave.setAutoSaveEnabled,
      getIsDirty: notes.getIsDirty,
      scheduleAutoSave: notes.scheduleAutoSave,
    });
  }

  async function handleAIToggle() {
    await handleAIToggleAction({
      getCurrentNote: () => {
        const note = notes.getCurrentNote();
        return note ? { id: note.id, ai_enabled: note.ai_enabled ?? false } : null;
      },
      updateCurrentAI: notes.updateCurrentNoteAIEnabled,
      reloadTree: tree.loadTree,
      toast,
      strings: {
        enabled: $_('component.editor.ai_enabled_success'),
        disabled: $_('component.editor.ai_disabled_success'),
        error: $_('component.editor.ai_toggle_error'),
      },
    });
  }

  async function handleEncryptionToggle() {
    await handleEncryptionToggleAction({
      getIsEncrypted: () => notes.getCurrentNote()?.content_encrypted !== false,
      confirm: dialog.confirm,
      toggleEncryption: notes.toggleEncryption,
      toast,
      strings: {
        decryptTitle: $_('component.editor.encryption_toggle.decrypt_confirm_title'),
        decryptMessage: $_('component.editor.encryption_toggle.decrypt_confirm_message'),
        decryptConfirm: $_('component.editor.toolbar.decrypt_note'),
        encryptTitle: $_('component.editor.encryption_toggle.encrypt_confirm_title'),
        encryptMessage: $_('component.editor.encryption_toggle.encrypt_confirm_message'),
        encryptConfirm: $_('component.editor.toolbar.encrypt_note'),
        cancel: $_('dialog.cancel'),
        decrypted: $_('component.editor.encryption_toggle.decrypted_success'),
        encrypted: $_('component.editor.encryption_toggle.encrypted_success'),
        error: $_('component.editor.encryption_toggle.error'),
      },
    });
  }

  // Open AI Actions dropdown
  function handleAIActionsClick(rect: DOMRect) {
    const currentNote = notes.getCurrentNote();
    if (!currentNote) return;
    openAIActionsMenuAction({
      rect,
      aiEnabled: Boolean(currentNote.ai_enabled),
      setTriggerRect: (nextRect) => {
        aiActionsTriggerRect = nextRect;
      },
      setOpen: (value) => {
        showAIActionsDropdown = value;
      },
      showDisabledError: () => {
        toast.error($_('error.ai_transform.ai_disabled'));
      },
    });
  }

  // Handle AI action selection from dropdown
  async function handleAIActionSelect(action: AIAction, customPrompt?: string) {
    const currentNote = notes.getCurrentNote();
    if (!currentNote || !editorView) return;

    await handleAIActionSelectAction({
      action,
      customPrompt,
      currentContent: currentNote.content,
      editorView,
      setDropdownOpen: (value) => {
        showAIActionsDropdown = value;
      },
      setDialogOpen: (value) => {
        showAITransformDialog = value;
      },
      setTransformState: (state) => {
        aiTransformState = state;
      },
      showTooShortError: () => {
        toast.error($_('error.ai_transform.too_short'));
      },
    });
  }

  // Apply transformed content from dialog
  function applyAITransform(transformedText: string) {
    applyAITransformAction({
      transformedText,
      editorView,
      aiTransformState,
      scheduleAutoSave: () => notes.scheduleAutoSave(),
      setDialogOpen: (value) => {
        showAITransformDialog = value;
      },
      setTransformState: (state) => {
        aiTransformState = state;
      },
      showSuccess: () => {
        toast.success($_('component.editor.ai_transform_success'));
      },
    });
  }

  // Menu coordination: close other menus when one opens
  function openMoreMenu(triggerRect: DOMRect) {
    openMoreMenuAction(
      {
        setShowColorPicker: (value) => {
          showColorPicker = value;
        },
        setMarkdownGuideDropdownOpen: (value) => ui.setMarkdownGuideDropdownOpen(value),
        setMoreMenuTriggerRect: (rect) => {
          moreMenuTriggerRect = rect;
        },
        setShowMoreMenu: (value) => {
          showMoreMenu = value;
        },
      },
      triggerRect
    );
  }

  function openColorPicker() {
    openColorPickerAction({
      setShowMoreMenu: (value) => {
        showMoreMenu = value;
      },
      setMarkdownGuideDropdownOpen: (value) => ui.setMarkdownGuideDropdownOpen(value),
      setShowColorPicker: (value) => {
        showColorPicker = value;
      },
    });
  }

  function openMarkdownHelp() {
    openMarkdownHelpAction({
      setShowMoreMenu: (value) => {
        showMoreMenu = value;
      },
      setShowColorPicker: (value) => {
        showColorPicker = value;
      },
      toggleMarkdownGuideDropdown: () => ui.toggleMarkdownGuideDropdown(),
    });
  }

  function handleExportNote() {
    exportNoteMarkdown(notes.getCurrentNote());
  }

  async function handleDelete() {
    await handleDeleteNote({
      goto,
      confirm: dialog.confirm,
      createDeleteCommand: (snapshot) => new DeleteCommand(snapshot),
      executeCommand: history.executeCommand,
      undoCommand: history.undo,
      getCurrentNote: () => notes.getCurrentNote(),
      loadNotes: notes.loadNotes,
      loadTree: tree.loadTree,
      clearCurrentNote: notes.clearCurrentNote,
      incrementTrash: trash.incrementTrashCount,
      decrementTrash: trash.decrementTrashCount,
      toast,
      strings: {
        confirmTitle: $_('dialog.confirm_title'),
        deleteConfirmMessage: $_('dialog.delete_note_confirm'),
        deleteConfirmText: $_('common.delete'),
        cancelText: $_('dialog.cancel'),
        deleteError: $_('component.editor.error_delete'),
        noteTrashed: $_('component.editor.note_trashed'),
        noteRestored: $_('component.editor.note_restored'),
        restoreError: $_('component.editor.error_restore'),
      },
    });
  }

  async function handleWikilinkClick(title: string) {
    await handleWikilinkAction(title, {
      goto,
      confirm: dialog.confirm,
      getCurrentNote: () => notes.getCurrentNote(),
      getAllNotes: () => notes.getNotes(),
      createNote: notes.createNote,
      loadFolders: folders.loadFolders,
      strings: {
        confirmTitle: $_('dialog.confirm_title'),
        cancelText: $_('dialog.cancel'),
        createMissingMessage: $_('dialog.create_missing_note'),
        createMissingConfirmText: $_('common.confirm'),
      },
    });
  }

  function handlePreviewClickLocal(e: MouseEvent) {
    handlePreviewClick(e, {
      featureTaskLists: FEATURE_FLAGS.taskLists,
      getLastTaskClickTime: () => lastTaskClickTime,
      setLastTaskClickTime: (value) => {
        lastTaskClickTime = value;
      },
      onWikilink: (title) => handleWikilinkClick(title),
      onToggleTask: (index, checked, lineNumber) => toggleTask(index, checked, lineNumber),
      log: console.log,
    });
  }

  function handleTocClickLocal(slug: string) {
    const noteContent = notes.getCurrentNote()?.content;
    handleTocClick(slug, {
      content: noteContent,
      liveEditorView: ui.getEditorMode() === 'live' ? editorView : undefined,
    });
  }

  /**
   * Handle Enter key in task lists.
   * Currently defers to default CodeMirror behavior for all cases.
   * Task reordering (checked → bottom) happens on checkbox toggle, not on Enter.
   */
  function handleNewlineWithTaskReorder(_view: import('@codemirror/view').EditorView): boolean {
    return false;
  }

  function toggleTask(checkboxIndex: number, checked: boolean, lineNumber?: number) {
    const cn = notes.getCurrentNote();
    const baseOptions = {
      editorView,
      checked,
      getContent: () => cn?.content ?? '',
      setContent: (content: string) => notes.updateCurrentNoteContent(content),
      scheduleAutoSave: () => notes.scheduleAutoSave(),
      queueTaskEvent: (
        noteId: string,
        taskText: string,
        index: number,
        status: 'completed' | 'reopened'
      ) => notes.queueTaskEvent(noteId, taskText, index, status),
      noteId: cn?.id,
      log: console.log,
    };

    if (lineNumber && lineNumber > 0) {
      toggleTaskByLine({
        ...baseOptions,
        lineNumber,
      });
      return;
    }

    toggleTaskByIndex({
      ...baseOptions,
      checkboxIndex,
    });
  }

  // Task Drag & Drop Reorder Handler
  function handleTaskReorder(
    fromTaskIndex: number,
    toTaskIndex: number,
    options?: { scrollIntoView?: boolean }
  ) {
    handleTaskReorderAction({
      editorView,
      fromTaskIndex,
      toTaskIndex,
      scheduleAutoSave: () => notes.scheduleAutoSave(),
      scrollIntoView: options?.scrollIntoView ?? true,
    });
  }

  function handleTaskReorderByLine(fromLine: number, toLine: number) {
    if (!editorView) return;
    const tasks = getTasksInDocument(editorView.state.doc);
    const fromTask = tasks.find((task) => task.lineNum === fromLine);
    const toTask = tasks.find((task) => task.lineNum === toLine);
    if (!fromTask || !toTask) return;
    handleTaskReorder(fromTask.index, toTask.index, { scrollIntoView: false });
  }

  // Image Resize Handler
  function handleImageResize(imageIndex: number, newWidth: number) {
    handleImageResizeAction({
      editorView,
      imageIndex,
      newWidth,
      getFallbackContent: () => notes.getCurrentNote()?.content ?? '',
      setFallbackContent: (content) => notes.updateCurrentNoteContent(content),
      scheduleAutoSave: () => notes.scheduleAutoSave(),
    });
  }

  // Upload Logic
  let uploading = $state(false);

  async function uploadImagesFromEditor(files: File[]) {
    await uploadImagesFromEditorAction({
      files,
      editorView,
      setUploading: (value) => {
        uploading = value;
      },
      toast: {
        success: (message) => toast.success(message),
        warning: (message) => toast.warning(message),
        error: (message) => toast.error(message),
      },
      messages: {
        success: (filename) => $_('component.editor.upload_success', { values: { filename } }),
        copiedToClipboard: $_('component.editor.upload_clipboard'),
        fallback: (url) => $_('component.editor.upload_fallback', { values: { url } }),
        error: (filename, message) =>
          $_('component.editor.status.error_upload', {
            values: {
              filename,
              error: message,
            },
          }),
      },
    });
  }

  function handleColorSelect(color: string) {
    if (!editorView) return;
    handleColorSelectAction(editorView, color);
  }

  function handleInsertLink(term: string, targetTitle: string) {
    handleInsertLinkAction({
      editorView,
      term,
      targetTitle,
      getFallbackContent: () => notes.getCurrentNote()?.content || '',
      setFallbackContent: (content) => notes.updateCurrentNoteContent(content),
      scheduleAutoSave: () => notes.scheduleAutoSave(),
    });
  }

  function handleSummaryUpdated(summary: string) {
    updateCurrentNoteSummary(notes.getCurrentNote(), summary);
  }

  // Upload Button Handler
  let fileInput: HTMLInputElement;

  function handleUploadButtonClick() {
    fileInput.click();
  }

  function handleFileInputChange(e: Event) {
    const files = extractFilesFromInputChangeEvent(e);
    if (files.length > 0) {
      uploadImagesFromEditor(files);
    }
    resetInputEventValue(e);
  }

  async function handleInsertTask() {
    const view = await ensureEditorReady({
      getEditorView: () => editorView,
      setEditorMode: (mode) => ui.setEditorMode(mode),
      tick,
    });
    if (!view) return;
    insertTask(view);
  }

  function handleInsertTable() {
    showTableInsertDialog = true;
  }

  async function handleTableInsert(rows: number, cols: number) {
    showTableInsertDialog = false;
    const view = await ensureEditorReady({
      getEditorView: () => editorView,
      setEditorMode: (mode) => ui.setEditorMode(mode),
      tick,
    });
    if (!view) return;
    insertTable(view, rows, cols);
  }

  async function handleIndent() {
    const view = await ensureEditorReady({
      getEditorView: () => editorView,
      setEditorMode: (mode) => ui.setEditorMode(mode),
      tick,
    });
    if (!view) return;
    indentSelection(view);
  }

  async function handleOutdent() {
    const view = await ensureEditorReady({
      getEditorView: () => editorView,
      setEditorMode: (mode) => ui.setEditorMode(mode),
      tick,
    });
    if (!view) return;
    outdentSelection(view);
  }

  const findReplaceHandlers = {
    getEditorView: () => editorView,
    getEditorMode: () => ui.getEditorMode(),
    getNoteId: () => noteId,
    getUrlHighlight: () => $page.url.searchParams.get('highlight'),
    setUrlHighlight: (value: string | null) => {
      const url = new URL(window.location.href);
      if (value) {
        url.searchParams.set('highlight', value);
      } else if (url.searchParams.has('highlight')) {
        url.searchParams.delete('highlight');
      }
      window.history.replaceState(window.history.state, '', url.toString());
    },
    setState: (
      partial: Partial<{
        show: boolean;
        query: string;
        showReplace: boolean;
        caseSensitive: boolean;
        pendingHighlightQuery: string | null;
        editorExtensionsReady: boolean;
        prevNoteId: string | null;
      }>
    ) => {
      if (partial.show !== undefined) showFindReplace = partial.show;
      if (partial.query !== undefined) findReplaceQuery = partial.query;
      if (partial.showReplace !== undefined) findReplaceShowReplace = partial.showReplace;
      if (partial.caseSensitive !== undefined) findReplaceCaseSensitive = partial.caseSensitive;
      if (partial.pendingHighlightQuery !== undefined)
        pendingHighlightQuery = partial.pendingHighlightQuery;
      if (partial.editorExtensionsReady !== undefined)
        editorExtensionsReady = partial.editorExtensionsReady;
      if (partial.prevNoteId !== undefined) prevNoteId = partial.prevNoteId;
    },
  };

  function getFindReplaceState(options?: {
    editorExtensionsReadyOverride?: boolean;
  }): FindReplaceState {
    return readFindReplaceState({
      show: showFindReplace,
      query: findReplaceQuery,
      showReplace: findReplaceShowReplace,
      caseSensitive: findReplaceCaseSensitive,
      pendingHighlightQuery,
      editorExtensionsReady: options?.editorExtensionsReadyOverride ?? editorExtensionsReady,
      prevNoteId,
    });
  }

  function setFindReplaceState(nextState: FindReplaceState) {
    writeFindReplaceState(nextState, {
      setShow: (value) => {
        showFindReplace = value;
      },
      setQuery: (value) => {
        findReplaceQuery = value;
      },
      setShowReplace: (value) => {
        findReplaceShowReplace = value;
      },
      setCaseSensitive: (value) => {
        findReplaceCaseSensitive = value;
      },
      setPendingHighlightQuery: (value) => {
        pendingHighlightQuery = value;
      },
      setEditorExtensionsReady: (value) => {
        editorExtensionsReady = value;
      },
      setPrevNoteId: (value) => {
        prevNoteId = value;
      },
    });
  }

  function openFindReplace(query?: string, options?: { replace?: boolean }) {
    const nextState = openFindReplaceUI(getFindReplaceState(), findReplaceHandlers, query, options);
    setFindReplaceState(nextState);
  }

  function closeFindReplace() {
    const nextState = closeFindReplaceUI(getFindReplaceState(), findReplaceHandlers);
    setFindReplaceState(nextState);
  }

  function getScrollRatio(element: HTMLElement): number {
    const maxScroll = element.scrollHeight - element.clientHeight;
    if (maxScroll <= 0) return 0;
    return Math.max(0, Math.min(1, element.scrollTop / maxScroll));
  }

  function setScrollByRatio(element: HTMLElement, ratio: number) {
    const maxScroll = element.scrollHeight - element.clientHeight;
    if (maxScroll <= 0) {
      element.scrollTop = 0;
      return;
    }
    element.scrollTop = maxScroll * Math.max(0, Math.min(1, ratio));
  }

  $effect(() => {
    const isDesktopSplit = !ui.getIsMobile() && ui.getEditorMode() === 'split';
    const previewScroller = previewScrollRef;
    const editorScroller = editorView?.scrollDOM;
    if (!isDesktopSplit || !previewScroller || !editorScroller) return;

    let syncingFromEditor = false;
    let syncingFromPreview = false;

    const syncPreviewFromEditor = () => {
      if (syncingFromPreview) return;
      syncingFromEditor = true;
      setScrollByRatio(previewScroller, getScrollRatio(editorScroller));
      requestAnimationFrame(() => {
        syncingFromEditor = false;
      });
    };

    const syncEditorFromPreview = () => {
      if (syncingFromEditor) return;
      syncingFromPreview = true;
      setScrollByRatio(editorScroller, getScrollRatio(previewScroller));
      requestAnimationFrame(() => {
        syncingFromPreview = false;
      });
    };

    // Keep preview position aligned when entering split mode.
    syncPreviewFromEditor();

    editorScroller.addEventListener('scroll', syncPreviewFromEditor, { passive: true });
    previewScroller.addEventListener('scroll', syncEditorFromPreview, { passive: true });
    return () => {
      editorScroller.removeEventListener('scroll', syncPreviewFromEditor);
      previewScroller.removeEventListener('scroll', syncEditorFromPreview);
    };
  });

  $effect(() => {
    const isDesktopSplit = !ui.getIsMobile() && ui.getEditorMode() === 'split';
    const previewScroller = previewScrollRef;
    const editorScroller = editorView?.scrollDOM;
    const renderedContentSnapshot = renderedContent;
    if (!isDesktopSplit || !previewScroller || !editorScroller) return;

    requestAnimationFrame(() => {
      void renderedContentSnapshot;
      setScrollByRatio(previewScroller, getScrollRatio(editorScroller));
    });
  });

  $effect(() => {
    maybeLoadDialog(showMoveDialog, dialogLoaders, loadMoveToFolderDialog, (next) => {
      dialogLoaders = next;
    });
  });

  $effect(() => {
    maybeLoadDialog(showVersionHistory, dialogLoaders, loadVersionHistoryDialog, (next) => {
      dialogLoaders = next;
    });
  });

  $effect(() => {
    maybeLoadDialog(ui.getMarkdownGuideOpen(), dialogLoaders, loadMarkdownGuideDialog, (next) => {
      dialogLoaders = next;
    });
  });

  $effect(() => {
    maybeLoadDialog(
      ui.getMarkdownGuideDropdownOpen(),
      dialogLoaders,
      loadMarkdownGuideDropdown,
      (next) => {
        dialogLoaders = next;
      }
    );
  });

  // Close FindReplaceBar when note changes (not on initial mount).
  // MUST be declared BEFORE the URL highlight effect: Svelte 5 runs effects
  // in declaration order, so close runs first, then re-open with ?highlight=.
  $effect(() => {
    const nextState = handleNoteChange(getFindReplaceState(), findReplaceHandlers);
    setFindReplaceState(nextState);
  });

  // URL highlight param: open FindReplaceBar when ?highlight= is present
  $effect(() => {
    const nextState = handleUrlHighlight(getFindReplaceState(), findReplaceHandlers);
    setFindReplaceState(nextState);
  });
</script>

<div class="flex flex-col h-full">
  <!-- Toolbar (fixed header, not in scroll container) -->
  {#if notes.getCurrentNote()}
    <EditorToolbar
      note={notes.getCurrentNote()}
      {editorView}
      isMobile={ui.getIsMobile()}
      editorMode={ui.getEditorMode()}
      autoSaveStatus={notes.getAutoSaveStatus()}
      autoSaveEnabled={autosave.getAutoSaveEnabled()}
      isDirty={notes.getIsDirty()}
      isSaving={notes.getIsSaving()}
      {uploading}
      {showAIActionsDropdown}
      {showMoreMenu}
      aiEnabled={notes.getCurrentNote()?.ai_enabled ?? false}
      syncing={getIsSyncing()}
      syncProgress={getSyncProgress()}
      pendingCount={getPendingCount()}
      isOnline={network.getIsOnline()}
      isEncryptionUnlocked={encryption.isEncryptionUnlocked()}
      focusModeActive={focusMode.isActive()}
      showSpellCheck={Boolean(notes.getCurrentNote()?.ai_enabled) &&
        (ui.getEditorMode() === 'edit' ||
          ui.getEditorMode() === 'split' ||
          ui.getEditorMode() === 'live')}
      onTitleInput={handleTitleInput}
      onOpenSidebar={() => ui.setSidebarOpen(true)}
      onSetEditorMode={settings.setEditorModePreference}
      onInsertTask={handleInsertTask}
      onInsertTable={handleInsertTable}
      onSave={handleSave}
      onUpload={handleUploadButtonClick}
      onShowHistory={() => (showVersionHistory = true)}
      onToggleFocus={focusMode.toggle}
      onToggleAutosave={handleAutoSaveToggle}
      onAIActions={handleAIActionsClick}
      onOpenMoreMenu={openMoreMenu}
    />
  {/if}

  <!-- Content area: flex column so editor/preview get constrained heights.
       CodeMirror and preview each scroll internally — required for Chrome PWA
       standalone mode where wheel events don't chain from .cm-scroller to
       an outer overflow-auto container. -->
  <div
    class="editor-scroll-container flex-1 {ui.getIsMobile()
      ? 'overflow-auto'
      : 'flex flex-col min-h-0'}"
  >
    {#if notes.getIsLoading() && !notes.getCurrentNote()}
      <div
        class="flex-1 flex items-center justify-center text-muted-foreground h-full"
        role="status"
        aria-live="polite"
      >
        {$_('common.loading')}
      </div>
    {:else if notes.getError()}
      <div class="flex-1 flex items-center justify-center text-destructive h-full" role="alert">
        {notes.getError()}
      </div>
    {:else if notes.getCurrentNote()}
      <!-- Editor / Preview area (relative for FindReplaceBar positioning) -->
      <div class="relative shrink-0">
        <!-- Find & Replace Bar -->
        {#if showFindReplace}
          <FindReplaceBar
            {editorView}
            initialQuery={findReplaceQuery}
            showReplace={findReplaceShowReplace}
            isReadOnly={ui.getEditorMode() === 'preview'}
            onClose={closeFindReplace}
            onQueryChange={(query, cs) => {
              findReplaceQuery = query;
              findReplaceCaseSensitive = cs;
            }}
          />
        {/if}
      </div>

      <!-- Editor / Preview area — flex-1 so it fills remaining height -->
      <div
        class="relative flex min-h-0 {ui.getIsMobile() ? '' : 'flex-1'}"
        bind:this={splitContainerRef}
      >
        {#if ui.getEditorMode() === 'live' && headings.length > 0}
          <div class="absolute inset-x-0 top-0 z-20 pointer-events-none">
            <TableOfContents {headings} onHeadingClick={handleTocClickLocal} />
          </div>
        {/if}
        <!-- Editor -->
        {#if ui.getEditorMode() === 'edit' || ui.getEditorMode() === 'split' || ui.getEditorMode() === 'live'}
          <div
            use:initEditor
            use:taskSortable={liveTaskSortableOptions}
            role="region"
            aria-label={$_('component.editor.editor_area')}
            class:flex-1={ui.getEditorMode() !== 'split'}
            class:desktop-split-editor={ui.getEditorMode() === 'split' && !ui.getIsMobile()}
            ondrop={(e) => handleEditorDrop(e, uploadImagesFromEditor)}
            ondragover={handleEditorDragOver}
            onpaste={(e) => handleEditorPaste(e, uploadImagesFromEditor)}
            style={ui.getEditorMode() === 'split'
              ? `width: ${ui.getSplitPosition()}%; min-height: 400px;`
              : ui.getIsMobile()
                ? 'min-height: 80vh;'
                : 'min-height: 400px;'}
          ></div>
        {/if}

        <!-- Split resize handle -->
        {#if ui.getEditorMode() === 'split'}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div
            class="split-resize-handle"
            class:active={isSplitResizing}
            role="separator"
            aria-orientation="vertical"
            aria-label={$_('component.editor.split_resize_handle')}
            onpointerdown={splitResizeController.onStart}
            onpointermove={splitResizeController.onMove}
            onpointerup={splitResizeController.onEnd}
            onpointercancel={splitResizeController.onEnd}
            ondblclick={splitResizeController.onDblClick}
          ></div>
        {/if}

        <!-- Preview -->
        {#if ui.getEditorMode() === 'preview' || ui.getEditorMode() === 'split'}
          <!-- Theme wrapper for preview (overflow-auto for internal scrolling) -->
          <div
            class="relative {ui.getIsMobile()
              ? ''
              : 'overflow-auto'} {ui.getEffectivePreviewThemeClass()}"
            class:flex-1={ui.getEditorMode() !== 'split'}
            style={ui.getEditorMode() === 'split' ? `width: ${100 - ui.getSplitPosition()}%;` : ''}
            bind:this={previewScrollRef}
          >
            <!-- Floating Table of Contents -->
            {#if headings.length > 0}
              <TableOfContents {headings} onHeadingClick={handleTocClickLocal} />
            {/if}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <!-- Intentional: Click handler delegates to interactive elements (wikilinks, checkboxes) in rendered markdown. These elements are natively interactive in the HTML output. -->
            <div
              class="markdown-preview"
              role="region"
              aria-label={$_('component.editor.preview_area')}
              onclick={handlePreviewClickLocal}
              use:taskCollapse={taskCollapseOptions}
              use:taskSortable={taskSortableOptions}
              use:imageResize={{ onResize: handleImageResize }}
              use:highlightSearchTerms={{
                query: showFindReplace ? findReplaceQuery : '',
                caseSensitive: findReplaceCaseSensitive,
              }}
            >
              {@html renderedContent}
            </div>
          </div>
        {/if}
      </div>

      <EditorStatusBar
        note={notes.getCurrentNote()!}
        backlinks={notes.getBacklinks()}
        {editorView}
        isMobile={ui.getIsMobile()}
        editorPanelsCollapsed={ui.getEditorPanelsCollapsed()}
        onTogglePanelsCollapsed={() => ui.toggleEditorPanelsCollapsed()}
        onInsertLink={handleInsertLink}
        onSummaryUpdated={handleSummaryUpdated}
      />
    {:else}
      <div class="flex-1 flex items-center justify-center text-muted-foreground h-full">
        {$_('component.editor.empty_state')}
      </div>
    {/if}
  </div>
</div>

<!-- All dialogs extracted to EditorDialogs sub-component -->
<EditorDialogs
  note={notes.getCurrentNote()}
  {noteId}
  {showMoveDialog}
  {dialogLoaders}
  onCloseMoveDialog={() => (showMoveDialog = false)}
  {showVersionHistory}
  onCloseVersionHistory={() => (showVersionHistory = false)}
  onVersionRestored={async () => {
    await notes.loadNote(noteId);
    toast.success($_('component.editor.version_restored'));
  }}
  markdownGuideOpen={ui.getMarkdownGuideOpen()}
  markdownGuideDropdownOpen={ui.getMarkdownGuideDropdownOpen()}
  onCloseMarkdownGuide={() => ui.setMarkdownGuideOpen(false)}
  onCloseMarkdownGuideDropdown={() => ui.setMarkdownGuideDropdownOpen(false)}
  {showColorPicker}
  onColorSelect={handleColorSelect}
  onCloseColorPicker={() => (showColorPicker = false)}
  {showMoreMenu}
  {moreMenuTriggerRect}
  editorMode={ui.getEditorMode()}
  isMobile={ui.getIsMobile()}
  aiEnabled={notes.getCurrentNote()?.ai_enabled ?? false}
  isEncrypted={notes.getCurrentNote()?.content_encrypted ?? false}
  onDelete={handleDelete}
  onMove={() => (showMoveDialog = true)}
  onExport={handleExportNote}
  onColorPicker={openColorPicker}
  onHelp={openMarkdownHelp}
  onIndent={handleIndent}
  onOutdent={handleOutdent}
  onAIToggle={handleAIToggle}
  onShare={() => (showShareDialog = true)}
  onEncryptionToggle={handleEncryptionToggle}
  onSetEditorMode={settings.setEditorModePreference}
  onCloseMoreMenu={() => (showMoreMenu = false)}
  {showAIActionsDropdown}
  {lazyDialogs}
  {aiActionsTriggerRect}
  onAIActionSelect={handleAIActionSelect}
  onCloseAIActionsDropdown={() => (showAIActionsDropdown = false)}
  {showTableInsertDialog}
  onCloseTableInsertDialog={() => (showTableInsertDialog = false)}
  onTableInsert={handleTableInsert}
  {showAITransformDialog}
  {aiTransformState}
  getCurrentContent={() => notes.getCurrentNote()?.content ?? ''}
  onApplyAITransform={applyAITransform}
  onCloseAITransformDialog={() => {
    showAITransformDialog = false;
    aiTransformState = null;
  }}
  {showShareDialog}
  onCloseShareDialog={() => (showShareDialog = false)}
/>

<!-- Hidden file input for image upload -->
<input
  type="file"
  accept="image/*"
  multiple
  bind:this={fileInput}
  onchange={handleFileInputChange}
  style="display:none"
  aria-hidden="true"
  tabindex="-1"
/>

<style>
  .split-resize-handle {
    width: 6px;
    cursor: col-resize;
    flex-shrink: 0;
    position: relative;
    background: var(--color-border, #333);
    transition: background-color var(--duration-fast) var(--ease-default);
  }

  .split-resize-handle:hover,
  .split-resize-handle.active {
    background: var(--color-primary, oklch(0.65 0.15 155));
  }

  /* Desktop split: keep only the right-side scrollbar visible (preview pane). */
  .desktop-split-editor :global(.cm-scroller) {
    scrollbar-width: none;
    -ms-overflow-style: none;
  }

  .desktop-split-editor :global(.cm-scroller::-webkit-scrollbar) {
    width: 0;
    height: 0;
    display: none;
  }
</style>
