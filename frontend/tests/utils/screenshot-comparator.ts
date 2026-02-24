import fs from 'node:fs/promises';
import path from 'node:path';

import pixelmatch from 'pixelmatch';
import { PNG } from 'pngjs';

export interface DiffRegion {
  x: number;
  y: number;
  width: number;
  height: number;
  diffPercentage: number;
}

export interface ComparisonResult {
  match: boolean;
  diffPercentage: number;
  diffPixelCount: number;
  totalPixels: number;
  diffImagePath: string | null;
  regions: DiffRegion[];
  severity: 'none' | 'minor' | 'moderate' | 'critical';
  metadata: {
    baselineSize: { width: number; height: number };
    currentSize: { width: number; height: number };
    sizeMismatch: boolean;
    timestamp: string;
    browser: string;
    viewport: string;
  };
}

export interface CompareOptions {
  threshold?: number;
  outputDir?: string;
  browser?: string;
  viewport?: string;
  regionGridSize?: number;
}

function classifySeverity(diffPercentage: number): ComparisonResult['severity'] {
  if (diffPercentage < 0.1) return 'none';
  if (diffPercentage < 1) return 'minor';
  if (diffPercentage < 5) return 'moderate';
  return 'critical';
}

function findDiffRegions(
  diffBuffer: Uint8Array,
  width: number,
  height: number,
  gridSize: number
): DiffRegion[] {
  const regions: DiffRegion[] = [];
  const cols = Math.ceil(width / gridSize);
  const rows = Math.ceil(height / gridSize);

  for (let row = 0; row < rows; row++) {
    for (let col = 0; col < cols; col++) {
      const startX = col * gridSize;
      const startY = row * gridSize;
      const regionWidth = Math.min(gridSize, width - startX);
      const regionHeight = Math.min(gridSize, height - startY);
      let diffCount = 0;
      const totalRegionPixels = regionWidth * regionHeight;

      for (let y = startY; y < startY + regionHeight; y++) {
        for (let x = startX; x < startX + regionWidth; x++) {
          const idx = (y * width + x) * 4;
          // If R channel > 0 in diff image, pixel differs
          if (diffBuffer[idx] > 0 && diffBuffer[idx + 1] === 0) {
            diffCount++;
          }
        }
      }

      const diffPct = (diffCount / totalRegionPixels) * 100;
      if (diffPct > 0.5) {
        regions.push({
          x: startX,
          y: startY,
          width: regionWidth,
          height: regionHeight,
          diffPercentage: Math.round(diffPct * 100) / 100,
        });
      }
    }
  }

  return regions;
}

export async function compareScreenshots(
  baselinePath: string,
  currentPath: string,
  name: string,
  options: CompareOptions = {}
): Promise<ComparisonResult> {
  const {
    threshold = 0.2,
    outputDir = './tests/results/diffs',
    browser = 'unknown',
    viewport = 'unknown',
    regionGridSize = 64,
  } = options;

  const timestamp = new Date().toISOString();

  // Read images
  const [baselineBuffer, currentBuffer] = await Promise.all([
    fs.readFile(baselinePath),
    fs.readFile(currentPath),
  ]);

  const baseline = PNG.sync.read(baselineBuffer);
  const current = PNG.sync.read(currentBuffer);

  const sizeMismatch = baseline.width !== current.width || baseline.height !== current.height;

  if (sizeMismatch) {
    return {
      match: false,
      diffPercentage: 100,
      diffPixelCount: baseline.width * baseline.height,
      totalPixels: baseline.width * baseline.height,
      diffImagePath: null,
      regions: [],
      severity: 'critical',
      metadata: {
        baselineSize: { width: baseline.width, height: baseline.height },
        currentSize: { width: current.width, height: current.height },
        sizeMismatch: true,
        timestamp,
        browser,
        viewport,
      },
    };
  }

  const { width, height } = baseline;
  const totalPixels = width * height;
  const diff = new PNG({ width, height });

  const diffPixelCount = pixelmatch(baseline.data, current.data, diff.data, width, height, {
    threshold,
  });

  const diffPercentage = Math.round((diffPixelCount / totalPixels) * 10000) / 100;
  const severity = classifySeverity(diffPercentage);
  const match = severity === 'none';

  let diffImagePath: string | null = null;

  if (!match) {
    await fs.mkdir(outputDir, { recursive: true });
    diffImagePath = path.join(outputDir, `${name}-diff.png`);
    await fs.writeFile(diffImagePath, PNG.sync.write(diff));

    // Create side-by-side comparison
    const sideByWidth = width * 3 + 4; // 2px gap between each
    const sideBySide = new PNG({ width: sideByWidth, height });

    // Fill with white background
    for (let i = 0; i < sideBySide.data.length; i += 4) {
      sideBySide.data[i] = 255;
      sideBySide.data[i + 1] = 255;
      sideBySide.data[i + 2] = 255;
      sideBySide.data[i + 3] = 255;
    }

    // Copy baseline, current, and diff side by side
    for (let y = 0; y < height; y++) {
      for (let x = 0; x < width; x++) {
        const srcIdx = (y * width + x) * 4;

        // Baseline (left)
        const dstLeft = (y * sideByWidth + x) * 4;
        sideBySide.data[dstLeft] = baseline.data[srcIdx];
        sideBySide.data[dstLeft + 1] = baseline.data[srcIdx + 1];
        sideBySide.data[dstLeft + 2] = baseline.data[srcIdx + 2];
        sideBySide.data[dstLeft + 3] = baseline.data[srcIdx + 3];

        // Current (middle)
        const dstMid = (y * sideByWidth + (x + width + 2)) * 4;
        sideBySide.data[dstMid] = current.data[srcIdx];
        sideBySide.data[dstMid + 1] = current.data[srcIdx + 1];
        sideBySide.data[dstMid + 2] = current.data[srcIdx + 2];
        sideBySide.data[dstMid + 3] = current.data[srcIdx + 3];

        // Diff (right)
        const dstRight = (y * sideByWidth + (x + width * 2 + 4)) * 4;
        sideBySide.data[dstRight] = diff.data[srcIdx];
        sideBySide.data[dstRight + 1] = diff.data[srcIdx + 1];
        sideBySide.data[dstRight + 2] = diff.data[srcIdx + 2];
        sideBySide.data[dstRight + 3] = diff.data[srcIdx + 3];
      }
    }

    const sideBySidePath = path.join(outputDir, `${name}-comparison.png`);
    await fs.writeFile(sideBySidePath, PNG.sync.write(sideBySide));
  }

  const regions = !match ? findDiffRegions(diff.data, width, height, regionGridSize) : [];

  return {
    match,
    diffPercentage,
    diffPixelCount,
    totalPixels,
    diffImagePath,
    regions,
    severity,
    metadata: {
      baselineSize: { width, height },
      currentSize: { width: current.width, height: current.height },
      sizeMismatch: false,
      timestamp,
      browser,
      viewport,
    },
  };
}
