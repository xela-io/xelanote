import { EditorState } from '@codemirror/state';
import { EditorView } from '@codemirror/view';
import { afterEach, describe, expect, it } from 'vitest';

import { insertTask } from './task-insert';

const views: EditorView[] = [];

function createView(doc: string, anchor: number): EditorView {
  const parent = document.createElement('div');
  document.body.appendChild(parent);
  const view = new EditorView({
    state: EditorState.create({
      doc,
      selection: { anchor },
    }),
    parent,
  });
  views.push(view);
  return view;
}

describe('insertTask', () => {
  afterEach(() => {
    for (const view of views.splice(0)) {
      view.destroy();
    }
    document.body.innerHTML = '';
  });

  it('inserts before first checked task when cursor is inside that list section', () => {
    const doc = '- [x] Done A\n- [x] Done B';
    const view = createView(doc, doc.length);

    insertTask(view);

    expect(view.state.doc.toString()).toBe('- [ ] \n- [x] Done A\n- [x] Done B');
  });

  it('does not jump to completed tasks above another heading section', () => {
    const doc = '# Heading A\n- [x] Done A\n## Heading B\nText';
    const view = createView(doc, doc.length);

    insertTask(view);

    expect(view.state.doc.toString()).toBe('# Heading A\n- [x] Done A\n## Heading B\nText\n- [ ] ');
  });
});
