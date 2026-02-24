<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    align?: 'start' | 'end' | 'between';
    wrap?: boolean;
    children?: Snippet;
  }

  const { align = 'end', wrap = true, children }: Props = $props();

  const justifyClasses: Record<NonNullable<Props['align']>, string> = {
    start: 'justify-start',
    end: 'justify-end',
    between: 'justify-between',
  };
</script>

<div class={`flex w-full items-center gap-2 ${wrap ? 'flex-wrap' : ''} ${justifyClasses[align]}`}>
  {@render children?.()}
</div>
