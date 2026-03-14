<script lang="ts">
  import {
    BookOpen,
    ChevronRight,
    Download,
    FileText,
    Filter,
    FolderPlus,
    Lock,
    Map as MapIcon,
    Moon,
    Plus,
    Search,
    Settings,
  } from 'lucide-svelte';
  import { onDestroy, onMount } from 'svelte';
  import { SvelteMap } from 'svelte/reactivity';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import type { Note } from '$lib/api';
  import { quickSearch, type QuickSearchFilters } from '$lib/api';
  import { getCommands, registerCommands } from '$lib/commands/command-registry';
  import * as encryption from '$lib/stores/encryption.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as folders from '$lib/stores/folders.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as searchStore from '$lib/stores/search.svelte';
  import { searchEncrypted } from '$lib/stores/search-index.svelte';
  import * as tabs from '$lib/stores/tabs.svelte';
  import * as toast from '$lib/stores/toast.svelte';
  import * as ui from '$lib/stores/ui.svelte';

  import FilterBar from './FilterBar.svelte';
  import FilterMenu from './FilterMenu.svelte';

  let query = $state('');
  let results = $state<Note[]>([]);
  let snippetMap = $state<Map<string, string>>(new Map());
  let selectedIndex = $state(0);
  let loading = $state(false);
  let filterMenuOpen = $state(false);
  // svelte-ignore non_reactive_update
  let inputRef: HTMLInputElement;

  // Command mode: activated when query starts with '>'
  const isCommandMode = $derived(query.startsWith('>'));
  const commandQuery = $derived(isCommandMode ? query.slice(1).trim() : '');
  const filteredCommands = $derived(isCommandMode ? getCommands(commandQuery) : []);

  // Icon map for commands
  const commandIcons: Record<string, typeof FileText> = {
    'new-note': FileText,
    'new-folder': FolderPlus,
    'toggle-theme': Moon,
    'open-graph': MapIcon,
    'open-settings': Settings,
    'open-journal': BookOpen,
    'export-note': Download,
  };

  // Register commands on mount
  onMount(() => {
    registerCommands([
      {
        id: 'new-note',
        label: $_('component.quick_switcher.cmd_new_note'),
        shortcut: 'Ctrl+N',
        action: async () => {
          ui.setQuickSwitcherOpen(false);
          const folderPath = folders.getSelectedFolder() || '/';
          const note = await notes.createNote('', '', folderPath);
          if (note?.id) goto(`/note/${note.id}`);
        },
      },
      {
        id: 'new-folder',
        label: $_('component.quick_switcher.cmd_new_folder'),
        action: () => {
          ui.setQuickSwitcherOpen(false);
          // Navigate to main view where sidebar folder creation is accessible
          goto('/');
        },
      },
      {
        id: 'toggle-theme',
        label: $_('component.quick_switcher.cmd_toggle_theme'),
        action: () => {
          ui.setQuickSwitcherOpen(false);
          ui.toggleTheme();
        },
      },
      ...(features.getGraphFeatureEnabled()
        ? [
            {
              id: 'open-graph',
              label: $_('component.quick_switcher.cmd_open_graph'),
              shortcut: 'Ctrl+G',
              action: () => {
                ui.setQuickSwitcherOpen(false);
                goto('/graph');
              },
            },
          ]
        : []),
      {
        id: 'open-settings',
        label: $_('component.quick_switcher.cmd_open_settings'),
        action: () => {
          ui.setQuickSwitcherOpen(false);
          goto('/settings');
        },
      },
      {
        id: 'open-journal',
        label: $_('component.quick_switcher.cmd_open_journal'),
        action: () => {
          ui.setQuickSwitcherOpen(false);
          goto('/journal');
        },
      },
      {
        id: 'export-note',
        label: $_('component.quick_switcher.cmd_export_note'),
        action: () => {
          const currentNote = notes.getCurrentNote();
          if (!currentNote) return;
          ui.setQuickSwitcherOpen(false);
          // Download as markdown
          const blob = new Blob([currentNote.content], { type: 'text/markdown' });
          const url = URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = `${currentNote.title || 'note'}.md`;
          a.click();
          URL.revokeObjectURL(url);
        },
      },
    ]);
  });

  type SnippetPart = { text: string; highlighted: boolean };
  function parseSnippet(snippet: string): SnippetPart[] {
    const parts: SnippetPart[] = [];
    const regex = /<mark>(.*?)<\/mark>/g;
    let lastIndex = 0;
    let match;
    while ((match = regex.exec(snippet)) !== null) {
      if (match.index > lastIndex) {
        parts.push({ text: snippet.slice(lastIndex, match.index), highlighted: false });
      }
      parts.push({ text: match[1], highlighted: true });
      lastIndex = regex.lastIndex;
    }
    if (lastIndex < snippet.length) {
      parts.push({ text: snippet.slice(lastIndex), highlighted: false });
    }
    return parts;
  }

  let debounceTimer: ReturnType<typeof setTimeout>;

  const currentFilters = $derived(searchStore.getFilters());
  const activeFilterCount = $derived(searchStore.getActiveFilterCount());

  $effect(() => {
    if (ui.getQuickSwitcherOpen() && inputRef) {
      inputRef.focus();
      query = '';
      results = [];
      selectedIndex = 0;
      filterMenuOpen = false;
    }
  });

  // Re-search when filters change (only in search mode)
  $effect(() => {
    if (ui.getQuickSwitcherOpen() && !isCommandMode) {
      // Trigger search when filters change
      void currentFilters;
      handleSearch();
    }
  });

  function handleInput() {
    if (isCommandMode) {
      // Commands are filtered reactively, no debounce needed
      selectedIndex = 0;
      return;
    }
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => handleSearch(), 200);
  }

  onDestroy(() => clearTimeout(debounceTimer));

  async function handleSearch() {
    if (isCommandMode) return;
    loading = true;
    try {
      // Build filter parameters
      const filters: QuickSearchFilters = {};

      if (currentFilters.folders.length > 0) {
        filters.folders = currentFilters.folders;
      }
      if (currentFilters.tags.length > 0) {
        filters.tags = currentFilters.tags;
      }

      // Convert date filters to absolute ISO8601 timestamps
      if (currentFilters.createdDate) {
        const dateRange = searchStore.getAbsoluteDateRange(currentFilters.createdDate);
        if (dateRange?.after) filters.created_after = dateRange.after;
        if (dateRange?.before) filters.created_before = dateRange.before;
      }
      if (currentFilters.updatedDate) {
        const dateRange = searchStore.getAbsoluteDateRange(currentFilters.updatedDate);
        if (dateRange?.after) filters.updated_after = dateRange.after;
        if (dateRange?.before) filters.updated_before = dateRange.before;
      }

      // Perform server search (with or without query)
      const response = await quickSearch(
        query || '',
        10,
        Object.keys(filters).length > 0 ? filters : undefined
      );

      // Decrypt titles for encrypted server results
      const serverResults = response.notes.map((note) => {
        if (note.title_encrypted && note.encrypted_title && encryption.isEncryptionUnlocked()) {
          const decrypted = encryption.decryptTitle(note.encrypted_title, note.id);
          if (decrypted) return { ...note, title: decrypted };
        }
        return note;
      });

      // Also search client-side encrypted index (if query is non-empty and vault unlocked)
      const newSnippetMap = new SvelteMap<string, string>();
      if (query.trim() && encryption.isEncryptionUnlocked()) {
        const encryptedResults = searchEncrypted(query, 10);
        const serverIds = new Set(serverResults.map((n) => n.id));

        for (const er of encryptedResults) {
          if (er.snippet) newSnippetMap.set(er.id, er.snippet);
          if (!serverIds.has(er.id)) {
            serverResults.push({
              id: er.id,
              title: er.title,
              content: '',
              folder_path: '',
              version: 0,
              created_at: '',
              updated_at: '',
              content_encrypted: true,
            });
          }
        }
      }

      snippetMap = newSnippetMap;
      results = serverResults;
      selectedIndex = 0;
    } catch (e) {
      console.error('Search failed:', e);
      toast.error($_('component.search.search_failed'));
      results = [];
    } finally {
      loading = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    // Ctrl+F to open filter menu (only in search mode)
    if (!isCommandMode && e.ctrlKey && e.key === 'f') {
      e.preventDefault();
      filterMenuOpen = !filterMenuOpen;
      return;
    }

    const maxIndex = isCommandMode ? filteredCommands.length - 1 : results.length; // includes "create" option

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        selectedIndex = Math.min(selectedIndex + 1, maxIndex);
        break;
      case 'ArrowUp':
        e.preventDefault();
        selectedIndex = Math.max(selectedIndex - 1, 0);
        break;
      case 'Enter':
        e.preventDefault();
        if (isCommandMode) {
          const cmd = filteredCommands[selectedIndex];
          if (cmd) {
            cmd.action();
          }
        } else if (selectedIndex === results.length && query.trim()) {
          createNewNote();
        } else if (results[selectedIndex]) {
          selectNote(results[selectedIndex].id, e.ctrlKey || e.metaKey);
        }
        break;
      case 'Escape':
        if (filterMenuOpen) {
          filterMenuOpen = false;
        } else {
          ui.setQuickSwitcherOpen(false);
        }
        break;
    }
  }

  function selectNote(id: string, newTab = false) {
    ui.setQuickSwitcherOpen(false);
    if (newTab) tabs.requestNewTab();
    const highlightParam = query.trim() ? `?highlight=${encodeURIComponent(query.trim())}` : '';
    goto(`/note/${id}${highlightParam}`);
  }

  async function createNewNote() {
    try {
      const folderPath = folders.getSelectedFolder() || '/';
      const note = await notes.createNote(query.trim(), '', folderPath);
      await folders.loadFolders();
      ui.setQuickSwitcherOpen(false);
      goto(`/note/${note.id}`);
    } catch (e) {
      console.error('Failed to create note:', e);
      toast.error($_('common.error'));
    }
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      ui.setQuickSwitcherOpen(false);
    }
  }

  function handleBackdropKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' || e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      ui.setQuickSwitcherOpen(false);
    }
  }
