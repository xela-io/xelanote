<script lang="ts">
  import { X, Tag, Plus, Loader2 } from 'lucide-svelte';
  import * as api from '$lib/api';
  import * as toast from '$lib/stores/toast.svelte';
  import type { Tag as TagType } from '$lib/api';
  import { _ } from 'svelte-i18n';

  interface Props {
    noteId: string;
    onTagsChanged?: (tags: TagType[]) => void;
  }

  const { noteId, onTagsChanged }: Props = $props();

  let noteTags = $state<TagType[]>([]);
  let allTags = $state<TagType[]>([]);
  let inputValue = $state('');
  let loading = $state(false);
  let saving = $state(false);
  let showSuggestions = $state(false);
  let inputElement = $state<HTMLInputElement | null>(null);
  let previousNoteId = $state<string | null>(null);

  // Filter suggestions based on input (exclude already assigned tags)
  // Shows all available tags when input is empty, filters when typing
  const suggestions = $derived(() => {
    if (!noteTags || !allTags) return [];

    const noteTagNames = new Set(noteTags.map((t) => t.name.toLowerCase()));
    const available = allTags.filter((t) => !noteTagNames.has(t.name.toLowerCase()));

    if (!inputValue.trim()) {
      // Show all available tags when input is empty (max 8)
      return available.slice(0, 8);
    }

    // Filter by input when typing
    return available
      .filter((t) => t.name.toLowerCase().includes(inputValue.toLowerCase()))
      .slice(0, 5);
  });

  // Load tags when noteId actually changes (not on every re-render)
  $effect(() => {
    // Only load if noteId changed AND we're not currently saving
    if (noteId && noteId !== previousNoteId && !saving) {
      previousNoteId = noteId;
      loadTags();
    }
  });

  async function loadTags() {
    // Don't reload while saving to prevent race conditions
    if (saving) return;
    // Skip loading when offline - tags are not available
    if (!navigator.onLine) return;

    loading = true;
    try {
      const [noteTagsResult, allTagsResult] = await Promise.all([
        api.getNoteTags(noteId),
        api.getTags(),
      ]);

      // Sichere Zuweisung mit Fallback zu leerem Array
      noteTags = Array.isArray(noteTagsResult) ? noteTagsResult : [];
      allTags = Array.isArray(allTagsResult) ? allTagsResult : [];
    } catch (e) {
      console.error('Failed to load tags:', e);
      // Bei Error: leere Arrays setzen statt alte Daten behalten
      noteTags = [];
      allTags = [];
      toast.error('Tags konnten nicht geladen werden');
    } finally {
      loading = false;
    }
  }

  async function addTag(tagName: string) {
    const trimmed = tagName.trim();
    if (!trimmed) return;

    // Check if already assigned
    if (noteTags.some((t) => t.name.toLowerCase() === trimmed.toLowerCase())) {
      inputValue = '';
      toast.info(`Tag "${trimmed}" ist bereits hinzugefügt`);
      return;
    }

    saving = true;
    try {
      const newTagNames = [...noteTags.map((t) => t.name), trimmed];
      const updatedTags = await api.setNoteTags(noteId, newTagNames);
      noteTags = updatedTags;
      inputValue = '';

      // Refresh all tags list (in case a new tag was created)
      allTags = await api.getTags();

      // Notify parent of tag changes
      onTagsChanged?.(noteTags);

      toast.success($_('component.tag_editor.added', { values: { tag: trimmed } }));
    } catch (e: unknown) {
      console.error('Failed to add tag:', e);
      toast.error(
        $_('component.tag_editor.add_error', {
          values: { error: e instanceof Error ? e.message : 'Unknown error' },
        })
      );
      // Don't clear input on error so user can retry
    } finally {
      saving = false;
    }
  }

  async function removeTag(tagToRemove: TagType) {
    saving = true;
    try {
      const newTagNames = noteTags.filter((t) => t.id !== tagToRemove.id).map((t) => t.name);
      const updatedTags = await api.setNoteTags(noteId, newTagNames);
      noteTags = updatedTags;

      // Notify parent of tag changes
      onTagsChanged?.(noteTags);

      toast.success(`Tag "${tagToRemove.name}" entfernt`);
    } catch (e: unknown) {
      console.error('Failed to remove tag:', e);
      toast.error(
        `Fehler beim Entfernen: ${e instanceof Error ? e.message : 'Unbekannter Fehler'}`
      );
    } finally {
      saving = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (suggestions().length > 0) {
        addTag(suggestions()[0].name);
      } else if (inputValue.trim()) {
        addTag(inputValue);
      }
    } else if (e.key === 'Escape') {
      showSuggestions = false;
      inputElement?.blur();
    }
  }

  function handleInput(e: Event) {
    inputValue = (e.target as HTMLInputElement).value;
    showSuggestions = true;
  }

  function selectSuggestion(tag: TagType) {
    addTag(tag.name);
    showSuggestions = false;
  }

  // Expose current tags for parent components
  export function getCurrentTags(): TagType[] {
    return noteTags;
  }

  // Set input value for editing before adding (used by TagSuggestionsPanel)
  export function setInputValue(value: string) {
    inputValue = value;
    showSuggestions = false;
  }

  // Focus the input element
  export function focusInput() {
    inputElement?.focus();
  }

  // Expose addTag for external use (e.g., TagSuggestionsPanel)
  export { addTag };
</script>

<div class="space-y-2">
  <div class="flex items-center gap-2 text-sm font-medium text-muted-foreground">
    <Tag size={14} />
    <span>Tags</span>
    {#if saving}
      <Loader2 size={14} class="animate-spin" />
    {/if}
  </div>

  {#if loading}
    <div class="text-sm text-muted-foreground">Laden...</div>
  {:else}
    <!-- Tag chips -->
    <div class="flex flex-wrap gap-1.5">
      {#each noteTags as tag (tag.id)}
        <div class="inline-flex items-center gap-1 px-2 py-0.5 bg-accent rounded-full text-sm">
          <span>{tag.name}</span>
          <button
            type="button"
            onclick={() => removeTag(tag)}
            disabled={saving}
            class="p-0.5 hover:bg-muted rounded-full transition-colors disabled:opacity-50"
            aria-label="Tag entfernen: {tag.name}"
          >
            <X size={12} />
          </button>
        </div>
      {/each}

      <!-- Add tag input -->
      <div class="relative">
        <div
          class="inline-flex items-center gap-1 px-2 py-0.5 border border-dashed border-border rounded-full"
        >
          <Plus size={12} class="text-muted-foreground" />
          <input
            bind:this={inputElement}
            type="text"
            value={inputValue}
            oninput={handleInput}
            onkeydown={handleKeydown}
            onfocus={() => (showSuggestions = true)}
            onblur={() => setTimeout(() => (showSuggestions = false), 150)}
            placeholder="Tag hinzufuegen"
            disabled={saving}
            class="w-24 text-sm bg-transparent border-0 outline-none placeholder:text-muted-foreground disabled:opacity-50"
          />
        </div>

        <!-- Suggestions dropdown -->
        {#if showSuggestions && suggestions().length > 0}
          <div class="absolute z-10 mt-1 w-48 bg-popover border border-border rounded-md shadow-md">
            {#each suggestions() as suggestion (suggestion.id)}
              <button
                type="button"
                class="w-full px-3 py-1.5 text-left text-sm hover:bg-accent"
                onmousedown={() => selectSuggestion(suggestion)}
              >
                {suggestion.name}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/if}
</div>
