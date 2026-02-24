/**
 * Vision Design Reviewer — uses the `claude` CLI (Claude Max subscription)
 * to analyze UI screenshots via Claude Vision.
 *
 * No API key needed: runs `claude -p` which authenticates through the
 * locally configured Claude Code session.
 */
import { execSync } from 'node:child_process';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface DesignReviewResult {
  page: string;
  viewport: string;
  overallScore: number; // 1-10
  consistency: number; // 1-10
  visualHierarchy: number; // 1-10
  whitespace: number; // 1-10
  readability: number; // 1-10
  issues: DesignIssue[];
  strengths: string[];
  recommendations: string[];
}

export interface DesignIssue {
  severity: 'minor' | 'moderate' | 'critical';
  area: string;
  description: string;
  suggestion: string;
}

export interface ConsistencyReport {
  overallConsistency: number; // 1-10
  colorConsistency: number;
  typographyConsistency: number;
  spacingConsistency: number;
  layoutConsistency: number;
  observations: string[];
  inconsistencies: Array<{
    pages: string[];
    description: string;
    severity: 'minor' | 'moderate' | 'critical';
  }>;
  recommendations: string[];
}

export interface VisionReviewReport {
  timestamp: string;
  pageReviews: DesignReviewResult[];
  consistencyReport: ConsistencyReport | null;
  averageScore: number;
  criticalIssueCount: number;
}

export interface ScreenshotEntry {
  path: string;
  page: string;
  viewport: string;
}

// ---------------------------------------------------------------------------
// Claude CLI helpers
// ---------------------------------------------------------------------------

/** Check whether the `claude` CLI is installed and reachable. */
export function isClaudeCliAvailable(): boolean {
  try {
    execSync('which claude', {
      encoding: 'utf-8',
      timeout: 5000,
      stdio: 'pipe',
    });
    return true;
  } catch {
    return false;
  }
}

/**
 * Invoke `claude -p` (print/pipe mode) with the given prompt piped via stdin.
 * The CLI uses the locally authenticated Claude Max session — no API key needed.
 */
function callClaude(prompt: string, timeoutMs = 300_000): string {
  try {
    const result = execSync('claude -p --allowedTools Read', {
      input: prompt,
      encoding: 'utf-8',
      timeout: timeoutMs,
      maxBuffer: 1024 * 1024 * 10, // 10 MB
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    return result;
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    throw new Error(`Claude CLI call failed: ${message}`);
  }
}

/** Extract JSON from Claude's text response (handles code-block wrapping). */
function parseJsonFromOutput<T>(output: string): T {
  const trimmed = output.trim();

  // 1. Direct parse
  try {
    return JSON.parse(trimmed);
  } catch {
    /* continue */
  }

  // 2. Extract from ```json ... ``` code block
  const codeBlockMatch = trimmed.match(/```(?:json)?\s*\n([\s\S]*?)\n```/);
  if (codeBlockMatch) {
    try {
      return JSON.parse(codeBlockMatch[1].trim());
    } catch {
      /* continue */
    }
  }

  // 3. Find the outermost JSON structure (array or object)
  const arrayMatch = trimmed.match(/\[[\s\S]*\]/);
  if (arrayMatch) {
    try {
      return JSON.parse(arrayMatch[0]);
    } catch {
      /* continue */
    }
  }

  const objectMatch = trimmed.match(/\{[\s\S]*\}/);
  if (objectMatch) {
    try {
      return JSON.parse(objectMatch[0]);
    } catch {
      /* continue */
    }
  }

  throw new Error(`Failed to parse JSON from Claude output:\n${trimmed.substring(0, 500)}`);
}

// ---------------------------------------------------------------------------
// Prompts
// ---------------------------------------------------------------------------

const SYSTEM_CONTEXT = `You are a senior UI/UX design reviewer analyzing screenshots of "Xelanote", a personal note-taking web application. The app uses a Gruvbox-inspired color scheme with Tailwind CSS.

Focus on practical design quality:
- Layout structure and alignment
- Whitespace and breathing room
- Visual hierarchy (headings, CTAs, grouping)
- Typography (sizing, weight, line-height)
- Consistency with the overall design language
- Readability and scannability
- Mobile-specific concerns (touch targets, text size, overflow)

Be specific and actionable. Reference concrete areas of the UI.`;

// ---------------------------------------------------------------------------
// Review functions
// ---------------------------------------------------------------------------

/**
 * Review a batch of screenshots. Each screenshot is read by Claude via the
 * Read tool, then evaluated for design quality.
 *
 * Returns one DesignReviewResult per screenshot.
 */
export function reviewPages(screenshots: ScreenshotEntry[]): DesignReviewResult[] {
  if (screenshots.length === 0) return [];

  const listing = screenshots.map((s) => `- ${s.path} → "${s.page}" (${s.viewport})`).join('\n');

  const prompt = `${SYSTEM_CONTEXT}

Read and analyze these UI screenshots. For each one, evaluate the design quality.

Screenshots to review:
${listing}

Read each image file listed above using the Read tool, then return a JSON array where each element has this exact structure:

[
  {
    "page": "<page name from the list>",
    "viewport": "<viewport from the list>",
    "overallScore": <number 1-10>,
    "consistency": <number 1-10>,
    "visualHierarchy": <number 1-10>,
    "whitespace": <number 1-10>,
    "readability": <number 1-10>,
    "issues": [
      {
        "severity": "minor" | "moderate" | "critical",
        "area": "<area of the page, e.g. Header, Sidebar, Content>",
        "description": "<what is wrong>",
        "suggestion": "<how to fix it>"
      }
    ],
    "strengths": ["<positive aspect 1>", "<positive aspect 2>"],
    "recommendations": ["<actionable suggestion 1>", "<actionable suggestion 2>"]
  }
]

Scoring guide:
- 1-3: Significant design problems, poor UX
- 4-5: Functional but needs improvement
- 6-7: Good design with minor issues
- 8-9: Excellent, polished design
- 10: Near-perfect

IMPORTANT: Return ONLY the JSON array. No markdown fences, no text before or after.`;

  const output = callClaude(prompt);
  return parseJsonFromOutput<DesignReviewResult[]>(output);
}

/**
 * Analyse cross-page design consistency by comparing multiple screenshots.
 */
export function reviewConsistency(screenshots: ScreenshotEntry[]): ConsistencyReport {
  if (screenshots.length < 2) {
    throw new Error('Need at least 2 screenshots for consistency review');
  }

  const listing = screenshots.map((s) => `- ${s.path} → "${s.page}" (${s.viewport})`).join('\n');

  const prompt = `${SYSTEM_CONTEXT}

Read these UI screenshots and analyze the CROSS-PAGE DESIGN CONSISTENCY. Focus on whether the pages feel like they belong to the same application and follow the same design system.

Screenshots:
${listing}

Read each image file listed above using the Read tool, then return a JSON object:

{
  "overallConsistency": <number 1-10>,
  "colorConsistency": <number 1-10>,
  "typographyConsistency": <number 1-10>,
  "spacingConsistency": <number 1-10>,
  "layoutConsistency": <number 1-10>,
  "observations": ["<general observation about the design system>"],
  "inconsistencies": [
    {
      "pages": ["<page1>", "<page2>"],
      "description": "<what is inconsistent between these pages>",
      "severity": "minor" | "moderate" | "critical"
    }
  ],
  "recommendations": ["<actionable suggestion for better consistency>"]
}

IMPORTANT: Return ONLY the JSON object. No markdown fences, no text before or after.`;

  const output = callClaude(prompt);
  return parseJsonFromOutput<ConsistencyReport>(output);
}
