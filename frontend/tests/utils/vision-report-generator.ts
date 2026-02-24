/**
 * Generates a standalone HTML report for the Vision Design Review.
 *
 * Features:
 * - Score dashboard per page (radar chart as inline SVG)
 * - Issues table with severity badges
 * - Strengths and recommendations
 * - Consistency analysis across pages
 * - Screenshots inline (Base64)
 */
import fs from 'node:fs/promises';
import path from 'node:path';

import type {
  ConsistencyReport,
  DesignReviewResult,
  ScreenshotEntry,
  VisionReviewReport,
} from './vision-design-reviewer';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function severityColor(severity: string): string {
  const map: Record<string, string> = {
    minor: '#eab308',
    moderate: '#f97316',
    critical: '#ef4444',
  };
  return map[severity] ?? '#6b7280';
}

function severityBadge(severity: string): string {
  const color = severityColor(severity);
  return `<span class="badge" style="background:${color}">${escapeHtml(severity)}</span>`;
}

function scoreColor(score: number): string {
  if (score >= 8) return '#22c55e';
  if (score >= 6) return '#eab308';
  if (score >= 4) return '#f97316';
  return '#ef4444';
}

function scoreBar(score: number, max = 10): string {
  const pct = (score / max) * 100;
  const color = scoreColor(score);
  return `<div class="score-bar"><div class="score-fill" style="width:${pct}%;background:${color}"></div><span class="score-label">${score}</span></div>`;
}

// ---------------------------------------------------------------------------
// Radar chart (inline SVG)
// ---------------------------------------------------------------------------

function radarChart(scores: Array<{ label: string; value: number }>, size = 220): string {
  const cx = size / 2;
  const cy = size / 2;
  const radius = size * 0.35;
  const n = scores.length;
  if (n < 3) return '';

  let svg = `<svg width="${size}" height="${size}" viewBox="0 0 ${size} ${size}" xmlns="http://www.w3.org/2000/svg">`;

  // Grid rings (at 2, 4, 6, 8, 10)
  for (let ring = 2; ring <= 10; ring += 2) {
    const r = (ring / 10) * radius;
    const pts: string[] = [];
    for (let i = 0; i < n; i++) {
      const angle = (Math.PI * 2 * i) / n - Math.PI / 2;
      pts.push(`${(cx + r * Math.cos(angle)).toFixed(1)},${(cy + r * Math.sin(angle)).toFixed(1)}`);
    }
    svg += `<polygon points="${pts.join(' ')}" fill="none" stroke="var(--border)" stroke-width="0.5"/>`;
  }

  // Axis lines + labels
  for (let i = 0; i < n; i++) {
    const angle = (Math.PI * 2 * i) / n - Math.PI / 2;
    const x2 = cx + radius * Math.cos(angle);
    const y2 = cy + radius * Math.sin(angle);
    svg += `<line x1="${cx}" y1="${cy}" x2="${x2.toFixed(1)}" y2="${y2.toFixed(1)}" stroke="var(--border)" stroke-width="0.5"/>`;

    const lx = cx + (radius + 24) * Math.cos(angle);
    const ly = cy + (radius + 24) * Math.sin(angle);
    svg += `<text x="${lx.toFixed(1)}" y="${ly.toFixed(1)}" text-anchor="middle" dominant-baseline="middle" font-size="10" fill="var(--muted)">${escapeHtml(scores[i].label)}</text>`;
  }

  // Data polygon
  const dataPts: string[] = [];
  for (let i = 0; i < n; i++) {
    const angle = (Math.PI * 2 * i) / n - Math.PI / 2;
    const r = (scores[i].value / 10) * radius;
    dataPts.push(
      `${(cx + r * Math.cos(angle)).toFixed(1)},${(cy + r * Math.sin(angle)).toFixed(1)}`
    );
  }
  svg += `<polygon points="${dataPts.join(' ')}" fill="rgba(37,99,235,0.15)" stroke="#2563eb" stroke-width="2"/>`;

  // Data dots + score values
  for (let i = 0; i < n; i++) {
    const angle = (Math.PI * 2 * i) / n - Math.PI / 2;
    const r = (scores[i].value / 10) * radius;
    const dx = cx + r * Math.cos(angle);
    const dy = cy + r * Math.sin(angle);
    svg += `<circle cx="${dx.toFixed(1)}" cy="${dy.toFixed(1)}" r="3.5" fill="#2563eb"/>`;
    svg += `<text x="${dx.toFixed(1)}" y="${(dy - 8).toFixed(1)}" text-anchor="middle" font-size="9" font-weight="600" fill="#2563eb">${scores[i].value}</text>`;
  }

  svg += '</svg>';
  return svg;
}

// ---------------------------------------------------------------------------
// Page card
// ---------------------------------------------------------------------------