</script>

{#if ui.getQuickSwitcherOpen()}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 bg-black/50 z-50"
    onclick={handleBackdropClick}
    onkeydown={handleBackdropKeydown}
    aria-hidden="true"
  ></div>

  <!-- Dialog -->
  <div
    class="fixed inset-0 z-50 flex items-start justify-center pt-[10vh] sm:pt-[20vh] px-4 sm:px-0"
    onclick={handleBackdropClick}
    onkeydown={handleBackdropKeydown}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
    aria-label={isCommandMode
      ? $_('component.quick_switcher.placeholder_commands')
      : $_('component.quick_switcher.placeholder')}
  >
    <div
      class="w-full max-w-lg bg-popover border border-border rounded-lg shadow-lg overflow-hidden relative"
    >
      <!-- Search input -->
      <div class="flex items-center gap-2 px-4 py-3 border-b border-border">
        {#if isCommandMode}
          <ChevronRight size={18} class="text-primary" />
        {:else}
          <Search size={18} class="text-muted-foreground" />
        {/if}
        <input
          bind:this={inputRef}
          bind:value={query}
          oninput={handleInput}
          onkeydown={handleKeydown}
          type="text"
          role="combobox"
          aria-expanded={isCommandMode
            ? filteredCommands.length > 0
            : results.length > 0 || query.trim() !== ''}
          aria-controls="qs-results"
          aria-activedescendant={selectedIndex >= 0
            ? isCommandMode
              ? `qs-cmd-${selectedIndex}`
              : selectedIndex === results.length
                ? 'qs-opt-create'
                : `qs-opt-${selectedIndex}`
            : undefined}
          autocomplete="off"
          placeholder={isCommandMode
            ? $_('component.quick_switcher.placeholder_commands')
            : $_('component.quick_switcher.placeholder')}
          class="flex-1 bg-transparent border-0 outline-none text-base text-foreground placeholder:text-muted-foreground"
        />
        {#if loading && !isCommandMode}
          <div
            class="animate-spin w-4 h-4 border-2 border-primary border-t-transparent rounded-full"
            role="status"
            aria-label={$_('common.loading')}
          ></div>
        {/if}
        {#if !isCommandMode}
          <button
            type="button"
            onclick={() => (filterMenuOpen = !filterMenuOpen)}
            class="p-1.5 hover:bg-accent rounded-md transition-colors relative"
            title="{$_('component.quick_switcher.filter')} (Ctrl+F)"
            aria-label={$_('component.quick_switcher.filter')}
            aria-expanded={filterMenuOpen}
          >
            <Filter size={16} class="text-muted-foreground" />
            {#if activeFilterCount > 0}
              <span
                class="absolute -top-1 -right-1 w-4 h-4 bg-primary text-primary-foreground text-xs rounded-full flex items-center justify-center"
              >
                {activeFilterCount}
              </span>
            {/if}
          </button>
        {/if}
      </div>

      {#if !isCommandMode}
        <!-- Filter Bar -->
        <FilterBar />

        <!-- Filter Menu -->
        <FilterMenu isOpen={filterMenuOpen} onClose={() => (filterMenuOpen = false)} />
      {/if}

      <!-- Results -->
      <div id="qs-results" role="listbox" class="max-h-[50vh] sm:max-h-80 overflow-y-auto">
        {#if isCommandMode}
          <!-- Command mode results -->
          {#if filteredCommands.length === 0}
            <div class="px-4 py-8 text-center text-muted-foreground text-sm">
              {$_('component.quick_switcher.no_commands')}
            </div>
          {:else}
            {#each filteredCommands as cmd, i (cmd.id)}
              {@const IconComponent = commandIcons[cmd.id]}
              <div
                id="qs-cmd-{i}"
                role="option"
                aria-selected={selectedIndex === i}
                tabindex="-1"
                onclick={() => cmd.action()}
                onkeydown={(e) => {
                  if (e.key === 'Enter') cmd.action();
                }}
                class="w-full text-left px-4 py-2.5 hover:bg-accent cursor-pointer flex items-center gap-3"
                class:bg-accent={selectedIndex === i}
              >
                {#if IconComponent}
                  <IconComponent size={16} class="text-muted-foreground shrink-0" />
                {/if}
                <span class="flex-1 truncate">{cmd.label}</span>
                {#if cmd.shortcut}
                  <kbd
                    class="text-xs text-muted-foreground bg-muted px-1.5 py-0.5 rounded border border-border font-mono shrink-0"
                  >
                    {cmd.shortcut}
                  </kbd>
                {/if}
              </div>
            {/each}
          {/if}
        {:else}
          <!-- Search mode results -->
          {#if results.length === 0 && query.trim() === '' && !searchStore.hasActiveFilters()}
            <div class="px-4 py-8 text-center text-muted-foreground text-sm">
              {$_('component.quick_switcher.hint')}
              <div class="mt-2 text-xs opacity-60">
                {$_('component.quick_switcher.hint_commands')}
              </div>
            </div>
          {:else if results.length === 0}
            <div class="px-4 py-8 text-center text-muted-foreground text-sm">
              {#if searchStore.hasActiveFilters()}
                {$_('component.quick_switcher.no_results')}
              {:else}
                {$_('component.quick_switcher.no_results_simple')}
              {/if}
            </div>
          {:else}
            {#each results as note, i (note.id)}
              <div
                id="qs-opt-{i}"
                role="option"
                aria-selected={selectedIndex === i}
                tabindex="-1"
                onclick={() => selectNote(note.id)}
                onkeydown={(e) => {
                  if (e.key === 'Enter') selectNote(note.id);
                }}
                class="w-full text-left px-4 py-2 hover:bg-accent cursor-pointer"
                class:bg-accent={selectedIndex === i}
              >
                <div class="flex items-center gap-2">
                  {#if note.content_encrypted}
                    <Lock size={14} class="text-muted-foreground shrink-0" />
                  {/if}
                  <span class="truncate">{note.title || note.id.substring(0, 8) + '...'}</span>
                  <span class="ml-auto text-xs text-muted-foreground truncate max-w-32 shrink-0">
                    {note.folder_path}
                  </span>
                </div>
                {#if snippetMap.has(note.id)}
                  <p class="text-xs text-muted-foreground mt-0.5 truncate pl-5">
                    {#each parseSnippet(snippetMap.get(note.id) ?? '') as part, pi (pi)}{#if part.highlighted}<mark
                          class="bg-primary/30 text-foreground rounded-sm">{part.text}</mark
                        >{:else}{part.text}{/if}{/each}
                  </p>
                {/if}
              </div>
            {/each}

            {#if query.trim()}
              <div
                id="qs-opt-create"
                role="option"
                aria-selected={selectedIndex === results.length}
                tabindex="-1"
                onclick={createNewNote}
                onkeydown={(e) => {
                  if (e.key === 'Enter') createNewNote();
                }}
                class="w-full text-left px-4 py-2 hover:bg-accent flex items-center gap-2 text-primary cursor-pointer"
                class:bg-accent={selectedIndex === results.length}
              >
                <Plus size={16} />
                <span>{$_('component.quick_switcher.create', { values: { query } })}</span>
              </div>
            {/if}
          {/if}
        {/if}
      </div>

      <!-- Footer -->
      <div class="px-4 py-2 border-t border-border text-xs text-muted-foreground flex gap-4">
        <span>↑↓ {$_('component.quick_switcher.navigate')}</span>
        <span>↵ {$_('component.quick_switcher.select')}</span>
        {#if !isCommandMode}
          <span>Ctrl+F {$_('component.quick_switcher.filter')}</span>
        {/if}
        <span>esc {$_('common.close')}</span>
      </div>
    </div>
  </div>
{/if}
