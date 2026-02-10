<script lang="ts">
  const { count = 10 } = $props<{ count?: number }>();
</script>

<div class="space-y-2 p-4 animate-pulse">
  {#each Array(count) as _, i (i)}
    <div class="flex items-center gap-3 p-2">
      <!-- Icon skeleton -->
      <div class="h-4 w-4 bg-muted rounded"></div>

      <!-- Text skeleton -->
      <div class="flex-1">
        <div class="h-4 bg-muted rounded" style="width: {60 + Math.random() * 30}%"></div>
      </div>

      <!-- Metadata skeleton -->
      <div class="h-3 w-16 bg-muted rounded"></div>
    </div>
  {/each}
</div>
