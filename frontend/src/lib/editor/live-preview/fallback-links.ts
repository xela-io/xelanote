export function extractMarkdownLinksFromText(
  text: string,
  baseOffset: number
): Array<{ from: number; to: number; label: string; href: string }> {
  const links: Array<{ from: number; to: number; label: string; href: string }> = [];

  let i = 0;
  while (i < text.length) {
    const labelStart = text.indexOf('[', i);
    if (labelStart === -1) break;
    const labelEnd = text.indexOf(']', labelStart + 1);
    if (labelEnd === -1 || labelEnd + 1 >= text.length || text[labelEnd + 1] !== '(') {
      i = labelStart + 1;
      continue;
    }

    let depth = 1;
    let hrefEnd = labelEnd + 2;
    while (hrefEnd < text.length && depth > 0) {
      const ch = text[hrefEnd];
      if (ch === '(') depth++;
      if (ch === ')') depth--;
      hrefEnd++;
    }

    if (depth !== 0) {
      i = labelStart + 1;
      continue;
    }

    const fullEnd = hrefEnd - 1;
    const label = text.slice(labelStart + 1, labelEnd);
    const href = text.slice(labelEnd + 2, fullEnd).trim();
    if (label.length > 0 && href.length > 0) {
      links.push({
        from: baseOffset + labelStart,
        to: baseOffset + fullEnd + 1,
        label,
        href,
      });
    }
    i = fullEnd + 1;
  }

  return links;
}
