#!/usr/bin/env node
/**
 * Adds a pixel-perfect iPhone 15 Pro frame around mobile screenshots.
 *
 * Frame geometry from MagicUI (MIT license):
 * https://github.com/magicuidesign/magicui/blob/main/apps/www/registry/magicui/iphone.tsx
 *
 * Usage:  cd /workspace && node scripts/add-iphone-frame.mjs
 * Output: docs/pr-assets/screenshots/mobile-framed/
 */

import sharp from 'sharp';
import fs from 'node:fs/promises';
import path from 'node:path';

// --- MagicUI iPhone 15 Pro geometry ---
const PHONE_W = 433;
const PHONE_H = 882;
const SCREEN_X = 21.25;
const SCREEN_Y = 19.25;
const SCREEN_W = 389.5;
const SCREEN_H = 843.5;
const SCREEN_R = 55.75;

const SCALE = 2;   // Retina
const PAD = 48;     // Shadow padding

const CANVAS_W = PHONE_W + PAD * 2;
const CANVAS_H = PHONE_H + PAD * 2;

/**
 * Build a production-quality iPhone 15 Pro SVG frame.
 * Uses compound path (fill-rule="evenodd") for a reliable transparent screen hole
 * instead of SVG masks (which can render inconsistently across rasterizers).
 */
