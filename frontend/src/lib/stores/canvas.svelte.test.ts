import { describe, expect, it } from 'vitest';

import type { CanvasData } from '$lib/api/types';
import {
  canvasToFlow,
  flowToCanvas,
  parseCanvas,
  serializeCanvas,
} from '$lib/stores/canvas.svelte';

describe('parseCanvas', () => {
  it('should parse valid canvas JSON', () => {
    const data = parseCanvas('{"nodes":[],"edges":[]}');
    expect(data).toEqual({ nodes: [], edges: [] });
  });

  it('should return empty canvas for empty string', () => {
    const data = parseCanvas('');
    expect(data).toEqual({ nodes: [], edges: [] });
  });

  it('should return empty canvas for empty object', () => {
    const data = parseCanvas('{}');
    expect(data).toEqual({ nodes: [], edges: [] });
  });

  it('should return null for invalid JSON', () => {
    const data = parseCanvas('not json');
    expect(data).toBeNull();
  });

  it('should parse canvas with nodes and edges', () => {
    const json = JSON.stringify({
      nodes: [{ id: 'n1', type: 'text', x: 0, y: 0, width: 100, height: 100, text: 'hello' }],
      edges: [{ id: 'e1', fromNode: 'n1', toNode: 'n2' }],
    });
    const data = parseCanvas(json);
    expect(data).not.toBeNull();
    expect(data!.nodes).toHaveLength(1);
    expect(data!.edges).toHaveLength(1);
  });
});

describe('serializeCanvas', () => {
  it('should serialize canvas data to formatted JSON', () => {
    const data: CanvasData = { nodes: [], edges: [] };
    const json = serializeCanvas(data);
    expect(json).toBe('{\n  "nodes": [],\n  "edges": []\n}');
  });
});

describe('canvasToFlow', () => {
  it('should convert text nodes', () => {
    const data: CanvasData = {
      nodes: [{ id: 'n1', type: 'text', x: 100, y: 200, width: 300, height: 150, text: 'hello' }],
      edges: [],
    };
    const { nodes, edges } = canvasToFlow(data);
    expect(nodes).toHaveLength(1);
    expect(edges).toHaveLength(0);
    expect(nodes[0].id).toBe('n1');
    expect(nodes[0].type).toBe('canvas-text');
    expect(nodes[0].position).toEqual({ x: 100, y: 200 });
  });

  it('should convert edges with source/target', () => {
    const data: CanvasData = {
      nodes: [
        { id: 'n1', type: 'text', x: 0, y: 0, width: 100, height: 100, text: 'a' },
        { id: 'n2', type: 'text', x: 200, y: 0, width: 100, height: 100, text: 'b' },
      ],
      edges: [
        {
          id: 'e1',
          fromNode: 'n1',
          toNode: 'n2',
          fromSide: 'right',
          toSide: 'left',
          toEnd: 'arrow',
        },
      ],
    };
    const { edges } = canvasToFlow(data);
    expect(edges).toHaveLength(1);
    expect(edges[0].source).toBe('n1');
    expect(edges[0].target).toBe('n2');
  });

  it('should compute parentId for nodes inside groups', () => {
    const data: CanvasData = {
      nodes: [
        { id: 'g1', type: 'group', x: 0, y: 0, width: 500, height: 400, label: 'Group' },
        { id: 'n1', type: 'text', x: 50, y: 50, width: 100, height: 100, text: 'inside' },
      ],
      edges: [],
    };
    const { nodes } = canvasToFlow(data);
    const textNode = nodes.find((n) => n.id === 'n1');
    expect(textNode).toBeDefined();
    expect(textNode!.parentId).toBe('g1');
    // Position should be relative to group
    expect(textNode!.position).toEqual({ x: 50, y: 50 });
  });

  it('should sort groups before other nodes', () => {
    const data: CanvasData = {
      nodes: [
        { id: 'n1', type: 'text', x: 50, y: 50, width: 100, height: 100, text: 'text' },
        { id: 'g1', type: 'group', x: 0, y: 0, width: 500, height: 400, label: 'Group' },
      ],
      edges: [],
    };
    const { nodes } = canvasToFlow(data);
    expect(nodes[0].id).toBe('g1');
    expect(nodes[1].id).toBe('n1');
  });
});

