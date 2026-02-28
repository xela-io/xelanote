// Quick-input parser for shopping list items.
// Multi-pass tokenizer: split → quantity → unit → name

export interface ParsedItem {
  name: string;
  quantity: number | null;
  unit: string | null;
}

// Unit normalization map (case-insensitive)
const UNIT_MAP: Record<string, string> = {
  stk: 'Stück',
  stück: 'Stück',
  st: 'Stück',
  g: 'g',
  gr: 'g',
  gramm: 'g',
  kg: 'kg',
  kilo: 'kg',
  ml: 'ml',
  milliliter: 'ml',
  l: 'l',
  liter: 'l',
  pkg: 'Packung',
  packung: 'Packung',
  pck: 'Packung',
  pack: 'Packung',
  bund: 'Bund',
  bd: 'Bund',
  dose: 'Dose',
  dos: 'Dose',
  flasche: 'Flasche',
  fl: 'Flasche',
  becher: 'Becher',
  tüte: 'Tüte',
  scheiben: 'Scheibe(n)',
  scheibe: 'Scheibe(n)',
  el: 'EL',
  esslöffel: 'EL',
  tl: 'TL',
  teelöffel: 'TL',
};

// Regex patterns
const LEADING_NUMBER = /^(\d+(?:[.,]\d+)?)\s*/;
const ATTACHED_UNIT = /^(\d+(?:[.,]\d+)?)\s*([a-zA-ZäöüÄÖÜß]+)\s+/;
const NX_PATTERN = /^(\d+)x\s*/i;

/**
 * Parse a single input token into a structured item.
 */
function parseToken(raw: string): ParsedItem {
  const token = raw.trim();
  if (!token) return { name: '', quantity: null, unit: null };

  // Pass 1: Check for "Nx" pattern (e.g., "3x Tomaten")
  const nxMatch = token.match(NX_PATTERN);
  if (nxMatch) {
    return {
      name: token.slice(nxMatch[0].length).trim(),
      quantity: parseNumber(nxMatch[1]),
      unit: null,
    };
  }

  // Pass 2: Check for attached unit (e.g., "500g Hack", "2,5l Milch")
  const attachedMatch = token.match(ATTACHED_UNIT);
  if (attachedMatch) {
    const unitKey = attachedMatch[2].toLowerCase();
    const normalizedUnit = UNIT_MAP[unitKey];
    if (normalizedUnit) {
      return {
        name: token.slice(attachedMatch[0].length).trim(),
        quantity: parseNumber(attachedMatch[1]),
        unit: normalizedUnit,
      };
    }
  }

  // Pass 3: Check for leading number + separate word as unit
  const numberMatch = token.match(LEADING_NUMBER);
  if (numberMatch) {
    const rest = token.slice(numberMatch[0].length).trim();
    const words = rest.split(/\s+/);

    if (words.length >= 2) {
      const possibleUnit = words[0].toLowerCase();
      const normalizedUnit = UNIT_MAP[possibleUnit];
      if (normalizedUnit) {
        return {
          name: words.slice(1).join(' ').trim(),
          quantity: parseNumber(numberMatch[1]),
          unit: normalizedUnit,
        };
      }
    }

    // Just a leading number, no unit (e.g., "3 Äpfel")
    return {
      name: rest,
      quantity: parseNumber(numberMatch[1]),
      unit: null,
    };
  }

  // Fallback: name only
  return { name: token, quantity: null, unit: null };
}

/**
 * Parse a comma/newline-separated input string into structured items.
 */
export function parseShoppingInput(input: string): ParsedItem[] {
  if (!input.trim()) return [];

  // Split by comma or newline
  const tokens = input
    .split(/[,\n]+/)
    .map((t) => t.trim())
    .filter(Boolean);

  return tokens.map(parseToken).filter((item) => item.name.length > 0);
}

/**
 * Parse a number string, supporting both "." and "," as decimal separator.
 */
function parseNumber(str: string): number | null {
  const normalized = str.replace(',', '.');
  const num = parseFloat(normalized);
  return isNaN(num) ? null : num;
}
