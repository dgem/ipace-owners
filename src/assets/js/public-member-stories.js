(function () {
  'use strict';

  var root = document.querySelector('[data-public-stories]');
  if (!root) return;

  function esc(value) {
    var node = document.createElement('div');
    node.textContent = value == null ? '' : String(value);
    return node.innerHTML;
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
      return '<figure class="story-media-card"><a href="' + esc(media.url) + '" target="_blank" rel="noopener noreferrer">' + preview + '</a><figcaption>' + esc(media.name) + '</figcaption></figure>';
    }).join('');
  }

  function consent() {
    try { return localStorage.getItem('ipace_analytics_consent') === 'yes'; } catch { return false; }
  }

  function reader() {
    try {
      var id = localStorage.getItem('ipace_story_reader');
      if (!id) {
        id = crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + '-' + Math.random().toString(36).slice(2);
        localStorage.setItem('ipace_story_reader', id);
      }
      return id;
    } catch { return ''; }
  }

  function referrerPath() {
    try {
      var value = new URL(document.referrer);
      return value.origin + value.pathname;
    } catch { return ''; }
  }

  function track(story, event, platform) {
    if (!consent()) return;
    var readerID = reader();
    if (!readerID) return;
    fetch('/api/story-event', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      keepalive: true,
      body: JSON.stringify({
        storyId: story.id,
        event: event,
        platform: platform || '',
        readerId: readerID,
        campaign: new URLSearchParams(location.search).get('campaign') || '',
        path: location.pathname,
        referrer: referrerPath()
      })
    });
  }

  function shareURL() {
    var url = new URL(location.href);
    url.searchParams.delete('site-mode');
    return url.href;
  }

  function shareOptions(story) {
    var url = shareURL();
    var text = story.title + ' — an I-PACE owner story';
    var encodedURL = encodeURIComponent(url);
    var encodedText = encodeURIComponent(text);
    return '<section class="story-share-options" data-story-share-options hidden aria-label="Share this experience">' +
      '<p>Share this public experience. We record share link clicks only when you have allowed story analytics.</p>' +
      '<div class="cluster">' +
      '<a class="btn btn--secondary btn--sm" data-share-platform="facebook" href="https://www.facebook.com/sharer/sharer.php?u=' + encodedURL + '" target="_blank" rel="noopener noreferrer">Facebook</a>' +
      '<a class="btn btn--secondary btn--sm" data-share-platform="x" href="https://twitter.com/intent/tweet?text=' + encodedText + '&url=' + encodedURL + '" target="_blank" rel="noopener noreferrer">X</a>' +
      '<a class="btn btn--secondary btn--sm" data-share-platform="whatsapp" href="https://wa.me/?text=' + encodedText + '%20' + encodedURL + '" target="_blank" rel="noopener noreferrer">WhatsApp</a>' +
      '<a class="btn btn--secondary btn--sm" data-share-platform="linkedin" href="https://www.linkedin.com/sharing/share-offsite/?url=' + encodedURL + '" target="_blank" rel="noopener noreferrer">LinkedIn</a>' +
      '<button class="btn btn--secondary btn--sm" data-copy-share="instagram" type="button">Copy for Instagram</button>' +
      '<button class="btn btn--secondary btn--sm" data-copy-share="tiktok" type="button">Copy for TikTok</button>' +
      '<button class="btn btn--secondary btn--sm" data-copy-share="copy" type="button">Copy link</button>' +
      '<button class="btn btn--secondary btn--sm" data-native-share type="button">More sharing options</button>' +
      '</div><p class="form-hint" data-share-status role="status" aria-live="polite"></p></section>';
  }

  function copyText(value) {
    if (navigator.clipboard && navigator.clipboard.writeText) return navigator.clipboard.writeText(value);
    return new Promise(function (resolve, reject) {
      var input = document.createElement('textarea');
      input.value = value;
      input.setAttribute('readonly', '');
      input.style.position = 'fixed';
      input.style.opacity = '0';
      document.body.appendChild(input);
      input.select();
      var copied = document.execCommand('copy');
      input.remove();
      copied ? resolve() : reject(new Error('Copy failed'));
    });
  }

  function bindSharing(story) {
    var toggle = root.querySelector('[data-story-share]');
    var panel = root.querySelector('[data-story-share-options]');
    toggle.addEventListener('click', function () {
      panel.hidden = !panel.hidden;
      toggle.setAttribute('aria-expanded', String(!panel.hidden));
      if (!panel.hidden) panel.querySelector('a,button').focus();
    });
    panel.querySelectorAll('[data-share-platform]').forEach(function (link) {
      link.addEventListener('click', function () { track(story, 'share', link.dataset.sharePlatform); });
    });
    panel.querySelectorAll('[data-copy-share]').forEach(function (button) {
      button.addEventListener('click', function () {
        var platform = button.dataset.copyShare;
        copyText(shareURL()).then(function () {
          track(story, 'share', platform);
          panel.querySelector('[data-share-status]').textContent = platform === 'copy'
            ? 'Story link copied.'
            : 'Story link copied. Open ' + (platform === 'instagram' ? 'Instagram' : 'TikTok') + ' and paste it into your post, profile or message.';
        }).catch(function () { panel.querySelector('[data-share-status]').textContent = 'Could not copy the link. Select and copy the page address instead.'; });
      });
    });
    var nativeButton = panel.querySelector('[data-native-share]');
    nativeButton.addEventListener('click', function () {
      if (!navigator.share) {
        panel.querySelector('[data-share-status]').textContent = 'Use one of the sharing buttons or copy the link.';
        return;
      }
      track(story, 'share', 'native');
      navigator.share({ title: story.title, text: 'An I-PACE owner story', url: shareURL() }).catch(function () {});
    });
  }

  fetch('/api/stories').then(function (response) { return response.json(); }).then(function (data) {
    var params = new URLSearchParams(location.search);
    var pathParts = location.pathname.split('/').filter(Boolean);
    var requested = params.get('story') || ((pathParts[0] === 'member-stories' && pathParts[1]) ? decodeURIComponent(pathParts[1]) : '');
    var campaign = params.get('campaign') || '';
    var stories = data.stories || [];
    var selected = stories.find(function (story) { return story.slug === requested || story.id === requested; });
    if (selected) {
      var joinLink = '/join/?story=' + encodeURIComponent(selected.id) + (campaign ? '&campaign=' + encodeURIComponent(campaign) : '');
      root.innerHTML = '<article class="member-story"><p class="eyebrow">Owners’ story</p><h2>' + esc(selected.title) + '</h2><p class="lede">By ' + esc(selected.displayName) + '</p><div class="story-markdown">' + markdown(selected.bodyMarkdown, selected.media) + '</div><section class="story-media-gallery" aria-label="Attached media">' + mediaGallery(selected.media) + '</section><aside class="story-evidence-summary"><h2>Evidence behind this story</h2><p>' + esc(selected.evidence.vehicleLabel || 'Registered I-PACE') + (selected.evidence.modelYear ? ' · model year ' + esc(selected.evidence.modelYear) : '') + '</p><p>' + Number(selected.evidence.serviceEventCount || 0) + ' linked service, fault or JLR contact records · ' + Number(selected.evidence.sohReadingCount || 0) + ' linked SoH readings.</p><p><a href="/member/dashboard/">Add your own vehicle evidence</a></p></aside><div class="cluster"><a class="btn btn--primary" data-story-cta href="' + joinLink + '">Add your voice</a><button class="btn btn--secondary" data-story-share type="button" aria-expanded="false">Share this story</button></div>' + shareOptions(selected) + '</article>';
      track(selected, 'view');
      setTimeout(function () { track(selected, 'engaged'); }, 30000);
      root.querySelector('[data-story-cta]').addEventListener('click', function () { track(selected, 'cta'); });
      bindSharing(selected);
      return;
    }
    root.innerHTML = '<div class="story-grid">' + stories.map(function (story) {
      var href = '/member-stories/' + encodeURIComponent(story.slug || story.id) + '/';
      return '<article class="card"><h2><a href="' + href + '">' + esc(story.title) + '</a></h2><p>By ' + esc(story.displayName) + '</p><p>' + esc(story.bodyMarkdown.slice(0, 220)) + '…</p><a href="' + href + '">Read this story</a></article>';
    }).join('') + '</div>';
  }).catch(function () { root.innerHTML = '<p>Owners’ stories are temporarily unavailable.</p>'; });
})();
