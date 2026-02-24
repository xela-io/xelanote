<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title: string;
    subtitle?: string | null;
    class?: string;
    containerClass?: string;
    titleClass?: string;
    subtitleClass?: string;
    leading?: Snippet;
    actions?: Snippet;
  }

  const {
    title,
    subtitle = null,
    class: className = '',
    containerClass = '',
    titleClass = '',
    subtitleClass = '',
    leading,
    actions,
  }: Props = $props();
</script>

<header class={`ui-page-header ${className}`.trim()}>
  <div class={containerClass}>
    <div class="ui-page-title-row">
      <div class="ui-page-title-group">
        {@render leading?.()}
        <div class="ui-page-title-stack">
          <h1 class={`ui-page-title ${titleClass}`.trim()}>{title}</h1>
          {#if subtitle}
            <p class={`ui-page-subtitle ${subtitleClass}`.trim()}>{subtitle}</p>
          {/if}
        </div>
      </div>
      {#if actions}
        <div class="flex items-center gap-2">{@render actions()}</div>
      {/if}
    </div>
  </div>
</header>
