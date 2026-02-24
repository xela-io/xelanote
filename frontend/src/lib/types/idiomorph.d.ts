// TypeScript declarations for idiomorph (no official types)
declare module 'idiomorph' {
  export interface MorphCallbacks {
    beforeNodeAdded?: (node: Node) => boolean | void;
    afterNodeAdded?: (node: Node) => void;
    beforeNodeMorphed?: (oldNode: Element, newNode: Element) => boolean | void;
    afterNodeMorphed?: (oldNode: Element, newNode: Element) => void;
    beforeNodeRemoved?: (node: Node) => boolean | void;
    afterNodeRemoved?: (node: Node) => void;
    beforeAttributeUpdated?: (
      attributeName: string,
      node: Element,
      mutationType: 'updated' | 'removed'
    ) => boolean | void;
  }

  export interface MorphOptions {
    morphStyle?: 'innerHTML' | 'outerHTML';
    ignoreActive?: boolean;
    ignoreActiveValue?: boolean;
    restoreFocus?: boolean;
    head?: {
      style?: 'merge' | 'append' | 'morph' | 'none';
    };
    callbacks?: MorphCallbacks;
  }

  export const Idiomorph: {
    morph(
      oldNode: Element | Document,
      newContent: string | Element | Element[],
      options?: MorphOptions
    ): void;
  };

  export default Idiomorph;
}
