/**
 * outreach-assistant.js — Human-controlled Facebook search-link and reply helper.
 * This module deliberately performs no network requests and no Facebook automation.
 */
(function () {
  'use strict';

  var suggestedQueries = [
    '"traction battery fault"',
    '"battery fault" I-PACE',
    '"restricted performance" I-PACE',
    '"HV battery" I-PACE',
    '"battery module" I-PACE',
    '"state of health" I-PACE',
    '"reduced range" I-PACE',
    'I-PACE battery recall',
    'I-PACE H484',
    'I-PACE H570',
  ];

  var issueGuidance = {
    'traction-fault': 'It may help to photograph the exact warning, note the mileage and state of charge, and ask for a written copy of any diagnostic trouble codes and the proposed repair before work begins.',
    'restricted-performance': 'It may help to record when the restriction appears, ambient temperature, state of charge, displayed range, mileage and any dashboard warnings, then ask the workshop for the diagnostic codes in writing.',
    'module-repair': 'Before authorising work, consider asking which module or cells were identified, how the diagnosis was made, whether the repair is covered by warranty or a campaign, and what post-repair battery test will be supplied.',
    'state-of-health': 'State-of-health figures can vary by tool and conditions, so it is useful to record the source, date, mileage, state of charge and any accompanying report rather than relying on the percentage alone.',
    recall: 'Campaign eligibility and remedy can depend on the VIN and market. The safest next step is to check with Jaguar or an authorised repairer and ask them to confirm the applicable campaign and proposed remedy in writing.',
    general: 'It may help to keep the exact warning text, dates, mileage, state of charge, diagnostic codes and written workshop response together so the history is clear.',
  };

  function uniqueLines(value) {
    var seen = {};
    return String(value || '').split(/\r?\n/).map(function (line) {
      return line.trim();
    }).filter(function (line) {
      var key = line.toLowerCase();
      if (!line || seen[key]) return false;
      seen[key] = true;
      return true;
    });
  }

  function normaliseGroupUrl(value) {
    try {
      var url = new URL(value);
      var hostname = url.hostname.toLowerCase();
      if (url.protocol !== 'https:' || !['facebook.com', 'www.facebook.com', 'm.facebook.com'].includes(hostname)) return '';
      var parts = url.pathname.split('/').filter(Boolean);
      if (parts.length < 2 || parts[0].toLowerCase() !== 'groups') return '';
      if (!/^[a-zA-Z0-9._-]+$/.test(parts[1])) return '';
      return 'https://www.facebook.com/groups/' + parts[1];
    } catch {
      return '';
    }
  }

  function buildSearchLinks(groupText, queryText, includeGlobal) {
    var rawGroups = uniqueLines(groupText);
    var groups = rawGroups.map(normaliseGroupUrl).filter(Boolean);
    var queries = uniqueLines(queryText).slice(0, 50);
    var links = [];

    queries.forEach(function (query) {
      groups.forEach(function (groupUrl) {
        links.push({
          scope: groupUrl.replace('https://www.facebook.com/groups/', ''),
          query: query,
          url: groupUrl + '/search/?q=' + encodeURIComponent(query),
        });
      });
      if (includeGlobal) {
        links.push({
          scope: 'All Facebook posts',
          query: query,
          url: 'https://www.facebook.com/search/posts/?q=' + encodeURIComponent(query),
        });
      }
    });

    return { links: links, invalidGroupCount: rawGroups.length - groups.length };
  }

  function draftReply(issue, includeInvitation) {
    var guidance = issueGuidance[issue] || issueGuidance.general;
    var draft = 'Sorry you are dealing with this. ' + guidance;
    if (includeInvitation) {
      draft += '\n\nI volunteer with the I-PACE Owners\' Advocacy Group. We are collecting owner-submitted evidence about battery health, faults and repair outcomes at https://ipace-owners.org/. Joining is optional, and detailed evidence remains private unless the owner consents to anonymised analysis.';
    }
    draft += '\n\nThis is general owner-to-owner information rather than a diagnosis or official Jaguar advice.';
    return draft;
  }

  function copyText(value) {
    if (!navigator.clipboard || !navigator.clipboard.writeText) return Promise.reject(new Error('Clipboard unavailable'));
    return navigator.clipboard.writeText(value);
  }

  function makeResultCard(item) {
    var article = document.createElement('article');
    article.className = 'outreach-result-card';
    var content = document.createElement('div');
    var scope = document.createElement('p');
    scope.className = 'outreach-result-card__scope';
    scope.textContent = item.scope;
    var query = document.createElement('p');
    query.className = 'outreach-result-card__query';
    query.textContent = item.query;
    content.appendChild(scope);
    content.appendChild(query);
    var link = document.createElement('a');
    link.className = 'btn btn--secondary btn--sm';
    link.href = item.url;
    link.target = '_blank';
    link.rel = 'noopener noreferrer';
    link.textContent = 'Open search';
    link.setAttribute('aria-label', 'Open Facebook search for ' + item.query + ' in ' + item.scope);
    article.appendChild(content);
    article.appendChild(link);
    return article;
  }

  function initialise() {
    var root = document.querySelector('[data-outreach-assistant]');
    if (!root) return;
    var form = root.querySelector('[data-outreach-search-form]');
    var groups = root.querySelector('[data-outreach-groups]');
    var queries = root.querySelector('[data-outreach-queries]');
    var globalSearch = root.querySelector('[data-outreach-global]');
    var results = root.querySelector('[data-outreach-results]');
    var reset = root.querySelector('[data-outreach-reset]');
    var issue = root.querySelector('[data-outreach-issue]');
    var invitation = root.querySelector('[data-outreach-invitation]');
    var draft = root.querySelector('[data-outreach-draft]');
    var refreshDraft = root.querySelector('[data-outreach-refresh-draft]');
    var copyDraft = root.querySelector('[data-outreach-copy-draft]');
    var copyStatus = root.querySelector('[data-outreach-copy-status]');
    queries.value = suggestedQueries.join('\n');

    function renderLinks() {
      var generated = buildSearchLinks(groups.value, queries.value, globalSearch.checked);
      results.replaceChildren();
      var summary = document.createElement('p');
      summary.className = 'outreach-results__summary';
      if (!generated.links.length) {
        summary.textContent = 'Add at least one search phrase and select Facebook-wide search or enter a valid group URL.';
        results.appendChild(summary);
        return;
      }
      summary.textContent = generated.links.length + ' search link' + (generated.links.length === 1 ? '' : 's') + ' generated.';
      if (generated.invalidGroupCount) summary.textContent += ' ' + generated.invalidGroupCount + ' invalid group URL' + (generated.invalidGroupCount === 1 ? ' was' : 's were') + ' ignored.';
      results.appendChild(summary);
      var grid = document.createElement('div');
      grid.className = 'outreach-results__grid';
      generated.links.forEach(function (item) { grid.appendChild(makeResultCard(item)); });
      results.appendChild(grid);
    }

    function renderDraft() {
      draft.value = draftReply(issue.value, invitation.checked);
      copyStatus.textContent = '';
    }

    form.addEventListener('submit', function (event) { event.preventDefault(); renderLinks(); });
    reset.addEventListener('click', function () { queries.value = suggestedQueries.join('\n'); renderLinks(); });
    refreshDraft.addEventListener('click', renderDraft);
    issue.addEventListener('change', renderDraft);
    invitation.addEventListener('change', renderDraft);
    copyDraft.addEventListener('click', function () {
      copyText(draft.value).then(function () {
        copyStatus.textContent = 'Reply copied.';
      }).catch(function () {
        copyStatus.textContent = 'Copy failed. Select the text and copy it manually.';
        draft.focus();
        draft.select();
      });
    });
    renderLinks();
    renderDraft();
  }

  window.ipaceOutreachAssistant = {
    buildSearchLinks: buildSearchLinks,
    draftReply: draftReply,
    normaliseGroupUrl: normaliseGroupUrl,
    suggestedQueries: suggestedQueries.slice(),
  };
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', initialise);
  else initialise();
})();