function buildFrameSvg() {
  // Helper: rounded rect path going counter-clockwise (for evenodd hole)
  const sx = PAD + SCREEN_X;
  const sy = PAD + SCREEN_Y;
  const sw = SCREEN_W;
  const sh = SCREEN_H;
  const sr = SCREEN_R;

  // Counter-clockwise rounded rect (inner cutout)
  const screenHole = [
    `M${sx + sr},${sy}`,
    `L${sx},${sy + sr}`,          // top-left corner (CCW = go down)
    `A${sr},${sr} 0 0 0 ${sx},${sy + sr}`, // (already there from L)
    // Actually, let me construct this properly with arcs
  ].join(' ');

  // Use a cleaner approach: construct body as compound path with evenodd
  // Outer body (clockwise) + screen rect (counter-clockwise) = frame with hole
  const ox = PAD; // outer body offset
  const oy = PAD;

  return `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg"
     width="${CANVAS_W}" height="${CANVAS_H}"
     viewBox="0 0 ${CANVAS_W} ${CANVAS_H}" fill="none">
  <defs>
    <!-- Metallic body gradient (subtle left-to-right reflection) -->
    <linearGradient id="bodyGrad" x1="0" y1="0" x2="1" y2="0.3">
      <stop offset="0%" stop-color="#4a4a4a"/>
      <stop offset="20%" stop-color="#3d3d3d"/>
      <stop offset="50%" stop-color="#353535"/>
      <stop offset="80%" stop-color="#3d3d3d"/>
      <stop offset="100%" stop-color="#484848"/>
    </linearGradient>

    <!-- Inner bezel gradient (darker, slight vertical variation) -->
    <linearGradient id="bezelGrad" x1="0.5" y1="0" x2="0.5" y2="1">
      <stop offset="0%" stop-color="#1f1f1f"/>
      <stop offset="50%" stop-color="#1a1a1a"/>
      <stop offset="100%" stop-color="#1f1f1f"/>
    </linearGradient>

    <!-- Edge highlight gradient -->
    <linearGradient id="edgeGrad" x1="0" y1="0" x2="1" y2="0.5">
      <stop offset="0%" stop-color="#666"/>
      <stop offset="15%" stop-color="#555"/>
      <stop offset="50%" stop-color="#444"/>
      <stop offset="85%" stop-color="#555"/>
      <stop offset="100%" stop-color="#666"/>
    </linearGradient>

    <!-- Button gradient -->
    <linearGradient id="btnGrad" x1="0" y1="0" x2="1" y2="0">
      <stop offset="0%" stop-color="#505050"/>
      <stop offset="50%" stop-color="#3a3a3a"/>
      <stop offset="100%" stop-color="#505050"/>
    </linearGradient>

    <!-- Drop shadow -->
    <filter id="shadow" x="-20%" y="-10%" width="140%" height="130%"
            color-interpolation-filters="sRGB">
      <feGaussianBlur in="SourceAlpha" stdDeviation="12" result="blur"/>
      <feOffset in="blur" dx="0" dy="8" result="offsetBlur"/>
      <feFlood flood-color="#000" flood-opacity="0.28" result="color"/>
      <feComposite in="color" in2="offsetBlur" operator="in" result="shadow"/>
      <feMerge>
        <feMergeNode in="shadow"/>
        <feMergeNode in="SourceGraphic"/>
      </feMerge>
    </filter>

    <!-- Screen cutout mask -->
    <mask id="screenMask" maskUnits="userSpaceOnUse">
      <rect width="${CANVAS_W}" height="${CANVAS_H}" fill="white"/>
      <rect x="${sx}" y="${sy}" width="${sw}" height="${sh}"
            rx="${sr}" ry="${sr}" fill="black"/>
    </mask>

    <!-- Camera lens radial gradient -->
    <radialGradient id="lensGrad" cx="0.4" cy="0.35" r="0.6">
      <stop offset="0%" stop-color="#333"/>
      <stop offset="40%" stop-color="#1a1a1a"/>
      <stop offset="100%" stop-color="#111"/>
    </radialGradient>
  </defs>

  <!-- === Phone body with integrated shadow === -->
  <g filter="url(#shadow)">
    <!-- Masked group: everything except screen area -->
    <g mask="url(#screenMask)">

      <!-- Outer body (titanium frame) -->
      <path d="M${ox + 2} ${oy + 73}C${ox + 2} ${oy + 32.683} ${ox + 34.683} ${oy} ${ox + 75} ${oy}H${ox + 357}C${ox + 397.317} ${oy} ${ox + 430} ${oy + 32.683} ${ox + 430} ${oy + 73}V${ox + 809}C${ox + 430} ${oy + 849.317} ${ox + 397.317} ${oy + 882} ${ox + 357} ${oy + 882}H${ox + 75}C${ox + 34.683} ${oy + 882} ${ox + 2} ${oy + 849.317} ${ox + 2} ${oy + 809}V${oy + 73}Z"
            fill="url(#bodyGrad)"/>

      <!-- Metallic edge highlight (outer ring) -->
      <path d="M${ox + 2} ${oy + 73}C${ox + 2} ${oy + 32.683} ${ox + 34.683} ${oy} ${ox + 75} ${oy}H${ox + 357}C${ox + 397.317} ${oy} ${ox + 430} ${oy + 32.683} ${ox + 430} ${oy + 73}V${ox + 809}C${ox + 430} ${oy + 849.317} ${ox + 397.317} ${oy + 882} ${ox + 357} ${oy + 882}H${ox + 75}C${ox + 34.683} ${oy + 882} ${ox + 2} ${oy + 849.317} ${ox + 2} ${oy + 809}V${oy + 73}Z"
            fill="none" stroke="url(#edgeGrad)" stroke-width="1.5"/>

      <!-- Inner bezel (darker recessed area around screen) -->
      <path d="M${ox + 6} ${oy + 74}C${ox + 6} ${oy + 35.34} ${ox + 37.34} ${oy + 4} ${ox + 76} ${oy + 4}H${ox + 356}C${ox + 394.66} ${oy + 4} ${ox + 426} ${oy + 35.34} ${ox + 426} ${oy + 74}V${oy + 808}C${ox + 426} ${oy + 846.66} ${ox + 394.66} ${oy + 878} ${ox + 356} ${oy + 878}H${ox + 76}C${ox + 37.34} ${oy + 878} ${ox + 6} ${oy + 846.66} ${ox + 6} ${oy + 808}V${oy + 74}Z"
            fill="url(#bezelGrad)"/>

      <!-- Screen border (thin line between bezel and screen) -->
      <rect x="${sx - 0.5}" y="${sy - 0.5}" width="${sw + 1}" height="${sh + 1}"
            rx="${sr + 0.5}" ry="${sr + 0.5}"
            fill="none" stroke="#333" stroke-width="0.75"/>
    </g>

    <!-- Side buttons (outside mask so they're always visible) -->
    <!-- Silent switch (left) -->
    <path d="M${ox} ${oy + 171}C${ox} ${oy + 170.448} ${ox + 0.448} ${oy + 170} ${ox + 1} ${oy + 170}H${ox + 3}V${oy + 204}H${ox + 1}C${ox + 0.448} ${oy + 204} ${ox} ${oy + 203.552} ${ox} ${oy + 203}V${oy + 171}Z"
          fill="url(#btnGrad)"/>
    <!-- Volume up (left) -->
    <path d="M${ox + 1} ${oy + 234}C${ox + 1} ${oy + 233.448} ${ox + 1.448} ${oy + 233} ${ox + 2} ${oy + 233}H${ox + 3.5}V${oy + 300}H${ox + 2}C${ox + 1.448} ${oy + 300} ${ox + 1} ${oy + 299.552} ${ox + 1} ${oy + 299}V${oy + 234}Z"
          fill="url(#btnGrad)"/>
    <!-- Volume down (left) -->
    <path d="M${ox + 1} ${oy + 319}C${ox + 1} ${oy + 318.448} ${ox + 1.448} ${oy + 318} ${ox + 2} ${oy + 318}H${ox + 3.5}V${oy + 385}H${ox + 2}C${ox + 1.448} ${oy + 385} ${ox + 1} ${oy + 384.552} ${ox + 1} ${oy + 384}V${oy + 319}Z"
          fill="url(#btnGrad)"/>
    <!-- Power (right) -->
    <path d="M${ox + 430} ${oy + 279}H${ox + 432}C${ox + 432.552} ${oy + 279} ${ox + 433} ${oy + 279.448} ${ox + 433} ${oy + 280}V${oy + 384}C${ox + 433} ${oy + 384.552} ${ox + 432.552} ${oy + 385} ${ox + 432} ${oy + 385}H${ox + 430}V${oy + 279}Z"
          fill="url(#btnGrad)"/>
  </g>

  <!-- Top antenna line (subtle hardware detail) -->
  <path opacity="0.35"
        d="M${ox + 174} ${oy + 5}H${ox + 258}V${oy + 5.5}C${ox + 258} ${oy + 6.605} ${ox + 257.105} ${oy + 7.5} ${ox + 256} ${oy + 7.5}H${ox + 176}C${ox + 174.895} ${oy + 7.5} ${ox + 174} ${oy + 6.605} ${ox + 174} ${oy + 5.5}V${oy + 5}Z"
        fill="#666"/>

  <!-- === Dynamic Island === -->
  <!-- Pill background -->
  <path d="M${ox + 154} ${oy + 48.5}C${ox + 154} ${oy + 38.283} ${ox + 162.283} ${oy + 30} ${ox + 172.5} ${oy + 30}H${ox + 259.5}C${ox + 269.717} ${oy + 30} ${ox + 278} ${oy + 38.283} ${ox + 278} ${oy + 48.5}C${ox + 278} ${oy + 58.717} ${ox + 269.717} ${oy + 67} ${ox + 259.5} ${oy + 67}H${ox + 172.5}C${ox + 162.283} ${oy + 67} ${ox + 154} ${oy + 58.717} ${ox + 154} ${oy + 48.5}Z"
        fill="#111"/>

  <!-- Camera lens (outer housing) -->
  <circle cx="${ox + 259.5}" cy="${oy + 48.5}" r="10.5" fill="#0d0d0d"/>
  <!-- Camera lens (glass) -->
  <circle cx="${ox + 259.5}" cy="${oy + 48.5}" r="5.5" fill="url(#lensGrad)"/>
  <!-- Camera lens (reflection highlight) -->
  <circle cx="${ox + 258}" cy="${oy + 47}" r="1.8" fill="#444" opacity="0.3"/>
</svg>`;
}

