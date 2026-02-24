<script lang="ts">
  import { AlertTriangle, FileQuestion, Home, RefreshCw } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { page } from '$app/state';
  import Logo from '$lib/components/Logo.svelte';

  const status = $derived(page.status);
  const errorMessage = $derived(page.error?.message ?? '');
  const errorId = $derived((page.error as { errorId?: string } | null)?.errorId);
  const isNotFound = $derived(status === 404);
</script>

<div class="flex min-h-full items-center justify-center p-4">
  <div class="w-full max-w-md text-center">
    <div class="mx-auto mb-6">
      <Logo size="xl" />
    </div>

    <div class="mb-4 flex justify-center text-muted-foreground">
      {#if isNotFound}
        <FileQuestion class="h-12 w-12" />
      {:else}
        <AlertTriangle class="h-12 w-12" />
      {/if}
    </div>

    <h1 class="mb-2 text-2xl font-bold text-foreground">
      {status}
    </h1>

    <h2 class="mb-2 text-lg font-semibold text-foreground">
      {#if isNotFound}
        {$_('error_page.not_found')}
      {:else}
        {$_('error_page.internal_error')}
      {/if}
    </h2>

    <p class="mb-6 text-sm text-muted-foreground">
      {#if errorMessage}
        {errorMessage}
      {:else if isNotFound}
        {$_('error_page.not_found_description')}
      {:else}
        {$_('error_page.internal_error_description')}
      {/if}
    </p>

    {#if errorId}
      <p class="mb-6 font-mono text-xs text-muted-foreground">
        {$_('error_page.error_id', { values: { errorId } })}
      </p>
    {/if}

    <div class="flex items-center justify-center gap-3">
      <a
        href="/"
        class="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:opacity-90 transition-opacity"
      >
        <Home class="h-4 w-4" />
        {$_('error_page.back_home')}
      </a>

      {#if !isNotFound}
        <button
          onclick={() => location.reload()}
          class="inline-flex items-center gap-2 rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted transition-colors"
        >
          <RefreshCw class="h-4 w-4" />
          {$_('error_page.try_again')}
        </button>
      {/if}
    </div>
  </div>
</div>