function pageCard(review: DesignReviewResult, screenshotBase64: string | null): string {
  const chart = radarChart([
    { label: 'Overall', value: review.overallScore },
    { label: 'Consistency', value: review.consistency },
    { label: 'Hierarchy', value: review.visualHierarchy },
    { label: 'Whitespace', value: review.whitespace },
    { label: 'Readability', value: review.readability },
  ]);

  const issueRows = review.issues
    .map(
      (i) => `
    <tr>
      <td>${severityBadge(i.severity)}</td>
      <td>${escapeHtml(i.area)}</td>
      <td>${escapeHtml(i.description)}</td>
      <td>${escapeHtml(i.suggestion)}</td>
    </tr>`
    )
    .join('');

  const strengthsList = review.strengths.map((s) => `<li>${escapeHtml(s)}</li>`).join('');

  const recoList = review.recommendations.map((r) => `<li>${escapeHtml(r)}</li>`).join('');

  const screenshotImg = screenshotBase64
    ? `<img src="data:image/png;base64,${screenshotBase64}" class="screenshot-thumb" alt="${escapeHtml(review.page)}" loading="lazy"/>`
    : '';

  return `
  <div class="card page-card">
    <div class="page-header">
      <h3>${escapeHtml(review.page)} <span class="viewport-tag">${escapeHtml(review.viewport)}</span></h3>
      <div class="overall-score" style="color:${scoreColor(review.overallScore)}">${review.overallScore}<span class="score-max">/10</span></div>
    </div>

    <div class="page-body">
      <div class="chart-col">
        ${chart}
        ${screenshotImg}
      </div>

      <div class="details-col">
        <h4>Scores</h4>
        <div class="scores-grid">
          <div class="score-row"><span>Overall</span>${scoreBar(review.overallScore)}</div>
          <div class="score-row"><span>Consistency</span>${scoreBar(review.consistency)}</div>
          <div class="score-row"><span>Hierarchy</span>${scoreBar(review.visualHierarchy)}</div>
          <div class="score-row"><span>Whitespace</span>${scoreBar(review.whitespace)}</div>
          <div class="score-row"><span>Readability</span>${scoreBar(review.readability)}</div>
        </div>

        ${
          review.strengths.length > 0
            ? `<h4>Strengths</h4><ul class="strengths">${strengthsList}</ul>`
            : ''
        }

        ${
          review.recommendations.length > 0
            ? `<h4>Recommendations</h4><ul class="recommendations">${recoList}</ul>`
            : ''
        }
      </div>
    </div>

    ${
      review.issues.length > 0
        ? `
    <details class="issues-section">
      <summary>${review.issues.length} Issue${review.issues.length !== 1 ? 's' : ''}</summary>
      <table class="issues-table">
        <thead><tr><th>Severity</th><th>Area</th><th>Issue</th><th>Suggestion</th></tr></thead>
        <tbody>${issueRows}</tbody>
      </table>
    </details>`
        : '<p class="no-issues">No issues found</p>'
    }
  </div>`;
}

// ---------------------------------------------------------------------------
// Consistency section
// ---------------------------------------------------------------------------

function consistencySection(report: ConsistencyReport): string {
  const chart = radarChart([
    { label: 'Overall', value: report.overallConsistency },
    { label: 'Color', value: report.colorConsistency },
    { label: 'Typography', value: report.typographyConsistency },
    { label: 'Spacing', value: report.spacingConsistency },
    { label: 'Layout', value: report.layoutConsistency },
  ]);

  const inconsistencyRows = report.inconsistencies
    .map(
      (i) => `
    <tr>
      <td>${severityBadge(i.severity)}</td>
      <td>${escapeHtml(i.pages.join(', '))}</td>
      <td>${escapeHtml(i.description)}</td>
    </tr>`
    )
    .join('');

  const observations = report.observations.map((o) => `<li>${escapeHtml(o)}</li>`).join('');

  const recommendations = report.recommendations.map((r) => `<li>${escapeHtml(r)}</li>`).join('');

  return `
  <h2>Cross-Page Consistency</h2>
  <div class="card">
    <div class="consistency-grid">
      <div class="chart-col">${chart}</div>
      <div class="details-col">
        <div class="scores-grid">
          <div class="score-row"><span>Overall</span>${scoreBar(report.overallConsistency)}</div>
          <div class="score-row"><span>Color</span>${scoreBar(report.colorConsistency)}</div>
          <div class="score-row"><span>Typography</span>${scoreBar(report.typographyConsistency)}</div>
          <div class="score-row"><span>Spacing</span>${scoreBar(report.spacingConsistency)}</div>
          <div class="score-row"><span>Layout</span>${scoreBar(report.layoutConsistency)}</div>
        </div>
      </div>
    </div>

    ${observations.length > 0 ? `<h4>Observations</h4><ul>${observations}</ul>` : ''}

    ${
      report.inconsistencies.length > 0
        ? `
    <h4>Inconsistencies</h4>
    <table class="issues-table">
      <thead><tr><th>Severity</th><th>Pages</th><th>Description</th></tr></thead>
      <tbody>${inconsistencyRows}</tbody>
    </table>`
        : '<p class="no-issues">No inconsistencies detected</p>'
    }

    ${recommendations.length > 0 ? `<h4>Recommendations</h4><ul>${recommendations}</ul>` : ''}
  </div>`;
}

