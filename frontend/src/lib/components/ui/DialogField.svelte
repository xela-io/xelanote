<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    forId?: string;
    label?: string;
    help?: string | null;
    error?: string | null;
    class?: string;
    children?: Snippet;
  }

  const {
    forId,
    label,
    help = null,
    error = null,
    class: className = '',
    children,
  }: Props = $props();
</script>

<div class={`space-y-2 ${className}`.trim()}>
  {#if label}
    <label for={forId} class="ui-label mb-0">{label}</label>
  {/if}
  {@render children?.()}
  {#if help}
    <p class="ui-form-help">{help}</p>
  {/if}
  {#if error}
    <div class="ui-alert ui-alert-danger text-sm">{error}</div>
  {/if}
</div>
