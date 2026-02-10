<script lang="ts">
  import DOMPurify from 'isomorphic-dompurify';
  import MarkdownIt from 'markdown-it';
  import { _ } from 'svelte-i18n';

  import { getChangelog } from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
    version: string;
  }

  const { open, onClose, version }: Props = $props();

  let changelogHtml = $state('');
  let isLoading = $state(true);
  let error = $state<string | null>(null);

  const md = new MarkdownIt({ html: false, linkify: true });

  $effect(() => {
    if (open) {
      isLoading = true;
      error = null;
      getChangelog()
        .then((raw) => {
          changelogHtml = DOMPurify.sanitize(md.render(raw));
        })
        .catch(() => {
          error = 'Changelog could not be loaded.';
        })
        .finally(() => {
          isLoading = false;
        });
    }
  });
</script>

<BaseDialog {open} title="Changelog — {version}" {onClose} size="lg" scrollable={true}>
  {#snippet content()}
    {#if isLoading}
      <div class="flex items-center justify-center py-12 text-muted-foreground">
        <span class="animate-pulse">{$_('common.loading')}</span>
      </div>
    {:else if error}
      <div class="text-sm text-destructive py-4">{error}</div>
    {:else}
      <div class="changelog-content prose prose-sm max-w-none">
        {@html changelogHtml}
      </div>
    {/if}
  {/snippet}

  {#snippet footer()}
    <button type="button" onclick={onClose} class="px-4 py-2 text-sm hover:bg-accent rounded-md">
      {$_('common.close')}
    </button>
  {/snippet}
</BaseDialog>

<style>
  .changelog-content :global(h1) {
    font-size: 1.5rem;
    font-weight: 700;
    margin-bottom: 0.75rem;
    color: var(--color-foreground);
  }

  .changelog-content :global(h2) {
    font-size: 1.2rem;
    font-weight: 600;
    margin-top: 1.5rem;
    margin-bottom: 0.5rem;
    padding-bottom: 0.25rem;
    border-bottom: 1px solid var(--color-border);
    color: var(--color-foreground);
  }

  .changelog-content :global(h3) {
    font-size: 1rem;
    font-weight: 600;
    margin-top: 1rem;
    margin-bottom: 0.25rem;
    color: var(--color-primary);
  }

  .changelog-content :global(ul) {
    list-style: disc;
    padding-left: 1.25rem;
    margin-bottom: 0.75rem;
  }

  .changelog-content :global(li) {
    font-size: 0.875rem;
    line-height: 1.5;
    margin-bottom: 0.25rem;
    color: var(--color-foreground);
  }

  .changelog-content :global(p) {
    font-size: 0.875rem;
    line-height: 1.5;
    margin-bottom: 0.5rem;
    color: var(--color-muted-foreground);
  }

  .changelog-content :global(a) {
    color: var(--color-primary);
    text-decoration: underline;
  }

  .changelog-content :global(code) {
    font-size: 0.8rem;
    padding: 0.1rem 0.3rem;
    border-radius: 0.25rem;
    background: var(--color-muted);
  }
</style>