// ---------------------------------------------------------------------------
// Main report generator
// ---------------------------------------------------------------------------

export async function generateVisionReport(
  report: VisionReviewReport,
  screenshots: ScreenshotEntry[],
  outputPath: string
): Promise<void> {
  // Load screenshots as base64
  const screenshotMap = new Map<string, string>();
  for (const entry of screenshots) {
    try {
      const buf = await fs.readFile(entry.path);
      screenshotMap.set(entry.page, buf.toString('base64'));
    } catch {
      // Screenshot may not exist — skip silently
    }
  }

  // Status
  const overallOk = report.averageScore >= 4 && report.criticalIssueCount === 0;
  const statusIcon = overallOk ? '&#10003;' : '&#10007;';
  const statusColor = overallOk ? '#22c55e' : '#ef4444';
  const totalIssues = report.pageReviews.reduce((n, r) => n + r.issues.length, 0);

  // Page cards
  const pageCards = report.pageReviews
    .map((r) => pageCard(r, screenshotMap.get(r.page) ?? null))
    .join('\n');

  // Consistency section
  const consistencyHtml = report.consistencyReport
    ? consistencySection(report.consistencyReport)
    : '';

  const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Vision Design Review - ${escapeHtml(report.timestamp)}</title>
  <style>
    :root {
      --bg: #fafaf9; --fg: #1c1917; --card: #ffffff; --border: #e7e5e4;
      --muted: #78716c; --primary: #2563eb; --primary-light: rgba(37,99,235,0.08);
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #1c1917; --fg: #fafaf9; --card: #292524; --border: #44403c;
        --muted: #a8a29e; --primary: #60a5fa; --primary-light: rgba(96,165,250,0.1);
      }
    }
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: 'Inter', system-ui, -apple-system, sans-serif;
      background: var(--bg); color: var(--fg);
      line-height: 1.6; padding: 2rem; max-width: 1400px; margin: 0 auto;
    }
    h1 { font-size: 1.5rem; }
    h2 { font-size: 1.25rem; margin: 2.5rem 0 1rem; border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; }
    h3 { font-size: 1.1rem; margin: 0; }
    h4 { font-size: 0.9rem; margin: 1rem 0 0.5rem; color: var(--muted); text-transform: uppercase; letter-spacing: 0.05em; }
    ul { padding-left: 1.25rem; font-size: 0.875rem; }
    li { margin-bottom: 0.25rem; }
    .timestamp { color: var(--muted); font-size: 0.875rem; margin-bottom: 2rem; }
    .card {
      background: var(--card); border: 1px solid var(--border);
      border-radius: 8px; padding: 1.25rem; margin-bottom: 1.25rem;
    }

    /* Summary grid */
    .summary-grid {
      display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
      gap: 1rem; margin-bottom: 2rem;
    }
    .summary-card { text-align: center; padding: 1rem; }
    .summary-card .number { font-size: 2rem; font-weight: 700; line-height: 1.2; }
    .summary-card .label { color: var(--muted); font-size: 0.8rem; }

    /* Page cards */
    .page-card { margin-bottom: 1.5rem; }
    .page-header {
      display: flex; justify-content: space-between; align-items: center;
      margin-bottom: 1rem; padding-bottom: 0.75rem; border-bottom: 1px solid var(--border);
    }
    .viewport-tag {
      font-size: 0.75rem; font-weight: 400; color: var(--muted);
      background: var(--primary-light); padding: 2px 8px; border-radius: 4px; margin-left: 0.5rem;
    }
    .overall-score { font-size: 2rem; font-weight: 700; }
    .score-max { font-size: 0.9rem; font-weight: 400; color: var(--muted); }
    .page-body { display: grid; grid-template-columns: auto 1fr; gap: 1.5rem; }
    @media (max-width: 800px) { .page-body { grid-template-columns: 1fr; } }
    .chart-col { display: flex; flex-direction: column; align-items: center; gap: 1rem; }
    .details-col { min-width: 0; }

    /* Score bars */
    .scores-grid { display: flex; flex-direction: column; gap: 0.4rem; }
    .score-row { display: flex; align-items: center; gap: 0.75rem; font-size: 0.8rem; }
    .score-row > span:first-child { width: 90px; color: var(--muted); }
    .score-bar {
      flex: 1; height: 18px; background: var(--border); border-radius: 4px;
      position: relative; overflow: hidden;
    }
    .score-fill { height: 100%; border-radius: 4px; transition: width 0.3s; }
    .score-label {
      position: absolute; right: 6px; top: 50%; transform: translateY(-50%);
      font-size: 0.7rem; font-weight: 600; color: var(--fg);
    }

    /* Badges */
    .badge {
      display: inline-block; padding: 2px 8px; border-radius: 4px;
      font-size: 0.7rem; font-weight: 600; color: #fff; text-transform: uppercase;
    }

    /* Issues table */
    .issues-section { margin-top: 1rem; }
    .issues-section summary {
      cursor: pointer; font-size: 0.85rem; font-weight: 600; color: var(--primary);
    }
    .issues-table { width: 100%; border-collapse: collapse; font-size: 0.8rem; margin-top: 0.5rem; }
    .issues-table th, .issues-table td {
      padding: 0.4rem 0.6rem; text-align: left; border-bottom: 1px solid var(--border);
    }
    .issues-table th { font-weight: 600; background: var(--bg); }
    .no-issues { font-size: 0.8rem; color: var(--muted); margin-top: 0.5rem; }

    /* Strengths & Recommendations */
    .strengths li::marker { content: '\\2713  '; color: #22c55e; }
    .recommendations li::marker { content: '\\25B8  '; color: var(--primary); }

    /* Consistency */
    .consistency-grid { display: grid; grid-template-columns: auto 1fr; gap: 1.5rem; }
    @media (max-width: 800px) { .consistency-grid { grid-template-columns: 1fr; } }

    /* Screenshot thumbnails */
    .screenshot-thumb {
      max-width: 220px; border: 1px solid var(--border); border-radius: 4px;
      cursor: pointer; transition: transform 0.2s;
    }
    .screenshot-thumb:hover { transform: scale(1.05); }

    /* Lightbox */
    .lightbox {
      display: none; position: fixed; inset: 0; background: rgba(0,0,0,0.85);
      z-index: 1000; align-items: center; justify-content: center; cursor: pointer;
    }
    .lightbox.active { display: flex; }
    .lightbox img { max-width: 95vw; max-height: 95vh; border-radius: 8px; }

    footer {
      margin-top: 3rem; padding-top: 1rem; border-top: 1px solid var(--border);
      color: var(--muted); font-size: 0.75rem;
    }
  </style>
</head>
<body>
  <header>
    <h1 style="color:${statusColor}">${statusIcon} Vision Design Review</h1>
    <p class="timestamp">Generated: ${escapeHtml(report.timestamp)} | Reviewed by Claude (claude-opus-4-6)</p>
  </header>

  <h2>Summary</h2>
  <div class="summary-grid">
    <div class="card summary-card">
      <div class="number" style="color:${scoreColor(report.averageScore)}">${report.averageScore}</div>
      <div class="label">Avg Score /10</div>
    </div>
    <div class="card summary-card">
      <div class="number">${report.pageReviews.length}</div>
      <div class="label">Pages Reviewed</div>
    </div>
    <div class="card summary-card">
      <div class="number" style="color:${report.criticalIssueCount > 0 ? '#ef4444' : '#22c55e'}">${report.criticalIssueCount}</div>
      <div class="label">Critical Issues</div>
    </div>
    <div class="card summary-card">
      <div class="number" style="color:${totalIssues > 10 ? '#f97316' : '#22c55e'}">${totalIssues}</div>
      <div class="label">Total Issues</div>
    </div>
    ${
      report.consistencyReport
        ? `<div class="card summary-card">
      <div class="number" style="color:${scoreColor(report.consistencyReport.overallConsistency)}">${report.consistencyReport.overallConsistency}</div>
      <div class="label">Consistency /10</div>
    </div>`
        : ''
    }
  </div>

  <h2>Page Reviews</h2>
  ${pageCards}

  ${consistencyHtml}

  <div id="lightbox" class="lightbox">
    <img id="lightbox-img" src="" alt="Screenshot"/>
  </div>

  <footer>
    Generated by Xelanote Vision Design Review | Powered by Claude Max (claude-opus-4-6)
  </footer>

  <script>
    // Lightbox for screenshots
    document.querySelectorAll('.screenshot-thumb').forEach(img => {
      img.addEventListener('click', () => {
        const lb = document.getElementById('lightbox');
        const lbImg = document.getElementById('lightbox-img');
        if (lb && lbImg) {
          lbImg.src = img.src;
          lb.classList.add('active');
        }
      });
    });
    document.getElementById('lightbox')?.addEventListener('click', (e) => {
      e.currentTarget.classList.remove('active');
    });
  </script>
</body>
</html>`;

  await fs.mkdir(path.dirname(outputPath), { recursive: true });
  await fs.writeFile(outputPath, html, 'utf8');
}
