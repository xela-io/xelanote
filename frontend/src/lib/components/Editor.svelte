<script lang="ts">
  import type { EditorView } from '@codemirror/view';
  import { Link } from 'lucide-svelte';
  import { tick } from 'svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import type { AIAction, Tag } from '$lib/api';
  import type { AITransformState } from '$lib/editor/ai-actions';
  import { ApiError } from '$lib/api';
  import { DeleteCommand } from '$lib/commands/DeleteCommand';
  import { FEATURE_FLAGS } from '$lib/config';
  import {
    createEditor,
    type EditorConfig,
    insertWikiLink,
    insertWikiLinkInContent,
    updateEditorContent,
    updateFocusMode,
  } from '$lib/editor/codemirror';
  import {
    closeFindReplace as closeFindReplaceUI,
    handleExtensionsReady,
    handleNoteChange,
    handleUrlHighlight,
    openFindReplace as openFindReplaceUI,
  } from '$lib/editor/find-replace-ui';
  import {
    handleEditorDragOver,
    handleEditorDrop,
    handleEditorPaste,
    uploadImages,
  } from '$lib/editor/image-upload';
  import { imageResize, updateImageWidthByIndex } from '$lib/editor/image-resize';
  import { applyAITransform as applyAITransformToEditor, prepareAITransform } from '$lib/editor/ai-actions';
  import {
    loadMarkdownGuideDialog,
    loadMarkdownGuideDropdown,
    loadMoveToFolderDialog,
    loadVersionHistoryDialog,
    type DialogLoaderState,
  } from '$lib/editor/dialog-loaders';
  import {
    handleSplitResizeDblClick,
    handleSplitResizeEnd,
    handleSplitResizeMove,
    handleSplitResizeStart,
  } from '$lib/editor/split-resize';
  import { indentSelection, outdentSelection } from '$lib/editor/indentation';
  import { extractHeadings, renderMarkdown } from '$lib/editor/markdown';
  import { highlightSearchTerms } from '$lib/editor/preview-highlight';
  import { taskCollapse } from '$lib/editor/task-collapse';
  import { insertTask } from '$lib/editor/task-insert';
  import { taskSortable } from '$lib/editor/task-sortable';
  import { toggleTaskByIndex } from '$lib/editor/task-toggle';
  import { getIsSyncing, getPendingCount, getSyncProgress } from '$lib/offline/sync-manager.svelte';
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
  import { calculateMoveChanges } from '$lib/utils/task-reorder';

  import AIActionsDropdown from './AIActionsDropdown.svelte';
  import AITransformDialog from './AITransformDialog.svelte';
  import ColorPickerPopover from './ColorPickerPopover.svelte';
  import EditorToolbar from './editor/EditorToolbar.svelte';
  import EditorMoreMenu from './EditorMoreMenu.svelte';
  import FindReplaceBar from './FindReplaceBar.svelte';
  import LinkSuggestionsPanel from './LinkSuggestionsPanel.svelte';
  import ShareDialog from './ShareDialog.svelte';
  import SummaryPanel from './SummaryPanel.svelte';
  import TableOfContents from './TableOfContents.svelte';
  import TagEditor from './TagEditor.svelte';
  import TagSuggestionsPanel from './TagSuggestionsPanel.svelte';

  interface Props {
    noteId: string;
  }

  const { noteId }: Props = $props();

  let editorView = $state<EditorView | undefined>(undefined);
  let renderedContent = $state('');
  let currentTags = $state<Tag[]>([]);
  let tagEditorRef: TagEditor | null = $state(null);
  let lastTaskClickTime = 0; // Timestamp-based debounce for task checkbox clicks
  let showMoveDialog = $state(false);
  let showVersionHistory = $state(false);
  let showColorPicker = $state(false);
  let showMoreMenu = $state(false);
  let showShareDialog = $state(false);
  let moreMenuTriggerRect = $state<DOMRect | null>(null);
  let showAIActionsDropdown = $state(false);
  let aiActionsTriggerRect = $state<DOMRect | null>(null);
  let showAITransformDialog = $state(false);
  let aiTransformState = $state<AITransformState | null>(null);
  // Find & Replace state
  let showFindReplace = $state(false);
  let findReplaceQuery = $state('');
  let findReplaceShowReplace = $state(false);
  let findReplaceCaseSensitive = $state(false);
  let pendingHighlightQuery = $state<string | null>(null);
  let editorExtensionsReady = $state(false);

  let dialogLoaders = $state<DialogLoaderState>({
    moveToFolderDialog: null,
    markdownGuideDialog: null,
    markdownGuideDropdown: null,
    versionHistoryDialog: null,
  });

  // Split resize state
  let isSplitResizing = $state(false);
  let splitContainerRef: HTMLDivElement | null = $state(null);

  function handleSplitResizeStartLocal(e: PointerEvent) {
    handleSplitResizeStart(e, {
      getContainerRect: () => splitContainerRef?.getBoundingClientRect() ?? null,
      setSplitPosition: (pos) => ui.setSplitPosition(pos),
      setActive: (active) => {
        isSplitResizing = active;
      },
    });
  }

  function handleSplitResizeMoveLocal(e: PointerEvent) {
    if (!isSplitResizing) return;
    handleSplitResizeMove(e, {
      getContainerRect: () => splitContainerRef?.getBoundingClientRect() ?? null,
      setSplitPosition: (pos) => ui.setSplitPosition(pos),
      setActive: (active) => {
        isSplitResizing = active;
      },
    });
  }

  function handleSplitResizeEndLocal() {
    handleSplitResizeEnd({
      getContainerRect: () => splitContainerRef?.getBoundingClientRect() ?? null,
      setSplitPosition: (pos) => ui.setSplitPosition(pos),
      setActive: (active) => {
        isSplitResizing = active;
      },
    });
  }

  function handleSplitResizeDblClickLocal() {
    handleSplitResizeDblClick({
      getContainerRect: () => splitContainerRef?.getBoundingClientRect() ?? null,
      setSplitPosition: (pos) => ui.setSplitPosition(pos),
      setActive: (active) => {
        isSplitResizing = active;
      },
    });
  }

  // Title to ID mapping for wikilink URLs
  const titleToIdMap = $derived.by(() => {
    const allNotes = notes.getNotes();
    const map = new SvelteMap<string, string>();
    for (const note of allNotes) {
      map.set(note.title.toLowerCase().trim(), note.id);
    }
    return map;
  });

  // Extract headings for TOC
  const headings = $derived.by(() => {
    const note = notes.getCurrentNote();
    if (note) {
      return extractHeadings(note.content);
    }
    return [];
  });

  // Reactive: update rendered content when note changes
  $effect(() => {
    const note = notes.getCurrentNote();
    if (note) {
      renderedContent = renderMarkdown(note.content, { titleToIdMap });
    }
  });

  // Reactive: load note when ID changes
  $effect(() => {
    if (noteId) {
      const currentNote = notes.getCurrentNote();
      // Only load if this is a different note or no note is loaded
      if (!currentNote || currentNote.id !== noteId) {
        notes.loadNote(noteId);
      }
    }
  });

  // Svelte Action: Initialisiert CodeMirror wenn das DOM-Element gerendert wird
  // Das ist das richtige Pattern für DOM-basierte Libraries wie CodeMirror
  function initEditor(node: HTMLElement) {
    const config: EditorConfig = {
      doc: notes.getCurrentNote()?.content ?? '',
      onChange: (content) => {
        notes.updateCurrentNoteContent(content);
        renderedContent = renderMarkdown(content, { titleToIdMap });

        // Schedule auto-save after content change
        notes.scheduleAutoSave();
      },
      onSave: handleSave,
      onWikilinkClick: handleWikilinkClick,
      onColorPicker: () => {
        openColorPicker();
      },
      onBeforeNewline: (view) => {
        return handleNewlineWithTaskReorder(view);
      },
      onFindReplace: (options) => {
        openFindReplace(undefined, options);
      },
      onExtensionsReady: () => {
        const nextState = handleExtensionsReady(
          {
            show: showFindReplace,
            query: findReplaceQuery,
            showReplace: findReplaceShowReplace,
            caseSensitive: findReplaceCaseSensitive,
            pendingHighlightQuery,
            editorExtensionsReady: true,
            prevNoteId,
          },
          findReplaceHandlers
        );
        showFindReplace = nextState.show;
        findReplaceQuery = nextState.query;
        findReplaceShowReplace = nextState.showReplace;
        pendingHighlightQuery = nextState.pendingHighlightQuery;
        editorExtensionsReady = true;
      },
    };

    editorView = createEditor(node, config);

    // Cleanup when element is destroyed
    return {
      destroy() {
        editorView?.destroy();
        editorView = undefined;
        editorExtensionsReady = false;
      },
    };
  }

  // Update editor when note content is loaded
  // But don't update during save operations to prevent focus loss
  $effect(() => {
    const note = notes.getCurrentNote();
    if (editorView && note && !notes.getIsDirty() && !notes.getIsLoading()) {
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
    function handleSpellCheckReplace(
      e: CustomEvent<{ from: number; to: number; replacement: string }>
    ) {
      if (!editorView) return;

      const { from, to, replacement } = e.detail;
      editorView.dispatch({
        changes: { from, to, insert: replacement },
      });
      notes.scheduleAutoSave();
    }

    // @ts-expect-error CustomEvent handler cast
    document.addEventListener('spell-check-replace', handleSpellCheckReplace);

    return () => {
      // @ts-expect-error CustomEvent handler cast
      document.removeEventListener('spell-check-replace', handleSpellCheckReplace);
    };
  });

  async function handleSave() {
    try {
      // Remember if editor had focus before save
      const hadFocus = editorView?.hasFocus;

      await notes.saveNote();

      // Restore focus after save (prevents keyboard from closing on mobile)
      if (hadFocus && editorView) {
        editorView.focus();
      }
    } catch (e) {
      if (e instanceof ApiError && e.status === 409) {
        // Version conflict detected
        const noteId = notes.getCurrentNote()?.id;
        if (noteId) {
          try {
            const latest = await api.getNote(noteId);
            toast.warning(
              $_('component.editor.conflict_warning', { values: { version: latest.version } }),
              {
                label: $_('component.editor.conflict_load_remote'),
                handler: () => notes.loadNote(noteId),
              }
            );
          } catch (fetchErr) {
            toast.error($_('component.editor.status.error_remote'));
            console.error('Failed to fetch remote version:', fetchErr);
          }
        }
      } else {
        toast.error($_('component.editor.status.error'));
        console.error('Failed to save:', e);
      }
    }
  }

  function handleTitleInput(e: Event) {
    const title = (e.currentTarget as HTMLInputElement).value;
    notes.updateCurrentNoteTitle(title);
    // Schedule auto-save after title change
    notes.scheduleAutoSave();
  }

  function handleAutoSaveToggle() {
    // Toggle auto-save
    autosave.setAutoSaveEnabled(!autosave.getAutoSaveEnabled());

    // If we just enabled auto-save and the note is dirty, schedule auto-save immediately
    if (autosave.getAutoSaveEnabled() && notes.getIsDirty()) {
      notes.scheduleAutoSave();
    }
  }

  async function handleAIToggle() {
    const currentNote = notes.getCurrentNote();
    if (!currentNote) return;

    const newValue = !currentNote.ai_enabled;

    try {
      await api.updateNoteAIEnabled(currentNote.id, newValue);
      // Update the local note state
      notes.updateCurrentNoteAIEnabled(newValue);
      // Update the tree to reflect the AI badge change
      await tree.loadTree();

      if (newValue) {
        toast.success($_('component.editor.ai_enabled_success'));
      } else {
        toast.info($_('component.editor.ai_disabled_success'));
      }
    } catch (e) {
      console.error('Failed to toggle AI enabled:', e);
      toast.error($_('component.editor.ai_toggle_error'));
    }
  }

  async function handleEncryptionToggle() {
    const currentNote = notes.getCurrentNote();
    if (!currentNote) return;

    const isCurrentlyEncrypted = currentNote.content_encrypted !== false;

    if (isCurrentlyEncrypted) {
      // Decrypt
      const confirmed = await dialog.confirm({
        title: $_('component.editor.encryption_toggle.decrypt_confirm_title'),
        message: $_('component.editor.encryption_toggle.decrypt_confirm_message'),
        confirmText: $_('component.editor.toolbar.decrypt_note'),
        cancelText: $_('dialog.cancel'),
      });
      if (!confirmed) return;
    } else {
      // Encrypt
      const confirmed = await dialog.confirm({
        title: $_('component.editor.encryption_toggle.encrypt_confirm_title'),
        message: $_('component.editor.encryption_toggle.encrypt_confirm_message'),
        confirmText: $_('component.editor.toolbar.encrypt_note'),
        cancelText: $_('dialog.cancel'),
        variant: 'danger',
      });
      if (!confirmed) return;
    }

    try {
      await notes.toggleEncryption();
      if (isCurrentlyEncrypted) {
        toast.success($_('component.editor.encryption_toggle.decrypted_success'));
      } else {
        toast.success($_('component.editor.encryption_toggle.encrypted_success'));
      }
    } catch (e) {
      console.error('Failed to toggle encryption:', e);
      toast.error($_('component.editor.encryption_toggle.error'));
    }
  }

  // Open AI Actions dropdown
  function handleAIActionsClick(rect: DOMRect) {
    const currentNote = notes.getCurrentNote();
    if (!currentNote) return;

    // Check if AI is enabled for this note
    if (!currentNote.ai_enabled) {
      toast.error($_('error.ai_transform.ai_disabled'));
      return;
    }

    // Capture button position for desktop dropdown placement
    aiActionsTriggerRect = rect;
    showAIActionsDropdown = true;
  }

  // Handle AI action selection from dropdown
  async function handleAIActionSelect(action: AIAction, customPrompt?: string) {
    showAIActionsDropdown = false;

    const currentNote = notes.getCurrentNote();
    if (!currentNote || !editorView) return;

    await prepareAITransform(action, customPrompt, {
      getCurrentContent: () => currentNote.content,
      getEditorView: () => editorView,
      setDialogOpen: (open) => {
        showAITransformDialog = open;
      },
      setTransformState: (state) => {
        aiTransformState = state;
      },
      showError: () => {
        toast.error($_('error.ai_transform.too_short'));
      },
    });
  }

  // Apply transformed content from dialog
  function applyAITransform(transformedText: string) {
    applyAITransformToEditor(editorView, aiTransformState, transformedText);

    // Schedule auto-save
    notes.scheduleAutoSave();

    // Close dialog and clear state
    showAITransformDialog = false;
    aiTransformState = null;

    toast.success($_('component.editor.ai_transform_success'));
  }

  // Menu coordination: close other menus when one opens
  function openMoreMenu(triggerRect: DOMRect) {
    showColorPicker = false;
    ui.setMarkdownGuideDropdownOpen(false);
    moreMenuTriggerRect = triggerRect;
    showMoreMenu = true;
  }

  function openColorPicker() {
    showMoreMenu = false;
    ui.setMarkdownGuideDropdownOpen(false);
    showColorPicker = true;
  }

  function openMarkdownHelp() {
    showMoreMenu = false;
    showColorPicker = false;
    ui.toggleMarkdownGuideDropdown();
  }

  function handleExportNote() {
    const currentNote = notes.getCurrentNote();
    if (!currentNote) return;

    const title = currentNote.title || 'Untitled';
    const content = currentNote.content || '';
    const markdown = `---\ntitle: "${title.replace(/"/g, '\\"')}"\n---\n\n${content}`;

    const sanitizedTitle = title.replace(/[<>:"/\\|?*]/g, '_').trim() || 'note';
    const blob = new Blob([markdown], { type: 'text/markdown;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${sanitizedTitle}.md`;
    a.click();
    URL.revokeObjectURL(url);
  }

  async function handleDelete() {
    const currentNote = notes.getCurrentNote();
    if (!currentNote) return;

    const confirmed = await dialog.confirm({
      title: $_('dialog.confirm_title'),
      message: $_('dialog.delete_note_confirm'),
      confirmText: $_('common.delete'),
      cancelText: $_('dialog.cancel'),
      variant: 'danger',
    });

    if (!confirmed) return;

    try {
      // Create snapshot for undo
      const snapshot = {
        noteId: currentNote.id,
        snapshot: {
          title: currentNote.title,
          content: currentNote.content,
          folder_path: currentNote.folder_path,
          version: currentNote.version,
        },
      };

      // Execute delete command
      const deleteCmd = new DeleteCommand(snapshot);
      const success = await history.executeCommand(deleteCmd);

      if (!success) {
        toast.error($_('component.editor.error_delete'));
        return;
      }

      // Update local state
      notes.clearCurrentNote();
      await notes.loadNotes();
      await tree.loadTree();

      // Update trash count
      trash.incrementTrashCount();

      // Show undo toast
      toast.undoToast($_('component.editor.note_trashed'), async () => {
        const success = await history.undo();
        if (success) {
          toast.success($_('component.editor.note_restored'));
          await notes.loadNotes();
          await tree.loadTree();
          trash.decrementTrashCount();
        } else {
          toast.error($_('component.editor.error_restore'));
        }
      });

      goto('/');
    } catch (e) {
      console.error('Failed to delete:', e);
      toast.error($_('component.editor.error_delete'));
    }
  }

  async function handleWikilinkClick(title: string) {
    // Try to find note by title
    const allNotes = notes.getNotes();
    const targetNote = allNotes.find((n) => n.title.toLowerCase() === title.toLowerCase());

    if (targetNote) {
      goto(`/note/${targetNote.id}`);
    } else {
      // Create new note in the same folder as the current note
      const confirmed = await dialog.confirm({
        title: $_('dialog.confirm_title'),
        message: $_('dialog.create_missing_note'),
        confirmText: $_('common.confirm'),
        cancelText: $_('dialog.cancel'),
      });

      if (confirmed) {
        const currentFolder = notes.getCurrentNote()?.folder_path || '/';
        const note = await notes.createNote(title, '', currentFolder);
        await folders.loadFolders();
        goto(`/note/${note.id}`);
      }
    }
  }

  function handlePreviewClick(e: MouseEvent) {
    const target = e.target as HTMLElement;

    // Wikilinks
    if (target.classList.contains('wikilink')) {
      e.preventDefault();
      const title = target.dataset.title;
      if (title) {
        handleWikilinkClick(title);
      }
      return;
    }

    // Task list checkboxes - handle clicks on checkbox or its label
    if (FEATURE_FLAGS.taskLists) {
      // Timestamp-based debounce: ignore clicks within 300ms of last click
      const now = Date.now();
      if (now - lastTaskClickTime < 300) {
        console.log(
          '[TaskSort] Ignoring click - debounce active (',
          now - lastTaskClickTime,
          'ms since last)'
        );
        e.preventDefault();
        return;
      }

      console.log('[TaskSort] Preview click detected, target:', target.tagName, target.className);

      const checkbox = target.matches('input.task-list-item-checkbox')
        ? (target as HTMLInputElement)
        : (target
            .closest('label')
            ?.querySelector('input.task-list-item-checkbox') as HTMLInputElement | null);

      console.log('[TaskSort] Checkbox found:', checkbox ? 'yes' : 'no');

      if (checkbox) {
        const previewContainer = checkbox.closest('.markdown-preview');
        console.log('[TaskSort] Preview container found:', previewContainer ? 'yes' : 'no');

        if (previewContainer) {
          const taskItem = checkbox.closest('li.task-list-item');
          const checkboxIndex = taskItem
            ? parseInt(taskItem.getAttribute('data-task-index') || '-1', 10)
            : -1;
          console.log('[TaskSort] Checkbox index:', checkboxIndex, 'checked:', checkbox.checked);

          if (checkboxIndex !== -1) {
            // Update timestamp before processing
            lastTaskClickTime = now;
            toggleTask(checkboxIndex, checkbox.checked);
          }
        }
      }
    }
  }

  function handleTocClick(slug: string) {
    // Find the heading element in the preview container
    const previewContainer = document.querySelector('.markdown-preview');
    if (previewContainer) {
      const heading = previewContainer.querySelector(`#${CSS.escape(slug)}`);
      if (heading) {
        heading.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }
    }
  }

  /**
   * Handle Enter key in task lists.
   * Currently defers to default CodeMirror behavior for all cases.
   * Task reordering (checked → bottom) happens on checkbox toggle, not on Enter.
   */
  function handleNewlineWithTaskReorder(_view: import('@codemirror/view').EditorView): boolean {
    return false;
  }

  function toggleTask(checkboxIndex: number, checked: boolean) {
    const cn = notes.getCurrentNote();
    toggleTaskByIndex({
      editorView,
      checkboxIndex,
      checked,
      getContent: () => cn?.content ?? '',
      setContent: (content) => notes.updateCurrentNoteContent(content),
      scheduleAutoSave: () => notes.scheduleAutoSave(),
      queueTaskEvent: (noteId, taskText, index, status) =>
        notes.queueTaskEvent(noteId, taskText, index, status),
      noteId: cn?.id,
      log: console.log,
    });
  }

  // Task Drag & Drop Reorder Handler
  function handleTaskReorder(fromTaskIndex: number, toTaskIndex: number) {
    if (!editorView) return;

    const doc = editorView.state.doc;
    const changes = calculateMoveChanges(doc, fromTaskIndex, toTaskIndex);

    if (changes.length > 0) {
      editorView.dispatch({ changes, scrollIntoView: true });
      notes.scheduleAutoSave();
    }
  }

  // Image Resize Handler
  function handleImageResize(imageIndex: number, newWidth: number) {
    // Get content from CodeMirror or notes store
    const content = editorView
      ? editorView.state.doc.toString()
      : (notes.getCurrentNote()?.content ?? '');

    const newContent = updateImageWidthByIndex(content, imageIndex, newWidth);

    if (editorView) {
      // Update via CodeMirror dispatch
      editorView.dispatch({
        changes: { from: 0, to: editorView.state.doc.length, insert: newContent },
      });
    } else {
      // Update via notes store (preview mode)
      notes.updateCurrentNoteContent(newContent);
    }

    notes.scheduleAutoSave();
  }

  // Upload Logic
  let uploading = $state(false);

  async function uploadImagesFromEditor(files: File[]) {
    await uploadImages(files, {
      editorView,
      onStatus: (value) => {
        uploading = value;
      },
      onSuccess: (_event, ctx) => {
        toast.success($_('component.editor.upload_success', { values: { filename: ctx?.filename } }));
      },
      onWarning: (event, ctx) => {
        if (event === 'copied_to_clipboard') {
          toast.warning($_('component.editor.upload_clipboard'));
        } else {
          toast.warning($_('component.editor.upload_fallback', { values: { url: ctx?.url } }));
        }
      },
      onError: (_event, ctx) => {
        toast.error(
          $_('component.editor.status.error_upload', {
            values: {
              filename: ctx?.filename,
              error: ctx?.error,
            },
          })
        );
      },
    });
  }

  function handleColorSelect(color: string) {
    if (!editorView) return;

    const selection = editorView.state.selection.main;
    const selectedText = editorView.state.doc.sliceString(selection.from, selection.to);

    // Wrap selected text in color syntax, or insert placeholder if no selection
    const text = selectedText || 'Text';
    const openTag = `{color:${color}}`;
    const closeTag = '{/color}';
    const fullText = `${openTag}${text}${closeTag}`;

    editorView.dispatch({
      changes: {
        from: selection.from,
        to: selection.to,
        insert: fullText,
      },
      // Position cursor after opening tag if no text was selected
      selection: selectedText
        ? { anchor: selection.from + fullText.length }
        : {
            anchor: selection.from + openTag.length,
            head: selection.from + openTag.length + text.length,
          },
    });

    // Focus editor
    editorView.focus();
  }

  // Upload Button Handler
  let fileInput: HTMLInputElement;

  function handleUploadButtonClick() {
    fileInput.click();
  }

  function handleFileInputChange(e: Event) {
    const files = Array.from((e.target as HTMLInputElement).files || []);
    if (files.length > 0) {
      uploadImagesFromEditor(files);
    }
    // Reset input
    (e.target as HTMLInputElement).value = '';
  }

  async function handleInsertTask() {
    // Switch to edit mode if in preview (e.g. on mobile)
    if (!editorView) {
      ui.setEditorMode('edit');
      await tick(); // Wait for editor DOM to render and initEditor action to run
      if (!editorView) return; // Editor still not ready
    }
    insertTask(editorView);
  }

  async function handleIndent() {
    if (!editorView) {
      ui.setEditorMode('edit');
      await tick();
      if (!editorView) return;
    }
    indentSelection(editorView);
  }

  async function handleOutdent() {
    if (!editorView) {
      ui.setEditorMode('edit');
      await tick();
      if (!editorView) return;
    }
    outdentSelection(editorView);
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
    setState: (partial: Partial<{
      show: boolean;
      query: string;
      showReplace: boolean;
      caseSensitive: boolean;
      pendingHighlightQuery: string | null;
      editorExtensionsReady: boolean;
      prevNoteId: string | null;
    }>) => {
      if (partial.show !== undefined) showFindReplace = partial.show;
      if (partial.query !== undefined) findReplaceQuery = partial.query;
      if (partial.showReplace !== undefined) findReplaceShowReplace = partial.showReplace;
      if (partial.caseSensitive !== undefined) findReplaceCaseSensitive = partial.caseSensitive;
      if (partial.pendingHighlightQuery !== undefined) pendingHighlightQuery = partial.pendingHighlightQuery;
      if (partial.editorExtensionsReady !== undefined) editorExtensionsReady = partial.editorExtensionsReady;
      if (partial.prevNoteId !== undefined) prevNoteId = partial.prevNoteId;
    },
  };

  function openFindReplace(query?: string, options?: { replace?: boolean }) {
    const nextState = openFindReplaceUI(
      {
        show: showFindReplace,
        query: findReplaceQuery,
        showReplace: findReplaceShowReplace,
        caseSensitive: findReplaceCaseSensitive,
        pendingHighlightQuery,
        editorExtensionsReady,
        prevNoteId,
      },
      findReplaceHandlers,
      query,
      options
    );
    showFindReplace = nextState.show;
    findReplaceQuery = nextState.query;
    findReplaceShowReplace = nextState.showReplace;
  }

  function closeFindReplace() {
    const nextState = closeFindReplaceUI(
      {
        show: showFindReplace,
        query: findReplaceQuery,
        showReplace: findReplaceShowReplace,
        caseSensitive: findReplaceCaseSensitive,
        pendingHighlightQuery,
        editorExtensionsReady,
        prevNoteId,
      },
      findReplaceHandlers
    );
    showFindReplace = nextState.show;
    findReplaceQuery = nextState.query;
    findReplaceShowReplace = nextState.showReplace;
  }

  $effect(() => {
    if (showMoveDialog) {
      loadMoveToFolderDialog(dialogLoaders).then((next) => {
        dialogLoaders = next;
      });
    }
  });

  $effect(() => {
    if (showVersionHistory) {
      loadVersionHistoryDialog(dialogLoaders).then((next) => {
        dialogLoaders = next;
      });
    }
  });

  $effect(() => {
    if (ui.getMarkdownGuideOpen()) {
      loadMarkdownGuideDialog(dialogLoaders).then((next) => {
        dialogLoaders = next;
      });
    }
  });

  $effect(() => {
    if (ui.getMarkdownGuideDropdownOpen()) {
      loadMarkdownGuideDropdown(dialogLoaders).then((next) => {
        dialogLoaders = next;
      });
    }
  });

  // Close FindReplaceBar when note changes (not on initial mount).
  // MUST be declared BEFORE the URL highlight effect: Svelte 5 runs effects
  // in declaration order, so close runs first, then re-open with ?highlight=.
  let prevNoteId: string | null = null;
  $effect(() => {
    const nextState = handleNoteChange(
      {
        show: showFindReplace,
        query: findReplaceQuery,
        showReplace: findReplaceShowReplace,
        caseSensitive: findReplaceCaseSensitive,
        pendingHighlightQuery,
        editorExtensionsReady,
        prevNoteId,
      },
      findReplaceHandlers
    );
    showFindReplace = nextState.show;
    findReplaceQuery = nextState.query;
    findReplaceShowReplace = nextState.showReplace;
    prevNoteId = nextState.prevNoteId;
  });

  // URL highlight param: open FindReplaceBar when ?highlight= is present
  $effect(() => {
    const nextState = handleUrlHighlight(
      {
        show: showFindReplace,
        query: findReplaceQuery,
        showReplace: findReplaceShowReplace,
        caseSensitive: findReplaceCaseSensitive,
        pendingHighlightQuery,
        editorExtensionsReady,
        prevNoteId,
      },
      findReplaceHandlers
    );
    showFindReplace = nextState.show;
    findReplaceQuery = nextState.query;
    findReplaceShowReplace = nextState.showReplace;
    pendingHighlightQuery = nextState.pendingHighlightQuery;
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
        (ui.getEditorMode() === 'edit' || ui.getEditorMode() === 'split')}
      onTitleInput={handleTitleInput}
      onOpenSidebar={() => ui.setSidebarOpen(true)}
      onSetEditorMode={settings.setEditorModePreference}
      onInsertTask={handleInsertTask}
      onSave={handleSave}
      onUpload={handleUploadButtonClick}
      onShowHistory={() => (showVersionHistory = true)}
      onToggleFocus={focusMode.toggle}
      onToggleAutosave={handleAutoSaveToggle}
      onAIActions={handleAIActionsClick}
      onOpenMoreMenu={openMoreMenu}
    />
  {/if}

  <!-- Content area (scrollable container for editor, backlinks, and tags) -->
  <div class="flex-1 overflow-auto">
    {#if notes.getIsLoading() && !notes.getCurrentNote()}
      <div
        class="flex-1 flex items-center justify-center text-muted-foreground h-full"
        role="status"
        aria-live="polite"
      >
        {$_('common.loading')}
      </div>
    {:else if notes.getError()}
      <div class="flex-1 flex items-center justify-center text-destructive h-full">
        {notes.getError()}
      </div>
    {:else if notes.getCurrentNote()}
      <!-- Editor / Preview area (relative for FindReplaceBar positioning) -->
      <div class="relative">
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

      <!-- Editor / Preview area -->
      <div class="flex min-h-0" bind:this={splitContainerRef}>
        <!-- Editor -->
        {#if ui.getEditorMode() === 'edit' || ui.getEditorMode() === 'split'}
          <div
            use:initEditor
            role="region"
            aria-label={$_('component.editor.editor_area')}
            class:flex-1={ui.getEditorMode() !== 'split'}
            ondrop={(e) => handleEditorDrop(e, uploadImagesFromEditor)}
            ondragover={handleEditorDragOver}
            onpaste={(e) => handleEditorPaste(e, uploadImagesFromEditor)}
            style={ui.getEditorMode() === 'split'
              ? `width: ${ui.getSplitPosition()}%; min-height: 400px;`
              : 'min-height: 400px;'}
          ></div>
        {/if}

        <!-- Split resize handle -->
        {#if ui.getEditorMode() === 'split'}
          <!-- svelte-ignore a11y_no_static_element_interactions -->
          <div
            class="split-resize-handle"
            class:active={isSplitResizing}
            onpointerdown={handleSplitResizeStartLocal}
            onpointermove={handleSplitResizeMoveLocal}
            onpointerup={handleSplitResizeEndLocal}
            onpointercancel={handleSplitResizeEndLocal}
            ondblclick={handleSplitResizeDblClickLocal}
          ></div>
        {/if}

        <!-- Preview -->
        {#if ui.getEditorMode() === 'preview' || ui.getEditorMode() === 'split'}
          <!-- Theme wrapper for preview -->
          <div
            class="relative {ui.getEffectivePreviewThemeClass()}"
            class:flex-1={ui.getEditorMode() !== 'split'}
            style={ui.getEditorMode() === 'split'
              ? `width: ${100 - ui.getSplitPosition()}%; min-height: 400px;`
              : 'min-height: 400px;'}
          >
            <!-- Floating Table of Contents -->
            {#if headings.length > 0}
              <TableOfContents {headings} onHeadingClick={handleTocClick} />
            {/if}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <!-- Intentional: Click handler delegates to interactive elements (wikilinks, checkboxes) in rendered markdown. These elements are natively interactive in the HTML output. -->
            {#key renderedContent}
              <div
                class="markdown-preview h-full"
                onclick={handlePreviewClick}
                use:taskCollapse={{
                  completedLabel: (n) =>
                    $_('component.editor.completed_count', { values: { count: n } }),
                  completedAriaLabel: (n) =>
                    $_('component.editor.completed_toggle', { values: { count: n } }),
                  noteId,
                }}
                use:taskSortable={{ onReorder: handleTaskReorder }}
                use:imageResize={{ onResize: handleImageResize }}
                use:highlightSearchTerms={{
                  query: showFindReplace ? findReplaceQuery : '',
                  caseSensitive: findReplaceCaseSensitive,
                }}
              >
                {@html renderedContent}
              </div>
            {/key}
          </div>
        {/if}
      </div>

      <!-- Backlinks panel -->
      {#if notes.getBacklinks().length > 0}
        <div class="border-t border-border p-4">
          <h3 class="text-sm font-medium flex items-center gap-2 mb-2">
            <Link size={14} />
            {$_('component.editor.backlinks_title', {
              values: { count: notes.getBacklinks().length },
            })}
          </h3>
          <div class="flex flex-wrap gap-2">
            {#each notes.getBacklinks() as backlink (backlink.id)}
              <a
                href="/note/{backlink.id}"
                class="text-sm px-2 py-1 bg-accent rounded-md hover:bg-accent/80"
              >
                {backlink.title}
              </a>
            {/each}
          </div>
        </div>
      {/if}

      <!-- Table of Contents is now rendered inside the preview container -->

      <!-- AI Summary Panel -->
      {#if notes.getCurrentNote()}
        <div class="border-t border-border p-4">
          <SummaryPanel
            note={notes.getCurrentNote()!}
            decryptedContent={notes.getCurrentNote()!.content_encrypted
              ? notes.getCurrentNote()!.content
              : undefined}
            onSummaryUpdated={(summary) => {
              // Update the note in the store with the new summary
              const currentNote = notes.getCurrentNote();
              if (currentNote) {
                currentNote.summary = summary;
                currentNote.summary_generated_at = new Date().toISOString();
              }
            }}
          />
        </div>
      {/if}

      <!-- Tag Suggestions Panel -->
      {#if notes.getCurrentNote() && FEATURE_FLAGS.tagSuggestions}
        <div class="border-t border-border p-4">
          <TagSuggestionsPanel
            noteId={notes.getCurrentNote()!.id}
            isEncrypted={notes.getCurrentNote()!.content_encrypted || false}
            plaintextContent={notes.getCurrentNote()!.content_encrypted
              ? notes.getCurrentNote()!.content
              : undefined}
            existingTagNames={currentTags.map((t) => t.name)}
            onAddTag={async (tagName) => {
              if (tagEditorRef) {
                tagEditorRef.setInputValue(tagName);
                tagEditorRef.focusInput();
              }
            }}
          />
        </div>
      {/if}

      <!-- Link Suggestions Panel -->
      {#if notes.getCurrentNote() && FEATURE_FLAGS.linkSuggestions}
        <div class="border-t border-border p-4">
          <LinkSuggestionsPanel
            noteId={notes.getCurrentNote()!.id}
            isEncrypted={notes.getCurrentNote()!.content_encrypted || false}
            plaintextContent={notes.getCurrentNote()!.content}
            onInsertLink={(term, targetTitle) => {
              if (editorView) {
                // Editor mode: use CodeMirror
                insertWikiLink(editorView, term, targetTitle);
                notes.scheduleAutoSave();
              } else {
                // Preview mode: modify content directly
                const content = notes.getCurrentNote()?.content || '';
                const { newContent, found } = insertWikiLinkInContent(content, term, targetTitle);
                if (found) {
                  notes.updateCurrentNoteContent(newContent);
                  notes.scheduleAutoSave();
                }
              }
            }}
          />
        </div>
      {/if}

      <!-- Tag editor panel -->
      <div class="border-t border-border p-4">
        <TagEditor
          bind:this={tagEditorRef}
          noteId={notes.getCurrentNote()!.id}
          onTagsChanged={(tags) => {
            currentTags = tags;
          }}
        />
      </div>
    {:else}
      <div class="flex-1 flex items-center justify-center text-muted-foreground h-full">
        {$_('component.editor.empty_state')}
      </div>
    {/if}
  </div>
</div>

<!-- Move to folder dialog -->
{#if showMoveDialog && notes.getCurrentNote()}
  {#if dialogLoaders.moveToFolderDialog}
    <svelte:component
      this={dialogLoaders.moveToFolderDialog}
      noteId={notes.getCurrentNote()!.id}
      currentFolder={notes.getCurrentNote()!.folder_path}
      onClose={() => (showMoveDialog = false)}
    />
  {/if}
{/if}

<!-- Markdown Guide Dropdown -->
{#if ui.getMarkdownGuideDropdownOpen() && dialogLoaders.markdownGuideDropdown}
  <svelte:component
    this={dialogLoaders.markdownGuideDropdown}
    onClose={() => ui.setMarkdownGuideDropdownOpen(false)}
  />
{/if}

<!-- Markdown Guide Dialog -->
{#if ui.getMarkdownGuideOpen() && dialogLoaders.markdownGuideDialog}
  <svelte:component
    this={dialogLoaders.markdownGuideDialog}
    onClose={() => ui.setMarkdownGuideOpen(false)}
  />
{/if}

<!-- Version History Dialog -->
{#if showVersionHistory && notes.getCurrentNote() && dialogLoaders.versionHistoryDialog}
  <svelte:component
    this={dialogLoaders.versionHistoryDialog}
    noteId={notes.getCurrentNote()!.id}
    noteTitle={notes.getCurrentNote()!.title}
    currentVersion={notes.getCurrentNote()!.version}
    currentContent={notes.getCurrentNote()!.content}
    onClose={() => (showVersionHistory = false)}
    onRestored={async () => {
      await notes.loadNote(noteId);
      toast.success($_('component.editor.version_restored'));
    }}
  />
{/if}

<!-- Color Picker Popover -->
{#if showColorPicker && FEATURE_FLAGS.colorSyntax}
  <ColorPickerPopover onSelect={handleColorSelect} onClose={() => (showColorPicker = false)} />
{/if}

<!-- More Menu (rendered outside toolbar to avoid overflow clipping) -->
{#if showMoreMenu}
  <EditorMoreMenu
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
    aiEnabled={notes.getCurrentNote()?.ai_enabled ?? false}
    isEncrypted={notes.getCurrentNote()?.content_encrypted ?? false}
    onClose={() => (showMoreMenu = false)}
    triggerRect={moreMenuTriggerRect}
  />
{/if}

<!-- AI Actions Dropdown (rendered outside toolbar to avoid overflow clipping) -->
{#if showAIActionsDropdown}
  <AIActionsDropdown
    onSelectAction={handleAIActionSelect}
    onClose={() => (showAIActionsDropdown = false)}
    triggerRect={aiActionsTriggerRect}
  />
{/if}

<!-- AI Transform Dialog -->
{#if showAITransformDialog && notes.getCurrentNote() && aiTransformState}
  <AITransformDialog
    noteId={notes.getCurrentNote()!.id}
    action={aiTransformState.action}
    customPrompt={aiTransformState.customPrompt}
    originalText={aiTransformState.originalText}
    selectionFrom={aiTransformState.selectionFrom}
    selectionTo={aiTransformState.selectionTo}
    isFullContent={aiTransformState.isFullContent}
    initialContentHash={aiTransformState.initialContentHash}
    getCurrentContent={() => notes.getCurrentNote()?.content ?? ''}
    onApply={applyAITransform}
    onClose={() => {
      showAITransformDialog = false;
      aiTransformState = null;
    }}
  />
{/if}

<!-- Share Note Dialog -->
{#if showShareDialog && notes.getCurrentNote()}
  <ShareDialog
    resourceType="note"
    resourceId={notes.getCurrentNote()!.id}
    isEncrypted={notes.getCurrentNote()!.content_encrypted ?? false}
    onClose={() => (showShareDialog = false)}
  />
{/if}

<!-- Hidden file input for image upload -->
<input
  type="file"
  accept="image/*"
  multiple
  bind:this={fileInput}
  onchange={handleFileInputChange}
  style="display:none"
/>

<style>
  .split-resize-handle {
    width: 6px;
    cursor: col-resize;
    flex-shrink: 0;
    position: relative;
    background: var(--color-border, #333);
    transition: background-color 0.15s ease;
  }

  .split-resize-handle:hover,
  .split-resize-handle.active {
    background: var(--color-primary, oklch(0.65 0.15 155));
  }
</style>
