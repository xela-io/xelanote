<script lang="ts">
  import { Globe } from 'lucide-svelte';
  import { locale } from 'svelte-i18n';

  function handleChange(e: Event) {
    const newLocale = (e.target as HTMLSelectElement).value;
    locale.set(newLocale);
    window.localStorage.setItem('locale', newLocale);
  }
</script>

<div class="language-selector">
  <Globe size={16} />
  <select value={$locale} onchange={handleChange}>
    <option value="de">Deutsch</option>
    <option value="en">English</option>
  </select>
</div>

<style>
  .language-selector {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--color-muted-foreground);
  }

  select {
    padding: 0.25rem 0.5rem;
    font-size: 0.875rem;
    background: transparent;
    border: 1px solid var(--color-border);
    border-radius: 4px;
    color: var(--color-muted-foreground);
    cursor: pointer;
  }

  select:hover {
    border-color: var(--color-primary);
    color: var(--color-foreground);
  }

  select:focus {
    outline: none;
    border-color: var(--color-primary);
  }
</style>
