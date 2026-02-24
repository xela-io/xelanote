<script lang="ts">
  import type { EditorView } from '@codemirror/view';
  import { _ } from 'svelte-i18n';

  import { FEATURE_FLAGS } from '$lib/config';
  import { imageResize } from '$lib/editor/image-resize';
  import { mathRenderer } from '$lib/editor/math-action';
  import { mermaidRenderer } from '$lib/editor/mermaid-action';
  import { highlightSearchTerms } from '$lib/editor/preview-highlight';
  import { previewRenderer } from '$lib/editor/preview-renderer';
  import { setupScrollSync, syncPreviewToEditor } from '$lib/editor/scroll-sync';
  import {
    setupElementScrollSync,
    syncPreviewToEditorElements,
  } from '$lib/editor/scroll-sync-elements';
  import { shikiHighlighter } from '$lib/editor/shiki-action';
  import { taskCollapse, type TaskCollapseOptions } from '$lib/editor/task-collapse';
  import { taskSortable, type TaskSortableOptions } from '$lib/editor/task-sortable';

  import TableOfContents from '../TableOfContents.svelte';

  interface Props {
    renderedContent: string;
    headings: ReturnType<typeof import('$lib/editor/markdown').extractHeadings>;
    editorView: EditorView | undefined;
    editorMode: string;
    isMobile: boolean;
    splitPosition: number;
    previewThemeClass: string;
    taskCollapseOptions: TaskCollapseOptions;
    taskSortableOptions: TaskSortableOptions;
    showFindReplace: boolean;
    findReplaceQuery: string;
    findReplaceCaseSensitive: boolean;
    onPreviewClick: (e: MouseEvent) => void;
    onHeadingClick: (slug: string) => void;
    onImageResize: (imageIndex: number, newWidth: number) => void;
  }

  const {
    renderedContent,
    headings,
    editorView,
    editorMode,
    isMobile,
    splitPosition,
    previewThemeClass,
    taskCollapseOptions,
    taskSortableOptions,
    showFindReplace,
    findReplaceQuery,
    findReplaceCaseSensitive,
    onPreviewClick,
    onHeadingClick,
    onImageResize,
  }: Props = $props();

  let previewScrollRef: HTMLDivElement | null = $state(null);
  let activePreviewHeadingSlug: string | null = $state(null);

  function updateActivePreviewHeading() {
    const scroller = previewScrollRef;
    if (!scroller) {
      activePreviewHeadingSlug = null;
      return;
    }

    const markdownPreview = scroller.querySelector<HTMLElement>('.markdown-preview');
    if (!markdownPreview) {
      activePreviewHeadingSlug = null;
      return;
    }

    const headingElements = Array.from(
      markdownPreview.querySelectorAll<HTMLElement>(
        'h1[id], h2[id], h3[id], h4[id], h5[id], h6[id]'
      )
    );
    if (headingElements.length === 0) {
      activePreviewHeadingSlug = null;
      return;
    }

    const maxScrollTop = Math.max(0, scroller.scrollHeight - scroller.clientHeight);
    if (maxScrollTop - scroller.scrollTop <= 24) {
      activePreviewHeadingSlug = headingElements[headingElements.length - 1]?.id ?? null;
      return;
    }

    const scrollerRect = scroller.getBoundingClientRect();
    const activationOffset = Math.min(140, Math.max(56, scroller.clientHeight * 0.18));
    let currentSlug = headingElements[0].id;

    for (const heading of headingElements) {
      const headingTop = heading.getBoundingClientRect().top - scrollerRect.top;
      if (headingTop <= activationOffset) {
        currentSlug = heading.id;
      } else {
        break;
      }
    }

    activePreviewHeadingSlug = currentSlug || null;
  }

  // Scroll sync: editor → preview
  $effect(() => {
    const isDesktopSplit = !isMobile && editorMode === 'split';
    const previewScroller = previewScrollRef;
    const editorScroller = editorView?.scrollDOM;
    if (!isDesktopSplit || !previewScroller || !editorScroller) return;

    if (FEATURE_FLAGS.elementScrollSync && editorView) {
      return setupElementScrollSync(editorView, previewScroller);
    }
    return setupScrollSync(editorScroller, previewScroller);
  });

  // Scroll sync: re-sync when content changes
  $effect(() => {
    const isDesktopSplit = !isMobile && editorMode === 'split';
    const previewScroller = previewScrollRef;
    const editorScroller = editorView?.scrollDOM;
    const renderedContentSnapshot = renderedContent;
    if (!isDesktopSplit || !previewScroller || !editorScroller) return;

    requestAnimationFrame(() => {
      void renderedContentSnapshot;
      if (FEATURE_FLAGS.elementScrollSync && editorView) {
        syncPreviewToEditorElements(previewScroller, editorView);
      } else {
        syncPreviewToEditor(previewScroller, editorScroller);
      }
    });
  });

  // Active heading tracking for TOC highlight
  $effect(() => {
    const mode = editorMode;
    const previewScroller = previewScrollRef;
    const renderedContentSnapshot = renderedContent;
    const shouldTrackPreviewHeading = mode === 'preview' || mode === 'split';

    if (!shouldTrackPreviewHeading || !previewScroller) {
      activePreviewHeadingSlug = null;
      return;
    }

    let rafId = 0;
    const scheduleUpdate = () => {
      if (rafId) return;
      rafId = requestAnimationFrame(() => {
        rafId = 0;
        void renderedContentSnapshot;
        updateActivePreviewHeading();
      });
    };

    scheduleUpdate();
    previewScroller.addEventListener('scroll', scheduleUpdate, { passive: true });
    window.addEventListener('resize', scheduleUpdate);

    return () => {
      previewScroller.removeEventListener('scroll', scheduleUpdate);
      window.removeEventListener('resize', scheduleUpdate);
      if (rafId) cancelAnimationFrame(rafId);
    };
  });
</script>

<!-- Theme wrapper for preview (overflow-auto for internal scrolling) -->
<div
  class="preview-pane-shell relative {isMobile ? '' : 'overflow-auto'} {previewThemeClass}"
  class:flex-1={editorMode !== 'split'}
  style={editorMode === 'split' ? `width: ${100 - splitPosition}%;` : ''}
  bind:this={previewScrollRef}
>
  <!-- Floating Table of Contents -->
  {#if headings.length > 0}
    <TableOfContents {headings} activeSlug={activePreviewHeadingSlug} {onHeadingClick} />
  {/if}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <!-- Intentional: Click handler delegates to interactive elements (wikilinks, checkboxes) in rendered markdown. These elements are natively interactive in the HTML output. -->
  <div
    class="markdown-preview"
    role="region"
    aria-label={$_('component.editor.preview_area')}
    onclick={onPreviewClick}
    use:previewRenderer={{ html: renderedContent }}
    use:shikiHighlighter={{ revision: renderedContent }}
    use:mathRenderer={{ revision: renderedContent }}
    use:mermaidRenderer={{ revision: renderedContent }}
    use:taskCollapse={taskCollapseOptions}
    use:taskSortable={taskSortableOptions}
    use:imageResize={{ onResize: onImageResize }}
    use:highlightSearchTerms={{
      query: showFindReplace ? findReplaceQuery : '',
      caseSensitive: findReplaceCaseSensitive,
    }}
  ></div>
</div>

<style>
  .preview-pane-shell {
    background: var(--surface-panel-bg-contrast);
    border-left: 1px solid var(--surface-panel-border);
  }

  @media (max-width: 639px) {
    .preview-pane-shell {
      border-left: 0;
    }
  }
</style>
