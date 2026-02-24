<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title?: string | null;
    description?: string | null;
    variant?: 'panel' | 'soft' | 'form';
    class?: string;
    titleClass?: string;
    descriptionClass?: string;
    headerSuffix?: Snippet;
    children?: Snippet;
  }

  const {
    title = null,
    description = null,
    variant = 'panel',
    class: className = '',
    titleClass = '',
    descriptionClass = '',
    headerSuffix,
    children,
  }: Props = $props();

  const variantClasses: Record<NonNullable<Props['variant']>, string> = {
    panel: 'ui-panel p-5 sm:p-6',
    soft: 'ui-panel-soft p-4 sm:p-5',
    form: 'ui-form-section',
  };
</script>

<section class={`${variantClasses[variant]} ${className}`.trim()}>
  {#if title || description || headerSuffix}
    <div class={description ? 'mb-4' : 'mb-2'}>
      <div class="flex items-start justify-between gap-3">
        <div class="min-w-0 flex-1">
          {#if title}
            <h3 class={`ui-form-section-title ${titleClass}`.trim()}>{title}</h3>
          {/if}
          {#if description}
            <p class={`mt-2 text-sm text-muted-foreground ${descriptionClass}`.trim()}>
              {description}
            </p>
          {/if}
        </div>
        {#if headerSuffix}
          <div class="shrink-0">{@render headerSuffix()}</div>
        {/if}
      </div>
    </div>
  {/if}

  {@render children?.()}
</section>
