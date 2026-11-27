import type { Locator } from '@playwright/test';

export function prepareStoryExample(example: string) {
  const placements: Array<'image' | 'video'> = [];
  const body = example.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_match, _alt, file) => {
    const index = placements.length;
    placements.push(/\.(mov|mp4|webm)$/i.test(file) ? 'video' : 'image');
    return 'E2EMEDIAPOSITION' + index + 'END';
  });
  if (placements.length !== 3 || placements.join(',') !== 'image,video,image') throw new Error('The staging story example must contain two image positions and one video position.');
  return { body, placements };
}

export async function insertStoryExampleMedia(root: Locator, placements: Array<'image' | 'video'>) {
  const editor = root.locator('[data-story-editor]');
  for (let index = 0; index < placements.length; index++) {
    await editor.evaluate((element, token) => {
      const walker = document.createTreeWalker(element, NodeFilter.SHOW_TEXT);
      let node: Node | null;
      while ((node = walker.nextNode())) {
        const offset = (node.nodeValue || '').indexOf(token);
        if (offset < 0) continue;
        const range = document.createRange();
        range.setStart(node, offset); range.setEnd(node, offset + token.length);
        const selection = window.getSelection();
        selection?.removeAllRanges(); selection?.addRange(range);
        document.dispatchEvent(new Event('selectionchange'));
        return;
      }
      throw new Error('A staging example media position was not found in the editor.');
    }, 'E2EMEDIAPOSITION' + index + 'END');
    await root.locator('[data-insert-pending-media="' + (placements[index] === 'video' ? 1 : 0) + '"]').click();
  }
}
