import { GFM, parser } from '@lezer/markdown';

export interface TaskMatch {
  line: number;
  checked: boolean;
  from: number;
  to: number;
}

export interface LinkMatch {
  from: number;
  to: number;
  label: string;
  href: string;
}

export interface ExtractionResult {
  tasks: TaskMatch[];
  links: LinkMatch[];
}

export interface SpikeCase {
  name: string;
  markdown: string;
  expectedTasks: number;
  expectedLinks: number;
}

const taskPattern = /^(\s*(?:[-*+]|\d+[.)]) )(\[[xX ]\])(\s+)(.*)$/;
const linkPattern = /\[([^\]]+)\]\(([^)]+)\)/g;

const gfmParser = parser.configure([GFM]);

function buildLineStarts(doc: string): number[] {
  const starts = [0];
  for (let i = 0; i < doc.length; i++) {
    if (doc.charCodeAt(i) === 10) starts.push(i + 1);
  }
  return starts;
}

function lineNumberAt(pos: number, lineStarts: number[]): number {
  let lo = 0;
  let hi = lineStarts.length - 1;
  while (lo <= hi) {
    const mid = (lo + hi) >> 1;
    if (lineStarts[mid] <= pos) lo = mid + 1;
    else hi = mid - 1;
  }
  return hi + 1;
}

function lineEndAt(lineNumber: number, lineStarts: number[], docLength: number): number {
  const next = lineStarts[lineNumber];
  return next === undefined ? docLength : next - 1;
}

export function extractRegexFeatures(doc: string): ExtractionResult {
  const tasks: TaskMatch[] = [];
  const links: LinkMatch[] = [];
  const lines = doc.split('\n');

  let offset = 0;
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const taskMatch = taskPattern.exec(line);
    if (taskMatch) {
      const taskBody = taskMatch[4].trim();
      if (taskBody.length > 0) {
        const markerFrom = offset + taskMatch[1].length;
        const markerTo = markerFrom + taskMatch[2].length;
        tasks.push({
          line: i + 1,
          checked: taskMatch[2].toLowerCase() === '[x]',
          from: markerFrom,
          to: markerTo,
        });
      }
    }
    offset += line.length + 1;
  }

  let match: RegExpExecArray | null;
  while ((match = linkPattern.exec(doc)) !== null) {
    links.push({
      from: match.index,
      to: match.index + match[0].length,
      label: match[1],
      href: match[2].trim(),
    });
  }

  return { tasks, links };
}

export function extractTreeFeatures(doc: string): ExtractionResult {
  const tasks: TaskMatch[] = [];
  const links: LinkMatch[] = [];
  const lineStarts = buildLineStarts(doc);
  const tree = gfmParser.parse(doc);
  const cursor = tree.cursor();

  do {
    const type = cursor.type.name;
    if (type === 'TaskMarker') {
      const marker = doc.slice(cursor.from, cursor.to).toLowerCase();
      const line = lineNumberAt(cursor.from, lineStarts);
      const end = lineEndAt(line, lineStarts, doc.length);
      const taskBody = doc.slice(cursor.to, end).trim();
      if (taskBody.length === 0) continue;
      tasks.push({
        line,
        checked: marker === '[x]',
        from: cursor.from,
        to: cursor.to,
      });
    } else if (type === 'Link') {
      const linkFrom = cursor.from;
      const linkTo = cursor.to;
      const linkCursor = cursor.node.cursor();
      const marks: Array<{ from: number; to: number }> = [];
      let urlRange: { from: number; to: number } | null = null;

      if (linkCursor.firstChild()) {
        do {
          if (linkCursor.type.name === 'LinkMark') {
            marks.push({ from: linkCursor.from, to: linkCursor.to });
          } else if (linkCursor.type.name === 'URL') {
            urlRange = { from: linkCursor.from, to: linkCursor.to };
          }
        } while (linkCursor.nextSibling());
      }

      if (marks.length >= 2 && urlRange) {
        links.push({
          from: linkFrom,
          to: linkTo,
          label: doc.slice(marks[0].to, marks[1].from),
          href: doc.slice(urlRange.from, urlRange.to).trim(),
        });
      }
    }
  } while (cursor.next());

  return { tasks, links };
}

function sumCounts(result: ExtractionResult): number {
  return result.tasks.length + result.links.length;
}

export interface BenchmarkResult {
  name: 'regex' | 'tree';
  avgMs: number;
  totalMatches: number;
}

export function benchmarkExtraction(
  doc: string,
  iterations = 200
): {
  regex: BenchmarkResult;
  tree: BenchmarkResult;
} {
  const now =
    typeof globalThis.performance !== 'undefined' &&
    typeof globalThis.performance.now === 'function'
      ? () => performance.now()
      : () => Date.now();

  for (let i = 0; i < 20; i++) {
    extractRegexFeatures(doc);
    extractTreeFeatures(doc);
  }

  let regexTotal = 0;
  let treeTotal = 0;
  let regexMatches = 0;
  let treeMatches = 0;

  for (let i = 0; i < iterations; i++) {
    let start = now();
    const regex = extractRegexFeatures(doc);
    regexTotal += now() - start;
    regexMatches += sumCounts(regex);

    start = now();
    const tree = extractTreeFeatures(doc);
    treeTotal += now() - start;
    treeMatches += sumCounts(tree);
  }

  return {
    regex: {
      name: 'regex',
      avgMs: regexTotal / iterations,
      totalMatches: regexMatches,
    },
    tree: {
      name: 'tree',
      avgMs: treeTotal / iterations,
      totalMatches: treeMatches,
    },
  };
}

export const spikeCases: SpikeCase[] = [
  {
    name: 'basic tasks and links',
    markdown: '- [ ] Task A\n- [x] Task B\n[Docs](https://example.com/docs)',
    expectedTasks: 2,
    expectedLinks: 1,
  },
  {
    name: 'ordered tasks',
    markdown: '1. [ ] One\n2) [x] Two',
    expectedTasks: 2,
    expectedLinks: 0,
  },
  {
    name: 'ignore empty task marker',
    markdown: '- [ ] \n- [ ] Visible',
    expectedTasks: 1,
    expectedLinks: 0,
  },
  {
    name: 'nested parentheses in URL',
    markdown: '[Spec](https://example.com/a_(b))',
    expectedTasks: 0,
    expectedLinks: 1,
  },
  {
    name: 'mixed content',
    markdown:
      '## Title\n- [ ] Open [[Wiki]]\n- [x] Done @due(2027-02-10)\nSee [Ref](https://a.com/x_(y))',
    expectedTasks: 2,
    expectedLinks: 1,
  },
];

export function calculateBugCount(
  cases: SpikeCase[],
  extractor: (doc: string) => ExtractionResult
): number {
  let bugs = 0;
  for (const testCase of cases) {
    const result = extractor(testCase.markdown);
    if (result.tasks.length !== testCase.expectedTasks) bugs++;
    if (result.links.length !== testCase.expectedLinks) bugs++;
  }
  return bugs;
}

export function generateLargeMarkdownDoc(lineCount = 3000): string {
  const chunks: string[] = [];
  for (let i = 0; i < lineCount; i++) {
    if (i % 12 === 0) {
      chunks.push(`- [ ] Task ${i} with [Link ${i}](https://example.com/${i}_(v2))`);
    } else if (i % 12 === 1) {
      chunks.push(`- [x] Done ${i}`);
    } else if (i % 12 === 2) {
      chunks.push(`## Heading ${i}`);
    } else {
      chunks.push(`Paragraph line ${i} with **bold** and _italic_.`);
    }
  }
  return chunks.join('\n');
}
