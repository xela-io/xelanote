<script lang="ts">
  import {
    BookOpen,
    CalendarCheck,
    ChevronDown,
    Code,
    ExternalLink,
    FolderTree,
    History,
    Key,
    LayoutDashboard,
    Link2,
    Lock,
    Monitor,
    Network,
    Package,
    Search,
    Server,
    ShieldCheck,
    Sparkles,
    Type,
    UtensilsCrossed,
    WifiOff,
  } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { browser } from '$app/environment';
  import LanguageSelector from '$lib/components/LanguageSelector.svelte';
  import Logo from '$lib/components/Logo.svelte';

  const features = [
    { icon: Type, key: 'editor' },
    { icon: Link2, key: 'wikilinks' },
    { icon: Network, key: 'graph' },
    { icon: Search, key: 'search' },
    { icon: WifiOff, key: 'offline' },
    { icon: Sparkles, key: 'ai' },
    { icon: History, key: 'versions' },
    { icon: FolderTree, key: 'organize' },
    { icon: UtensilsCrossed, key: 'recipes' },
    { icon: BookOpen, key: 'journal' },
    { icon: LayoutDashboard, key: 'canvas' },
    { icon: CalendarCheck, key: 'tasks' },
  ];

  const securityFeatures = [
    { icon: Lock, key: 'e2e' },
    { icon: Key, key: 'zero' },
    { icon: ShieldCheck, key: '2fa' },
    { icon: Code, key: 'open' },
  ];

  const deployItems = [
    { icon: Package, key: 'single_binary' },
    { icon: Server, key: 'sqlite' },
    { icon: Code, key: 'docker' },
    { icon: Monitor, key: 'platforms' },
  ];

  // Scroll-reveal: IntersectionObserver adds .visible when element enters viewport
  function reveal(node: HTMLElement) {
    if (!browser) return;
    const prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (prefersReduced) {
      node.classList.add('visible');
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            entry.target.classList.add('visible');
            observer.unobserve(entry.target);
          }
        }
      },
      { threshold: 0.15, rootMargin: '0px 0px -40px 0px' }
    );
    observer.observe(node);
    return { destroy: () => observer.disconnect() };
  }

  // Terminal typing effect
  let terminalText = $state('');
  let terminalVisible = $state(false);
  const fullCommand = 'docker compose up -d';

  function typeTerminal(node: HTMLElement) {
    if (!browser) return;
    const prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (prefersReduced) {
      terminalText = fullCommand;
      terminalVisible = true;
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && !terminalVisible) {
          terminalVisible = true;
          let i = 0;
          const timer = setInterval(() => {
            terminalText = fullCommand.slice(0, ++i);
            if (i >= fullCommand.length) clearInterval(timer);
          }, 60);
          observer.unobserve(node);
        }
      },
      { threshold: 0.5 }
    );
    observer.observe(node);
    return { destroy: () => observer.disconnect() };
  }

  // Parallax on hero background
  let heroOffset = $state(0);

  onMount(() => {
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    const onScroll = () => {
      heroOffset = window.scrollY * 0.3;
    };
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  });
</script>

<svelte:head>
  <title>xelanote — {$_('page.landing.hero_title')}</title>
  <meta name="description" content={$_('page.landing.hero_subtitle')} />
</svelte:head>