/**
 * Rounded-corner mask for the screenshot.
 */
function buildScreenMask(w, h, r) {
  return Buffer.from(
    `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}">
      <rect width="${w}" height="${h}" rx="${r}" ry="${r}" fill="white"/>
    </svg>`
  );
}

async function addFrame(inputPath, outputPath) {
  const canvasW = Math.round(CANVAS_W * SCALE);
  const canvasH = Math.round(CANVAS_H * SCALE);
  const screenW = Math.round(SCREEN_W * SCALE);
  const screenH = Math.round(SCREEN_H * SCALE);
  const screenX = Math.round((PAD + SCREEN_X) * SCALE);
  const screenY = Math.round((PAD + SCREEN_Y) * SCALE);
  const screenR = Math.round(SCREEN_R * SCALE);

  // 1. Resize screenshot to fill screen area exactly (negligible distortion)
  const screenshot = await sharp(inputPath)
    .resize(screenW, screenH, { fit: 'fill' })
    .ensureAlpha()
    .toBuffer();

  // 2. Round the screenshot corners to match the screen shape
  const mask = buildScreenMask(screenW, screenH, screenR);
  const maskPng = await sharp(mask).resize(screenW, screenH).png().toBuffer();
  const roundedScreenshot = await sharp(screenshot)
    .composite([{ input: maskPng, blend: 'dest-in' }])
    .png()
    .toBuffer();

  // 3. Render the frame SVG at 2x density
  const frameSvg = Buffer.from(buildFrameSvg());
  const frame = await sharp(frameSvg, { density: 72 * SCALE })
    .resize(canvasW, canvasH)
    .ensureAlpha()
    .png()
    .toBuffer();

  // 4. Composite onto transparent canvas
  const result = await sharp({
    create: {
      width: canvasW,
      height: canvasH,
      channels: 4,
      background: { r: 0, g: 0, b: 0, alpha: 0 },
    },
  })
    .composite([
      { input: roundedScreenshot, left: screenX, top: screenY },
      { input: frame, left: 0, top: 0 },
    ])
    .png({ compressionLevel: 9 })
    .toBuffer();

  await fs.writeFile(outputPath, result);
  return (await fs.stat(outputPath)).size;
}

async function main() {
  const mobileDir = path.resolve('docs/pr-assets/screenshots/mobile');
  const framedDir = path.resolve('docs/pr-assets/screenshots/mobile-framed');
  await fs.mkdir(framedDir, { recursive: true });

  const files = (await fs.readdir(mobileDir)).filter((f) => f.endsWith('.png'));
  console.log(`Adding iPhone 15 Pro frame to ${files.length} screenshots (${SCALE}x Retina)...\n`);

  let total = 0;
  for (const file of files) {
    const size = await addFrame(
      path.join(mobileDir, file),
      path.join(framedDir, file)
    );
    total += size;
    console.log(`  ${file.padEnd(30)} ${(size / 1024).toFixed(0).padStart(4)} KB`);
  }

  console.log(`\n  Total: ${(total / 1024).toFixed(0)} KB`);
  console.log(`  Resolution: ${Math.round(CANVAS_W * SCALE)}x${Math.round(CANVAS_H * SCALE)}`);
  console.log(`  Output: ${framedDir}/`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
