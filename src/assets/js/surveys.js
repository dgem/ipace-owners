(function () {
  'use strict';
  function token() { var u = window.firebase && window.firebase.auth().currentUser; return u ? u.getIdToken() : Promise.reject(new Error('Sign in first.')); }
  function request(path, method, body) { return token().then(function (t) { return fetch(path, { method: method, headers: { 'Authorization': 'Bearer ' + t, 'Content-Type': 'application/json' }, body: body ? JSON.stringify(body) : undefined }).then(function (r) { if (r.status === 204) return {}; return r.json().then(function (d) { if (!r.ok) throw new Error(d.error || 'Request failed.'); return d; }); }); }); }
  function esc(v) { var d = document.createElement('div'); d.textContent = v || ''; return d.innerHTML; }
  function inlineMarkdown(v) { var html = esc(v); html = html.replace(/\[([^\]]+)\]\((https?:\/\/[A-Za-z0-9._~:/?#@!$&'()*+,;=%-]+)\)/g, '<a href="$2" rel="noopener noreferrer" target="_blank">$1</a>'); html = html.replace(/\*\*([^*\n]+)\*\*/g, '<strong>$1</strong>'); return html.replace(/\*([^*\n]+)\*/g, '<em>$1</em>'); }
  function markdown(v) { var html = '', list = false; String(v || '').replace(/\r\n?/g, '\n').split('\n').forEach(function (line) { var item = line.match(/^\s*[-*]\s+(.+)$/); if (item) { if (!list) { html += '<ul>'; list = true; } html += '<li>' + inlineMarkdown(item[1]) + '</li>'; return; } if (list) { html += '</ul>'; list = false; } if (line.trim()) html += '<p>' + inlineMarkdown(line) + '</p>'; }); return html + (list ? '</ul>' : ''); }
  function dateInputValue(date) { return date.toISOString().slice(0, 10); }

  function setupAdmin(root) {
    var form = root.querySelector('[data-survey-editor]'), options = root.querySelector('[data-survey-options]'), status = root.querySelector('[data-survey-admin-status]'), list = root.querySelector('[data-survey-admin-list]');
    function add(o) { var row = document.createElement('div'); row.className = 'survey-option-editor'; row.innerHTML = '<input data-option-id type="hidden" value="' + esc(o && o.id) + '"><label class="form-label">Option<textarea class="survey-option-editor__input" data-option-label aria-label="Option label" maxlength="2000" required rows="4" placeholder="Option text. Markdown supported.">' + esc(o && o.label) + '</textarea></label><div class="cluster"><label><input data-option-text type="checkbox" ' + (o && o.allowsText ? 'checked' : '') + '> Ask for a short explanation</label><button class="btn btn--secondary btn--sm" type="button">Remove</button></div>'; row.querySelector('button').onclick = function () { row.remove(); }; options.appendChild(row); }
    function clear() { var start = new Date(), end = new Date(start); end.setDate(end.getDate() + 6); form.reset(); form.querySelector('[data-survey-id]').value = ''; form.querySelector('[data-survey-start]').value = dateInputValue(start); form.querySelector('[data-survey-end]').value = dateInputValue(end); options.innerHTML = ''; add(); add(); status.textContent = ''; }
    function edit(s) { form.querySelector('[data-survey-id]').value = s.id; form.querySelector('[data-survey-title]').value = s.title; form.querySelector('[data-survey-description]').value = s.description || ''; form.querySelector('[data-survey-question]').value = s.question || ''; form.querySelector('[data-survey-cta]').value = s.callToAction || ''; form.querySelector('[data-survey-start]').value = s.startsOn; form.querySelector('[data-survey-end]').value = s.endsOn; form.querySelector('[data-survey-multiple]').checked = s.multiple; form.querySelector('[data-survey-results]').checked = s.showResults; options.innerHTML = ''; s.options.forEach(add); window.scrollTo({ top: 0, behavior: 'smooth' }); }
    function render(surveys) { list.innerHTML = ''; if (!surveys.length) { list.textContent = 'No surveys yet.'; return; } surveys.forEach(function (s) { var item = document.createElement('article'); item.className = 'card'; item.innerHTML = '<h3>' + esc(s.title) + '</h3><p>' + esc(s.startsOn) + ' to ' + esc(s.endsOn) + ' · ' + (s.multiple ? 'Multiple choice' : 'Single choice') + '</p><div class="cluster"><button class="btn btn--secondary btn--sm" type="button">Edit</button><button class="btn btn--danger btn--sm" type="button">Delete</button></div>'; var buttons = item.querySelectorAll('button'); buttons[0].onclick = function () { edit(s); }; buttons[1].onclick = function () { if (confirm('Delete this survey? Existing responses will no longer be visible.')) request('/api/admin/surveys?id=' + encodeURIComponent(s.id), 'DELETE').then(load).catch(function (e) { status.textContent = e.message; }); }; list.appendChild(item); }); }
    var started = false;
    function load() { request('/api/admin/surveys', 'GET').then(function (d) { render(d.surveys); }).catch(function (e) { list.textContent = e.message; }); }
    function start() { if (started) return; started = true; load(); window.setInterval(load, 30000); window.addEventListener('focus', load); }
    root.querySelector('[data-survey-add-option]').onclick = function () { add(); }; root.querySelector('[data-survey-new]').onclick = clear; root.querySelector('[data-survey-refresh]').onclick = load;
    form.onsubmit = function (e) { e.preventDefault(); var payload = { id: form.querySelector('[data-survey-id]').value, title: form.querySelector('[data-survey-title]').value, description: form.querySelector('[data-survey-description]').value, question: form.querySelector('[data-survey-question]').value, callToAction: form.querySelector('[data-survey-cta]').value, startsOn: form.querySelector('[data-survey-start]').value, endsOn: form.querySelector('[data-survey-end]').value, multiple: form.querySelector('[data-survey-multiple]').checked, showResults: form.querySelector('[data-survey-results]').checked, options: Array.prototype.map.call(options.children, function (x) { return { id: x.querySelector('[data-option-id]').value, label: x.querySelector('[data-option-label]').value, allowsText: x.querySelector('[data-option-text]').checked }; }) }; request('/api/admin/surveys', payload.id ? 'PUT' : 'POST', payload).then(function () { status.textContent = 'Survey saved.'; clear(); load(); }).catch(function (err) { status.textContent = err.message; }); }; clear(); document.addEventListener('admin:data', start);
  }
  function setupMember(root) {
    function render(all) {
      var requestedID = new URLSearchParams(window.location.search).get('id');
      var active = all.filter(function (result) { return result.canRespond && (!requestedID || result.survey.id === requestedID); });
      root.innerHTML = '';
      if (!active.length) { root.innerHTML = '<section class="dashboard-panel"><h2 class="dashboard-panel__title">No open surveys at the moment</h2><p>We will let you know when there is another opportunity to share your view.</p><a class="btn btn--secondary" href="/member/survey-history/">View past surveys</a></section>'; return; }
      active.forEach(function (result) {
        var s = result.survey, article = document.createElement('article');
        article.className = 'dashboard-panel survey-member-card';
        article.innerHTML = '<div class="survey-member-card__intro"><p class="dashboard-panel__eyebrow">Member survey</p><h2 class="survey-member-card__title">' + esc(s.title) + '</h2><div class="survey-markdown survey-member-card__copy">' + markdown(s.description) + markdown(s.question) + markdown(s.callToAction) + '</div><p class="survey-member-card__status">Open now · ' + esc(s.startsOn) + ' to ' + esc(s.endsOn) + '</p></div><form class="survey-options"></form><p role="status" aria-live="polite"></p>';
        var form = article.querySelector('form');
        form.insertAdjacentHTML('beforeend', '<fieldset><legend>' + (s.multiple ? 'Select every outcome you would support' : 'Select the outcome you would support') + '</legend><p class="form-hint">' + (s.multiple ? 'You can choose more than one option.' : 'Choose one option.') + '</p><div data-survey-choice-list></div></fieldset>');
        var choices = form.querySelector('[data-survey-choice-list]');
        s.options.forEach(function (o, index) {
          var selected = (result.myOptionIds || []).indexOf(o.id) > -1, text = result.myTextByOption && result.myTextByOption[o.id] || '';
          choices.insertAdjacentHTML('beforeend', '<div class="survey-choice"><label><input type="' + (s.multiple ? 'checkbox' : 'radio') + '" name="survey" value="' + esc(o.id) + '" ' + (selected ? 'checked' : '') + '><span class="survey-choice__number" aria-hidden="true">' + (index + 1) + '</span><span class="survey-markdown survey-choice__copy">' + markdown(o.label) + '</span></label>' + (o.allowsText ? '<div class="survey-choice__text" ' + (selected ? '' : 'hidden') + '><label for="survey-text-' + esc(o.id) + '">Tell us more <span>(up to 250 characters)</span></label><textarea id="survey-text-' + esc(o.id) + '" data-option-text-id="' + esc(o.id) + '" maxlength="250" rows="3" placeholder="Please describe">' + esc(text) + '</textarea></div>' : '') + '</div>');
        });
        form.insertAdjacentHTML('beforeend', '<button class="btn btn--primary survey-options__submit" type="submit">' + ((result.myOptionIds || []).length ? 'Update your response' : 'Submit your response') + '</button>');
        form.onchange = function () { Array.prototype.forEach.call(form.querySelectorAll('[data-option-text-id]'), function (field) { field.closest('.survey-choice__text').hidden = !form.querySelector('input[value="' + field.getAttribute('data-option-text-id') + '"]').checked; }); };
        form.onsubmit = function (e) { e.preventDefault(); var selected = Array.prototype.map.call(form.querySelectorAll('input:checked'), function (input) { return input.value; }), textByOption = {}; selected.forEach(function (id) { var field = form.querySelector('[data-option-text-id="' + id + '"]'); if (field) textByOption[id] = field.value; }); request('/api/member/survey-response', 'POST', { surveyId: s.id, optionIds: selected, textByOption: textByOption }).then(function () { window.location.assign('/member/survey-results/?id=' + encodeURIComponent(s.id)); }).catch(function (err) { article.querySelector('[role=status]').textContent = err.message; }); };
        root.appendChild(article);
      });
    }
    var started = false;
    function load() { request('/api/member/surveys', 'GET').then(function (d) { render(d.surveys); }).catch(function (e) { root.textContent = e.message; }); }
    function start() { if (started) return; started = true; load(); window.setInterval(load, 60000); window.addEventListener('focus', load); }
    document.addEventListener('member:data', start);
  }
  function resultMarkup(result) {
    var s = result.survey;
    if (!result.myOptionIds || !result.myOptionIds.length) return '<section class="dashboard-panel"><h2 class="dashboard-panel__title">Submit your response first</h2><p>Aggregate results become available after you have submitted your own response.</p><a class="btn btn--primary" href="/member/surveys/">Take the survey</a></section>';
    if (!s.showResults) return '<section class="dashboard-panel"><h2 class="dashboard-panel__title">Results are not published for this survey</h2><p>Your response has been saved.</p><a class="btn btn--secondary" href="/member/surveys/?id=' + encodeURIComponent(s.id) + '">Edit your response</a></section>';
    return '<article class="dashboard-panel survey-member-card"><p class="dashboard-panel__eyebrow">Response saved</p><h2 class="survey-member-card__title">' + esc(s.title) + '</h2><p class="survey-member-card__status">Thank you — your response is included below.</p><div class="survey-results"><div class="survey-results__header"><h3>Current results</h3><span>' + result.total + ' response' + (result.total === 1 ? '' : 's') + '</span></div>' + s.options.map(function (o, index) { return '<div class="survey-result"><span class="survey-choice__number" aria-hidden="true">' + (index + 1) + '</span><div class="survey-markdown">' + markdown(o.label) + '</div><strong>' + ((result.counts && result.counts[o.id]) || 0) + '</strong></div>'; }).join('') + '</div><div class="cluster survey-results__actions"><a class="btn btn--secondary" href="/member/surveys/?id=' + encodeURIComponent(s.id) + '">Edit your response</a><a class="btn btn--secondary" href="/member/survey-history/">Past surveys</a></div></article>';
  }
  function setupResults(root) {
    var started = false;
    function load() {
      var id = new URLSearchParams(window.location.search).get('id');
      request('/api/member/surveys', 'GET').then(function (data) {
        var result = data.surveys.filter(function (item) { return item.survey.id === id; })[0];
        root.innerHTML = result ? resultMarkup(result) : '<section class="dashboard-panel"><h2 class="dashboard-panel__title">Survey not found</h2><a class="btn btn--primary" href="/member/surveys/">View open surveys</a></section>';
      }).catch(function (error) { root.textContent = error.message; });
    }
    function start() { if (started) return; started = true; load(); }
    document.addEventListener('member:data', start);
  }
  function setupHistory(root) {
    var started = false;
    function load() { request('/api/member/surveys', 'GET').then(function (data) { var closed = data.surveys.filter(function (result) { return !result.canRespond; }); root.innerHTML = closed.length ? closed.map(function (result) { return '<article class="dashboard-panel survey-history-item"><h2 class="dashboard-panel__title">' + esc(result.survey.title) + '</h2><p>Closed ' + esc(result.survey.endsOn) + '</p>' + ((result.myOptionIds || []).length && result.survey.showResults ? '<a class="btn btn--secondary" href="/member/survey-results/?id=' + encodeURIComponent(result.survey.id) + '">View results</a>' : '<p class="form-hint">Results are available only to members who responded.</p>') + '</article>'; }).join('') : '<section class="dashboard-panel"><h2 class="dashboard-panel__title">No past surveys yet</h2><p>Closed surveys will appear here.</p></section>'; }).catch(function (error) { root.textContent = error.message; }); }
    function start() { if (started) return; started = true; load(); }
    document.addEventListener('member:data', start);
  }
  function setupMemberDashboard(root) {
    var started = false;
    function render(surveys) {
      var active = surveys.filter(function (result) { return result.canRespond; }), closed = surveys.filter(function (result) { return !result.canRespond; }), historyAction = closed.length ? '<a class="btn btn--secondary" href="/member/survey-history/">Past surveys</a>' : '';
      if (!active.length) { root.innerHTML = '<h2 class="survey-dashboard-callout__title">Member surveys</h2><p>There are no open surveys right now.</p>' + historyAction; return; }
      var names = active.map(function (result) { return '<li>' + esc(result.survey.title) + '</li>'; }).join('');
      root.innerHTML = '<p class="dashboard-panel__eyebrow">Your voice matters</p><h2 class="survey-dashboard-callout__title">Help steer our discussions with JLR</h2><p>There ' + (active.length === 1 ? 'is an open member survey' : 'are ' + active.length + ' open member surveys') + ' waiting for your response.</p><ul>' + names + '</ul><div class="cluster"><a class="btn btn--primary" href="/member/surveys/">Take the survey' + (active.length === 1 ? '' : 's') + '</a>' + historyAction + '</div>';
    }
    function load() { request('/api/member/surveys', 'GET').then(function (data) { render(data.surveys); }).catch(function () { root.textContent = 'Member surveys are currently unavailable.'; }); }
    function start() { if (started) return; started = true; load(); window.setInterval(load, 60000); window.addEventListener('focus', load); }
    document.addEventListener('member:data', start);
  }
  document.querySelectorAll('[data-survey-admin]').forEach(setupAdmin); document.querySelectorAll('[data-member-surveys]').forEach(setupMember);
  document.querySelectorAll('[data-member-survey-results]').forEach(setupResults);
  document.querySelectorAll('[data-member-survey-history]').forEach(setupHistory);
  document.querySelectorAll('[data-member-survey-summary]').forEach(setupMemberDashboard);
}());