describe('flowToCanvas', () => {
  it('should convert flow nodes back to canvas format', () => {
    const flowNodes = [
      {
        id: 'n1',
        type: 'canvas-text',
        position: { x: 100, y: 200 },
        measured: { width: 300, height: 150 },
        data: { type: 'text', text: 'hello', id: 'n1', x: 100, y: 200, width: 300, height: 150 },
      },
    ];
    const data = flowToCanvas(flowNodes, []);
    expect(data.nodes).toHaveLength(1);
    expect(data.nodes[0].type).toBe('text');
    expect(data.nodes[0].x).toBe(100);
    expect(data.nodes[0].y).toBe(200);
    expect((data.nodes[0] as { text: string }).text).toBe('hello');
  });

  it('should convert positions back to absolute for grouped nodes', () => {
    const flowNodes = [
      {
        id: 'g1',
        type: 'canvas-group',
        position: { x: 100, y: 100 },
        measured: { width: 500, height: 400 },
        data: { type: 'group', label: 'Group', id: 'g1', x: 100, y: 100, width: 500, height: 400 },
      },
      {
        id: 'n1',
        type: 'canvas-text',
        position: { x: 50, y: 50 },
        parentId: 'g1',
        measured: { width: 200, height: 100 },
        data: { type: 'text', text: 'hello', id: 'n1', x: 150, y: 150, width: 200, height: 100 },
      },
    ];
    const data = flowToCanvas(flowNodes, []);
    const textNode = data.nodes.find((n) => n.id === 'n1');
    expect(textNode).toBeDefined();
    // Should be absolute: parent(100,100) + relative(50,50) = (150,150)
    expect(textNode!.x).toBe(150);
    expect(textNode!.y).toBe(150);
  });

  it('should strip parentId from canvas output', () => {
    const flowNodes = [
      {
        id: 'g1',
        type: 'canvas-group',
        position: { x: 0, y: 0 },
        measured: { width: 500, height: 400 },
        data: { type: 'group', label: 'G', id: 'g1', x: 0, y: 0, width: 500, height: 400 },
      },
      {
        id: 'n1',
        type: 'canvas-text',
        position: { x: 10, y: 10 },
        parentId: 'g1',
        measured: { width: 100, height: 100 },
        data: { type: 'text', text: 't', id: 'n1', x: 10, y: 10, width: 100, height: 100 },
      },
    ];
    const data = flowToCanvas(flowNodes, []);
    const json = JSON.stringify(data);
    expect(json).not.toContain('parentId');
  });

  it('should convert edges back to canvas format', () => {
    const flowEdges = [
      {
        id: 'e1',
        source: 'n1',
        target: 'n2',
        data: { fromNode: 'n1', toNode: 'n2', fromSide: 'right', toSide: 'left', toEnd: 'arrow' },
        label: 'connects',
      },
    ];
    const data = flowToCanvas([], flowEdges);
    expect(data.edges).toHaveLength(1);
    expect(data.edges[0].fromNode).toBe('n1');
    expect(data.edges[0].toNode).toBe('n2');
    expect(data.edges[0].label).toBe('connects');
  });
});

describe('roundtrip idempotence', () => {
  it('should preserve data through canvasToFlow -> flowToCanvas roundtrip', () => {
    const original: CanvasData = {
      nodes: [
        { id: 'n1', type: 'text', x: 100, y: 200, width: 300, height: 150, text: 'hello world' },
        { id: 'n2', type: 'file', x: 500, y: 200, width: 300, height: 200, file: 'My Note' },
        {
          id: 'n3',
          type: 'link',
          x: 100,
          y: 500,
          width: 300,
          height: 200,
          url: 'https://example.com',
        },
      ],
      edges: [{ id: 'e1', fromNode: 'n1', toNode: 'n2' }],
    };

    const { nodes, edges } = canvasToFlow(original);
    const result = flowToCanvas(nodes, edges);

    // All nodes should be preserved
    expect(result.nodes).toHaveLength(3);
    expect(result.edges).toHaveLength(1);

    // Check node IDs preserved
    const resultIds = result.nodes.map((n) => n.id).sort();
    expect(resultIds).toEqual(['n1', 'n2', 'n3']);

    // Check positions preserved
    const n1 = result.nodes.find((n) => n.id === 'n1')!;
    expect(n1.x).toBe(100);
    expect(n1.y).toBe(200);
    expect(n1.type).toBe('text');

    // Check edge preserved
    expect(result.edges[0].fromNode).toBe('n1');
    expect(result.edges[0].toNode).toBe('n2');
  });

  it('should preserve data through parse -> serialize -> parse roundtrip', () => {
    const original: CanvasData = {
      nodes: [{ id: 'n1', type: 'text', x: 10, y: 20, width: 100, height: 80, text: 'test' }],
      edges: [],
    };

    const json = serializeCanvas(original);
    const parsed = parseCanvas(json);
    expect(parsed).not.toBeNull();
    expect(parsed!.nodes).toHaveLength(1);
    expect(parsed!.nodes[0].id).toBe('n1');
    expect((parsed!.nodes[0] as { text: string }).text).toBe('test');
  });
});
