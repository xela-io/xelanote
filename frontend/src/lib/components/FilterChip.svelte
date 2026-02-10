<script lang="ts">
  import { X, Folder, Tag, Calendar } from 'lucide-svelte';

  interface Props {
    type: 'folder' | 'tag' | 'date';
    label: string;
    onRemove: () => void;
  }

  const { type, label, onRemove }: Props = $props();

  const iconMap = {
    folder: Folder,
    tag: Tag,
    date: Calendar,
  } as const;

  const colorMap = {
    folder: 'text-primary',
    tag: 'text-success',
    date: 'text-purple-600',
  } as const;

  const Icon = $derived(iconMap[type] ?? Tag);
  const iconColor = $derived(colorMap[type] ?? 'text-muted-foreground');
</script>

<div class="inline-flex items-center gap-1.5 px-2.5 py-1 bg-muted rounded-full text-sm">
  <Icon size={14} class={iconColor} />
  <span class="text-foreground">{label}</span>
  <button
    type="button"
    onclick={onRemove}
    class="ml-0.5 p-0.5 hover:bg-accent rounded-full transition-colors"
    aria-label="Remove filter"
  >
    <X size={12} class="text-muted-foreground" />
  </button>
</div>
