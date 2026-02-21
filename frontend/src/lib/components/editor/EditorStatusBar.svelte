<script lang="ts">
  import type { EditorView } from '@codemirror/view';
  import { _ } from 'svelte-i18n';

  import type { Backlink, Note } from '$lib/api';
  import { FEATURE_FLAGS } from '$lib/config';

  import EditorPanels from './EditorPanels.svelte';

  interface Props {
    note: Note;
    backlinks: Backlink[];
    editorView: EditorView | undefined;
    isMobile: boolean;
    editorPanelsCollapsed: boolean;
    onTogglePanelsCollapsed: () => void;
    onInsertLink: (term: string, targetTitle: string) => void;
    onSummaryUpdated: (summary: string) => void;
  }

  const {
    note,
    backlinks,
    editorView,
    isMobile,
    editorPanelsCollapsed,
    onTogglePanelsCollapsed,
    onInsertLink,
    onSummaryUpdated,
  }: Props = $props();
</script>

{#if !isMobile}
  <div class="shrink-0 px-4 pb-2 pt-1 border-t border-border">
    <button
      type="button"
      class="text-xs text-muted-foreground hover:text-foreground transition-colors"
      onclick={onTogglePanelsCollapsed}
      aria-expanded={!editorPanelsCollapsed}
    >
      {editorPanelsCollapsed
        ? $_('component.editor.show_bottom_panels')
        : $_('component.editor.hide_bottom_panels')}
    </button>
  </div>
{/if}

{#if isMobile || !editorPanelsCollapsed}
  <div class={isMobile ? '' : 'shrink-0 overflow-auto max-h-[40vh]'}>
    <EditorPanels
      {note}
      {backlinks}
      showTagSuggestions={FEATURE_FLAGS.tagSuggestions}
      showLinkSuggestions={FEATURE_FLAGS.linkSuggestions}
      {editorView}
      {onInsertLink}
      {onSummaryUpdated}
    />
  </div>
{/if}
