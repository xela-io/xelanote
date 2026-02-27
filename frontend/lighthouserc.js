module.exports = {
  ci: {
    collect: {
      url: ['http://localhost:4173/login'], // Login-Seite: öffentlich, kein Auth nötig
      numberOfRuns: 3,
      settings: {
        chromeFlags: '--no-sandbox --headless --disable-gpu',
        onlyCategories: ['performance', 'accessibility', 'best-practices'],
        // PWA-Audits werden automatisch mitgeprüft, auch ohne Kategorie
      },
    },
    assert: {
      assertions: {
        // Kategorie-Scores (PWA-Kategorie existiert seit Lighthouse 12 nicht mehr)
        'categories:performance': ['error', { minScore: 0.7 }],
        'categories:accessibility': ['error', { minScore: 0.9 }],
        'categories:best-practices': ['error', { minScore: 0.8 }],
        // Einzelne PWA-Audits
        'service-worker': 'error',
        'installable-manifest': 'error',
        'apple-touch-icon': 'error',
        'splash-screen': 'warn',
        'themed-omnibox': 'warn',
      },
    },
    upload: {
      target: 'filesystem',
      outputDir: './lighthouse-reports',
    },
  },
};
