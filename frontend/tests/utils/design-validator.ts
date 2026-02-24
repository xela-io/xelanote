import type { Page } from '@playwright/test';

export interface DesignViolation {
  type: 'typography' | 'color' | 'spacing' | 'layout' | 'responsive' | 'contrast';
  message: string;
  element: string;
  selector: string;
  actual: string;
  expected?: string;
  severity: 'info' | 'warning' | 'error';
}

export interface DesignAuditResult {
  violations: DesignViolation[];
  summary: {
    total: number;
    bySeverity: Record<string, number>;
    byType: Record<string, number>;
  };
  page: string;
}

// Xelanote Design System constants
const _ALLOWED_FONT_FAMILIES = ['inter', 'ui-sans-serif', 'system-ui', 'ui-monospace', 'monospace'];
const ALLOWED_FONT_WEIGHTS = [400, 500, 600, 700];
const MIN_TOUCH_TARGET = 44; // WCAG AA minimum
const MIN_MOBILE_FONT_SIZE = 12; // px
const MAX_CONTENT_WIDTH = 1600; // px

export async function validateTypography(page: Page): Promise<DesignViolation[]> {
  return page.evaluate(
    ({ allowedWeights, minMobileFontSize }) => {
      const violations: DesignViolation[] = [];
      const isMobile = window.innerWidth < 768;

      const textElements = document.querySelectorAll(
        'p, h1, h2, h3, h4, h5, h6, span, a, li, td, th, label, button, input, textarea, select'
      );

      textElements.forEach((el) => {
        const styles = getComputedStyle(el);
        const fontSize = parseFloat(styles.fontSize);
        const fontWeight = parseInt(styles.fontWeight);
        const lineHeight = parseFloat(styles.lineHeight);

        // Check minimum font size on mobile
        if (isMobile && fontSize < minMobileFontSize && el.textContent?.trim()) {
          const selector = el.id ? `#${el.id}` : `${el.tagName.toLowerCase()}.${el.className}`;
          violations.push({
            type: 'typography',
            message: `Font size ${fontSize}px is below ${minMobileFontSize}px minimum on mobile`,
            element: el.tagName.toLowerCase(),
            selector,
            actual: `${fontSize}px`,
            expected: `>= ${minMobileFontSize}px`,
            severity: 'warning',
          });
        }

        // Check font weight is in allowed set
        if (!allowedWeights.includes(fontWeight) && el.textContent?.trim()) {
          const selector = el.id ? `#${el.id}` : `${el.tagName.toLowerCase()}.${el.className}`;
          violations.push({
            type: 'typography',
            message: `Font weight ${fontWeight} is not in the allowed set`,
            element: el.tagName.toLowerCase(),
            selector,
            actual: `${fontWeight}`,
            expected: allowedWeights.join(', '),
            severity: 'info',
          });
        }

        // Check line-height ratio (should be >= 1.2 for readability)
        if (fontSize > 0 && lineHeight / fontSize < 1.2 && el.textContent?.trim()) {
          const ratio = Math.round((lineHeight / fontSize) * 100) / 100;
          const selector = el.id ? `#${el.id}` : `${el.tagName.toLowerCase()}.${el.className}`;
          violations.push({
            type: 'typography',
            message: `Line-height ratio ${ratio} is below 1.2 minimum`,
            element: el.tagName.toLowerCase(),
            selector,
            actual: `${ratio}`,
            expected: '>= 1.2',
            severity: 'info',
          });
        }
      });

      return violations;
    },
    {
      allowedWeights: ALLOWED_FONT_WEIGHTS,
      minMobileFontSize: MIN_MOBILE_FONT_SIZE,
    }
  );
}

export async function validateLayout(page: Page): Promise<DesignViolation[]> {
  return page.evaluate(
    ({ maxContentWidth: _maxContentWidth }) => {
      const violations: DesignViolation[] = [];

      // Check for horizontal overflow
      if (document.documentElement.scrollWidth > window.innerWidth) {
        violations.push({
          type: 'layout',
          message: `Page has horizontal overflow: ${document.documentElement.scrollWidth}px > ${window.innerWidth}px`,
          element: 'html',
          selector: 'html',
          actual: `${document.documentElement.scrollWidth}px`,
          expected: `<= ${window.innerWidth}px`,
          severity: 'error',
        });
      }

      // Check images for broken aspect ratios
      const images = document.querySelectorAll('img');
      images.forEach((img) => {
        if (img.complete && img.naturalWidth > 0 && img.width > 0 && img.height > 0) {
          const naturalRatio = img.naturalWidth / img.naturalHeight;
          const displayRatio = img.width / img.height;
          const ratioDiff = Math.abs(naturalRatio - displayRatio);

          if (ratioDiff > 0.1) {
            violations.push({
              type: 'layout',
              message: `Image has distorted aspect ratio`,
              element: 'img',
              selector: img.src.substring(img.src.lastIndexOf('/') + 1),
              actual: `${displayRatio.toFixed(2)}`,
              expected: `${naturalRatio.toFixed(2)}`,
              severity: 'warning',
            });
          }
        }
      });

      return violations;
    },
    { maxContentWidth: MAX_CONTENT_WIDTH }
  );
}

