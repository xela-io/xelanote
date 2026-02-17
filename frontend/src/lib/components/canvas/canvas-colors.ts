// Canvas color presets matching the Gruvbox theme palette.
// Preset indices 1-6 map to the JSON Canvas spec color field.

export const CANVAS_COLOR_PRESETS = [
  { id: '1', name: 'Red', cssVar: '--canvas-red', bgVar: '--canvas-red-bg' },
  { id: '2', name: 'Orange', cssVar: '--canvas-orange', bgVar: '--canvas-orange-bg' },
  { id: '3', name: 'Yellow', cssVar: '--canvas-yellow', bgVar: '--canvas-yellow-bg' },
  { id: '4', name: 'Green', cssVar: '--canvas-green', bgVar: '--canvas-green-bg' },
  { id: '5', name: 'Blue', cssVar: '--canvas-blue', bgVar: '--canvas-blue-bg' },
  { id: '6', name: 'Purple', cssVar: '--canvas-purple', bgVar: '--canvas-purple-bg' },
] as const;

/**
 * Get the CSS color value for a canvas color preset or hex value.
 */
export function getCanvasColor(color: string | undefined): string | undefined {
  if (!color) return undefined;
  const preset = CANVAS_COLOR_PRESETS.find((p) => p.id === color);
  if (preset) return `var(${preset.cssVar})`;
  if (color.startsWith('#')) return color;
  return undefined;
}

/**
 * Get the CSS background color for a canvas color preset.
 */
export function getCanvasBgColor(color: string | undefined): string | undefined {
  if (!color) return undefined;
  const preset = CANVAS_COLOR_PRESETS.find((p) => p.id === color);
  if (preset) return `var(${preset.bgVar})`;
  if (color.startsWith('#')) return `${color}20`; // Hex with low opacity
  return undefined;
}
