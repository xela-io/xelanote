// Line primitive extraction: heading, blockquote, list marker, task checkbox detection

export interface LinePrimitives {
  heading: { indentLength: number; level: number; spacingLength: number } | null;
  blockquote: { indentLength: number; spacingLength: number } | null;
  markerPrefixLength: number | null;
  taskRegex: { markerLength: number; markerTokenLength: number; checked: boolean } | null;
  listMarker: { indentLength: number; marker: string; spacingLength: number } | null;
}

export function parseLinePrimitives(text: string): LinePrimitives {
  const headingMatch = /^(\s{0,3})(#{1,6})(\s+)/.exec(text);
  const blockquoteMatch = /^(\s{0,3})>(\s*)/.exec(text);
  const markerPrefixMatch = /^(\s*(?:[-*+]|\d+[.)])\s+)/.exec(text);
  const taskRegexMatch = /^(\s*(?:[-*+]|\d+[.)]) )(\[[xX ]\])(\s+)/.exec(text);
  const listMarkerMatch = /^(\s*)([-*+]|\d+[.)])(\s+)/.exec(text);

  return {
    heading: headingMatch
      ? {
          indentLength: headingMatch[1].length,
          level: headingMatch[2].length,
          spacingLength: headingMatch[3].length,
        }
      : null,
    blockquote: blockquoteMatch
      ? {
          indentLength: blockquoteMatch[1].length,
          spacingLength: blockquoteMatch[2].length,
        }
      : null,
    markerPrefixLength: markerPrefixMatch ? markerPrefixMatch[1].length : null,
    taskRegex: taskRegexMatch
      ? {
          markerLength: taskRegexMatch[1].length,
          markerTokenLength: taskRegexMatch[2].length + taskRegexMatch[3].length,
          checked: taskRegexMatch[2].toLowerCase() === '[x]',
        }
      : null,
    listMarker: listMarkerMatch
      ? {
          indentLength: listMarkerMatch[1].length,
          marker: listMarkerMatch[2],
          spacingLength: listMarkerMatch[3].length,
        }
      : null,
  };
}
