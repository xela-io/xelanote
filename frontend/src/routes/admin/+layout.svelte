<script lang="ts">
  import { goto } from '$app/navigation';
  import * as admin from '$lib/stores/admin.svelte';
  import * as auth from '$lib/stores/auth.svelte';

  const { children } = $props();
  let checking = $state(true);
  let authorized = $state(false);

  // Check admin access
  $effect(() => {
    const isAuthenticated = auth.isAuthenticated();
    const isAdmin = auth.isAdmin();

    if (!isAuthenticated) {
      goto('/login');
      return;
    }

    if (!isAdmin) {
      goto('/');
      return;
    }

    authorized = true;
    checking = false;
  });

  // Cleanup on unmount
  $effect(() => {
    return () => {
      admin.resetAdminState();
    };
  });
</script>

{#if checking}
  <div class="flex items-center justify-center h-full">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
  </div>
{:else if authorized}
  {@render children()}
{/if}
