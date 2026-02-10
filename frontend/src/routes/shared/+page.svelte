<script lang="ts">
  import { _ } from 'svelte-i18n';

  import SharedWithMeList from '$lib/components/SharedWithMeList.svelte';
</script>

<svelte:head>
  <title>{$_('sharing.shared_with_me')} - xelanote</title>
</svelte:head>

<SharedWithMeList />
