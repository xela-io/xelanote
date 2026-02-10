<script lang="ts">
  import {
    ArrowRight,
    Link,
    Search,
    Type,
    Link2,
    Lock,
    Key,
    Code,
    Network,
    History,
    WifiOff,
    Sparkles,
    FolderTree,
    ShieldCheck,
    Server,
    Monitor,
    Palette,
    Globe,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';
  import LanguageSelector from '$lib/components/LanguageSelector.svelte';
  import Logo from '$lib/components/Logo.svelte';
</script>

<div class="about-container">
  <div class="about-card">
    <div class="header">
      <h1><Logo size="xl" /></h1>
      <p class="main-title">{$_('page.about.main_title')}</p>
      <p class="subtitle">
        {$_('page.about.subtitle')}
      </p>
    </div>

    <section class="section">
      <h2>{$_('page.about.features_title')}</h2>
      <div class="features-grid">
        <div class="feature-card">
          <div class="icon-wrapper">
            <Link2 size={24} />
          </div>
          <h3>{$_('page.about.feature_wikilinks_title')}</h3>
          <p>{$_('page.about.feature_wikilinks_description')}</p>
        </div>
        <div class="feature-card">
          <div class="icon-wrapper">
            <Link size={24} />
          </div>
          <h3>{$_('page.about.feature_backlinks_title')}</h3>
          <p>{$_('page.about.feature_backlinks_description')}</p>
        </div>
        <div class="feature-card">
          <div class="icon-wrapper">
            <Network size={24} />
          </div>
          <h3>{$_('page.about.feature_graph_title')}</h3>
          <p>{$_('page.about.feature_graph_description')}</p>
        </div>
        <div class="feature-card">
          <div class="icon-wrapper">
            <Search size={24} />
          </div>
          <h3>{$_('page.about.feature_fulltextsearch_title')}</h3>
          <p>{$_('page.about.feature_fulltextsearch_description')}</p>
        </div>
        <div class="feature-card">
          <div class="icon-wrapper">
            <Type size={24} />
          </div>
          <h3>{$_('page.about.feature_markdowneditor_title')}</h3>
          <p>{$_('page.about.feature_markdowneditor_description')}</p>
        </div>
        <div class="feature-card">
          <div class="icon-wrapper">
            <History size={24} />
          </div>
          <h3>{$_('page.about.feature_versionhistory_title')}</h3>
          <p>{$_('page.about.feature_versionhistory_description')}</p>
        </div>
        <div class="feature-card">
          <div class="icon-wrapper">
            <WifiOff size={24} />
          </div>
          <h3>{$_('page.about.feature_offline_title')}</h3>
          <p>{$_('page.about.feature_offline_description')}</p>
        </div>
        <div class="feature-card">
          <div class="icon-wrapper">
            <Sparkles size={24} />
          </div>
          <h3>{$_('page.about.feature_ai_title')}</h3>
          <p>{$_('page.about.feature_ai_description')}</p>
        </div>
        <div class="feature-card">
          <div class="icon-wrapper">
            <FolderTree size={24} />
          </div>
          <h3>{$_('page.about.feature_organization_title')}</h3>
          <p>{$_('page.about.feature_organization_description')}</p>
        </div>
      </div>
    </section>

    <section class="section">
      <h2>{$_('page.about.security_title')}</h2>
      <div class="security-grid">
        <div class="security-card">
          <div class="icon-wrapper">
            <Lock size={24} />
          </div>
          <h3>{$_('page.about.e2e_title')}</h3>
          <p>{$_('page.about.e2e_description')}</p>
        </div>
        <div class="security-card">
          <div class="icon-wrapper">
            <Key size={24} />
          </div>
          <h3>{$_('page.about.zero_knowledge_title')}</h3>
          <p>{$_('page.about.zero_knowledge_description')}</p>
        </div>
        <div class="security-card">
          <div class="icon-wrapper">
            <ShieldCheck size={24} />
          </div>
          <h3>{$_('page.about.twofa_title')}</h3>
          <p>{$_('page.about.twofa_description')}</p>
        </div>
        <div class="security-card">
          <div class="icon-wrapper">
            <Code size={24} />
          </div>
          <h3>{$_('page.about.open_source_title')}</h3>
          <p>{$_('page.about.open_source_description')}</p>
        </div>
      </div>
    </section>

    <section class="section">
      <h2>{$_('page.about.platform_title')}</h2>
      <div class="platform-grid">
        <div class="platform-card">
          <div class="icon-wrapper">
            <Server size={24} />
          </div>
          <h3>{$_('page.about.platform_selfhosted_title')}</h3>
          <p>{$_('page.about.platform_selfhosted_description')}</p>
        </div>
        <div class="platform-card">
          <div class="icon-wrapper">
            <Monitor size={24} />
          </div>
          <h3>{$_('page.about.platform_desktop_title')}</h3>
          <p>{$_('page.about.platform_desktop_description')}</p>
        </div>
        <div class="platform-card">
          <div class="icon-wrapper">
            <Palette size={24} />
          </div>
          <h3>{$_('page.about.platform_themes_title')}</h3>
          <p>{$_('page.about.platform_themes_description')}</p>
        </div>
        <div class="platform-card">
          <div class="icon-wrapper">
            <Globe size={24} />
          </div>
          <h3>{$_('page.about.platform_i18n_title')}</h3>
          <p>{$_('page.about.platform_i18n_description')}</p>
        </div>
      </div>
    </section>

    <a href="/login" class="login-link">
      {$_('page.about.login_and_start')}
      <ArrowRight size={18} />
    </a>
    <div class="register-link">
      {$_('page.login.no_account_yet')} <a href="/register">{$_('common.register')}</a>
    </div>
    <div class="language-link">
      <LanguageSelector />
    </div>
  </div>
</div>

<style>
  .about-container {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
    min-height: 100dvh;
    background: var(--color-background);
    color: var(--color-foreground);
    padding: 2rem 1rem;
  }

  .about-card {
    width: 100%;
    max-width: 900px;
    padding: 3rem;
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.05);
  }

  .header {
    text-align: center;
    margin-bottom: 3rem;
  }

  h1 {
    margin: 0;
    font-size: 2rem;
    font-weight: 700;
    text-align: center;
    color: var(--color-foreground);
  }

  .main-title {
    margin: 1rem 0 0;
    font-size: 1.75rem;
    font-weight: 600;
    line-height: 1.2;
    color: var(--color-foreground);
  }

  .subtitle {
    margin: 0.75rem auto 0;
    max-width: 600px;
    color: var(--color-muted-foreground);
    font-size: 1.1rem;
  }

  /* Sections */
  .section {
    margin-bottom: 2.5rem;
    padding-top: 2rem;
    border-top: 1px solid var(--color-border);
  }

  .section h2 {
    text-align: center;
    font-size: 1.5rem;
    font-weight: 600;
    margin: 0 0 1.5rem 0;
  }

  /* Features Grid */
  .features-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1.5rem;
  }

  @media (max-width: 768px) {
    .features-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 480px) {
    .features-grid {
      grid-template-columns: 1fr;
    }
  }

  .feature-card {
    background: var(--color-background);
    padding: 1.5rem;
    border-radius: 8px;
    border: 1px solid var(--color-border);
    text-align: center;
    transition:
      transform 0.2s,
      box-shadow 0.2s;
  }

  @media (prefers-reduced-motion: reduce) {
    .feature-card,
    .security-card,
    .platform-card {
      transition: none;
    }
  }

  .feature-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.05);
  }

  .icon-wrapper {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    background: color-mix(in oklch, var(--color-primary), transparent 80%);
    border-radius: 50%;
    color: var(--color-primary);
    margin-bottom: 0.75rem;
  }

  h3 {
    font-size: 1.05rem;
    font-weight: 600;
    margin: 0 0 0.4rem 0;
  }

  .feature-card p,
  .security-card p,
  .platform-card p {
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
    line-height: 1.5;
    margin: 0;
  }

  /* Security Grid */
  .security-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 1.5rem;
  }

  @media (max-width: 768px) {
    .security-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 480px) {
    .security-grid {
      grid-template-columns: 1fr;
    }
  }

  .security-card {
    background: var(--color-background);
    padding: 1.5rem;
    border-radius: 8px;
    border: 1px solid var(--color-border);
    text-align: center;
    transition:
      transform 0.2s,
      box-shadow 0.2s;
  }

  .security-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.05);
  }

  .security-card h3 {
    font-size: 0.95rem;
    font-weight: 600;
    margin: 0 0 0.4rem 0;
  }

  /* Platform Grid */
  .platform-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 1.5rem;
  }

  @media (max-width: 768px) {
    .platform-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 480px) {
    .platform-grid {
      grid-template-columns: 1fr;
    }
  }

  .platform-card {
    background: var(--color-background);
    padding: 1.5rem;
    border-radius: 8px;
    border: 1px solid var(--color-border);
    text-align: center;
    transition:
      transform 0.2s,
      box-shadow 0.2s;
  }

  .platform-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.05);
  }

  .platform-card h3 {
    font-size: 0.95rem;
    font-weight: 600;
    margin: 0 0 0.4rem 0;
  }

  /* CTA */
  .login-link {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.8rem;
    font-size: 1rem;
    font-weight: 600;
    color: var(--color-primary-foreground);
    background: var(--color-primary);
    border: none;
    border-radius: 6px;
    cursor: pointer;
    text-decoration: none;
    transition: background 0.2s;
  }

  .login-link:hover {
    background: color-mix(in oklch, var(--color-primary), black 15%);
  }

  .register-link {
    margin-top: 1.5rem;
    text-align: center;
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
  }

  .register-link a {
    color: var(--color-primary);
    text-decoration: none;
    font-weight: 600;
  }

  .register-link a:hover {
    text-decoration: underline;
  }

  .language-link {
    display: flex;
    justify-content: center;
    margin-top: 1.5rem;
    padding-top: 1rem;
    border-top: 1px solid var(--color-border);
  }

  /* Mobile padding */
  @media (max-width: 480px) {
    .about-card {
      padding: 1.5rem;
    }
  }
</style>