export async function validateTouchTargets(page: Page): Promise<DesignViolation[]> {
  return page.evaluate(
    ({ minSize }) => {
      const violations: DesignViolation[] = [];
      const isMobile = window.innerWidth < 768;

      if (!isMobile) return violations;

      const interactiveElements = document.querySelectorAll(
        'a, button, input, select, textarea, [role="button"], [role="link"], [role="tab"], [role="checkbox"], [role="radio"]'
      );

      interactiveElements.forEach((el) => {
        const rect = el.getBoundingClientRect();
        if (rect.width === 0 || rect.height === 0) return;

        // Skip hidden elements
        const styles = getComputedStyle(el);
        if (styles.display === 'none' || styles.visibility === 'hidden') return;

        if (rect.width < minSize || rect.height < minSize) {
          const selector = el.id
            ? `#${el.id}`
            : `${el.tagName.toLowerCase()}${el.className ? '.' + String(el.className).split(' ')[0] : ''}`;
          violations.push({
            type: 'responsive',
            message: `Touch target too small: ${Math.round(rect.width)}x${Math.round(rect.height)}px`,
            element: el.tagName.toLowerCase(),
            selector,
            actual: `${Math.round(rect.width)}x${Math.round(rect.height)}px`,
            expected: `>= ${minSize}x${minSize}px`,
            severity: 'warning',
          });
        }
      });

      return violations;
    },
    { minSize: MIN_TOUCH_TARGET }
  );
}

export async function validateContrast(page: Page): Promise<DesignViolation[]> {
  return page.evaluate(() => {
    const violations: DesignViolation[] = [];

    function luminance(r: number, g: number, b: number): number {
      const a = [r, g, b].map((v) => {
        v /= 255;
        return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
      });
      return a[0] * 0.2126 + a[1] * 0.7152 + a[2] * 0.0722;
    }

    function contrastRatio(l1: number, l2: number): number {
      const lighter = Math.max(l1, l2);
      const darker = Math.min(l1, l2);
      return (lighter + 0.05) / (darker + 0.05);
    }

    function parseColor(color: string): { r: number; g: number; b: number } | null {
      const match = color.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
      if (match) {
        return {
          r: parseInt(match[1]),
          g: parseInt(match[2]),
          b: parseInt(match[3]),
        };
      }
      return null;
    }

    const textElements = document.querySelectorAll(
      'p, h1, h2, h3, h4, h5, h6, span, a, li, label, button'
    );

    textElements.forEach((el) => {
      if (!el.textContent?.trim()) return;

      const styles = getComputedStyle(el);
      const fontSize = parseFloat(styles.fontSize);
      const fontWeight = parseInt(styles.fontWeight);
      const isLargeText = fontSize >= 18 || (fontSize >= 14 && fontWeight >= 700);

      const fg = parseColor(styles.color);
      const bg = parseColor(styles.backgroundColor);

      if (!fg || !bg) return;
      // Skip transparent backgrounds
      if (styles.backgroundColor === 'rgba(0, 0, 0, 0)') return;

      const fgLum = luminance(fg.r, fg.g, fg.b);
      const bgLum = luminance(bg.r, bg.g, bg.b);
      const ratio = contrastRatio(fgLum, bgLum);

      const minRatio = isLargeText ? 3 : 4.5; // WCAG AA

      if (ratio < minRatio) {
        const selector = el.id
          ? `#${el.id}`
          : `${el.tagName.toLowerCase()}${el.className ? '.' + String(el.className).split(' ')[0] : ''}`;
        violations.push({
          type: 'contrast',
          message: `Contrast ratio ${ratio.toFixed(2)}:1 below WCAG AA minimum ${minRatio}:1`,
          element: el.tagName.toLowerCase(),
          selector,
          actual: `${ratio.toFixed(2)}:1`,
          expected: `>= ${minRatio}:1`,
          severity: 'error',
        });
      }
    });

    return violations;
  });
}

export async function validateHeadingHierarchy(page: Page): Promise<DesignViolation[]> {
  return page.evaluate(() => {
    const violations: DesignViolation[] = [];
    const headings = document.querySelectorAll('h1, h2, h3, h4, h5, h6');
    let lastLevel = 0;

    headings.forEach((h) => {
      const level = parseInt(h.tagName[1]);

      if (lastLevel > 0 && level > lastLevel + 1) {
        violations.push({
          type: 'typography',
          message: `Heading hierarchy skipped from h${lastLevel} to h${level}`,
          element: h.tagName.toLowerCase(),
          selector: h.textContent?.substring(0, 30) ?? h.tagName,
          actual: `h${level}`,
          expected: `h${lastLevel + 1} or lower`,
          severity: 'warning',
        });
      }

      lastLevel = level;
    });

    return violations;
  });
}

export async function runDesignAudit(page: Page, routeName: string): Promise<DesignAuditResult> {
  const allViolations: DesignViolation[] = [];

  const [typography, layout, touchTargets, contrast, headings] = await Promise.all([
    validateTypography(page),
    validateLayout(page),
    validateTouchTargets(page),
    validateContrast(page),
    validateHeadingHierarchy(page),
  ]);

  allViolations.push(...typography, ...layout, ...touchTargets, ...contrast, ...headings);

  const bySeverity: Record<string, number> = {};
  const byType: Record<string, number> = {};

  for (const v of allViolations) {
    bySeverity[v.severity] = (bySeverity[v.severity] ?? 0) + 1;
    byType[v.type] = (byType[v.type] ?? 0) + 1;
  }

  return {
    violations: allViolations,
    summary: {
      total: allViolations.length,
      bySeverity,
      byType,
    },
    page: routeName,
  };
}
