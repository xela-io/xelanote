<script lang="ts">
  import { Check, Copy } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';

  interface Props {
    onClose: () => void;
  }

  const { onClose }: Props = $props();

  type TabType = 'syntax' | 'wikilinks' | 'code';
  let activeTab = $state<TabType>('syntax');

  // Copy to clipboard state
  let copiedCode = $state<string | null>(null);

  async function copyToClipboard(code: string) {
    try {
      await navigator.clipboard.writeText(code);
      copiedCode = code;
      setTimeout(() => {
        copiedCode = null;
      }, 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  }

  // Content for each tab (reactive for locale changes)
  const g = 'dialog.markdown_guide';

  const syntaxContent = $derived([
    {
      title: $_(`${g}.syntax.headings.title`),
      description: $_(`${g}.syntax.headings.description`),
      examples: [
        $_(`${g}.syntax.headings.example_h1`),
        $_(`${g}.syntax.headings.example_h2`),
        $_(`${g}.syntax.headings.example_h3`),
      ],
    },
    {
      title: $_(`${g}.syntax.formatting.title`),
      description: $_(`${g}.syntax.formatting.description`),
      examples: [
        $_(`${g}.syntax.formatting.example_bold`),
        $_(`${g}.syntax.formatting.example_italic`),
        $_(`${g}.syntax.formatting.example_bold_italic`),
        $_(`${g}.syntax.formatting.example_strikethrough`),
      ],
    },
    {
      title: $_(`${g}.syntax.lists.title`),
      description: $_(`${g}.syntax.lists.description`),
      examples: [
        $_(`${g}.syntax.lists.example_unordered`),
        $_(`${g}.syntax.lists.example_ordered`),
      ],
    },
    {
      title: $_(`${g}.syntax.quotes.title`),
      description: $_(`${g}.syntax.quotes.description`),
      examples: [$_(`${g}.syntax.quotes.example`)],
    },
    {
      title: $_(`${g}.syntax.due_dates.title`),
      description: $_(`${g}.syntax.due_dates.description`),
      examples: [
        $_(`${g}.syntax.due_dates.example_basic`),
        $_(`${g}.syntax.due_dates.example_task`),
      ],
    },
  ]);

  const wikilinksContent = $derived([
    {
      title: $_(`${g}.wikilinks.simple.title`),
      description: $_(`${g}.wikilinks.simple.description`),
      examples: [$_(`${g}.wikilinks.simple.example`)],
    },
    {
      title: $_(`${g}.wikilinks.alias.title`),
      description: $_(`${g}.wikilinks.alias.description`),
      examples: [$_(`${g}.wikilinks.alias.example`)],
    },
    {
      title: $_(`${g}.wikilinks.standard.title`),
      description: $_(`${g}.wikilinks.standard.description`),
      examples: [
        $_(`${g}.wikilinks.standard.example_markdown`),
        $_(`${g}.wikilinks.standard.example_url`),
      ],
    },
    {
      title: $_(`${g}.wikilinks.backlinks.title`),
      description: $_(`${g}.wikilinks.backlinks.description`),
      examples: [],
    },
  ]);

  const codeContent = $derived([
    {
      title: $_(`${g}.code.inline.title`),
      description: $_(`${g}.code.inline.description`),
      examples: [$_(`${g}.code.inline.example`)],
    },
    {
      title: $_(`${g}.code.block.title`),
      description: $_(`${g}.code.block.description`),
      examples: [$_(`${g}.code.block.example`)],
    },
    {
      title: $_(`${g}.code.block_language.title`),
      description: $_(`${g}.code.block_language.description`),
      examples: [$_(`${g}.code.block_language.example`)],
    },
    {
      title: $_(`${g}.code.images.title`),
      description: $_(`${g}.code.images.description`),
      examples: [$_(`${g}.code.images.example_external`), $_(`${g}.code.images.example_upload`)],
    },
  ]);
</script>

<BaseDialog
  open={true}
  title={$_('dialog.markdown_guide.title')}
  {onClose}
  size="lg"
  scrollable={true}
>
  {#snippet content()}
    <!-- Tabs -->
    <div class="flex border-b border-border -mx-4 -mt-4 mb-4">
      <button
        type="button"
        onclick={() => (activeTab = 'syntax')}
        class="flex-1 px-4 py-2 text-sm font-medium border-b-2 transition-colors"
        class:border-primary={activeTab === 'syntax'}
        class:text-primary={activeTab === 'syntax'}
        class:border-transparent={activeTab !== 'syntax'}
        class:text-muted-foreground={activeTab !== 'syntax'}
      >
        {$_('dialog.markdown_guide.tab_syntax')}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'wikilinks')}
        class="flex-1 px-4 py-2 text-sm font-medium border-b-2 transition-colors"
        class:border-primary={activeTab === 'wikilinks'}
        class:text-primary={activeTab === 'wikilinks'}
        class:border-transparent={activeTab !== 'wikilinks'}
        class:text-muted-foreground={activeTab !== 'wikilinks'}
      >
        {$_('dialog.markdown_guide.tab_wikilinks')}
      </button>
      <button
        type="button"
        onclick={() => (activeTab = 'code')}
        class="flex-1 px-4 py-2 text-sm font-medium border-b-2 transition-colors"
        class:border-primary={activeTab === 'code'}
        class:text-primary={activeTab === 'code'}
        class:border-transparent={activeTab !== 'code'}
        class:text-muted-foreground={activeTab !== 'code'}
      >
        {$_('dialog.markdown_guide.tab_code')}
      </button>
    </div>

    <!-- Tab Content -->
    <div class="space-y-6">
      {#if activeTab === 'syntax'}
        {#each syntaxContent as section (section.title)}
          <div class="space-y-2">
            <h3 class="text-base font-semibold">{section.title}</h3>
            <p class="text-sm text-muted-foreground">{section.description}</p>
            {#each section.examples as example (example)}
              <div class="relative group">
                <pre
                  class="bg-muted p-3 rounded-md text-sm font-mono overflow-x-auto">{example}</pre>
                <button
                  type="button"
                  onclick={() => copyToClipboard(example)}
                  class="absolute top-2 right-2 p-1.5 bg-background border border-border rounded-md opacity-0 group-hover:opacity-100 [@media(pointer:coarse)]:opacity-100 transition-opacity"
                  title={$_('dialog.markdown_guide.copy_tooltip')}
                >
                  {#if copiedCode === example}
                    <Check size={14} class="text-success" />
                  {:else}
                    <Copy size={14} />
                  {/if}
                </button>
              </div>
            {/each}
          </div>
        {/each}
      {:else if activeTab === 'wikilinks'}
        {#each wikilinksContent as section (section.title)}
          <div class="space-y-2">
            <h3 class="text-base font-semibold">{section.title}</h3>
            <p class="text-sm text-muted-foreground">{section.description}</p>
            {#each section.examples as example (example)}
              <div class="relative group">
                <pre
                  class="bg-muted p-3 rounded-md text-sm font-mono overflow-x-auto">{example}</pre>
                <button
                  type="button"
                  onclick={() => copyToClipboard(example)}
                  class="absolute top-2 right-2 p-1.5 bg-background border border-border rounded-md opacity-0 group-hover:opacity-100 [@media(pointer:coarse)]:opacity-100 transition-opacity"
                  title={$_('dialog.markdown_guide.copy_tooltip')}
                >
                  {#if copiedCode === example}
                    <Check size={14} class="text-success" />
                  {:else}
                    <Copy size={14} />
                  {/if}
                </button>
              </div>
            {/each}
          </div>
        {/each}

        <div class="border border-border rounded-md p-4 bg-muted/50">
          <h4 class="text-sm font-semibold mb-2">
            {$_('dialog.markdown_guide.wikilinks.behavior.title')}
          </h4>
          <ul class="text-sm text-muted-foreground space-y-1 list-disc list-inside">
            <li>
              <strong>{$_('dialog.markdown_guide.wikilinks.behavior.existing_label')}</strong>
              {$_('dialog.markdown_guide.wikilinks.behavior.existing_text')}
            </li>
            <li>
              <strong>{$_('dialog.markdown_guide.wikilinks.behavior.nonexisting_label')}</strong>
              {$_('dialog.markdown_guide.wikilinks.behavior.nonexisting_text')}
            </li>
            <li>
              <strong>{$_('dialog.markdown_guide.wikilinks.behavior.detection_label')}</strong>
              {$_('dialog.markdown_guide.wikilinks.behavior.detection_text')}
            </li>
          </ul>
        </div>
      {:else if activeTab === 'code'}
        {#each codeContent as section (section.title)}
          <div class="space-y-2">
            <h3 class="text-base font-semibold">{section.title}</h3>
            <p class="text-sm text-muted-foreground">{section.description}</p>
            {#each section.examples as example (example)}
              <div class="relative group">
                <pre
                  class="bg-muted p-3 rounded-md text-sm font-mono overflow-x-auto">{example}</pre>
                <button
                  type="button"
                  onclick={() => copyToClipboard(example)}
                  class="absolute top-2 right-2 p-1.5 bg-background border border-border rounded-md opacity-0 group-hover:opacity-100 [@media(pointer:coarse)]:opacity-100 transition-opacity"
                  title={$_('dialog.markdown_guide.copy_tooltip')}
                >
                  {#if copiedCode === example}
                    <Check size={14} class="text-success" />
                  {:else}
                    <Copy size={14} />
                  {/if}
                </button>
              </div>
            {/each}
          </div>
        {/each}

        <div class="border border-border rounded-md p-4 bg-muted/50">
          <h4 class="text-sm font-semibold mb-2">
            {$_('dialog.markdown_guide.code.upload.title')}
          </h4>
          <ul class="text-sm text-muted-foreground space-y-1 list-disc list-inside">
            <li>{$_('dialog.markdown_guide.code.upload.drag_drop')}</li>
            <li>{$_('dialog.markdown_guide.code.upload.paste')}</li>
            <li>{$_('dialog.markdown_guide.code.upload.toolbar_button')}</li>
          </ul>
        </div>
      {/if}
    </div>
  {/snippet}

  {#snippet footer()}
    <div class="flex items-center justify-between w-full text-xs text-muted-foreground">
      <div class="flex items-center gap-4">
        <span
          ><kbd class="px-2 py-1 bg-muted border border-border rounded text-xs font-mono"
            >Ctrl+/</kbd
          >
          {$_('dialog.markdown_guide.shortcut_open')}</span
        >
        <span
          ><kbd class="px-2 py-1 bg-muted border border-border rounded text-xs font-mono">Esc</kbd>
          {$_('common.close')}</span
        >
      </div>
      <div>xelanote Markdown Guide</div>
    </div>
  {/snippet}
</BaseDialog>
