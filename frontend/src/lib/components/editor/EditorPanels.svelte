<script lang="ts">
  import type { EditorView } from '@codemirror/view';
  import { Link } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { Backlink, Note, Tag } from '$lib/api';

  import LinkSuggestionsPanel from '../LinkSuggestionsPanel.svelte';
  import SummaryPanel from '../SummaryPanel.svelte';
  import TagEditor from '../TagEditor.svelte';
  import TagSuggestionsPanel from '../TagSuggestionsPanel.svelte';

  const {
    note,
    backlinks = [],
    showTagSuggestions = false,
    showLinkSuggestions = false,
    editorView: _editorView,
    onInsertLink,
    onSummaryUpdated,
  } = $props<{
    note: Note;
    backlinks?: Backlink[];
    showTagSuggestions?: boolean;
    showLinkSuggestions?: boolean;
    editorView?: EditorView;
    onInsertLink: (term: string, targetTitle: string) => void;
    onSummaryUpdated: (summary: string) => void;
  }>();

  let currentTags = $state<Tag[]>([]);
  let tagEditorRef: TagEditor | null = $state(null);
</script>

<!-- Backlinks panel -->
{#if backlinks.length > 0}
  <section class="editor-panel-section">
    <h3 class="text-sm font-medium flex items-center gap-2 mb-2">
      <Link size={14} />
      {$_('component.editor.backlinks_title', {
        values: { count: backlinks.length },
      })}
    </h3>
    <div class="flex flex-wrap gap-2">
      {#each backlinks as backlink (backlink.id)}
        <a
          href="/note/{backlink.id}"
          class="text-sm px-2 py-1 bg-accent rounded-md hover:bg-accent/80"
        >
          {backlink.title}
        </a>
      {/each}
    </div>
  </section>
{/if}

<!-- AI Summary Panel -->
<section class="editor-panel-section">
  <SummaryPanel
    {note}
    decryptedContent={note.content_encrypted ? note.content : undefined}
    {onSummaryUpdated}
  />
</section>

<!-- Tag Suggestions Panel -->
{#if showTagSuggestions}
  <section class="editor-panel-section">
    <TagSuggestionsPanel
      noteId={note.id}
      isEncrypted={note.content_encrypted || false}
      existingTagNames={currentTags.map((t) => t.name)}
      onAddTag={async (tagName) => {
        if (tagEditorRef) {
          tagEditorRef.setInputValue(tagName);
          tagEditorRef.focusInput();
        }
      }}
    />
  </section>
{/if}

<!-- Link Suggestions Panel -->
{#if showLinkSuggestions}
  <section class="editor-panel-section">
    <LinkSuggestionsPanel
      noteId={note.id}
      isEncrypted={note.content_encrypted || false}
      noteContent={note.content_encrypted ? undefined : note.content}
      onInsertLink={(term, targetTitle) => {
        onInsertLink(term, targetTitle);
      }}
    />
  </section>
{/if}

<!-- Tag editor panel -->
{#if !note.content_encrypted}
  <section class="editor-panel-section">
    <TagEditor
      bind:this={tagEditorRef}
      noteId={note.id}
      onTagsChanged={(tags) => {
        currentTags = tags;
      }}
    />
  </section>
{/if}

<style>
  .editor-panel-section {
    margin: 0.65rem 0.75rem 0;
    padding: 0.85rem;
    border: 1px solid var(--surface-panel-border);
    border-radius: 0.9rem;
    background: var(--surface-panel-bg);
    box-shadow: inset 0 1px 0 var(--surface-panel-inset-highlight);
  }

  @media (max-width: 639px) {
    .editor-panel-section {
      margin: 0.5rem 0.5rem 0;
      padding: 0.75rem;
      border-radius: 0.8rem;
    }
  }
</style>
