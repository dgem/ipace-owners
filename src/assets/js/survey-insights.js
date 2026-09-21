(function () {
  "use strict";

  var root = document.querySelector("[data-survey-insights]");
  if (!root) return;
  var runButton = root.querySelector("[data-insights-run]");
  var status = root.querySelector("[data-insights-status]");
  var output = root.querySelector("[data-insights-report]");
  var surveyID = new URLSearchParams(window.location.search).get("id");
  var active = false;
  var pending = null;

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

  function render(report, adminStats, publicStats) {
    var themeCounts = makeThemeCounts(report.items);
    var sentiment = makeSentiment(report.items);
    var optionRows = (report.survey.options || []).map(function (option) {
      var feelings = sentiment[option.id] || { positive: 0, mixed: 0, negative: 0, unclassified: 0 };
      return "<tr><th scope=\"row\">" + escapeHTML(optionName(report.survey, option.id)) + "</th><td>" + (report.counts[option.id] || 0) + "</td><td>" + feelings.positive + "</td><td>" + feelings.mixed + "</td><td>" + feelings.negative + "</td><td>" + feelings.unclassified + "</td></tr>";
    }).join("");
    var proposed = { good: 0, bad: 0, ugly: 0 };
    var quoteMarkup = report.quotes.map(function (quote, index) {
      var checked = proposed[quote.kind] < 3 ? ' checked' : '';
      proposed[quote.kind] += 1;
      return '<label class="survey-insights__quote"><input type="checkbox" data-quote-index="' + index + '"' + checked + '><span><strong>' + escapeHTML(quote.kind) + '</strong> — “' + escapeHTML(quote.text) + '”</span></label>';
    }).join("");
    output.hidden = false;
    output.innerHTML = '<section class="stack"><h3>Draft findings</h3><p class="form-hint">Review and edit the AI wording. The deck uses exact stored survey counts.</p><label for="survey-insights-overview">Summary</label><textarea id="survey-insights-overview" data-insights-overview rows="4" maxlength="650"></textarea><h4>Themes in optional comments</h4><div class="table-wrapper"><table class="data-table"><thead><tr><th>Theme</th><th>Comment entries</th></tr></thead><tbody>' + themeCounts.map(function (row) { return '<tr><td>' + escapeHTML(row.name) + '</td><td>' + row.count + '</td></tr>'; }).join("") + '</tbody></table></div><h4>Survey choices and comment sentiment</h4><p class="form-hint">Choice counts are respondents. Sentiment columns count only classified comments entered for that option; multiple comments may come from one respondent. Unclassified comments are excluded from sentiment and theme totals. Scroll the table sideways on a small screen.</p><div class="table-wrapper"><table class="data-table"><thead><tr><th>Outcome</th><th>Selected</th><th>Positive</th><th>Mixed</th><th>Negative</th><th>Unclassified</th></tr></thead><tbody>' + optionRows + '</tbody></table></div><h4>Questions and actions for JLR</h4><div data-insights-actions></div><h4>Choose anonymous quotes</h4><p class="form-hint">Choose three per good, bad and ugly category where suitable comments are available. Check that each quote has permission and reveals no person, registration or other identifying detail.</p><div class="survey-insights__quotes">' + (quoteMarkup || '<p>No suitable quote candidates were found.</p>') + '</div><label class="survey-insights__confirmation"><input type="checkbox" data-insights-quote-review> I have checked permission and reviewed every selected quote for identifying details.</label><div class="cluster"><button class="btn btn--primary" type="button" data-insights-download>Download PowerPoint</button></div><p class="form-hint">Meeting use only. AI labels and selected quotes need human review; self-selected survey responses are not a representative sample of all I-PACE owners.</p></section>';
    output.querySelector("[data-insights-overview]").value = report.overview;
    var actions = output.querySelector("[data-insights-actions]");
    actions.innerHTML = report.actions.map(function (_, index) {
      return '<label class="survey-insights__action">Action ' + (index + 1) + '<textarea data-insights-action="' + index + '" rows="2" maxlength="200"></textarea></label>';
    }).join("");
    report.actions.forEach(function (action, index) { actions.querySelector('[data-insights-action="' + index + '"]').value = action; });
    output.querySelector("[data-insights-download]").onclick = function () {
      var selected = Array.prototype.map.call(output.querySelectorAll("[data-quote-index]:checked"), function (checkbox) { return report.quotes[Number(checkbox.getAttribute("data-quote-index"))]; });
      var buckets = {};
      selected.forEach(function (quote) { buckets[quote.kind] = (buckets[quote.kind] || 0) + 1; });
      if (Object.keys(buckets).some(function (kind) { return buckets[kind] > 3; })) { status.textContent = "Choose no more than three quotes in each category."; return; }
      if (selected.length && !output.querySelector("[data-insights-quote-review]").checked) { status.textContent = "Review the permission and identifying details for selected quotes first."; return; }
      var overview = output.querySelector("[data-insights-overview]").value.trim();
      var editedActions = Array.prototype.map.call(actions.querySelectorAll("[data-insights-action]"), function (field) { return field.value.trim(); });
      if (!overview || editedActions.some(function (value) { return !value; })) { status.textContent = "Complete the summary and all three actions first."; return; }
      status.textContent = "Preparing PowerPoint…";
      window.ipaceMakeSurveyDeck({ survey: report.survey, counts: report.counts, preferredCounts: report.preferredCounts, totalResponses: report.totalResponses, items: report.items, overview: overview, actions: editedActions, quotes: selected, themeCounts: themeCounts, sentiment: sentiment }, adminStats, publicStats).then(function () {
        status.textContent = "PowerPoint downloaded. Keep the file private until the meeting.";
      }).catch(function () { status.textContent = "Could not create PowerPoint. Please try again."; });
    };
  }

  function analyse() {
    if (active) return;
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
      return Promise.all([retryRequest("/api/admin/stats", "GET", null, 0), fetch("/api/public-stats").then(function (response) { if (!response.ok) throw new Error("Could not load published statistics."); return response.json(); })]);
    }).then(function (stats) {
      render(report, stats[0], stats[1]);
      pending = null;
      runButton.textContent = "Analyse survey comments";
      var unclassified = report.items.filter(function (item) { return item.sentiment === "unclassified"; }).length;
      status.textContent = "Analysis complete: " + report.totalResponses + " survey responses; " + report.items.length + " optional comments processed" + (unclassified ? ", " + unclassified + " unclassified" : "") + ". Review the draft below.";
    }).catch(function (error) {
      runButton.textContent = pending ? "Resume analysis" : "Analyse survey comments";
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
})();
