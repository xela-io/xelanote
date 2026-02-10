<script lang="ts">
  import { ChevronLeft, ChevronRight, Flame, Loader2 } from 'lucide-svelte';
  import { SvelteDate } from 'svelte/reactivity';
  import { _ } from 'svelte-i18n';

  import * as journal from '$lib/stores/journal.svelte';

  const yearCalendarYear = $derived(journal.getYearCalendarYear());
  const yearCalendarLoading = $derived(journal.getYearCalendarLoading());
  const yearCalendarError = $derived(journal.getYearCalendarError());
  const yearDatesSet = $derived(journal.getYearDatesSet());
  const prevDecDates = $derived(journal.getPrevDecDates());
  const today = $derived(journal.getTodayDate());
  const journalLoading = $derived(journal.getJournalLoading());

  const streaks = $derived(journal.calculateStreaks(yearDatesSet, prevDecDates));

  const monthLabels = [
    'Jan',
    'Feb',
    'Mar',
    'Apr',
    'May',
    'Jun',
    'Jul',
    'Aug',
    'Sep',
    'Oct',
    'Nov',
    'Dec',
  ];
  const dayLabels = ['Mo', '', 'We', '', 'Fr', '', ''];

  // Cell size (w-3 h-3 = 12px) + 2px gap
  const CELL_SIZE = 14; // 12px cell + 2px gap
  const LABEL_COL_WIDTH = 20; // width for day labels column

  interface CellData {
    date: string;
    hasEntry: boolean;
    isToday: boolean;
    weekIndex: number;
    dayIndex: number;
  }

  // Build grid: columns = weeks, rows = days (Mon-Sun)
  const gridData = $derived.by(() => {
    const year = yearCalendarYear;

    const jan1 = new Date(year, 0, 1);
    let jan1Day = jan1.getDay() - 1; // Monday-based
    if (jan1Day < 0) jan1Day = 6;

    const dec31 = new Date(year, 11, 31);
    let dec31Day = dec31.getDay() - 1;
    if (dec31Day < 0) dec31Day = 6;

    // Start from Monday of the week containing Jan 1
    const startDate = new Date(year, 0, 1 - jan1Day);
    // End at Sunday of the week containing Dec 31
    const endDate = new Date(year, 11, 31 + (6 - dec31Day));

    // Build a 2D array: weeks[weekIdx][dayIdx]
    const weeks: (CellData | null)[][] = [];
    const d = new SvelteDate(startDate);
    let weekIdx = 0;
    let currentWeek: (CellData | null)[] = [];

    while (d <= endDate) {
      const dayIdx = d.getDay() - 1 < 0 ? 6 : d.getDay() - 1;
      const dateStr = journal.formatDate(d);

      // Ensure correct position in week array
      while (currentWeek.length < dayIdx) {
        currentWeek.push(null);
      }

      currentWeek.push({
        date: dateStr,
        hasEntry: yearDatesSet.has(dateStr),
        isToday: dateStr === today,
        weekIndex: weekIdx,
        dayIndex: dayIdx,
      });

      if (dayIdx === 6) {
        weeks.push(currentWeek);
        currentWeek = [];
        weekIdx++;
      }

      d.setDate(d.getDate() + 1);
    }

    // Push remaining week if any
    if (currentWeek.length > 0) {
      while (currentWeek.length < 7) currentWeek.push(null);
      weeks.push(currentWeek);
    }

    return weeks;
  });

  // Month label positions (column index where each month starts)
  const monthLabelPositions = $derived.by(() => {
    const year = yearCalendarYear;
    const positions: Array<{ label: string; col: number }> = [];

    const jan1 = new Date(year, 0, 1);
    let jan1Day = jan1.getDay() - 1;
    if (jan1Day < 0) jan1Day = 6;
    const startDate = new Date(year, 0, 1 - jan1Day);

    for (let m = 0; m < 12; m++) {
      const firstOfMonth = new Date(year, m, 1);
      const diffDays = Math.round(
        (firstOfMonth.getTime() - startDate.getTime()) / (1000 * 60 * 60 * 24)
      );
      const col = Math.floor(diffDays / 7);

      // Skip if same column as previous label
      if (positions.length > 0 && positions[positions.length - 1].col === col) continue;

      positions.push({ label: monthLabels[m], col });
    }

    return positions;
  });

  function handleCellClick(date: string) {
    if (journalLoading) return;
    journal.openJournalForDate(date);
  }

  function formatTooltipDate(dateStr: string): string {
    const d = new Date(dateStr + 'T00:00:00');
    return d.toLocaleDateString(undefined, {
      weekday: 'short',
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  }

  // Keyboard navigation for a11y (roving tabindex)
  let gridEl = $state<HTMLDivElement | null>(null);
  let focusedIndex = $state(-1);

  function handleGridKeydown(e: KeyboardEvent) {
    if (!gridEl) return;
    const cells = gridEl.querySelectorAll<HTMLButtonElement>('[data-cell]');
    if (cells.length === 0) return;

    let newIndex = focusedIndex;

    switch (e.key) {
      case 'ArrowRight': // Next week column (same day of week)
        newIndex = Math.min(focusedIndex + 7, cells.length - 1);
        e.preventDefault();
        break;
      case 'ArrowLeft': // Previous week column
        newIndex = Math.max(focusedIndex - 7, 0);
        e.preventDefault();
        break;
      case 'ArrowDown': // Next day in week
        newIndex = Math.min(focusedIndex + 1, cells.length - 1);
        e.preventDefault();
        break;
      case 'ArrowUp': // Previous day in week
        newIndex = Math.max(focusedIndex - 1, 0);
        e.preventDefault();
        break;
      case 'Home':
        newIndex = 0;
        e.preventDefault();
        break;
      case 'End':
        newIndex = cells.length - 1;
        e.preventDefault();
        break;
      case 'Enter':
      case ' ':
        if (focusedIndex >= 0 && focusedIndex < cells.length) {
          cells[focusedIndex].click();
          e.preventDefault();
        }
        return;
      default:
        return;
    }

    if (newIndex !== focusedIndex && newIndex >= 0 && newIndex < cells.length) {
      focusedIndex = newIndex;
      cells[newIndex].focus();
    }
  }

  function handleGridFocus() {
    if (focusedIndex < 0) focusedIndex = 0;
    const cells = gridEl?.querySelectorAll<HTMLButtonElement>('[data-cell]');
    if (cells && cells[focusedIndex]) {
      cells[focusedIndex].focus();
    }
  }
</script>

<div class="border border-border rounded-lg p-4">
  <!-- Header: Year navigation + Streaks -->
  <div class="flex items-center justify-between mb-3">
    <h3 class="text-sm font-semibold text-muted-foreground uppercase tracking-wide">
      {$_('page.journal.yearOverview')}
    </h3>

    <div class="flex items-center gap-4">
      <!-- Streak display -->
      {#if streaks.current > 0 || streaks.todayDone}
        <div class="flex items-center gap-1.5 text-sm text-muted-foreground">
          <Flame size={16} class="text-orange-500" />
          <span class="font-medium"
            >{streaks.current}
            {$_('page.journal.days', { values: { count: streaks.current } })}</span
          >
          {#if streaks.longest > streaks.current}
            <span class="text-xs">({$_('page.journal.longestStreak')}: {streaks.longest})</span>
          {/if}
        </div>
      {/if}

      <!-- Year navigation -->
      <div class="flex items-center gap-1">
        <button
          class="p-1 hover:bg-accent rounded disabled:opacity-50"
          onclick={() => journal.previousYear()}
          disabled={yearCalendarLoading}
          title={$_('page.journal.previousYear')}
        >
          <ChevronLeft size={16} />
        </button>
        <span class="font-medium text-sm min-w-[3rem] text-center">{yearCalendarYear}</span>
        <button
          class="p-1 hover:bg-accent rounded disabled:opacity-50"
          onclick={() => journal.nextYear()}
          disabled={yearCalendarLoading}
          title={$_('page.journal.nextYear')}
        >
          <ChevronRight size={16} />
        </button>
      </div>
    </div>
  </div>

  {#if yearCalendarLoading}
    <div class="flex items-center justify-center py-8">
      <Loader2 class="w-6 h-6 animate-spin text-muted-foreground" />
    </div>
  {:else if yearCalendarError}
    <div class="text-sm text-destructive py-4 text-center">{yearCalendarError}</div>
  {:else}
    <!-- Heatmap grid -->
    <div class="overflow-x-auto">
      <div class="relative" style="padding-left: {LABEL_COL_WIDTH}px;">
        <!-- Month labels row -->
        <div class="relative h-4 mb-1">
          {#each monthLabelPositions as { label, col } (col)}
            <span
              class="text-[10px] text-muted-foreground absolute leading-none"
              style="left: {col * CELL_SIZE}px;">{label}</span
            >
          {/each}
        </div>

        <!-- Grid area: day labels + cells -->
        <div class="relative">
          <!-- Day labels (absolute positioned to align with rows) -->
          <div class="absolute" style="left: -{LABEL_COL_WIDTH}px; top: 0;">
            {#each dayLabels as label, _i (_i)}
              <div class="flex items-center justify-end pr-1" style="height: {CELL_SIZE}px;">
                {#if label}
                  <span class="text-[9px] text-muted-foreground leading-none">{label}</span>
                {/if}
              </div>
            {/each}
          </div>

          <!-- Cells grid -->
          <div
            class="flex gap-[2px]"
            role="grid"
            aria-label={$_('page.journal.yearOverview')}
            bind:this={gridEl}
            onkeydown={handleGridKeydown}
            onfocus={handleGridFocus}
            tabindex="0"
          >
            {#each gridData as week, _weekIdx (_weekIdx)}
              <div class="flex flex-col gap-[2px]" role="row">
                {#each week as cell, _dayIdx (_dayIdx)}
                  {#if cell}
                    <button
                      data-cell
                      role="gridcell"
                      class="heatmap-cell w-3 h-3 rounded-[2px] transition-colors
                        {cell.hasEntry
                        ? 'bg-primary/40 hover:bg-primary/60'
                        : 'bg-muted/50 hover:bg-muted'}
                        {cell.isToday
                        ? 'ring-[1.5px] ring-primary ring-offset-1 ring-offset-background'
                        : ''}"
                      onclick={() => handleCellClick(cell.date)}
                      disabled={journalLoading}
                      title={formatTooltipDate(cell.date)}
                      tabindex="-1"
                      aria-label="{formatTooltipDate(cell.date)}{cell.hasEntry
                        ? ` - ${$_('page.journal.legendEntry')}`
                        : ''}"
                    ></button>
                  {:else}
                    <div class="w-3 h-3" role="presentation"></div>
                  {/if}
                {/each}
              </div>
            {/each}
          </div>
        </div>

        <!-- Legend -->
        <div class="flex items-center gap-3 mt-2">
          <div class="flex items-center gap-1">
            <div class="w-3 h-3 rounded-[2px] bg-muted/50"></div>
            <span class="text-xs text-muted-foreground">{$_('page.journal.legendNoEntry')}</span>
          </div>
          <div class="flex items-center gap-1">
            <div class="w-3 h-3 rounded-[2px] bg-primary/40"></div>
            <span class="text-xs text-muted-foreground">{$_('page.journal.legendEntry')}</span>
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>
