(function () {
  'use strict';

  function esc(value) {
    var node = document.createElement('div');
    node.textContent = value == null ? '' : String(value);
    return node.innerHTML.replace(/"/g, '&quot;').replace(/'/g, '&#39;');
  }

  function markdown(value, items) {
    var byId = {};
    (items || []).forEach(function (item) { byId[item.id] = item; });
    var text = esc(value || '');
    text = text.replace(/!\[([^\]]*)\]\(story-media:([a-zA-Z0-9_-]+)\)/g, function (match, alt, id) {
      var media = byId[id];
      if (!media || !media.url) return alt || '';
      if (media.contentType.indexOf('video/') === 0) return '<video src="' + esc(media.url) + '" controls preload="metadata"></video>';
      return '<a class="story-inline-image" href="' + esc(media.url) + '" target="_blank" rel="noopener noreferrer"><img src="' + esc(media.url) + '" alt="' + (alt || esc(media.name)) + '"></a>';
    }).replace(/^### (.+)$/gm, '<h3>$1</h3>')
      .replace(/^## (.+)$/gm, '<h2>$1</h2>')
      .replace(/^# (.+)$/gm, '<h2>$1</h2>')
      .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
      .replace(/\*(.+?)\*/g, '<em>$1</em>')
      .replace(/^[-*] (.+)$/gm, '<li>$1</li>')
      .replace(/(<li>.*<\/li>\n?)+/g, '<ul>$&</ul>')
      .replace(/^> ?(.+)$/gm, '<blockquote>$1</blockquote>');
    return text.split(/\n{2,}/).map(function (block) {
      return /^<(h2|h3|ul|blockquote)/.test(block) ? block : '<p>' + block.replace(/\n/g, '<br>') + '</p>';
    }).join('');
  }

  function mediaGallery(items) {
    return (items || []).filter(function (media) { return media.url; }).map(function (media) {
      var image = media.contentType.indexOf('image/') === 0;
      var preview = image ? '<img src="' + esc(media.url) + '" alt="' + esc(media.name) + '" loading="lazy">' : '<video src="' + esc(media.url) + '" controls preload="metadata"></video>';
      return '<figure class="story-media-card">' + (image ? '<a href="' + esc(media.url) + '" target="_blank" rel="noopener noreferrer">' + preview + '</a>' : preview) + '<figcaption>' + esc(media.name) + '</figcaption></figure>';
    }).join('');
  }

  window.ipaceStoryRenderer = { markdown: markdown, mediaGallery: mediaGallery };
})();
