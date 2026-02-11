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
  <div class="border-t border-border p-4">
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
  </div>
{/if}

<!-- AI Summary Panel -->
<div class="border-t border-border p-4">
  <SummaryPanel
    {note}
    decryptedContent={note.content_encrypted ? note.content : undefined}
    {onSummaryUpdated}
  />
</div>

<!-- Tag Suggestions Panel -->
{#if showTagSuggestions}
  <div class="border-t border-border p-4">
    <TagSuggestionsPanel
      noteId={note.id}
      isEncrypted={note.content_encrypted || false}
      plaintextContent={note.content_encrypted ? note.content : undefined}
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
{#if showLinkSuggestions}
  <div class="border-t border-border p-4">
    <LinkSuggestionsPanel
      noteId={note.id}
      isEncrypted={note.content_encrypted || false}
      plaintextContent={note.content}
      onInsertLink={(term, targetTitle) => {
        onInsertLink(term, targetTitle);
      }}
    />
  </div>
{/if}

<!-- Tag editor panel -->
<div class="border-t border-border p-4">
  <TagEditor
    bind:this={tagEditorRef}
    noteId={note.id}
    onTagsChanged={(tags) => {
      currentTags = tags;
    }}
  />
</div>
