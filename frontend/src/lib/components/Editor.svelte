<script lang="ts">
  import type { Component } from 'svelte';
  import { goto } from '$app/navigation';
  import { tick } from 'svelte';
  import { _ } from 'svelte-i18n';
  import {
    createEditor,
    updateEditorContent,
    updateFocusMode,
    insertWikiLink,
    insertWikiLinkInContent,
    type EditorConfig,
  } from '$lib/editor/codemirror';
  import { SvelteMap } from 'svelte/reactivity';
  import { renderMarkdown, extractHeadings } from '$lib/editor/markdown';
  import * as notes from '$lib/stores/notes.svelte';
  import * as folders from '$lib/stores/folders.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import * as autosave from '$lib/stores/autosave.svelte';
  import * as toast from '$lib/stores/toast.svelte';
  import * as trash from '$lib/stores/trash.svelte';
  import * as tree from '$lib/stores/tree.svelte';
  import * as history from '$lib/stores/history.svelte';
  import * as dialog from '$lib/stores/dialog.svelte';
  import * as focusMode from '$lib/stores/focus-mode.svelte';
  import { DeleteCommand } from '$lib/commands/DeleteCommand';
  import * as api from '$lib/api';
  import { ApiError } from '$lib/api';
  import {
    Edit,
    Eye,
    Columns,
    Save,
    Link,
    ImagePlus,
    Check,
    AlertCircle,
    Loader2,
    History,
    ListTodo,
    MoreVertical,
    Maximize2,
    Minimize2,
    Wand2,
    Menu,
    WifiOff,
    RefreshCw,
    Lock,
  } from 'lucide-svelte';
  import * as network from '$lib/stores/network.svelte';
  import { getIsSyncing, getSyncProgress, getPendingCount } from '$lib/offline/sync-manager.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';
  import type { EditorView } from '@codemirror/view';
  import { FEATURE_FLAGS } from '$lib/config';
  import ColorPickerPopover from './ColorPickerPopover.svelte';
  import EditorMoreMenu from './EditorMoreMenu.svelte';
  import ShareDialog from './ShareDialog.svelte';
  import TagEditor from './TagEditor.svelte';
  import TagSuggestionsPanel from './TagSuggestionsPanel.svelte';
  import LinkSuggestionsPanel from './LinkSuggestionsPanel.svelte';
  import SpellCheckToggle from './SpellCheckToggle.svelte';
  import FindReplaceBar from './FindReplaceBar.svelte';
  import { sanitizeSearchQuery, clearSearch } from '$lib/editor/find-replace';
  import { highlightSearchTerms } from '$lib/editor/preview-highlight';
  import AIActionsDropdown from './AIActionsDropdown.svelte';
  import AITransformDialog from './AITransformDialog.svelte';
  import type { Tag, AIAction } from '$lib/api';
  import Breadcrumb from './Breadcrumb.svelte';
  import TableOfContents from './TableOfContents.svelte';
  import SummaryPanel from './SummaryPanel.svelte';

  import { page } from '$app/stores';
  import { taskCollapse } from '$lib/editor/task-collapse';
  import { taskSortable } from '$lib/editor/task-sortable';
  import { imageResize, updateImageWidthByIndex } from '$lib/editor/image-resize';
  import { calculateMoveChanges } from '$lib/utils/task-reorder';

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
  let moreMenuButtonRef: HTMLButtonElement | null = $state(null);
  let moreMenuTriggerRect = $state<DOMRect | null>(null);
  let showAIActionsDropdown = $state(false);
  let aiActionsButtonRef: HTMLButtonElement | null = $state(null);
  let aiActionsTriggerRect = $state<DOMRect | null>(null);
  let showAITransformDialog = $state(false);
  let aiTransformState = $state<{
    action: AIAction;
    customPrompt?: string;
    originalText: string;
    selectionFrom: number;
    selectionTo: number;
    isFullContent: boolean;
    initialContentHash: string;
  } | null>(null);
  // Find & Replace state
  let showFindReplace = $state(false);
  let findReplaceQuery = $state('');
  let findReplaceShowReplace = $state(false);
  let findReplaceCaseSensitive = $state(false);
  let pendingHighlightQuery = $state<string | null>(null);
  let editorExtensionsReady = $state(false);

  let MoveToFolderDialogComponent = $state<Component<Record<string, unknown>> | null>(null);
  let markdownGuideDialogComponent = $state<Component<Record<string, unknown>> | null>(null);
  let MarkdownGuideDropdownComponent = $state<Component<Record<string, unknown>> | null>(null);
  let VersionHistoryDialogComponent = $state<Component<Record<string, unknown>> | null>(null);

  // Split resize state
  let isSplitResizing = $state(false);
  let splitContainerRef: HTMLDivElement | null = $state(null);

  function handleSplitResizeStart(e: PointerEvent) {
    e.preventDefault();
    isSplitResizing = true;
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
    document.body.style.userSelect = 'none';
    document.body.style.cursor = 'col-resize';
  }

  function handleSplitResizeMove(e: PointerEvent) {
    if (!isSplitResizing || !splitContainerRef) return;
    const rect = splitContainerRef.getBoundingClientRect();
    const pos = ((e.clientX - rect.left) / rect.width) * 100;
    ui.setSplitPosition(pos);
  }

  function handleSplitResizeEnd() {
    isSplitResizing = false;
    document.body.style.userSelect = '';
    document.body.style.cursor = '';
  }

  function handleSplitResizeDblClick() {
    ui.setSplitPosition(50);
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

  // Svelte Action: Scroll-Fade for toolbar overflow indicator
  function scrollFade(node: HTMLElement) {
    const wrapper = node.parentElement!;
    function update() {
      const hasOverflow = node.scrollWidth > node.clientWidth;
      const atEnd = node.scrollLeft + node.clientWidth >= node.scrollWidth - 2;
      wrapper.style.setProperty('--scroll-fade', hasOverflow && !atEnd ? '1' : '0');
    }
    update();
    node.addEventListener('scroll', update, { passive: true });
    const ro = new ResizeObserver(update);
    ro.observe(node);
    return {
      destroy() {
        node.removeEventListener('scroll', update);
        ro.disconnect();
      },
    };
  }

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
        editorExtensionsReady = true;
        if (pendingHighlightQuery) {
          openFindReplace(pendingHighlightQuery);
          pendingHighlightQuery = null;
        }
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

  // Client-side hash for conflict detection (first 16 hex chars of SHA-256)
  async function computeContentHash(content: string): Promise<string> {
    const encoder = new TextEncoder();
    const data = encoder.encode(content);
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray
      .slice(0, 8)
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');
  }

  // Open AI Actions dropdown
  function handleAIActionsClick() {
    const currentNote = notes.getCurrentNote();
    if (!currentNote) return;

    // Check if AI is enabled for this note
    if (!currentNote.ai_enabled) {
      toast.error($_('error.ai_transform.ai_disabled'));
      return;
    }

    // Capture button position for desktop dropdown placement
    if (aiActionsButtonRef) {
      aiActionsTriggerRect = aiActionsButtonRef.getBoundingClientRect();
    }
    showAIActionsDropdown = true;
  }

  // Handle AI action selection from dropdown
  async function handleAIActionSelect(action: AIAction, customPrompt?: string) {
    showAIActionsDropdown = false;

    const currentNote = notes.getCurrentNote();
    if (!currentNote || !editorView) return;

    // Get selection or full content
    const selection = editorView.state.selection.main;
    const hasSelection = selection.from !== selection.to;

    let textToTransform: string;
    let selectionFrom: number;
    let selectionTo: number;
    let isFullContent: boolean;

    if (hasSelection) {
      // Transform selected text
      textToTransform = editorView.state.doc.sliceString(selection.from, selection.to);
      selectionFrom = selection.from;
      selectionTo = selection.to;
      isFullContent = false;
    } else {
      // Transform entire note content
      textToTransform = currentNote.content;
      selectionFrom = 0;
      selectionTo = editorView.state.doc.length;
      isFullContent = true;
    }

    // Validate content length
    if (textToTransform.trim().length < 10) {
      toast.error($_('error.ai_transform.too_short'));
      return;
    }

    // Compute hash for conflict detection
    const initialContentHash = await computeContentHash(currentNote.content);

    // Store state and open dialog
    aiTransformState = {
      action,
      customPrompt,
      originalText: textToTransform,
      selectionFrom,
      selectionTo,
      isFullContent,
      initialContentHash,
    };
    showAITransformDialog = true;
  }

  // Apply transformed content from dialog
  function applyAITransform(transformedText: string) {
    if (!editorView || !aiTransformState) return;

    const { selectionFrom, selectionTo } = aiTransformState;

    editorView.dispatch({
      changes: {
        from: selectionFrom,
        to: selectionTo,
        insert: transformedText,
      },
    });

    // Schedule auto-save
    notes.scheduleAutoSave();

    // Close dialog and clear state
    showAITransformDialog = false;
    aiTransformState = null;

    toast.success($_('component.editor.ai_transform_success'));
  }

  // Menu coordination: close other menus when one opens
  function openMoreMenu() {
    showColorPicker = false;
    ui.setMarkdownGuideDropdownOpen(false);
    if (moreMenuButtonRef) {
      moreMenuTriggerRect = moreMenuButtonRef.getBoundingClientRect();
    }
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
            toggleTaskByIndex(checkboxIndex, checkbox.checked);
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

  // Types for task sorting
  interface TaskInfo {
    lineNum: number;
    from: number;
    to: number;
    isChecked: boolean;
    lineFrom: number;
    lineTo: number;
    indent: string;
  }

  interface ListBoundary {
    startLine: number;
    endLine: number;
  }

  /**
   * Handle Enter key in task lists.
   * Currently defers to default CodeMirror behavior for all cases.
   * Task reordering (checked → bottom) happens on checkbox toggle, not on Enter.
   */
  function handleNewlineWithTaskReorder(_view: import('@codemirror/view').EditorView): boolean {
    return false;
  }

  /**
   * Find the boundaries of a task list containing the given line.
   * A task list ends when: empty line, heading, or document boundary.
   * This finds ALL contiguous list items, regardless of indent.
   */
  function findTaskListBoundary(
    doc: import('@codemirror/state').Text,
    taskLineNum: number
  ): ListBoundary {
    let startLine = taskLineNum;
    let endLine = taskLineNum;

    // Scan upward - find all contiguous list items
    for (let i = taskLineNum - 1; i >= 1; i--) {
      const line = doc.line(i);
      const text = line.text;

      // Empty line or heading = boundary
      if (text.trim() === '' || /^#{1,6}\s/.test(text)) break;

      // Any list item continues the list
      if (/^\s*[-*+]/.test(text)) {
        startLine = i;
      } else {
        // Non-list, non-empty line = boundary
        break;
      }
    }

    // Scan downward - find all contiguous list items
    for (let i = taskLineNum + 1; i <= doc.lines; i++) {
      const line = doc.line(i);
      const text = line.text;

      // Empty line or heading = boundary
      if (text.trim() === '' || /^#{1,6}\s/.test(text)) break;

      // Any list item continues the list
      if (/^\s*[-*+]/.test(text)) {
        endLine = i;
      } else {
        // Non-list, non-empty line = boundary
        break;
      }
    }

    return { startLine, endLine };
  }

  /**
   * Calculate target position for a task after toggling.
   * Returns the line number where the task should end up.
   *
   * Algorithm:
   * 1. Create the desired order: all unchecked first, then all checked
   * 2. Find where the current task should be in this order
   * 3. Return the line number of the task currently at that position
   */
  function calculateTargetPosition(
    tasksInList: TaskInfo[],
    currentTask: TaskInfo,
    isNowChecked: boolean
  ): number {
    // Sort tasks by line number (they should already be, but ensure it)
    const sortedTasks = [...tasksInList].sort((a, b) => a.lineNum - b.lineNum);

    // Create the target order: unchecked first (preserve relative order), then checked
    // Apply the NEW checked state to the current task
    const withNewState = sortedTasks.map((t) => ({
      ...t,
      isChecked: t.lineNum === currentTask.lineNum ? isNowChecked : t.isChecked,
    }));

    const uncheckedInOrder = withNewState.filter((t) => !t.isChecked);
    const checkedInOrder = withNewState.filter((t) => t.isChecked);
    const targetOrder = [...uncheckedInOrder, ...checkedInOrder];

    // Find the index of the current task in the target order
    const targetIndex = targetOrder.findIndex((t) => t.lineNum === currentTask.lineNum);

    console.log(
      '[TaskSort] calculateTargetPosition:',
      'unchecked:',
      uncheckedInOrder.length,
      'checked:',
      checkedInOrder.length,
      'targetIndex:',
      targetIndex,
      'sortedTasks[targetIndex].lineNum:',
      sortedTasks[targetIndex].lineNum
    );

    // The target line is the line of the task currently at that index
    return sortedTasks[targetIndex].lineNum;
  }

  /**
   * Find task list boundaries from a string content (for preview mode without CodeMirror).
   */
  function findTaskListBoundaryFromString(lines: string[], taskLineIndex: number): ListBoundary {
    let startLine = taskLineIndex;
    let endLine = taskLineIndex;

    // Scan upward
    for (let i = taskLineIndex - 1; i >= 0; i--) {
      const text = lines[i];
      if (text.trim() === '' || /^#{1,6}\s/.test(text)) break;
      if (/^\s*[-*+]/.test(text)) {
        startLine = i;
      } else {
        break;
      }
    }

    // Scan downward
    for (let i = taskLineIndex + 1; i < lines.length; i++) {
      const text = lines[i];
      if (text.trim() === '' || /^#{1,6}\s/.test(text)) break;
      if (/^\s*[-*+]/.test(text)) {
        endLine = i;
      } else {
        break;
      }
    }

    // Convert to 1-based line numbers for consistency
    return { startLine: startLine + 1, endLine: endLine + 1 };
  }

  function toggleTaskByIndex(checkboxIndex: number, checked: boolean) {
    console.log('[TaskSort] toggleTaskByIndex called, index:', checkboxIndex, 'checked:', checked);

    // Get content - either from CodeMirror or from the notes store
    let content: string;
    const useEditorView = editorView != null; // Check for both null and undefined

    if (useEditorView) {
      content = editorView!.state.doc.toString();
    } else {
      // Preview mode: get content from notes store
      const currentNote = notes.getCurrentNote();
      if (!currentNote) {
        console.log('[TaskSort] No current note, returning');
        return;
      }
      content = currentNote.content;
      console.log('[TaskSort] Using content from notes store, length:', content.length);
    }

    // Find all task items in the markdown source
    // Match: -, *, + followed by [ ] or [x] or [X]
    const lines = content.split('\n');
    const tasks: TaskInfo[] = [];

    let charOffset = 0;
    for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
      const line = lines[lineIndex];
      const lineMatch = /^(\s*[-*+]\s*)\[([xX ])\]/.exec(line);
      if (lineMatch) {
        const indentMatch = line.match(/^(\s*)/);
        tasks.push({
          lineNum: lineIndex + 1, // 1-based
          from: charOffset + lineMatch[1].length,
          to: charOffset + lineMatch[1].length + 3,
          isChecked: lineMatch[2].toLowerCase() === 'x',
          lineFrom: charOffset,
          lineTo: charOffset + line.length,
          indent: indentMatch ? indentMatch[1] : '',
        });
      }
      charOffset += line.length + 1; // +1 for newline
    }

    console.log('[TaskSort] Found', tasks.length, 'tasks in document');

    if (checkboxIndex < 0 || checkboxIndex >= tasks.length) {
      console.log('[TaskSort] Invalid checkbox index, returning');
      return;
    }

    const task = tasks[checkboxIndex];
    const newCheckboxText = checked ? '[x]' : '[ ]';

    // Find list boundaries
    const boundary = useEditorView
      ? findTaskListBoundary(editorView!.state.doc, task.lineNum)
      : findTaskListBoundaryFromString(lines, task.lineNum - 1);
    console.log('[TaskSort] List boundary:', boundary);

    // Find all tasks within this list boundary
    const tasksInList = tasks.filter(
      (t) => t.lineNum >= boundary.startLine && t.lineNum <= boundary.endLine
    );
    console.log('[TaskSort] Tasks in list:', tasksInList.length);

    // Calculate target position
    const targetLineNum = calculateTargetPosition(tasksInList, task, checked);
    console.log('[TaskSort] Current line:', task.lineNum, 'Target line:', targetLineNum);

    // Check if we need to move the line
    const needsMove = targetLineNum !== task.lineNum;
    console.log('[TaskSort] Needs move:', needsMove);

    if (useEditorView) {
      // CodeMirror mode - apply changes via dispatch
      const doc = editorView!.state.doc;

      if (!needsMove) {
        editorView!.dispatch({
          changes: { from: task.from, to: task.to, insert: newCheckboxText },
        });
      } else {
        const currentLine = doc.line(task.lineNum);
        const lineText = currentLine.text;
        const toggledLineText =
          lineText.substring(0, task.from - currentLine.from) +
          newCheckboxText +
          lineText.substring(task.to - currentLine.from);

        if (targetLineNum < 1 || targetLineNum > doc.lines) {
          editorView!.dispatch({
            changes: { from: task.from, to: task.to, insert: newCheckboxText },
          });
          notes.scheduleAutoSave();
          return;
        }

        let changes: { from: number; to: number; insert: string }[];

        if (targetLineNum < task.lineNum) {
          const targetLine = doc.line(targetLineNum);
          const insertPos = targetLine.from;
          const deleteFrom = currentLine.from;
          const deleteTo = currentLine.to + (currentLine.to < doc.length ? 1 : 0);
          changes = [
            { from: insertPos, to: insertPos, insert: toggledLineText + '\n' },
            { from: deleteFrom, to: deleteTo, insert: '' },
          ];
        } else {
          const targetLine = doc.line(targetLineNum);
          let deleteFrom: number;
          let deleteTo: number;

          if (currentLine.from > 0) {
            deleteFrom = currentLine.from - 1;
            deleteTo = currentLine.to;
          } else {
            deleteFrom = currentLine.from;
            deleteTo = currentLine.to + (currentLine.to < doc.length ? 1 : 0);
          }

          const insertPos = targetLine.to;
          changes = [
            { from: deleteFrom, to: deleteTo, insert: '' },
            { from: insertPos, to: insertPos, insert: '\n' + toggledLineText },
          ];
        }

        editorView!.dispatch({
          changes,
          scrollIntoView: true,
        });
      }
    } else {
      // Preview mode - manipulate content string directly
      const currentLineIndex = task.lineNum - 1;
      const currentLineText = lines[currentLineIndex];

      // Toggle the checkbox in the line
      const toggledLineText = currentLineText.replace(/\[([xX ])\]/, newCheckboxText);

      if (!needsMove) {
        // Just toggle, no move
        lines[currentLineIndex] = toggledLineText;
      } else {
        // Remove current line and insert at target position
        const targetLineIndex = targetLineNum - 1;

        // Remove the line
        lines.splice(currentLineIndex, 1);

        // When moving down: we want to insert AFTER the target element
        // splice(index, 0, item) inserts BEFORE index, so we need targetLineIndex
        // (which after removal is the position AFTER the shifted target element)
        // When moving up: we want to insert BEFORE the target, so just use targetLineIndex
        const newTargetIndex = targetLineIndex;

        console.log(
          '[TaskSort] Moving line from index',
          currentLineIndex,
          'to index',
          newTargetIndex
        );

        // Insert at new position
        lines.splice(newTargetIndex, 0, toggledLineText);
      }

      // Update the note content
      const newContent = lines.join('\n');
      console.log('[TaskSort] Updating note content, new length:', newContent.length);
      notes.updateCurrentNoteContent(newContent);
    }

    // Queue task event for sending after next successful save
    const taskLine = lines[task.lineNum - 1];
    const taskText = taskLine.replace(/^\s*[-*+]\s*\[[xX ]\]\s*/, '').trim();
    const cn = notes.getCurrentNote();
    if (taskText && cn?.id) {
      notes.queueTaskEvent(
        cn.id,
        taskText.substring(0, 500),
        checkboxIndex,
        checked ? 'completed' : 'reopened'
      );
    }

    // Trigger auto-save
    notes.scheduleAutoSave();
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

  // Image Upload: Drag & Drop
  function handleEditorDrop(e: DragEvent) {
    const files = Array.from(e.dataTransfer?.files || []);
    const imageFiles = files.filter((f) => f.type.startsWith('image/'));

    if (imageFiles.length > 0) {
      e.preventDefault();
      uploadImages(imageFiles);
    }
  }

  function handleEditorDragOver(e: DragEvent) {
    // Check if dragging files (not notes/folders)
    const types = e.dataTransfer?.types || [];
    if (types.includes('Files')) {
      e.preventDefault();
      e.dataTransfer!.dropEffect = 'copy';
    }
  }

  // Image Upload: Paste from Clipboard
  function handleEditorPaste(e: ClipboardEvent) {
    const items = Array.from(e.clipboardData?.items || []);
    const imageItems = items.filter((item) => item.type.startsWith('image/'));

    if (imageItems.length > 0) {
      e.preventDefault();
      const files = imageItems.map((item) => item.getAsFile()).filter(Boolean) as File[];
      uploadImages(files);
    }
  }

  // Upload Logic
  let uploading = $state(false);

  async function uploadImages(files: File[]) {
    uploading = true;

    for (const file of files) {
      try {
        const { url } = await api.uploadImage(file);

        // Insert markdown at cursor
        const markdown = `\n![${file.name}](${url})\n`;
        const inserted = insertTextAtCursor(markdown);

        if (inserted) {
          toast.success($_('component.editor.upload_success', { values: { filename: file.name } }));
        } else {
          // Fallback: Copy to clipboard if editor not ready
          try {
            await navigator.clipboard.writeText(markdown);
            toast.warning($_('component.editor.upload_clipboard'));
          } catch {
            toast.warning($_('component.editor.upload_fallback', { values: { url } }));
          }
        }
      } catch (err: unknown) {
        console.error('Upload failed:', err);
        toast.error(
          $_('component.editor.status.error_upload', {
            values: {
              filename: file.name,
              error: err instanceof Error ? err.message : String(err),
            },
          })
        );
      }
    }

    uploading = false;
  }

  function insertTextAtCursor(text: string): boolean {
    if (!editorView) {
      console.warn('Editor not ready, cannot insert text');
      return false;
    }

    const selection = editorView.state.selection.main;
    editorView.dispatch({
      changes: {
        from: selection.from,
        to: selection.to,
        insert: text,
      },
      selection: { anchor: selection.from + text.length },
    });

    // Focus editor
    editorView.focus();
    return true;
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
      uploadImages(files);
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

    const doc = editorView.state.doc;
    const selection = editorView.state.selection.main;
    const cursorLine = doc.lineAt(selection.from);

    // Find the nearest task list by scanning upward from cursor
    // This handles cases where cursor is below or within a task list
    let nearestTaskListEnd = -1;
    for (let i = cursorLine.number; i >= 1; i--) {
      const line = doc.line(i);
      if (/^\s*[-*+]\s*\[[xX ]\]/.test(line.text)) {
        nearestTaskListEnd = i;
        break;
      }
    }

    // If we found a task list nearby, get its full boundaries
    const tasksInList: Array<{ lineNum: number; isChecked: boolean }> = [];

    if (nearestTaskListEnd > 0) {
      const boundary = findTaskListBoundary(doc, nearestTaskListEnd);

      // Find all tasks in this list boundary
      for (let i = boundary.startLine; i <= boundary.endLine; i++) {
        const line = doc.line(i);
        const match = /^(\s*[-*+]\s*)\[([xX ])\]/.exec(line.text);
        if (match) {
          tasksInList.push({
            lineNum: i,
            isChecked: match[2].toLowerCase() === 'x',
          });
        }
      }
    }

    // Find the first checked task
    const firstCheckedTask = tasksInList.find((t) => t.isChecked);

    // If there are checked tasks AND cursor is at or after the first checked task
    // (including below the list), insert the new task BEFORE the first checked task
    if (firstCheckedTask && cursorLine.number >= firstCheckedTask.lineNum) {
      const targetLine = doc.line(firstCheckedTask.lineNum);
      const text = '- [ ] \n';

      editorView.dispatch({
        changes: { from: targetLine.from, insert: text },
        // Position cursor at end of new task (before the newline)
        selection: { anchor: targetLine.from + text.length - 1 },
      });
    } else {
      // Original behavior: insert at cursor position
      const insertPos = selection.from === selection.to ? selection.from : cursorLine.to;

      const isAtLineStart = insertPos === cursorLine.from;
      const isEmptyLine = cursorLine.text.trim() === '';

      // Auf neuer Zeile einfügen, außer Zeile ist leer oder Cursor am Anfang
      const prefix = isAtLineStart || isEmptyLine ? '' : '\n';
      const text = `${prefix}- [ ] `;

      editorView.dispatch({
        changes: { from: insertPos, to: insertPos, insert: text },
        selection: { anchor: insertPos + text.length },
      });
    }

    editorView.focus();
  }

  async function handleIndent() {
    if (!editorView) {
      ui.setEditorMode('edit');
      await tick();
      if (!editorView) return;
    }

    const state = editorView.state;
    const selection = state.selection.main;
    const doc = state.doc;

    // Finde alle Zeilen in der Selektion
    const startLine = doc.lineAt(selection.from);
    const endLine = doc.lineAt(selection.to);

    const changes: { from: number; to: number; insert: string }[] = [];

    for (let lineNum = startLine.number; lineNum <= endLine.number; lineNum++) {
      const line = doc.line(lineNum);
      // Tab am Zeilenanfang einfügen
      changes.push({ from: line.from, to: line.from, insert: '\t' });
    }

    editorView.dispatch({ changes });
    editorView.focus();
  }

  async function handleOutdent() {
    if (!editorView) {
      ui.setEditorMode('edit');
      await tick();
      if (!editorView) return;
    }

    const state = editorView.state;
    const selection = state.selection.main;
    const doc = state.doc;

    // Finde alle Zeilen in der Selektion
    const startLine = doc.lineAt(selection.from);
    const endLine = doc.lineAt(selection.to);

    const changes: { from: number; to: number; insert: string }[] = [];

    for (let lineNum = startLine.number; lineNum <= endLine.number; lineNum++) {
      const line = doc.line(lineNum);
      const text = line.text;

      // Entferne führenden Tab oder bis zu 4 Spaces
      if (text.startsWith('\t')) {
        changes.push({ from: line.from, to: line.from + 1, insert: '' });
      } else if (text.startsWith('    ')) {
        changes.push({ from: line.from, to: line.from + 4, insert: '' });
      } else if (text.startsWith('   ')) {
        changes.push({ from: line.from, to: line.from + 3, insert: '' });
      } else if (text.startsWith('  ')) {
        changes.push({ from: line.from, to: line.from + 2, insert: '' });
      } else if (text.startsWith(' ')) {
        changes.push({ from: line.from, to: line.from + 1, insert: '' });
      }
    }

    if (changes.length > 0) {
      editorView.dispatch({ changes });
    }
    editorView.focus();
  }

  function openFindReplace(query?: string, options?: { replace?: boolean }) {
    findReplaceQuery = query ?? '';
    findReplaceShowReplace = options?.replace ?? false;

    // If editor has selection and no query provided, use selected text
    if (!query && editorView) {
      const selection = editorView.state.selection.main;
      if (selection.from !== selection.to) {
        findReplaceQuery = editorView.state.doc.sliceString(selection.from, selection.to);
      }
    }

    showFindReplace = true;
  }

  function closeFindReplace() {
    if (showFindReplace) {
      showFindReplace = false;
      findReplaceQuery = '';
      findReplaceShowReplace = false;
      if (editorView) {
        clearSearch(editorView);
      }
      // Remove ?highlight= from URL
      const url = new URL(window.location.href);
      if (url.searchParams.has('highlight')) {
        url.searchParams.delete('highlight');
        window.history.replaceState(window.history.state, '', url.toString());
      }
    }
  }

  async function loadMoveToFolderDialog() {
    if (MoveToFolderDialogComponent) return;
    const module = await import('./MoveToFolderDialog.svelte');
    MoveToFolderDialogComponent = module.default;
  }

  async function loadMarkdownGuideDialog() {
    if (markdownGuideDialogComponent) return;
    const module = await import('./MarkdownGuideDialog.svelte');
    markdownGuideDialogComponent = module.default;
  }

  async function loadMarkdownGuideDropdown() {
    if (MarkdownGuideDropdownComponent) return;
    const module = await import('./MarkdownGuideDropdown.svelte');
    MarkdownGuideDropdownComponent = module.default;
  }

  async function loadVersionHistoryDialog() {
    if (VersionHistoryDialogComponent) return;
    const { default: component } = await import('./VersionHistoryDialog.svelte');
    VersionHistoryDialogComponent = component;
  }

  $effect(() => {
    if (showMoveDialog) {
      loadMoveToFolderDialog();
    }
  });

  $effect(() => {
    if (showVersionHistory) {
      loadVersionHistoryDialog();
    }
  });

  $effect(() => {
    if (ui.getMarkdownGuideOpen()) {
      loadMarkdownGuideDialog();
    }
  });

  $effect(() => {
    if (ui.getMarkdownGuideDropdownOpen()) {
      loadMarkdownGuideDropdown();
    }
  });

  // Close FindReplaceBar when note changes (not on initial mount).
  // MUST be declared BEFORE the URL highlight effect: Svelte 5 runs effects
  // in declaration order, so close runs first, then re-open with ?highlight=.
  let prevNoteId: string | null = null;
  $effect(() => {
    if (prevNoteId !== null && prevNoteId !== noteId) {
      // Inline close without clearing pendingHighlightQuery
      showFindReplace = false;
      findReplaceQuery = '';
      findReplaceShowReplace = false;
      if (editorView) {
        clearSearch(editorView);
      }
    }
    prevNoteId = noteId;
  });

  // URL highlight param: open FindReplaceBar when ?highlight= is present
  $effect(() => {
    const query = $page.url.searchParams.get('highlight');
    if (query) {
      const sanitized = sanitizeSearchQuery(query);
      const isPreviewOnly = ui.getEditorMode() === 'preview';

      if (isPreviewOnly || (editorExtensionsReady && editorView)) {
        // Preview mode: no editor needed, just open bar for preview highlights
        // Edit/Split mode with ready editor: open immediately
        openFindReplace(sanitized);
      } else {
        // Edit/Split mode, editor not yet ready → defer until onExtensionsReady
        pendingHighlightQuery = sanitized;
      }
    }
  });
</script>

<div class="flex flex-col h-full">
  <!-- Toolbar (fixed header, not in scroll container) -->
  {#if notes.getCurrentNote()}
    <div class="flex-shrink-0 z-10 border-b border-border bg-background">
      <!-- Breadcrumb Navigation - hidden on mobile -->
      {#if !ui.getIsMobile()}
        <div class="px-4 pt-2">
          <Breadcrumb
            folderPath={notes.getCurrentNote()!.folder_path}
            noteTitle={notes.getCurrentNote()!.title}
          />
        </div>
      {/if}
      <!-- Mobile: Two-row layout | Desktop: Single-row layout -->
      <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between px-4 py-2 gap-2">
        <!-- Title (auto-sized on mobile so save icon sits next to text, limited on desktop) -->
        <div class="flex items-center gap-1 flex-shrink min-w-0 sm:max-w-[30%]">
          <input
            type="text"
            value={notes.getCurrentNote()?.title ?? ''}
            oninput={handleTitleInput}
            autocorrect="on"
            autocapitalize="words"
            spellcheck="true"
            inputmode="text"
            aria-label={$_('component.editor.title_input')}
            class="text-lg font-semibold bg-transparent border-0 outline-none focus:ring-1 focus:ring-ring rounded px-1 min-w-0 {ui.getIsMobile()
              ? ''
              : 'w-full'}"
            style={ui.getIsMobile()
              ? `width: ${Math.max((notes.getCurrentNote()?.title ?? '').length, 2) + 1}ch`
              : ''}
          />
          {#if ui.getIsMobile()}
            <span class="flex-shrink-0">
              {#if notes.getAutoSaveStatus() === 'saving'}
                <Loader2 size={16} class="animate-spin text-muted-foreground" />
              {:else if notes.getAutoSaveStatus() === 'saved'}
                <Check size={16} class="text-success" />
              {:else if notes.getAutoSaveStatus() === 'error'}
                <AlertCircle size={16} class="text-destructive" />
              {/if}
            </span>
          {/if}
        </div>

        <!-- Offline/Sync status pill -->
        {#if getIsSyncing()}
          <div
            class="flex-shrink-0 flex items-center gap-1 px-2 py-0.5 rounded-full text-xs text-white bg-blue-500"
          >
            <RefreshCw size={12} class="animate-spin" />
            <span
              >{getSyncProgress().total > 0
                ? `${getSyncProgress().current}/${getSyncProgress().total}`
                : 'Sync...'}</span
            >
          </div>
        {:else if !network.getIsOnline() && !encryption.isEncryptionUnlocked()}
          <div
            class="flex-shrink-0 flex items-center gap-1 px-2 py-0.5 rounded-full text-xs text-white bg-amber-600"
          >
            <Lock size={12} />
            <span>Gesperrt</span>
          </div>
        {:else if !network.getIsOnline() && getPendingCount() > 0}
          <div
            class="flex-shrink-0 flex items-center gap-1 px-2 py-0.5 rounded-full text-xs text-white bg-amber-500"
          >
            <WifiOff size={12} />
            <span>{getPendingCount()}</span>
          </div>
        {:else if !network.getIsOnline()}
          <div
            class="flex-shrink-0 flex items-center gap-1 px-2 py-0.5 rounded-full text-xs text-white bg-amber-500"
          >
            <WifiOff size={12} />
            <span>Offline</span>
          </div>
        {/if}

        <!-- Buttons (horizontally scrollable with fade indicator) + fixed More button -->
        <div class="flex items-center gap-1 flex-1 min-w-0">
          <div class="toolbar-scroll-wrapper">
            <div class="toolbar-buttons flex items-center gap-1" use:scrollFade>
              <!-- Sidebar toggle - always visible on mobile since MobileHeader is hidden on note pages -->
              {#if ui.getIsMobile()}
                <button
                  type="button"
                  onclick={() => ui.setSidebarOpen(true)}
                  class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
                  aria-label="Menü öffnen"
                >
                  <Menu size={16} />
                </button>
              {/if}

              <!-- Editor mode toggles - always visible -->
              <div
                class="flex rounded-md border border-border flex-shrink-0"
                role="group"
                aria-label={$_('component.editor.toolbar.mode_group')}
              >
                <button
                  type="button"
                  onclick={() => settings.setEditorModePreference('edit')}
                  class="p-2 hover:bg-accent rounded-l-md toolbar-btn"
                  class:rounded-r-md={ui.getIsMobile()}
                  class:bg-accent={ui.getEditorMode() === 'edit'}
                  aria-label={$_('component.editor.toolbar.mode_edit')}
                  aria-pressed={ui.getEditorMode() === 'edit'}
                >
                  <Edit size={16} />
                </button>
                {#if !ui.getIsMobile()}
                  <button
                    type="button"
                    onclick={() => settings.setEditorModePreference('split')}
                    class="p-2 hover:bg-accent border-x border-border toolbar-btn"
                    class:bg-accent={ui.getEditorMode() === 'split'}
                    aria-label={$_('component.editor.toolbar.mode_split')}
                    aria-pressed={ui.getEditorMode() === 'split'}
                  >
                    <Columns size={16} />
                  </button>
                {/if}
                <button
                  type="button"
                  onclick={() => settings.setEditorModePreference('preview')}
                  class="p-2 hover:bg-accent rounded-r-md toolbar-btn"
                  class:border-l={ui.getIsMobile()}
                  class:border-border={ui.getIsMobile()}
                  class:bg-accent={ui.getEditorMode() === 'preview'}
                  aria-label={$_('component.editor.toolbar.mode_preview')}
                  aria-pressed={ui.getEditorMode() === 'preview'}
                >
                  <Eye size={16} />
                </button>
              </div>

              <!-- Formatting tools - always visible -->
              {#if FEATURE_FLAGS.taskLists}
                <button
                  type="button"
                  onclick={handleInsertTask}
                  class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
                  aria-label={$_('component.editor.toolbar.task')}
                >
                  <ListTodo size={16} />
                </button>
              {/if}

              <!-- Divider -->
              <div class="w-px h-6 bg-border mx-1 flex-shrink-0"></div>

              <!-- Save button - always visible -->
              <button
                type="button"
                onclick={handleSave}
                disabled={!notes.getIsDirty() || notes.getIsSaving()}
                class="p-2 hover:bg-accent rounded-md disabled:opacity-50 flex-shrink-0 toolbar-btn"
                aria-label={$_('component.editor.toolbar.save')}
              >
                <Save size={16} />
              </button>

              <!-- Upload button - always visible -->
              <button
                type="button"
                onclick={handleUploadButtonClick}
                disabled={uploading}
                class="p-2 hover:bg-accent rounded-md disabled:opacity-50 flex-shrink-0 toolbar-btn"
                aria-label={$_('component.editor.toolbar.upload')}
              >
                <ImagePlus size={16} />
              </button>

              <!-- History button - always visible -->
              <button
                type="button"
                onclick={() => (showVersionHistory = true)}
                class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
                aria-label={$_('component.editor.toolbar.history')}
              >
                <History size={16} />
              </button>

              <!-- Focus Mode toggle - hidden on mobile -->
              {#if !ui.getIsMobile()}
                <button
                  type="button"
                  onclick={() => focusMode.toggle()}
                  class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
                  class:bg-accent={focusMode.isActive()}
                  aria-label={$_('component.editor.toolbar.focus_mode')}
                  aria-pressed={focusMode.isActive()}
                >
                  {#if focusMode.isActive()}
                    <Minimize2 size={16} />
                  {:else}
                    <Maximize2 size={16} />
                  {/if}
                </button>
              {/if}

              <!-- Spell Check toggle - only in edit mode with AI enabled -->
              {#if FEATURE_FLAGS.spellCheck && notes.getCurrentNote()?.ai_enabled && (ui.getEditorMode() === 'edit' || ui.getEditorMode() === 'split')}
                <SpellCheckToggle {editorView} />
              {/if}

              <!-- AI Actions button - only when AI enabled -->
              {#if notes.getCurrentNote()?.ai_enabled && (ui.getEditorMode() === 'edit' || ui.getEditorMode() === 'split')}
                <div class="flex-shrink-0">
                  <button
                    bind:this={aiActionsButtonRef}
                    type="button"
                    onclick={handleAIActionsClick}
                    class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
                    class:bg-accent={showAIActionsDropdown}
                    aria-label={$_('component.editor.ai_actions')}
                    title={$_('component.editor.ai_actions_tooltip')}
                    aria-expanded={showAIActionsDropdown}
                    aria-haspopup="menu"
                  >
                    <Wand2 size={16} />
                  </button>
                </div>
              {/if}

              <!-- Auto-save toggle -->
              <button
                type="button"
                onclick={handleAutoSaveToggle}
                class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
                class:bg-accent={autosave.getAutoSaveEnabled()}
                aria-label={$_('component.editor.toolbar.autosave')}
                aria-pressed={autosave.getAutoSaveEnabled()}
              >
                {#if notes.getAutoSaveStatus() === 'saving'}
                  <Loader2 size={16} class="animate-spin" />
                {:else if notes.getAutoSaveStatus() === 'saved'}
                  <Check size={16} class="text-success" />
                {:else if notes.getAutoSaveStatus() === 'error'}
                  <AlertCircle size={16} class="text-destructive" />
                {:else}
                  <Save size={16} class={autosave.getAutoSaveEnabled() ? 'text-primary' : ''} />
                {/if}
              </button>
            </div>
          </div>
          <!-- More menu button - fixed right, always visible outside scroll area -->
          <button
            bind:this={moreMenuButtonRef}
            onclick={openMoreMenu}
            class="p-2 hover:bg-accent rounded-md flex-shrink-0 toolbar-btn"
            aria-expanded={showMoreMenu}
            aria-haspopup="menu"
            title={$_('component.editor.toolbar.more_options')}
          >
            <MoreVertical size={16} />
          </button>
        </div>
      </div>
    </div>
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
            ondrop={handleEditorDrop}
            ondragover={handleEditorDragOver}
            onpaste={handleEditorPaste}
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
            onpointerdown={handleSplitResizeStart}
            onpointermove={handleSplitResizeMove}
            onpointerup={handleSplitResizeEnd}
            onpointercancel={handleSplitResizeEnd}
            ondblclick={handleSplitResizeDblClick}
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
  {#if MoveToFolderDialogComponent}
    <MoveToFolderDialogComponent
      noteId={notes.getCurrentNote()!.id}
      currentFolder={notes.getCurrentNote()!.folder_path}
      onClose={() => (showMoveDialog = false)}
    />
  {/if}
{/if}

<!-- Markdown Guide Dropdown -->
{#if ui.getMarkdownGuideDropdownOpen() && MarkdownGuideDropdownComponent}
  <MarkdownGuideDropdownComponent onClose={() => ui.setMarkdownGuideDropdownOpen(false)} />
{/if}

<!-- Markdown Guide Dialog -->
{#if ui.getMarkdownGuideOpen() && markdownGuideDialogComponent}
  {@const MarkdownGuideDialog = markdownGuideDialogComponent}
  <MarkdownGuideDialog onClose={() => ui.setMarkdownGuideOpen(false)} />
{/if}

<!-- Version History Dialog -->
{#if showVersionHistory && notes.getCurrentNote() && VersionHistoryDialogComponent}
  <VersionHistoryDialogComponent
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
