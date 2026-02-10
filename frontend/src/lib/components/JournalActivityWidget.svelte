<script lang="ts">
  import { Flame, Loader2 } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import * as journal from '$lib/stores/journal.svelte';

  const yearCalendarLoading = $derived(journal.getYearCalendarLoading());
  const yearCalendarError = $derived(journal.getYearCalendarError());

  const currentYear = $derived(parseInt(journal.getTodayDate().substring(0, 4), 10));

  const yearDates = $derived(journal.getYearDatesSetForYear(currentYear));
  const prevDec = $derived(journal.getPrevDecDatesForYear(currentYear));
  const streaks = $derived(journal.calculateStreaks(yearDates, prevDec));

  const sparklineData = $derived.by(() => {
    const today = journal.getTodayDate();
    const yr = parseInt(today.substring(0, 4), 10);
    const yDates = journal.getYearDatesSetForYear(yr);
    const pDec = journal.getPrevDecDatesForYear(yr);
    const days: { date: string; hasEntry: boolean; isToday: boolean }[] = [];

    for (let i = 29; i >= 0; i--) {
      // eslint-disable-next-line svelte/prefer-svelte-reactivity
      const d = new Date(today + 'T00:00:00');
      d.setDate(d.getDate() - i);
      const dateStr = journal.formatDate(d);
      const inCurrentYear = d.getFullYear() === yr;
      days.push({
        date: dateStr,
        hasEntry: inCurrentYear ? yDates.has(dateStr) : pDec.includes(dateStr),
        isToday: i === 0,
      });
    }
    return days;
  });

  const entryCount = $derived(sparklineData.filter((d) => d.hasEntry).length);

  const sparklineLabel = $derived(
    $_('page.journal.recentActivity') +
      ': ' +
      entryCount +
      ' / 30 ' +
      $_('page.journal.days', { values: { count: 30 } })
  );

  // Show widget only if there's any activity
  const showWidget = $derived(streaks.current > 0 || streaks.longest > 0);

  onMount(() => {
    journal.ensureYearCacheLoaded(new Date().getFullYear());
  });
</script>

{#if showWidget}
  <div class="border border-border rounded-lg p-3">
    {#if yearCalendarLoading}
      <div class="flex items-center justify-center py-2">
        <Loader2 class="w-4 h-4 animate-spin text-muted-foreground" />
      </div>
    {:else if yearCalendarError}
      <div class="text-xs text-destructive text-center">{yearCalendarError}</div>
    {:else}
      <!-- Streak header -->
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center gap-1.5 text-sm">
          <Flame size={16} class="text-orange-500" />
          <span class="font-medium">
            {streaks.current}
            {$_('page.journal.days', { values: { count: streaks.current } })}
          </span>
        </div>
        {#if streaks.longest > streaks.current}
          <span class="text-xs text-muted-foreground">
            {$_('page.journal.longestStreak')}: {streaks.longest}
          </span>
        {/if}
      </div>

      <!-- 30-day sparkline -->
      <div role="img" aria-label={sparklineLabel}>
        <div class="flex gap-[2px]" aria-hidden="true">
          {#each sparklineData as day (day.date)}
            <div
              class="flex-1 h-2.5 min-w-0 rounded-[2px]
                {day.hasEntry ? 'bg-primary/40' : 'bg-muted/50'}
                {day.isToday
                ? 'ring-[1.5px] ring-primary ring-offset-1 ring-offset-background'
                : ''}"
            ></div>
          {/each}
        </div>
        <div class="flex justify-between mt-1" aria-hidden="true">
          <span class="text-[9px] text-muted-foreground"
            >{$_('page.journal.daysAgo', { values: { count: 30 } })}</span
          >
          <span class="text-[9px] text-muted-foreground">{$_('page.journal.today')}</span>
        </div>
      </div>
    {/if}
  </div>
{/if}
