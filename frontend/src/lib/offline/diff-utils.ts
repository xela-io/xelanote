// Minimal line-based diff utility for the conflict dialog.
// Uses a simple LCS (Longest Common Subsequence) algorithm on lines.

export interface DiffLine {
  type: 'same' | 'added' | 'removed';
  content: string;
  lineNumber?: number;
}

/**
 * Compute a line-based diff between two texts.
 * Returns an array of DiffLine objects showing additions, removals, and unchanged lines.
 */
export function diffLines(oldText: string, newText: string): DiffLine[] {
  const oldLines = oldText.split('\n');
  const newLines = newText.split('\n');

  // Build LCS table
  const m = oldLines.length;
  const n = newLines.length;
  const dp: number[][] = Array.from({ length: m + 1 }, () => Array(n + 1).fill(0));

  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      if (oldLines[i - 1] === newLines[j - 1]) {
        dp[i][j] = dp[i - 1][j - 1] + 1;
      } else {
        dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1]);
      }
    }
  }

  // Backtrack to build diff
  const result: DiffLine[] = [];
  let i = m;
  let j = n;

  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && oldLines[i - 1] === newLines[j - 1]) {
      result.push({ type: 'same', content: oldLines[i - 1], lineNumber: j });
      i--;
      j--;
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      result.push({ type: 'added', content: newLines[j - 1], lineNumber: j });
      j--;
    } else {
      result.push({ type: 'removed', content: oldLines[i - 1], lineNumber: i });
      i--;
    }
  }

  return result.reverse();
}
