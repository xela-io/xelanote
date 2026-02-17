<script lang="ts">
  import { ExternalLink, FileText, Group, Type } from 'lucide-svelte';

  type ToolbarAction = 'add-text' | 'add-file' | 'add-link' | 'add-group';

  const { onAction }: { onAction: (action: ToolbarAction) => void } = $props();

  const tools: { action: ToolbarAction; icon: typeof Type; label: string }[] = [
    { action: 'add-text', icon: Type, label: 'Text' },
    { action: 'add-file', icon: FileText, label: 'Note' },
    { action: 'add-link', icon: ExternalLink, label: 'Link' },
    { action: 'add-group', icon: Group, label: 'Group' },
  ];
</script>

<div class="canvas-toolbar">
  {#each tools as tool (tool.action)}
    <button class="canvas-toolbar-btn" title={tool.label} onclick={() => onAction(tool.action)}>
      <tool.icon size={18} />
    </button>
  {/each}
</div>

<style>
  .canvas-toolbar {
    display: flex;
    align-items: center;
    padding: 6px 12px;
    gap: 4px;
    background: color-mix(in oklch, var(--color-card) 88%, transparent);
    backdrop-filter: blur(12px) saturate(1.2);
    border: 1px solid color-mix(in oklch, var(--color-border) 60%, transparent);
    border-radius: 1rem;
    box-shadow: 0 8px 28px color-mix(in oklch, var(--color-foreground) 10%, transparent);
  }

  .canvas-toolbar-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    border: none;
    background: transparent;
    border-radius: 6px;
    color: var(--color-foreground);
    cursor: pointer;
    transition: all 150ms ease;
  }

  .canvas-toolbar-btn:hover {
    background: var(--color-accent);
    transform: translateY(-1px);
  }

  .canvas-toolbar-btn:active {
    transform: scale(0.97);
  }

  @media (pointer: coarse) {
    .canvas-toolbar-btn {
      width: 44px;
      height: 44px;
    }
  }
</style>
