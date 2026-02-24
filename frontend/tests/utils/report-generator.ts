import fs from 'node:fs/promises';
import path from 'node:path';

export interface TestResult {
  name: string;
  status: 'passed' | 'failed' | 'skipped';
  duration: number;
  error?: string;
  screenshots?: string[];
}

export interface VisualDiff {
  name: string;
  baselinePath: string;
  currentPath: string;
  diffPath?: string;
  diffPercentage: number;
  severity: 'none' | 'minor' | 'moderate' | 'critical';
}

export interface DesignIssue {
  page: string;
  type: string;
  severity: string;
  message: string;
  selector: string;
}

export interface A11yIssue {
  page: string;
  id: string;
  impact: string;
  description: string;
  nodeCount: number;
}

export interface VisionReviewSummary {
  averageScore: number;
  criticalIssueCount: number;
  pageCount: number;
  reportPath: string;
}

export interface ReportData {
  timestamp: string;
  duration: number;
  tests: TestResult[];
  visualDiffs: VisualDiff[];
  designIssues: DesignIssue[];
  a11yIssues: A11yIssue[];
  errors: Array<{
    type: string;
    message: string;
    url: string;
    severity: string;
  }>;
  visionReview?: VisionReviewSummary;
}

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function severityBadge(severity: string): string {
  const colors: Record<string, string> = {
    none: '#22c55e',
    minor: '#eab308',
    moderate: '#f97316',
    critical: '#ef4444',
    error: '#ef4444',
    warning: '#f97316',
    info: '#3b82f6',
    low: '#22c55e',
    medium: '#eab308',
    high: '#f97316',
    serious: '#ef4444',
  };
  const color = colors[severity] ?? '#6b7280';
  return `<span style="display:inline-block;padding:2px 8px;border-radius:4px;font-size:12px;font-weight:600;color:#fff;background:${color}">${escapeHtml(severity)}</span>`;
}

