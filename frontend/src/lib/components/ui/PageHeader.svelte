<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title: string;
    subtitle?: string | null;
    class?: string;
    containerClass?: string;
    titleClass?: string;
    subtitleClass?: string;
    mobileHeaderMode?: 'default' | 'topbar';
    mobileSingleRow?: boolean;
    mobileHideSubtitle?: boolean;
    mobileSticky?: boolean;
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
    mobileHeaderMode = 'default',
    mobileSingleRow = false,
    mobileHideSubtitle = false,
    mobileSticky = true,
    leading,
    actions,
  }: Props = $props();
</script>

<header
  class={`ui-page-header ${mobileSticky ? 'ui-page-header-mobile-sticky' : ''} ${className}`.trim()}
>
  <div class={containerClass}>
    <div
      class={`ui-page-title-row ${
        mobileHeaderMode === 'topbar' ? 'ui-mobile-topbar' : ''
      } ${mobileSingleRow ? 'flex-nowrap' : ''}`.trim()}
    >
      <div class={`ui-page-title-group ${mobileHeaderMode === 'topbar' ? 'flex-1' : ''}`.trim()}>
        {@render leading?.()}
        <div
          class={`ui-page-title-stack ${
            mobileHeaderMode === 'topbar' ? 'ui-mobile-topbar-title' : ''
          }`.trim()}
        >
          <h1 class={`ui-page-title ${titleClass}`.trim()}>{title}</h1>
          {#if subtitle}
            <p
              class={`ui-page-subtitle ${
                mobileHeaderMode === 'topbar' && mobileHideSubtitle ? 'hidden sm:block' : ''
              } ${subtitleClass}`.trim()}
            >
              {subtitle}
            </p>
          {/if}
        </div>
      </div>
      {#if actions}
        <div
          class={`${
            mobileHeaderMode === 'topbar' ? 'ui-mobile-topbar-actions' : 'flex items-center gap-2'
          }`.trim()}
        >
          {@render actions()}
        </div>
      {/if}
    </div>
  </div>
</header>
