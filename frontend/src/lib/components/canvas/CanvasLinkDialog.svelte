<script lang="ts">
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';

  interface Props {
    open: boolean;
    linkUrl: string;
    linkUrlError: string | null;
    onUrlChange: (url: string) => void;
    onSubmit: () => void;
    onClose: () => void;
  }

  const {
    open,
    linkUrl,
    linkUrlError,
    onUrlChange,
    onSubmit,
    onClose,
  }: Props = $props();
</script>

<BaseDialog
  {open}
  title={$_('component.canvas.link_dialog.title')}
  {onClose}
  size="sm"
>
  {#snippet content()}
    <div class="space-y-3">
      <label for="canvas-link-input" class="text-sm font-medium text-foreground">
        {$_('component.canvas.link_dialog.url_label')}
      </label>
      <input
        id="canvas-link-input"
        type="text"
        value={linkUrl}
        oninput={(e) => onUrlChange((e.currentTarget as HTMLInputElement).value)}
        placeholder={$_('component.canvas.link_dialog.placeholder')}
        class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring"
        onkeydown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault();
            onSubmit();
          }
        }}
      />
      {#if linkUrlError}
        <p class="text-sm text-red-600" role="alert">{linkUrlError}</p>
      {/if}
    </div>
  {/snippet}
  {#snippet footer()}
    <button
      type="button"
      onclick={onClose}
      class="px-4 py-2 text-sm hover:bg-accent rounded-md"
    >
      {$_('common.cancel')}
    </button>
    <button
      type="button"
      onclick={onSubmit}
      class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md"
    >
      {$_('component.canvas.link_dialog.add_button')}
    </button>
  {/snippet}
</BaseDialog>