export async function generateHtmlReport(data: ReportData, outputPath: string): Promise<void> {
  const passed = data.tests.filter((t) => t.status === 'passed').length;
  const failed = data.tests.filter((t) => t.status === 'failed').length;
  const _skipped = data.tests.filter((t) => t.status === 'skipped').length;
  const total = data.tests.length;

  const visualCritical = data.visualDiffs.filter((d) => d.severity === 'critical').length;
  const _visualModerate = data.visualDiffs.filter((d) => d.severity === 'moderate').length;

  const overallStatus = failed > 0 || visualCritical > 0 ? 'failed' : 'passed';
  const statusIcon = overallStatus === 'passed' ? '&#10003;' : '&#10007;';
  const statusColor = overallStatus === 'passed' ? '#22c55e' : '#ef4444';

  const html = `<!DOCTYPE html>
<html lang="de">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Xelanote Test Report - ${data.timestamp}</title>
  <style>
    :root {
      --bg: #fafaf9;
      --fg: #1c1917;
      --card: #ffffff;
      --border: #e7e5e4;
      --muted: #78716c;
      --primary: #2563eb;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #1c1917;
        --fg: #fafaf9;
        --card: #292524;
        --border: #44403c;
        --muted: #a8a29e;
        --primary: #60a5fa;
      }
    }
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body {
      font-family: 'Inter', system-ui, -apple-system, sans-serif;
      background: var(--bg);
      color: var(--fg);
      line-height: 1.6;
      padding: 2rem;
      max-width: 1200px;
      margin: 0 auto;
    }
    h1 { font-size: 1.5rem; margin-bottom: 0.5rem; }
    h2 { font-size: 1.25rem; margin: 2rem 0 1rem; border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; }
    h3 { font-size: 1rem; margin: 1.5rem 0 0.5rem; }
    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 8px;
      padding: 1.25rem;
      margin-bottom: 1rem;
    }
    .summary-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 1rem;
      margin-bottom: 2rem;
    }
    .summary-card {
      text-align: center;
      padding: 1rem;
    }
    .summary-card .number {
      font-size: 2rem;
      font-weight: 700;
      line-height: 1.2;
    }
    .summary-card .label {
      color: var(--muted);
      font-size: 0.875rem;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.875rem;
    }
    th, td {
      padding: 0.5rem 0.75rem;
      text-align: left;
      border-bottom: 1px solid var(--border);
    }
    th {
      font-weight: 600;
      background: var(--card);
      position: sticky;
      top: 0;
    }
    tr:hover td { background: var(--card); }
    .status-passed { color: #22c55e; }
    .status-failed { color: #ef4444; }
    .status-skipped { color: var(--muted); }
    .timestamp {
      color: var(--muted);
      font-size: 0.875rem;
    }
    .filter-bar {
      display: flex;
      gap: 0.5rem;
      margin-bottom: 1rem;
      flex-wrap: wrap;
    }
    .filter-btn {
      padding: 4px 12px;
      border: 1px solid var(--border);
      border-radius: 4px;
      background: var(--card);
      color: var(--fg);
      cursor: pointer;
      font-size: 0.8rem;
    }
    .filter-btn.active {
      background: var(--primary);
      color: #fff;
      border-color: var(--primary);
    }
    .collapsible { cursor: pointer; user-select: none; }
    .collapsible::before { content: '\\25B6 '; font-size: 0.75rem; }
    .collapsible.open::before { content: '\\25BC '; }
    .collapse-content { display: none; margin-top: 0.5rem; }
    .collapse-content.show { display: block; }
    pre {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 4px;
      padding: 0.75rem;
      overflow-x: auto;
      font-size: 0.8rem;
    }
    @media print {
      body { padding: 1rem; }
      .filter-bar { display: none; }
    }
  </style>
</head>
<body>
  <header>
    <h1 style="color:${statusColor}">${statusIcon} Xelanote Test Report</h1>
    <p class="timestamp">Generated: ${escapeHtml(data.timestamp)} | Duration: ${(data.duration / 1000).toFixed(1)}s</p>
  </header>

  <h2>Executive Summary</h2>
  <div class="summary-grid">
    <div class="card summary-card">
      <div class="number" style="color:${statusColor}">${passed}/${total}</div>
      <div class="label">Tests Passed</div>
    </div>
    <div class="card summary-card">
      <div class="number" style="color:${failed > 0 ? '#ef4444' : '#22c55e'}">${failed}</div>
      <div class="label">Tests Failed</div>
    </div>
    <div class="card summary-card">
      <div class="number" style="color:${visualCritical > 0 ? '#ef4444' : '#22c55e'}">${visualCritical}</div>
      <div class="label">Critical Visual Diffs</div>
    </div>
    <div class="card summary-card">
      <div class="number">${data.designIssues.length}</div>
      <div class="label">Design Issues</div>
    </div>
    <div class="card summary-card">
      <div class="number">${data.a11yIssues.length}</div>
      <div class="label">A11y Issues</div>
    </div>
    <div class="card summary-card">
      <div class="number" style="color:${data.errors.length > 0 ? '#f97316' : '#22c55e'}">${data.errors.length}</div>
      <div class="label">Runtime Errors</div>
    </div>
  </div>

  <h2>Test Results</h2>
  <div class="card">
    <table>
      <thead>
        <tr>
          <th>Test</th>
          <th>Status</th>
          <th>Duration</th>
          <th>Error</th>
        </tr>
      </thead>
      <tbody>
        ${data.tests
          .map(
            (t) => `
        <tr>
          <td>${escapeHtml(t.name)}</td>
          <td class="status-${t.status}">${t.status}</td>
          <td>${(t.duration / 1000).toFixed(1)}s</td>
          <td>${t.error ? `<pre>${escapeHtml(t.error.substring(0, 200))}</pre>` : '-'}</td>
        </tr>`
          )
          .join('')}
      </tbody>
    </table>
  </div>

  ${
    data.visualDiffs.length > 0
      ? `
  <h2>Visual Regression (${data.visualDiffs.length} diffs)</h2>
  <div class="card">
    <table>
      <thead>
        <tr>
          <th>Screenshot</th>
          <th>Severity</th>
          <th>Diff %</th>
        </tr>
      </thead>
      <tbody>
        ${data.visualDiffs
          .sort((a, b) => b.diffPercentage - a.diffPercentage)
          .map(
            (d) => `
        <tr>
          <td>${escapeHtml(d.name)}</td>
          <td>${severityBadge(d.severity)}</td>
          <td>${d.diffPercentage.toFixed(2)}%</td>
        </tr>`
          )
          .join('')}
      </tbody>
    </table>
  </div>`
      : ''
  }

  ${
    data.designIssues.length > 0
      ? `
  <h2>Design Audit (${data.designIssues.length} issues)</h2>
  <div class="card">
    <table>
      <thead>
        <tr>
          <th>Page</th>
          <th>Type</th>
          <th>Severity</th>
          <th>Message</th>
          <th>Selector</th>
        </tr>
      </thead>
      <tbody>
        ${data.designIssues
          .map(
            (d) => `
        <tr>
          <td>${escapeHtml(d.page)}</td>
          <td>${escapeHtml(d.type)}</td>
          <td>${severityBadge(d.severity)}</td>
          <td>${escapeHtml(d.message)}</td>
          <td><code>${escapeHtml(d.selector.substring(0, 60))}</code></td>
        </tr>`
          )
          .join('')}
      </tbody>
    </table>
  </div>`
      : ''
  }

  ${
    data.a11yIssues.length > 0
      ? `
  <h2>Accessibility (${data.a11yIssues.length} issues)</h2>
  <div class="card">
    <table>
      <thead>
        <tr>
          <th>Page</th>
          <th>Rule</th>
          <th>Impact</th>
          <th>Description</th>
          <th>Nodes</th>
        </tr>
      </thead>
      <tbody>
        ${data.a11yIssues
          .map(
            (a) => `
        <tr>
          <td>${escapeHtml(a.page)}</td>
          <td><code>${escapeHtml(a.id)}</code></td>
          <td>${severityBadge(a.impact)}</td>
          <td>${escapeHtml(a.description)}</td>
          <td>${a.nodeCount}</td>
        </tr>`
          )
          .join('')}
      </tbody>
    </table>
  </div>`
      : ''
  }

  ${
    data.errors.length > 0
      ? `
  <h2>Runtime Errors (${data.errors.length})</h2>
  <div class="card">
    <table>
      <thead>
        <tr>
          <th>Type</th>
          <th>Severity</th>
          <th>URL</th>
          <th>Message</th>
        </tr>
      </thead>
      <tbody>
        ${data.errors
          .map(
            (e) => `
        <tr>
          <td>${escapeHtml(e.type)}</td>
          <td>${severityBadge(e.severity)}</td>
          <td>${escapeHtml(e.url)}</td>
          <td>${escapeHtml(e.message.substring(0, 150))}</td>
        </tr>`
          )
          .join('')}
      </tbody>
    </table>
  </div>`
      : ''
  }

  ${
    data.visionReview
      ? `
  <h2>AI Vision Design Review</h2>
  <div class="card">
    <div style="display:flex;gap:2rem;align-items:center;flex-wrap:wrap">
      <div style="text-align:center">
        <div style="font-size:2rem;font-weight:700;color:${data.visionReview.averageScore >= 7 ? '#22c55e' : data.visionReview.averageScore >= 4 ? '#eab308' : '#ef4444'}">${data.visionReview.averageScore}/10</div>
        <div style="color:var(--muted);font-size:0.8rem">Avg Design Score</div>
      </div>
      <div style="text-align:center">
        <div style="font-size:2rem;font-weight:700">${data.visionReview.pageCount}</div>
        <div style="color:var(--muted);font-size:0.8rem">Pages Reviewed</div>
      </div>
      <div style="text-align:center">
        <div style="font-size:2rem;font-weight:700;color:${data.visionReview.criticalIssueCount > 0 ? '#ef4444' : '#22c55e'}">${data.visionReview.criticalIssueCount}</div>
        <div style="color:var(--muted);font-size:0.8rem">Critical Issues</div>
      </div>
      <div style="flex:1;text-align:right">
        <a href="${escapeHtml(data.visionReview.reportPath)}" style="color:var(--primary);font-size:0.85rem">Full Vision Report &rarr;</a>
      </div>
    </div>
  </div>`
      : ''
  }

  <footer style="margin-top:3rem;padding-top:1rem;border-top:1px solid var(--border);color:var(--muted);font-size:0.8rem">
    Generated by Xelanote Test Suite | ${escapeHtml(data.timestamp)}
  </footer>

  <script>
    document.querySelectorAll('.collapsible').forEach(el => {
      el.addEventListener('click', () => {
        el.classList.toggle('open');
        const content = el.nextElementSibling;
        if (content) content.classList.toggle('show');
      });
    });

    document.querySelectorAll('.filter-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        btn.classList.toggle('active');
        // Filter logic would go here
      });
    });
  </script>
</body>
</html>`;

  await fs.mkdir(path.dirname(outputPath), { recursive: true });
  await fs.writeFile(outputPath, html, 'utf8');

  // Also write raw JSON data
  const jsonPath = outputPath.replace('.html', '.json');
  await fs.writeFile(jsonPath, JSON.stringify(data, null, 2), 'utf8');
}
