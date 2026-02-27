<script lang="ts">
  import { onDestroy, onMount } from 'svelte';

  interface Props {
    iframeUrl: string;
    onToken: (token: string) => void;
    onExpired?: () => void;
    onError?: () => void;
  }

  const { iframeUrl, onToken, onExpired, onError }: Props = $props();

  let iframeRef = $state<HTMLIFrameElement | null>(null);
  let timeoutId: ReturnType<typeof setTimeout> | null = null;

  function parseIframeUrl(url: string): URL | null {
    try {
      if (typeof window !== 'undefined') {
        return new URL(url, window.location.origin);
      }
      return new URL(url);
    } catch {
      return null;
    }
  }

  // Derive the expected origin from the iframe URL for security validation
  function getExpectedOrigin(url: string): string {
    const parsed = parseIframeUrl(url);
    if (!parsed) {
      return '';
    }
    return parsed.origin;
  }

  function buildIframeSrc(url: string): string {
    const parsed = parseIframeUrl(url);
    if (!parsed) {
      return url;
    }

    if (typeof window !== 'undefined' && !parsed.searchParams.has('parent_origin')) {
      parsed.searchParams.set('parent_origin', window.location.origin);
    }

    return parsed.toString();
  }

  function handleMessage(event: MessageEvent) {
    // Origin validation: only accept messages from the server hosting the CAPTCHA page
    const expectedOrigin = getExpectedOrigin(buildIframeSrc(iframeUrl));
    if (expectedOrigin && event.origin !== expectedOrigin) {
      return;
    }

    if (!event.data || typeof event.data !== 'object') {
      return;
    }

    switch (event.data.type) {
      case 'captcha-token':
        if (timeoutId) {
          clearTimeout(timeoutId);
          timeoutId = null;
        }
        onToken(event.data.token);
        break;
      case 'captcha-expired':
        onExpired?.();
        break;
      case 'captcha-error':
        onError?.();
        break;
    }
  }

  /**
   * Reset the CAPTCHA widget by sending a message to the iframe.
   */
  export function reset() {
    if (iframeRef?.contentWindow) {
      const expectedOrigin = getExpectedOrigin(buildIframeSrc(iframeUrl));
      iframeRef.contentWindow.postMessage({ type: 'captcha-reset' }, expectedOrigin || '*');
    }
  }

  onMount(() => {
    window.addEventListener('message', handleMessage);

    // Timeout: if no token received within 10s, assume iframe failed to load
    timeoutId = setTimeout(() => {
      timeoutId = null;
      onError?.();
    }, 10000);
  });

  onDestroy(() => {
    window.removeEventListener('message', handleMessage);
    if (timeoutId) {
      clearTimeout(timeoutId);
      timeoutId = null;
    }
  });
</script>

<iframe
  bind:this={iframeRef}
  src={buildIframeSrc(iframeUrl)}
  title="CAPTCHA verification"
  class="captcha-iframe"
  scrolling="no"
></iframe>

<style>
  .captcha-iframe {
    width: 300px;
    height: 65px;
    border: none;
    overflow: hidden;
  }
</style>
