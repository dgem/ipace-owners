(function () {
  "use strict";

  var root = document.querySelector("[data-survey-insights]");
  if (!root) return;
  var runButton = root.querySelector("[data-insights-run]");
  var status = root.querySelector("[data-insights-status]");
  var output = root.querySelector("[data-insights-report]");
  var saved = root.querySelector("[data-insights-saved]");
  var surveyID = new URLSearchParams(window.location.search).get("id");
  var active = false;
  var pending = null;
  var currentAnalysisID = null;
  var currentReport = null;
  var currentStats = null;
  var savedDecks = {};

  function escapeHTML(value) {
    var element = document.createElement("span");
    element.textContent = String(value == null ? "" : value);
    return element.innerHTML;
  }

  function request(path, method, body) {
    var user = window.firebase && window.firebase.auth().currentUser;
    if (!user) return Promise.reject(new Error("Sign in as an administrator first."));
    return user.getIdToken().then(function (token) {
      return fetch(path, {
        method: method,
        headers: window.ipaceAuthHeaders ? window.ipaceAuthHeaders({ Authorization: "Bearer " + token, "Content-Type": "application/json" }) : { Authorization: "Bearer " + token, "Content-Type": "application/json" },
        body: body ? JSON.stringify(body) : undefined
      }).then(function (response) {
        return response.json().catch(function () { return {}; }).then(function (data) {
          if (!response.ok) {
            var error = new Error(data.error || "The analysis request failed (HTTP " + response.status + ").");
            error.status = response.status;
            throw error;
          }
          return data;
        });
      });
    });
  }

  function requestBlob(path) {
    var user = window.firebase && window.firebase.auth().currentUser;
    if (!user) return Promise.reject(new Error("Sign in as an administrator first."));
    return user.getIdToken().then(function (token) {
      return fetch(path, { headers: window.ipaceAuthHeaders ? window.ipaceAuthHeaders({ Authorization: "Bearer " + token }) : { Authorization: "Bearer " + token } }).then(function (response) {
        if (!response.ok) throw new Error("Could not download the saved PowerPoint.");
        return response.blob();
      });
    });
  }

  function downloadBlob(blob, name) {
    var url = window.URL.createObjectURL(blob);
    var link = document.createElement("a");
    link.href = url;
    link.download = name;
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.setTimeout(function () { window.URL.revokeObjectURL(url); }, 60000);
  }

  function dateLabel(value) {
    var date = new Date(value);
    return isNaN(date.getTime()) ? "Saved analysis" : date.toLocaleString("en-GB", { dateStyle: "medium", timeStyle: "short" });
  }

  function loadArchive() {
    return request("/api/admin/survey-insight-archive?surveyId=" + encodeURIComponent(surveyID), "GET").then(function (data) {
      var analyses = data.analyses || [];
      if (analyses.length && !pending) runButton.textContent = "Re-run AI on latest responses";
      savedDecks = {};
      saved.innerHTML = analyses.length ? analyses.map(function (analysis) {
        var decks = (analysis.decks || []).map(function (deck) {
          savedDecks[deck.id] = deck;
          var quoteCount = (deck.quoteIndexes || []).length;
          return '<li><span>PowerPoint ' + escapeHTML(dateLabel(deck.createdAt)) + ' · ' + quoteCount + (quoteCount === 1 ? ' quote' : ' quotes') + '</span> <button class="btn btn--secondary" type="button" data-saved-deck="' + escapeHTML(deck.id) + '" data-analysis-id="' + escapeHTML(analysis.id) + '">Download</button> <button class="btn btn--secondary" type="button" data-edit-deck="' + escapeHTML(deck.id) + '" data-analysis-id="' + escapeHTML(analysis.id) + '">Edit selection</button></li>';
        }).join("");
        return '<article class="survey-insights__saved"><div><strong>' + escapeHTML(dateLabel(analysis.createdAt)) + '</strong><span>' + Number(analysis.responseCount || 0).toLocaleString("en-GB") + ' responses · ' + Number(analysis.commentCount || 0).toLocaleString("en-GB") + ' comments · ' + (analysis.decks || []).length + ' PowerPoints</span></div><button class="btn btn--secondary" type="button" data-saved-analysis="' + escapeHTML(analysis.id) + '">Open analysis</button>' + (decks ? '<ul class="survey-insights__saved-decks">' + decks + '</ul>' : '') + '</article>';
      }).join("") : '<p>No saved analyses yet. Run the analysis to create the first one.</p>';
    }).catch(function () { saved.innerHTML = '<p>Could not load saved analyses. Reload the page to try again.</p>'; });
  }

  function loadStats() {
    return Promise.all([retryRequest("/api/admin/stats", "GET", null, 0), fetch("/api/public-stats").then(function (response) { if (!response.ok) throw new Error("Could not load published statistics."); return response.json(); })]);
  }

  function openSavedAnalysis(id, preset) {
    pending = null;
    status.textContent = "Opening saved analysis…";
    return Promise.all([request("/api/admin/survey-insight-archive?surveyId=" + encodeURIComponent(surveyID) + "&analysisId=" + encodeURIComponent(id), "GET"), loadStats()]).then(function (values) {
      currentAnalysisID = id;
      currentReport = values[0];
      currentStats = values[1];
      render(currentReport, currentStats[0], currentStats[1], preset);
      runButton.textContent = "Re-run AI on latest responses";
      status.textContent = "Saved analysis opened. Select quotes and generate a new PowerPoint version.";
    }).catch(function (error) { status.textContent = error.message; });
  }

  function retryRequest(path, method, body, attempt) {
    return request(path, method, body).catch(function (error) {
      if (attempt >= 2 || [429, 502, 503, 504].indexOf(error.status) === -1 && !(error instanceof TypeError)) throw error;
      status.textContent = "Temporary server error. Retrying this page (" + (attempt + 2) + "/3)…";
      return new Promise(function (resolve) { window.setTimeout(resolve, (attempt + 1) * 1500); }).then(function () {
        return retryRequest(path, method, body, attempt + 1);
      });
    });
  }

  function optionName(survey, id) {
    var option = (survey.options || []).find(function (item) { return item.id === id; });
    return option ? (option.name || option.label || id) : id;
  }

  function makeThemeCounts(items) {
    var counts = {};
    items.forEach(function (item) {
      (item.themes || []).filter(function (theme, index, all) { return all.indexOf(theme) === index; }).forEach(function (theme) {
        counts[theme] = (counts[theme] || 0) + 1;
      });
    });
    return Object.keys(counts).map(function (name) { return { name: name, count: counts[name] }; }).sort(function (a, b) { return b.count - a.count || a.name.localeCompare(b.name); });
  }

  function makeSentiment(items) {
    var rows = {};
    items.forEach(function (item) {
      if (!rows[item.optionId]) rows[item.optionId] = { positive: 0, mixed: 0, negative: 0, unclassified: 0 };
      rows[item.optionId][item.sentiment] += 1;
    });
    return rows;
  }

  function makeSignalCounts(items) {
    var rows = {};
    items.forEach(function (item) {
      (item.signals || []).forEach(function (signal) {
        if (!rows[item.optionId]) rows[item.optionId] = {};
        if (!rows[item.optionId][signal.name]) rows[item.optionId][signal.name] = { positive: 0, mixed: 0, negative: 0 };
        if (rows[item.optionId][signal.name][signal.sentiment] != null) rows[item.optionId][signal.name][signal.sentiment] += 1;
      });
    });
    return rows;
  }

  function selectedQuoteIndexes() {
    return Array.prototype.map.call(output.querySelectorAll("[data-quote-index]:checked"), function (checkbox) {
      return Number(checkbox.getAttribute("data-quote-index"));
    });
  }

  function updateQuoteSelection(report) {
    var indexes = selectedQuoteIndexes();
    var selected = indexes.map(function (index) { return report.quotes[index]; });
    var counts = { good: 0, bad: 0, ugly: 0 };
    var available = { good: 0, bad: 0, ugly: 0 };
    report.quotes.forEach(function (quote) { available[quote.kind] += 1; });
    selected.forEach(function (quote) { counts[quote.kind] += 1; });
    ["good", "bad", "ugly"].forEach(function (kind) {
      var badge = output.querySelector('[data-quote-count="' + kind + '"]');
      var target = Math.min(3, available[kind]);
      badge.textContent = kind + " " + counts[kind] + "/" + target;
      badge.classList.toggle("is-valid", counts[kind] === target);
      badge.classList.toggle("is-invalid", counts[kind] !== target);
    });
    var valid = ["good", "bad", "ugly"].every(function (kind) { return counts[kind] === Math.min(3, available[kind]); });
    var feedback = output.querySelector("[data-quote-feedback]");
    feedback.textContent = valid ? "Ready: selected all available quotes, up to three per section." : "Select up to three quotes per section, or all available when there are fewer. Current selection: good " + counts.good + ", bad " + counts.bad + ", ugly " + counts.ugly + ".";
    feedback.classList.toggle("is-valid", valid);
    feedback.classList.toggle("is-invalid", !valid);
    Array.prototype.forEach.call(output.querySelectorAll("[data-quote-option]"), function (section) {
      var optionID = section.getAttribute("data-quote-option");
      var quotes = selected.filter(function (quote) { return quote.optionId === optionID; });
      section.querySelector("[data-option-count]").textContent = quotes.length + " selected";
      section.querySelector("[data-option-count]").classList.toggle("has-selection", quotes.length > 0);
      section.querySelector("[data-option-preview]").innerHTML = quotes.length ? quotes.map(function (quote) {
        return '<li><strong>' + escapeHTML(quote.kind) + ' · ' + escapeHTML(quote.sentiment) + '</strong> — “' + escapeHTML(quote.text) + '”</li>';
      }).join("") : '<li>No quotes selected from this option.</li>';
    });
    return valid;
  }

  function encodeBytes(bytes) {
    var chunks = [];
    for (var start = 0; start < bytes.length; start += 0x8000) {
      chunks.push(String.fromCharCode.apply(null, bytes.subarray(start, start + 0x8000)));
    }
    return window.btoa(chunks.join(""));
  }

  function render(report, adminStats, publicStats, preset) {
    var themeCounts = makeThemeCounts(report.items);
    var sentiment = makeSentiment(report.items);
    var signalCounts = makeSignalCounts(report.items);
    var optionRows = (report.survey.options || []).map(function (option) {
      var feelings = sentiment[option.id] || { positive: 0, mixed: 0, negative: 0, unclassified: 0 };
      return '<tr><th scope="row">' + escapeHTML(optionName(report.survey, option.id)) + '</th><td>' + (report.counts[option.id] || 0) + '</td><td>' + feelings.positive + '</td><td>' + feelings.mixed + '</td><td>' + feelings.negative + '</td><td>' + feelings.unclassified + '</td></tr>';
    }).join("");
    var signalRows = (report.survey.options || []).map(function (option) {
      return Object.keys(signalCounts[option.id] || {}).map(function (name) {
        var row = signalCounts[option.id][name];
        return '<tr><th scope="row">' + escapeHTML(optionName(report.survey, option.id)) + '</th><td>' + escapeHTML(name.replace(/-/g, ' ')) + '</td><td>' + row.positive + '</td><td>' + row.mixed + '</td><td>' + row.negative + '</td></tr>';
      }).join('');
    }).join('');
    var sorted = report.quotes.map(function (quote, index) { return { quote: quote, index: index }; });
    var sentimentOrder = { positive: 0, mixed: 1, negative: 2 };
    sorted.sort(function (left, right) {
      return (sentimentOrder[left.quote.sentiment] == null ? 3 : sentimentOrder[left.quote.sentiment]) - (sentimentOrder[right.quote.sentiment] == null ? 3 : sentimentOrder[right.quote.sentiment]) || left.index - right.index;
    });
    var proposed = { good: 0, bad: 0, ugly: 0 };
    var presetIndexes = preset && preset.quoteIndexes;
    var quoteMarkup = (report.survey.options || []).map(function (option, optionIndex) {
      var entries = sorted.filter(function (entry) { return entry.quote.optionId === option.id; });
      var cards = entries.map(function (entry) {
        var quote = entry.quote;
        var checked = presetIndexes ? presetIndexes.indexOf(entry.index) !== -1 : proposed[quote.kind] < 3;
        proposed[quote.kind] += 1;
        return '<label class="survey-insights__quote" data-quote-card data-kind="' + escapeHTML(quote.kind) + '" data-sentiment="' + escapeHTML(quote.sentiment) + '"><input type="checkbox" data-quote-index="' + entry.index + '"' + (checked ? ' checked' : '') + '><span><strong>' + escapeHTML(quote.sentiment) + ' · ' + escapeHTML(quote.kind) + '</strong> — <span data-quote-text>“' + escapeHTML(quote.text) + '”</span></span></label>';
      }).join("");
      var editorID = 'survey-quote-editor-' + optionIndex;
      return '<section class="survey-insights__quote-option" data-quote-option="' + escapeHTML(option.id) + '"><div class="survey-insights__quote-heading"><h5>' + escapeHTML(optionName(report.survey, option.id)) + '</h5><span class="survey-insights__option-count" data-option-count></span><button class="btn btn--secondary" type="button" data-option-edit aria-expanded="false" aria-controls="' + editorID + '">Edit quotes</button></div><ul class="survey-insights__selected" data-option-preview></ul><div class="survey-insights__quote-editor" data-option-editor id="' + editorID + '" hidden><div class="survey-insights__quote-filters"><label>Quote section <select data-quote-filter><option value="all">All sections</option><option value="good">Good</option><option value="bad">Bad</option><option value="ugly">Ugly</option></select></label><label>Comment sentiment <select data-sentiment-filter><option value="all">All sentiments</option><option value="positive">Positive</option><option value="mixed">Mixed</option><option value="negative">Negative</option></select></label><label>Search quotes <input type="search" data-quote-search maxlength="80" placeholder="Words or phrase"></label><label class="survey-insights__regex-toggle"><input type="checkbox" data-quote-regex> Use regular expression</label></div><p class="form-hint" data-quote-shown role="status"></p><p class="survey-insights__quote-pattern-error" data-quote-pattern-error role="alert" hidden></p><div class="survey-insights__quote-list">' + cards + '</div><p class="form-hint" data-quote-empty hidden></p></div></section>';
    }).join("");
    output.hidden = false;
    output.innerHTML = [
      '<section class="stack"><h3>Draft findings</h3><p class="form-hint">Review and edit the AI wording. The deck uses exact stored survey counts.</p>',
      '<label for="survey-insights-overview">Summary</label><textarea id="survey-insights-overview" data-insights-overview rows="4" maxlength="650"></textarea>',
      '<h4>Themes in optional comments</h4><div class="table-wrapper"><table class="data-table"><thead><tr><th>Theme</th><th>Comment entries</th></tr></thead><tbody>',
      themeCounts.map(function (row) { return '<tr><td>' + escapeHTML(row.name) + '</td><td>' + row.count + '</td></tr>'; }).join(""),
      '</tbody></table></div><h4>Survey choices and comment sentiment</h4><p class="form-hint">Choice counts are respondents. Sentiment columns count only classified comments entered for that option; multiple comments may come from one respondent. Unclassified comments are excluded from sentiment and theme totals. Scroll the table sideways on a small screen.</p>',
      '<div class="table-wrapper"><table class="data-table"><thead><tr><th>Outcome</th><th>Selected</th><th>Positive</th><th>Mixed</th><th>Negative</th><th>Unclassified</th></tr></thead><tbody>', optionRows, '</tbody></table></div>',
      '<h4>Key phrases by option</h4><p class="form-hint">Controlled phrase labels are counted by the experience expressed in each classified comment. A comment can carry up to three labels.</p><div class="table-wrapper"><table class="data-table"><thead><tr><th>Outcome</th><th>Phrase</th><th>Positive</th><th>Mixed</th><th>Negative</th></tr></thead><tbody>', signalRows || '<tr><td colspan="5">No supported phrases identified.</td></tr>', '</tbody></table></div>',
      '<h4>Questions and actions for JLR</h4><div data-insights-actions></div><h4>Choose anonymous quotes</h4>',
      '<p class="form-hint">Select three quotes for each Owner voices slide, or all available when fewer than three exist. Open an option to filter and choose its quotes. Check permission and remove identifying details.</p>',
      '<div class="survey-insights__quote-counts"><span data-quote-count="good"></span><span data-quote-count="bad"></span><span data-quote-count="ugly"></span></div>',
      '<p class="survey-insights__quote-feedback" data-quote-feedback role="status" aria-live="polite"></p>',
      '<div class="survey-insights__quotes">', quoteMarkup, '</div>',
      '<label class="survey-insights__confirmation"><input type="checkbox" data-insights-quote-review> I have checked permission and reviewed every selected quote for identifying details.</label>',
      '<div class="cluster"><button class="btn btn--primary" type="button" data-insights-download>Generate and save PowerPoint</button></div>',
      '<p class="form-hint">Meeting use only. Saved reports and decks are admin-only. AI labels and selected quotes need human review; responses are not a representative sample of all I-PACE owners.</p></section>'
    ].join("");
    output.querySelector("[data-insights-overview]").value = preset && preset.overview || report.overview;
    var actions = output.querySelector("[data-insights-actions]");
    actions.innerHTML = report.actions.map(function (_, index) { return '<label class="survey-insights__action">Action ' + (index + 1) + '<textarea data-insights-action="' + index + '" rows="2" maxlength="200"></textarea></label>'; }).join("");
    report.actions.forEach(function (action, index) { actions.querySelector('[data-insights-action="' + index + '"]').value = preset && preset.actions && preset.actions[index] || action; });
    output.onchange = function (event) {
      if (event.target.matches("[data-quote-index]")) {
        output.querySelector("[data-insights-quote-review]").checked = false;
        updateQuoteSelection(report);
      }
      if (event.target.matches("[data-quote-filter], [data-sentiment-filter]")) filterQuoteCards(event.target.closest("[data-quote-option]"));
      if (event.target.matches("[data-quote-regex]")) filterQuoteCards(event.target.closest("[data-quote-option]"));
    };
    output.oninput = function (event) {
      if (event.target.matches("[data-quote-search]")) filterQuoteCards(event.target.closest("[data-quote-option]"));
    };
    output.onclick = function (event) {
      var button = event.target.closest("[data-option-edit]");
      if (!button) return;
      var editor = button.closest("[data-quote-option]").querySelector("[data-option-editor]");
      editor.hidden = !editor.hidden;
      button.setAttribute("aria-expanded", String(!editor.hidden));
      button.textContent = editor.hidden ? "Edit quotes" : "Close editor";
      if (!editor.hidden) filterQuoteCards(button.closest("[data-quote-option]"));
    };
    updateQuoteSelection(report);
    output.querySelector("[data-insights-download]").onclick = function () {
      if (!updateQuoteSelection(report)) {
        status.textContent = "The quote selection is incomplete. Choose up to three in each section, or all available when fewer than three exist.";
        output.querySelector("[data-quote-feedback]").scrollIntoView({ block: "center", behavior: "smooth" });
        return;
      }
      if (!output.querySelector("[data-insights-quote-review]").checked) { status.textContent = "Review permission and identifying details for all nine selected quotes first."; return; }
      var overview = output.querySelector("[data-insights-overview]").value.trim();
      var editedActions = Array.prototype.map.call(actions.querySelectorAll("[data-insights-action]"), function (field) { return field.value.trim(); });
      if (!overview || editedActions.some(function (value) { return !value; })) { status.textContent = "Complete the summary and all three actions first."; return; }
      if (!currentAnalysisID) { status.textContent = "Save the completed analysis before making a PowerPoint."; return; }
      var indexes = selectedQuoteIndexes();
      var selected = indexes.map(function (index) { return report.quotes[index]; });
      var button = output.querySelector("[data-insights-download]");
      button.disabled = true;
      status.textContent = "Preparing and saving PowerPoint…";
      var deckReport = { survey: report.survey, counts: report.counts, preferredCounts: report.preferredCounts, textCounts: report.textCounts, totalResponses: report.totalResponses, items: report.items, overview: overview, actions: editedActions, quotes: selected, themeCounts: themeCounts, sentiment: sentiment };
      window.ipaceMakeSurveyDeck(deckReport, adminStats, publicStats, function (bytes) {
        return request("/api/admin/survey-insight-deck", "POST", { surveyId: surveyID, analysisId: currentAnalysisID, quoteIndexes: indexes, overview: overview, actions: editedActions, pptx: encodeBytes(bytes) }).then(function (savedDeck) {
          downloadBlob(new Blob([bytes], { type: "application/vnd.openxmlformats-officedocument.presentationml.presentation" }), "ipace-owner-survey-" + savedDeck.id + ".pptx");
          return loadArchive();
        });
      }).then(function () { status.textContent = "PowerPoint saved and downloaded. You can reopen this analysis to make another version."; }).catch(function (error) { status.textContent = error.message || "Could not save PowerPoint. Please try again."; }).finally(function () { button.disabled = false; });
    };
  }

  function filterQuoteCards(section) {
    var kind = section.querySelector("[data-quote-filter]").value;
    var sentiment = section.querySelector("[data-sentiment-filter]").value;
    var search = section.querySelector("[data-quote-search]").value.trim();
    var regexMode = section.querySelector("[data-quote-regex]").checked;
    var patternError = section.querySelector("[data-quote-pattern-error]");
    var pattern = null;
    if (search && regexMode) {
      try {
        if (/[(){}]/.test(search) || (search.match(/[+*?]/g) || []).length > 2) throw new Error("Pattern is too complex.");
        pattern = new RegExp(search, "i");
      } catch {
        patternError.textContent = "Invalid or overly complex regular expression. Edit the pattern or turn off regex mode.";
        patternError.hidden = false;
        return;
      }
    }
    patternError.hidden = true;
    var shown = 0;
    Array.prototype.forEach.call(section.querySelectorAll("[data-quote-card]"), function (card) {
      var quoteText = card.querySelector("[data-quote-text]").textContent;
      card.hidden = kind !== "all" && card.getAttribute("data-kind") !== kind || sentiment !== "all" && card.getAttribute("data-sentiment") !== sentiment || search && !(pattern ? pattern.test(quoteText) : quoteText.toLowerCase().indexOf(search.toLowerCase()) !== -1);
      if (!card.hidden) shown += 1;
    });
    section.querySelector("[data-quote-shown]").textContent = shown + (shown === 1 ? " quote shown" : " quotes shown") + ". Select or clear quotes below.";
    var empty = section.querySelector("[data-quote-empty]");
    empty.hidden = shown !== 0;
    empty.textContent = section.querySelectorAll("[data-quote-card]").length ? "No quotes match these filters." : "No quote candidates were generated for this option. Re-run AI after more responses arrive.";
  }

  function analyse() {
    if (active) return;
    if (!pending) currentAnalysisID = null;
    active = true;
    runButton.disabled = true;
    output.hidden = true;
    if (!pending) pending = { report: { id: surveyID, items: [], findings: [], quotes: [] }, offset: 0 };
    var report = pending.report;
    function page(offset) {
      status.textContent = "Completed " + offset + (report.totalResponses == null ? "" : " of " + report.totalResponses) + " responses. Analysing from response " + (offset + 1) + "…";
      return retryRequest("/api/admin/survey-insights", "POST", { id: surveyID, offset: offset }, 0).then(function (batch) {
        if (offset && batch.totalResponses !== report.totalResponses) { pending = null; throw new Error("Survey responses changed during analysis. Restart for a consistent snapshot."); }
        if (!offset) {
          report.survey = batch.survey;
          report.counts = batch.counts;
          report.preferredCounts = batch.preferredCounts;
          report.textCounts = batch.textCounts;
          report.totalResponses = batch.totalResponses;
        }
        report.items = report.items.concat(batch.items);
        report.quotes = report.quotes.concat(batch.quotes);
        if (batch.finding) report.findings.push(batch.finding);
        pending.offset = batch.nextOffset;
        if (batch.hasMore) {
          if (batch.nextOffset <= offset) throw new Error("Analysis did not advance. Restart.");
          return page(batch.nextOffset);
        }
        return report;
      });
    }
    (report.totalResponses != null && pending.offset >= report.totalResponses ? Promise.resolve(report) : page(pending.offset)).then(function () {
      if (report.overview && report.actions) return { overview: report.overview, actions: report.actions };
      status.textContent = "Summarising " + report.items.length + " optional comments…";
      return retryRequest("/api/admin/survey-insights-summary", "POST", { id: surveyID, expectedResponses: report.totalResponses, items: report.items, findings: report.findings, quotes: [] }, 0);
    }).then(function (summary) {
      report.overview = summary.overview;
      report.actions = summary.actions;
      if (currentAnalysisID) return loadStats();
      report.expectedResponses = report.totalResponses;
      status.textContent = "Saving completed analysis…";
      return request("/api/admin/survey-insight-archive", "POST", report).then(function (archive) {
        currentAnalysisID = archive.id;
        return loadArchive().then(loadStats);
      });
    }).then(function (stats) {
      currentReport = report;
      currentStats = stats;
      render(report, stats[0], stats[1]);
      pending = null;
      runButton.textContent = "Re-run AI on latest responses";
      var unclassified = report.items.filter(function (item) { return item.sentiment === "unclassified"; }).length;
      status.textContent = "Analysis complete: " + report.totalResponses + " survey responses; " + report.items.length + " optional comments processed" + (unclassified ? ", " + unclassified + " unclassified" : "") + ". Review the draft below.";
    }).catch(function (error) {
      runButton.textContent = pending ? "Resume analysis" : "Re-run AI on latest responses";
      status.textContent = error.message + (pending ? " Completed " + pending.offset + " of " + (pending.report.totalResponses || "?") + " responses. Resume analysis retries from response " + (pending.offset + 1) + " in this tab; do not refresh." : "");
    }).finally(function () { active = false; runButton.disabled = false; });
  }

  if (!surveyID) {
    runButton.disabled = true;
    status.textContent = "Choose a survey from the survey administration page.";
    return;
  }
  root.querySelector("[data-insights-back]").href = "/admin/survey-results/?id=" + encodeURIComponent(surveyID);
  runButton.onclick = analyse;
  saved.onclick = function (event) {
    var analysisButton = event.target.closest("[data-saved-analysis]");
    if (analysisButton) { openSavedAnalysis(analysisButton.getAttribute("data-saved-analysis")); return; }
    var editButton = event.target.closest("[data-edit-deck]");
    if (editButton) { openSavedAnalysis(editButton.getAttribute("data-analysis-id"), savedDecks[editButton.getAttribute("data-edit-deck")]); return; }
    var deckButton = event.target.closest("[data-saved-deck]");
    if (deckButton) {
      var path = "/api/admin/survey-insight-deck?surveyId=" + encodeURIComponent(surveyID) + "&analysisId=" + encodeURIComponent(deckButton.getAttribute("data-analysis-id")) + "&deckId=" + encodeURIComponent(deckButton.getAttribute("data-saved-deck"));
      status.textContent = "Downloading saved PowerPoint…";
      requestBlob(path).then(function (blob) { downloadBlob(blob, "ipace-owner-survey-" + deckButton.getAttribute("data-saved-deck") + ".pptx"); status.textContent = "Saved PowerPoint downloaded."; }).catch(function (error) { status.textContent = error.message; });
    }
  };
  if (window.ipaceIdentityReadyPromise) window.ipaceIdentityReadyPromise.then(function (user) { if (user) loadArchive(); });
  document.addEventListener("identity:login", loadArchive);
  if (window.firebase && window.firebase.auth().currentUser) loadArchive();
})();
