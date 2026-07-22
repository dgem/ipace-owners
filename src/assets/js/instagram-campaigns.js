(function () {
  'use strict';

  async function request(path, body) {
    var user = window.firebase && window.firebase.auth().currentUser;
    if (!user) throw new Error('Sign in as an administrator first.');
    var token = await user.getIdToken();
    var response = await fetch(path, {
      method: 'POST',
      headers: { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    var data = await response.json();
    if (!response.ok) throw new Error(data.error || 'The Instagram campaign request failed.');
    return data;
  }

  function initialise(root) {
    var draftForm = root.querySelector('[data-instagram-draft]');
    var publishForm = root.querySelector('[data-instagram-publish]');
    var mediaPath = root.querySelector('[data-instagram-media-path]');
    var caption = root.querySelector('[data-instagram-caption]');
    var mediaReviewed = root.querySelector('[data-instagram-media-reviewed]');
    var count = root.querySelector('[data-instagram-caption-count]');
    var previewButton = root.querySelector('[data-instagram-preview-button]');
    var preview = root.querySelector('[data-instagram-preview]');
    var previewVideo = root.querySelector('[data-instagram-video]');
    var previewCaption = root.querySelector('[data-instagram-caption-preview]');
    var campaignID = root.querySelector('[data-instagram-campaign-id]');
    var confirm = root.querySelector('[data-instagram-confirm]');
    var confirmHint = root.querySelector('[data-instagram-confirm-hint]');
    var publishButton = root.querySelector('[data-instagram-publish-button]');
    var status = root.querySelector('[data-instagram-status]');
    var current = null;

    function draft() { return { mediaPath: mediaPath.value, caption: caption.value, mediaReviewed: mediaReviewed.checked }; }
    function invalidate() {
      current = null;
      preview.hidden = true;
      confirm.value = '';
      confirm.disabled = true;
      publishButton.disabled = true;
      confirmHint.textContent = 'Validate the draft first.';
    }
    function updateCount() { count.textContent = caption.value.length; }
    caption.addEventListener('input', function () { updateCount(); invalidate(); });
    mediaPath.addEventListener('input', invalidate);
    mediaReviewed.addEventListener('change', invalidate);
    updateCount();

    draftForm.addEventListener('submit', async function (event) {
      event.preventDefault();
      previewButton.disabled = true;
      status.textContent = 'Validating the draft…';
      try {
        current = await request(root.getAttribute('data-preview-endpoint'), draft());
        previewVideo.src = current.mediaPath;
        previewCaption.textContent = current.caption;
        campaignID.textContent = current.campaignId;
        preview.hidden = false;
        confirm.disabled = false;
        publishButton.disabled = false;
        confirmHint.textContent = 'Type “' + current.confirmation + '” exactly.';
        status.textContent = current.configured ? 'Preview complete. Nothing has been published.' : 'Preview complete, but Instagram publishing is not configured on this environment.';
      } catch (error) {
        invalidate();
        status.textContent = error.message;
      }
      previewButton.disabled = false;
    });

    publishForm.addEventListener('submit', async function (event) {
      event.preventDefault();
      if (!current) return;
      publishButton.disabled = true;
      status.textContent = 'Uploading, processing and publishing the Reel…';
      try {
        var result = await request(root.getAttribute('data-publish-endpoint'), {
          campaignId: current.campaignId,
          mediaPath: current.mediaPath,
          caption: current.caption,
          mediaReviewed: true,
          confirmation: confirm.value
        });
        confirm.value = '';
        confirm.disabled = true;
        status.textContent = 'Published successfully as Instagram media ' + result.mediaId + '.';
      } catch (error) {
        status.textContent = error.message;
        publishButton.disabled = false;
      }
    });
  }

  document.querySelectorAll('[data-instagram-campaign]').forEach(initialise);
}());
