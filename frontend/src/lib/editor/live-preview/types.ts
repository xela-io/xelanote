export interface TreeTaskFeature {
  line: number;
  from: number;
  to: number;
  checked: boolean;
}

export interface TreeLinkFeature {
  line: number;
  from: number;
  to: number;
  label: string;
  href: string;
}

export interface TreeInlineFeature {
  line: number;
  from: number;
  to: number;
  text: string;
  className: 'cm-live-preview-code' | 'cm-live-preview-strong' | 'cm-live-preview-em';
}

export interface TreeWikilinkFeature {
  line: number;
  from: number;
  to: number;
  title: string;
  label: string;
}

export interface TreeDueDateFeature {
  line: number;
  from: number;
  to: number;
  date: string;
}

export interface TreeFeatureCollection {
  tasksByLine: Map<number, TreeTaskFeature>;
  linksByLine: Map<number, TreeLinkFeature[]>;
  inlineByLine: Map<number, TreeInlineFeature[]>;
  wikilinksByLine: Map<number, TreeWikilinkFeature[]>;
  dueDatesByLine: Map<number, TreeDueDateFeature[]>;
}
