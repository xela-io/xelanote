<script lang="ts">
  import { Calendar, CalendarCheck, CalendarClock, CalendarX2, FileText } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { type DueDateItem, getDueDates } from '$lib/api';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import { getDueDateStatus } from '$lib/editor/markdown';

  let dueDates = $state<DueDateItem[]>([]);
  let isLoading = $state(true);
  let error = $state<string | null>(null);
  let showCompleted = $state(false);

  async function loadDueDates() {
    isLoading = true;
    error = null;
    try {
      dueDates = await getDueDates(showCompleted);
    } catch (e) {
      error = e instanceof Error ? e.message : 'Unknown error';
    } finally {
      isLoading = false;
    }
  }

  // Load on mount + reload when showCompleted changes
  $effect(() => {
    void showCompleted; // eslint: track reactivity
    loadDueDates();
  });

  // Group due dates by status
  const grouped = $derived(() => {
    const groups: Record<string, DueDateItem[]> = {
      overdue: [],
      today: [],
      soon: [],
      future: [],
    };
    for (const item of dueDates) {
      const status = getDueDateStatus(item.due_date);
      groups[status].push(item);
    }
    return groups;
  });

  function formatDate(dateStr: string): string {
    const date = new Date(dateStr + 'T00:00:00');
    return date.toLocaleDateString(undefined, {
      weekday: 'short',
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  }

  function stripDueDateSyntax(text: string): string {
    return text.replace(/@due\(\d{4}-\d{2}-\d{2}\)/g, '').trim();
  }
</script>

<div data-body-scroll class="h-full overflow-y-auto">
  <div class="max-w-4xl mx-auto px-6 py-8">
    <!-- Header -->
    <div class="flex flex-col gap-3 mb-6 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex items-center gap-2 sm:gap-3 min-w-0">
        <MobileSidebarInlineToggle />
        <CalendarClock size={28} class="text-muted-foreground" />
        <h1 class="text-2xl font-bold">{$_('page.due_dates.title')}</h1>
        {#if dueDates.length > 0}
          <span class="text-muted-foreground">({dueDates.length})</span>
        {/if}
      </div>

      <label
        class="flex items-center gap-2 text-sm cursor-pointer select-none self-start sm:self-auto"
      >
        <input
          type="checkbox"
          bind:checked={showCompleted}
          class="rounded border-border accent-primary"
        />
        <span class="text-muted-foreground">{$_('page.due_dates.show_completed')}</span>
      </label>
    </div>

    <!-- Loading State -->
    {#if isLoading}
      <div class="text-center py-12 text-muted-foreground" role="status" aria-live="polite">
        {$_('common.loading')}
      </div>
    {:else if error}
      <div class="text-center py-12 text-destructive">
        {error}
      </div>
    {:else if dueDates.length === 0}
      <!-- Empty State -->
      <div class="text-center py-20">
        <CalendarClock size={64} class="mx-auto text-muted-foreground/50 mb-4" />
        <h2 class="text-xl font-semibold text-muted-foreground mb-2">
          {$_('page.due_dates.empty_title')}
        </h2>
        <p class="text-muted-foreground">{$_('page.due_dates.empty_description')}</p>
      </div>
    {:else}
      <!-- Sections -->
      {#each [{ key: 'overdue', icon: CalendarX2, iconClass: 'text-destructive' }, { key: 'today', icon: CalendarCheck, iconClass: 'text-warning' }, { key: 'soon', icon: Calendar, iconClass: 'text-warning/70' }, { key: 'future', icon: Calendar, iconClass: 'text-muted-foreground' }] as section (section.key)}
        {@const items = grouped()[section.key]}
        {#if items.length > 0}
          <div class="mb-8">
            <div class="flex items-center gap-2 mb-3">
              <section.icon size={18} class={section.iconClass} />
              <h2 class="text-lg font-semibold">
                {$_(`page.due_dates.${section.key}`)}
              </h2>
              <span class="text-sm text-muted-foreground">({items.length})</span>
            </div>

            <div class="space-y-2">
              {#each items as item (item.id)}
                <div
                  class="border border-border rounded-lg p-3 hover:shadow-md transition-shadow bg-card flex items-start gap-3"
                  class:opacity-60={item.is_completed}
                >
                  <!-- Date Badge -->
                  <span class="due-date due-date-{section.key} text-xs shrink-0 mt-0.5">
                    {formatDate(item.due_date)}
                  </span>

                  <!-- Content -->
                  <div class="flex-1 min-w-0">
                    <p
                      class="text-sm"
                      class:line-through={item.is_completed}
                      class:text-muted-foreground={item.is_completed}
                    >
                      {stripDueDateSyntax(item.line_text)}
                    </p>
                    <button
                      onclick={() => goto(`/note/${item.note_id}`)}
                      class="flex items-center gap-1 text-xs text-primary hover:underline mt-1"
                    >
                      <FileText size={12} />
                      {item.note_title}
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      {/each}
    {/if}
  </div>
</div>
