#!/usr/bin/env node
// Generate dark-mode variants of Apple splash screens.
// Reads existing PNGs from static/splash/, creates dark versions
// with #282828 background + centered icon-512.png.

import { readdir } from 'node:fs/promises';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import sharp from 'sharp';

const __dirname = dirname(fileURLToPath(import.meta.url));
const SPLASH_DIR = join(__dirname, '..', 'static', 'splash');
const ICON_PATH = join(__dirname, '..', 'static', 'icon-512.png');
const DARK_BG = '#282828';
const ICON_SCALE = 0.2; // 20% of shorter dimension

async function main() {
  const files = await readdir(SPLASH_DIR);
  const splashFiles = files.filter(
    (f) => f.startsWith('apple-splash-') && f.endsWith('.png') && !f.includes('-dark')
  );

  console.log(`Found ${splashFiles.length} splash screens to generate dark variants for`);

  const iconBuffer = await sharp(ICON_PATH).png().toBuffer();

  for (const file of splashFiles) {
    const match = file.match(/apple-splash-(\d+)-(\d+)\.png/);
    if (!match) {
      console.warn(`Skipping ${file}: could not parse dimensions`);
      continue;
    }

    const width = parseInt(match[1], 10);
    const height = parseInt(match[2], 10);
    const iconSize = Math.round(Math.min(width, height) * ICON_SCALE);

    const resizedIcon = await sharp(iconBuffer)
      .resize(iconSize, iconSize, { fit: 'contain', background: { r: 0, g: 0, b: 0, alpha: 0 } })
      .png()
      .toBuffer();

    const darkFile = file.replace('.png', '-dark.png');
    const outputPath = join(SPLASH_DIR, darkFile);

    await sharp({
      create: {
        width,
        height,
        channels: 4,
        background: DARK_BG,
      },
    })
      .composite([
        {
          input: resizedIcon,
          gravity: 'centre',
        },
      ])
      .png()
      .toFile(outputPath);

    console.log(`  ✓ ${darkFile} (${width}×${height}, icon ${iconSize}px)`);
  }

  console.log(`\nDone! Generated ${splashFiles.length} dark splash screens.`);
}

main().catch((err) => {
  console.error('Error:', err);
  process.exit(1);
});