<div class="landing-page">
  <!-- Navigation -->
  <nav class="landing-nav">
    <div class="nav-inner">
      <Logo size="lg" />
      <div class="nav-actions">
        <LanguageSelector />
        <a
          href="https://github.com/xela-io/xelanote"
          target="_blank"
          rel="noopener noreferrer"
          class="nav-link"
        >
          <ExternalLink size={14} />
          GitHub
        </a>
      </div>
    </div>
  </nav>

  <!-- Hero -->
  <section class="hero">
    <div class="hero-bg" style="translate: 0 {heroOffset}px"></div>
    <div class="hero-glow"></div>
    <div class="hero-content">
      <h1 class="hero-entrance" style="--delay: 0">{$_('page.landing.hero_title')}</h1>
      <p class="hero-subtitle hero-entrance" style="--delay: 1">
        {$_('page.landing.hero_subtitle')}
      </p>
      <div class="hero-cta hero-entrance" style="--delay: 2">
        <a
          href="https://github.com/xela-io/xelanote"
          target="_blank"
          rel="noopener noreferrer"
          class="btn btn-primary"
        >
          <ExternalLink size={18} />
          {$_('page.landing.cta_github')}
        </a>
      </div>
    </div>
    <button
      class="scroll-hint hero-entrance"
      style="--delay: 3"
      onclick={() => document.querySelector('.section')?.scrollIntoView({ behavior: 'smooth' })}
      aria-label="Scroll down"
    >
      <ChevronDown size={28} />
    </button>
  </section>

  <!-- Features -->
  <section class="section">
    <div class="section-inner">
      <h2 class="section-title reveal-up" use:reveal>{$_('page.landing.features_title')}</h2>
      <p class="section-subtitle reveal-up" use:reveal>{$_('page.landing.features_subtitle')}</p>
      <div class="features-grid">
        {#each features as f, i (f.key)}
          <div class="feature-card reveal-up" style="--stagger: {i}" use:reveal>
            <div class="feature-icon">
              <svelte:component this={f.icon} size={24} />
            </div>
            <h3>{$_(`page.landing.feature_${f.key}_title`)}</h3>
            <p>{$_(`page.landing.feature_${f.key}_desc`)}</p>
          </div>
        {/each}
      </div>
    </div>
  </section>

  <!-- Security -->
  <section class="section section-alt">
    <div class="section-inner">
      <h2 class="section-title reveal-up" use:reveal>{$_('page.landing.security_title')}</h2>
      <p class="section-subtitle reveal-up" use:reveal>{$_('page.landing.security_subtitle')}</p>
      <div class="security-grid">
        {#each securityFeatures as f, i (f.key)}
          <div class="security-card reveal-up" style="--stagger: {i}" use:reveal>
            <div class="security-icon">
              <svelte:component this={f.icon} size={28} />
            </div>
            <h3>{$_(`page.landing.security_${f.key}_title`)}</h3>
            <p>{$_(`page.landing.security_${f.key}_desc`)}</p>
          </div>
        {/each}
      </div>
    </div>
  </section>

  <!-- Deploy -->
  <section class="section section-dark">
    <div class="section-inner">
      <h2 class="section-title reveal-up" use:reveal>{$_('page.landing.deploy_title')}</h2>
      <p class="section-subtitle reveal-up" use:reveal>{$_('page.landing.deploy_subtitle')}</p>

      <div class="terminal reveal-up" use:reveal use:typeTerminal>
        <div class="terminal-bar">
          <span class="terminal-dot" style="background: #ff5f57"></span>
          <span class="terminal-dot" style="background: #ffbd2e"></span>
          <span class="terminal-dot" style="background: #28c840"></span>
        </div>
        <pre><code
            ><span class="terminal-prompt">$</span> {terminalText}<span class="terminal-cursor"
            ></span></code
          ></pre>
      </div>

      <ul class="deploy-list">
        {#each deployItems as { icon, key }, i (key)}
          <li class="reveal-up" style="--stagger: {i}" use:reveal>
            <svelte:component this={icon} size={20} />
            <span>{$_(`page.landing.deploy_${key}`)}</span>
          </li>
        {/each}
      </ul>
    </div>
  </section>

  <!-- Final CTA -->
  <section class="section section-cta">
    <div class="section-inner reveal-up" use:reveal>
      <h2 class="cta-title">{$_('page.landing.cta_final_title')}</h2>
      <p class="cta-subtitle">{$_('page.landing.cta_final_subtitle')}</p>
      <a
        href="https://github.com/xela-io/xelanote"
        target="_blank"
        rel="noopener noreferrer"
        class="btn btn-primary btn-lg"
      >
        <ExternalLink size={20} />
        {$_('page.landing.cta_github')}
      </a>
    </div>
  </section>

  <!-- Footer -->
  <footer class="landing-footer">
    <div class="footer-inner">
      <div class="footer-brand">
        <Logo size="md" />
      </div>
      <div class="footer-links">
        <a href="https://github.com/xela-io/xelanote" target="_blank" rel="noopener noreferrer">
          {$_('page.landing.footer_source')}
        </a>
      </div>
      <div class="footer-lang">
        <LanguageSelector />
      </div>
    </div>
  </footer>
</div>

<style>
  /* ===== Landing Page Root ===== */
  .landing-page {
    min-height: 100vh;
    min-height: 100dvh;
    background: var(--color-background);
    color: var(--color-foreground);
    overflow-y: auto;
  }

  /* ===== Navigation ===== */
  .landing-nav {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    z-index: 10;
    padding: 1.25rem 0;
  }

  .nav-inner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    max-width: 1200px;
    margin: 0 auto;
    padding: 0 2rem;
    --color-sidebar-primary: oklch(75% 0.1 230);
  }

  .nav-actions {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .nav-link {
    font-size: 0.875rem;
    font-weight: 500;
    text-decoration: none;
    color: oklch(78% 0.02 85);
    padding: 0.5rem 1rem;
    border-radius: var(--radius-md);
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    transition:
      color var(--duration-base) var(--ease-default),
      background var(--duration-base) var(--ease-default);
  }

  .nav-link:hover {
    color: oklch(95% 0.02 85);
    background: oklch(30% 0.02 60 / 0.5);
  }

  /* ===== Hero Section ===== */
  .hero {
    position: relative;
    min-height: 100vh;
    min-height: 100dvh;
    display: flex;
    align-items: center;
    justify-content: center;
    text-align: center;
    padding: 6rem 2rem 4rem;
    overflow: hidden;
  }

  .hero-bg {
    position: absolute;
    inset: -20% 0 0 0;
    background:
      radial-gradient(ellipse at 50% 30%, oklch(22% 0.05 230 / 0.5), transparent 60%),
      radial-gradient(ellipse at 80% 80%, oklch(20% 0.04 160 / 0.3), transparent 50%),
      linear-gradient(180deg, oklch(14% 0.01 60), oklch(17% 0.015 60) 50%, oklch(14% 0.01 60));
    will-change: translate;
  }

  .hero-bg::after {
    content: '';
    position: absolute;
    inset: 0;
    background-image: radial-gradient(oklch(35% 0.02 60) 1px, transparent 1px);
    background-size: 40px 40px;
    opacity: 0.25;
  }

  /* Floating glow orb */
  .hero-glow {
    position: absolute;
    width: 500px;
    height: 500px;
    border-radius: 50%;
    background: radial-gradient(circle, oklch(45% 0.15 230 / 0.15), transparent 70%);
    top: 20%;
    left: 50%;
    translate: -50% -50%;
    animation: glow-float 12s ease-in-out infinite;
    pointer-events: none;
    filter: blur(40px);
  }

  @keyframes glow-float {
    0%,
    100% {
      translate: -50% -50%;
      scale: 1;
    }
    33% {
      translate: -30% -40%;
      scale: 1.1;
    }
    66% {
      translate: -70% -55%;
      scale: 0.9;
    }
  }

  .hero-content {
    position: relative;
    z-index: 1;
    max-width: 800px;
  }

  /* Hero staggered entrance */
  .hero-entrance {
    opacity: 0;
    translate: 0 24px;
    animation: hero-enter 0.8s cubic-bezier(0.16, 1, 0.3, 1) forwards;
    animation-delay: calc(var(--delay, 0) * 0.15s + 0.2s);
  }

  @keyframes hero-enter {
    to {
      opacity: 1;
      translate: 0 0;
    }
  }

  .hero h1 {
    font-size: clamp(2rem, 5vw, 3.5rem);
    font-weight: 800;
    line-height: 1.15;
    margin: 0 0 1.5rem;
    background: linear-gradient(
      135deg,
      oklch(95% 0.02 85) 0%,
      oklch(78% 0.1 230) 50%,
      oklch(76% 0.08 180) 100%
    );
    background-size: 200% 200%;
    background-clip: text;
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    animation:
      hero-enter 0.8s cubic-bezier(0.16, 1, 0.3, 1) forwards,
      hero-gradient 8s ease-in-out 1s infinite;
    animation-delay: calc(var(--delay, 0) * 0.15s + 0.2s);
  }

  @keyframes hero-gradient {
    0%,
    100% {
      background-position: 0% 50%;
    }
    50% {
      background-position: 100% 50%;
    }
  }

  .hero-subtitle {
    font-size: clamp(1rem, 2vw, 1.25rem);
    color: oklch(70% 0.02 85);
    line-height: 1.65;
    margin: 0 auto 2.5rem;
    max-width: 600px;
  }

  .hero-cta {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    flex-wrap: wrap;
  }

  /* ===== Scroll Hint ===== */
  .scroll-hint {
    position: absolute;
    bottom: 2rem;
    left: 50%;
    z-index: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    border: none;
    border-radius: 50%;
    background: oklch(95% 0.02 85 / 0.1);
    color: oklch(70% 0.02 85);
    cursor: pointer;
    animation:
      hero-enter 0.8s cubic-bezier(0.16, 1, 0.3, 1) forwards,
      scroll-bounce 2s ease-in-out 1.2s infinite;
    transition: background var(--duration-base) var(--ease-default);
  }

  .scroll-hint:hover {
    background: oklch(95% 0.02 85 / 0.2);
    color: oklch(90% 0.02 85);
  }

  @keyframes scroll-bounce {
    0%,
    100% {
      translate: -50% 0;
    }
    50% {
      translate: -50% 10px;
    }
  }

  /* ===== Scroll Reveal ===== */
  .reveal-up {
    opacity: 0;
    translate: 0 32px;
    transition:
      opacity 0.6s cubic-bezier(0.16, 1, 0.3, 1),
      translate 0.6s cubic-bezier(0.16, 1, 0.3, 1);
    transition-delay: calc(var(--stagger, 0) * 0.06s);
  }

  .reveal-up:global(.visible) {
    opacity: 1;
    translate: 0 0;
  }

  /* ===== Buttons ===== */
  .btn {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1.75rem;
    font-size: 1rem;
    font-weight: 600;
    text-decoration: none;
    border-radius: var(--radius-lg);
    transition:
      transform var(--duration-base) var(--ease-default),
      background var(--duration-base) var(--ease-default),
      box-shadow var(--duration-base) var(--ease-default),
      border-color var(--duration-base) var(--ease-default);
    cursor: pointer;
    border: none;
  }

  .btn-primary {
    background: oklch(50% 0.14 230);
    color: oklch(98% 0.01 85);
    box-shadow: 0 2px 12px oklch(50% 0.14 230 / 0.3);
  }

  .btn-primary:hover {
    background: oklch(45% 0.14 230);
    transform: translateY(-2px);
    box-shadow: 0 4px 20px oklch(50% 0.14 230 / 0.4);
  }

  .btn-ghost {
    background: transparent;
    color: oklch(78% 0.02 85);
    border: 1px solid oklch(40% 0.02 60);
  }

  .btn-ghost:hover {
    color: oklch(95% 0.02 85);
    border-color: oklch(55% 0.02 60);
    background: oklch(25% 0.02 60);
  }

  .btn-lg {
    padding: 1rem 2.5rem;
    font-size: 1.125rem;
  }

  /* ===== Sections ===== */
  .section {
    padding: 5rem 2rem;
  }

  .section-inner {
    max-width: 1200px;
    margin: 0 auto;
  }

  .section-title {
    text-align: center;
    font-size: clamp(1.5rem, 3vw, 2.25rem);
    font-weight: 700;
    margin: 0 0 0.75rem;
  }

  .section-subtitle {
    text-align: center;
    color: var(--color-muted-foreground);
    font-size: 1.1rem;
    margin: 0 auto 3rem;
    max-width: 600px;
    line-height: 1.6;
  }

  .section-alt {
    background: var(--color-card);
  }

  .section-dark {
    background: oklch(14% 0.01 60);
    color: oklch(90% 0.02 85);
  }

  .section-dark .section-title {
    color: oklch(95% 0.02 85);
  }

  .section-dark .section-subtitle {
    color: oklch(65% 0.02 85);
  }

  /* ===== Features Grid ===== */
  .features-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 1.25rem;
  }

  .feature-card {
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-xl);
    padding: 1.5rem;
    text-align: center;
    transition:
      transform 0.3s cubic-bezier(0.16, 1, 0.3, 1),
      box-shadow 0.3s cubic-bezier(0.16, 1, 0.3, 1),
      border-color 0.3s ease;
  }

  .feature-card:hover {
    transform: translateY(-6px);
    box-shadow: 0 12px 32px oklch(0% 0 0 / 0.08);
    border-color: color-mix(in oklch, var(--color-primary), transparent 60%);
  }

  .feature-card:hover .feature-icon {
    scale: 1.1;
    background: color-mix(in oklch, var(--color-primary), transparent 75%);
  }

  .feature-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    border-radius: var(--radius-lg);
    background: color-mix(in oklch, var(--color-primary), transparent 85%);
    color: var(--color-primary);
    margin-bottom: 1rem;
    transition:
      scale 0.3s cubic-bezier(0.16, 1, 0.3, 1),
      background 0.3s ease;
  }

  .feature-card h3 {
    font-size: 1rem;
    font-weight: 600;
    margin: 0 0 0.5rem;
  }

  .feature-card p {
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
    line-height: 1.5;
    margin: 0;
  }

  /* ===== Security Grid ===== */
  .security-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 1.5rem;
  }

  .security-card {
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-top: 3px solid var(--color-primary);
    border-radius: var(--radius-xl);
    padding: 2rem 1.5rem;
    text-align: center;
    transition:
      transform 0.3s cubic-bezier(0.16, 1, 0.3, 1),
      box-shadow 0.3s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .security-card:hover {
    transform: translateY(-6px);
    box-shadow: 0 12px 32px oklch(0% 0 0 / 0.08);
  }

  .security-card:hover .security-icon {
    scale: 1.1;
  }

  .security-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 56px;
    height: 56px;
    border-radius: 50%;
    background: color-mix(in oklch, var(--color-primary), transparent 88%);
    color: var(--color-primary);
    margin-bottom: 1.25rem;
    transition: scale 0.3s cubic-bezier(0.16, 1, 0.3, 1);
  }

  .security-card h3 {
    font-size: 1.05rem;
    font-weight: 600;
    margin: 0 0 0.5rem;
  }

  .security-card p {
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
    line-height: 1.5;
    margin: 0;
  }

  /* ===== Terminal Mockup ===== */
  .terminal {
    max-width: 520px;
    margin: 2rem auto;
    border-radius: var(--radius-xl);
    overflow: hidden;
    background: oklch(12% 0.01 60);
    border: 1px solid oklch(25% 0.02 60);
    box-shadow: 0 16px 48px oklch(0% 0 0 / 0.3);
  }

  .terminal-bar {
    display: flex;
    gap: 8px;
    padding: 14px 16px;
    background: oklch(18% 0.01 60);
  }

  .terminal-dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
  }

  .terminal pre {
    margin: 0;
    padding: 1.5rem;
    font-family: 'JetBrains Mono Variable', ui-monospace, monospace;
    font-size: 1rem;
    line-height: 1.6;
    color: oklch(75% 0.12 160);
    min-height: 3.5rem;
  }

  .terminal code {
    font-family: inherit;
  }

  .terminal-prompt {
    color: oklch(55% 0.02 85);
    margin-right: 0.75rem;
  }

  .terminal-cursor {
    display: inline-block;
    width: 2px;
    height: 1.1em;
    background: oklch(75% 0.12 160);
    vertical-align: text-bottom;
    margin-left: 2px;
    animation: cursor-blink 1s step-end infinite;
  }

  @keyframes cursor-blink {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0;
    }
  }

  /* ===== Deploy List ===== */
  .deploy-list {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1rem;
    max-width: 700px;
    margin: 2.5rem auto 0;
    list-style: none;
    padding: 0;
  }

  .deploy-list li {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 0.95rem;
    color: oklch(75% 0.02 85);
  }

  .deploy-list li :global(svg) {
    color: oklch(65% 0.1 160);
    flex-shrink: 0;
  }

  /* ===== CTA Section ===== */
  .section-cta {
    background: linear-gradient(135deg, oklch(38% 0.12 230), oklch(32% 0.1 260));
    color: oklch(95% 0.02 85);
    text-align: center;
    padding: 5rem 2rem;
  }

  .cta-title {
    font-size: clamp(1.5rem, 3vw, 2rem);
    font-weight: 700;
    margin: 0 0 0.75rem;
    color: oklch(98% 0.01 85);
  }

  .cta-subtitle {
    font-size: 1.1rem;
    color: oklch(80% 0.04 230);
    margin: 0 0 2rem;
  }

  .section-cta .btn-primary {
    background: oklch(98% 0.01 85);
    color: oklch(25% 0.1 230);
    box-shadow: 0 4px 16px oklch(0% 0 0 / 0.15);
  }

  .section-cta .btn-primary:hover {
    background: oklch(100% 0 0);
    box-shadow: 0 6px 24px oklch(0% 0 0 / 0.2);
  }

  /* ===== Footer ===== */
  .landing-footer {
    background: oklch(12% 0.01 60);
    color: oklch(65% 0.02 85);
    padding: 2.5rem 2rem;
  }

  .footer-inner {
    max-width: 1200px;
    margin: 0 auto;
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 1.5rem;
  }

  .footer-brand {
    --color-sidebar-primary: oklch(58% 0.06 230);
  }

  .footer-links {
    display: flex;
    gap: 1.5rem;
  }

  .footer-links a {
    color: oklch(58% 0.02 85);
    text-decoration: none;
    font-size: 0.875rem;
    transition: color var(--duration-base);
  }

  .footer-links a:hover {
    color: oklch(82% 0.02 85);
  }

  .footer-lang {
    --color-foreground: oklch(65% 0.02 85);
    --color-border: oklch(28% 0.02 60);
    --color-background: oklch(18% 0.01 60);
    --color-popover: oklch(18% 0.01 60);
    --color-popover-foreground: oklch(75% 0.02 85);
  }

  /* ===== Responsive ===== */
  @media (max-width: 1024px) {
    .features-grid {
      grid-template-columns: repeat(3, 1fr);
    }
  }

  @media (max-width: 768px) {
    .section {
      padding: 3.5rem 1.5rem;
    }
    .hero {
      padding: 5rem 1.5rem 3rem;
    }
    .features-grid {
      grid-template-columns: repeat(2, 1fr);
    }
    .security-grid {
      grid-template-columns: repeat(2, 1fr);
    }
    .deploy-list {
      grid-template-columns: 1fr;
    }
    .footer-inner {
      flex-direction: column;
      text-align: center;
    }
    .footer-links {
      flex-wrap: wrap;
      justify-content: center;
    }
    .hero-glow {
      width: 300px;
      height: 300px;
    }
  }

  @media (max-width: 480px) {
    .section {
      padding: 2.5rem 1rem;
    }
    .hero {
      padding: 4rem 1rem 2rem;
    }
    .nav-inner {
      padding: 0 1rem;
    }
    .features-grid {
      grid-template-columns: 1fr;
    }
    .security-grid {
      grid-template-columns: 1fr;
    }
    .section-cta {
      padding: 3.5rem 1rem;
    }
  }

  /* ===== Reduced Motion ===== */
  @media (prefers-reduced-motion: reduce) {
    .hero h1 {
      animation: none;
      background-position: 0% 50%;
    }
    .hero-entrance {
      animation: none;
      opacity: 1;
      translate: 0 0;
    }
    .scroll-hint {
      animation: none;
      opacity: 1;
      translate: -50% 0;
    }
    .hero-glow {
      animation: none;
    }
    .terminal-cursor {
      animation: none;
    }
    .reveal-up {
      opacity: 1;
      translate: 0 0;
      transition: none;
    }
    .feature-card,
    .security-card,
    .feature-icon,
    .security-icon,
    .btn {
      transition: none;
    }
    .feature-card:hover,
    .security-card:hover,
    .btn:hover {
      transform: none;
    }
    .feature-card:hover .feature-icon,
    .security-card:hover .security-icon {
      scale: 1;
    }
  }
</style>
