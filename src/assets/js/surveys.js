(function () {
  "use strict";
  function authHeaders(headers) {
    return window.ipaceAuthHeaders ? window.ipaceAuthHeaders(headers) : headers;
  }
  function token() {
    var u = window.firebase && window.firebase.auth().currentUser;
    return u ? u.getIdToken() : Promise.reject(new Error("Sign in first."));
  }
  function request(path, method, body) {
    return token().then(function (t) {
      return fetch(path, {
        method: method,
        headers: authHeaders({
          Authorization: "Bearer " + t,
          "Content-Type": "application/json"
        }),
        body: body ? JSON.stringify(body) : undefined
      }).then(function (r) {
        if (r.status === 204) return {};
        return r.json().then(function (d) {
          if (!r.ok) throw new Error(d.error || "Request failed.");
          return d;
        });
      });
    });
  }
  function esc(v) {
    var d = document.createElement("div");
    d.textContent = v || "";
    return d.innerHTML;
  }
  function inlineMarkdown(v) {
    var html = esc(v);
    html = html.replace(
      /\[([^\]]+)\]\((https?:\/\/[A-Za-z0-9._~:/?#@!$&'()*+,;=%-]+)\)/g,
      '<a href="$2" rel="noopener noreferrer" target="_blank">$1</a>'
    );
    html = html.replace(/\*\*([^*\n]+)\*\*/g, "<strong>$1</strong>");
    return html.replace(/\*([^*\n]+)\*/g, "<em>$1</em>");
  }
  function markdown(v) {
    var html = "",
      list = false;
    String(v || "")
      .replace(/\r\n?/g, "\n")
      .split("\n")
      .forEach(function (line) {
        var item = line.match(/^\s*[-*]\s+(.+)$/);
        if (item) {
          if (!list) {
            html += "<ul>";
            list = true;
          }
          html += "<li>" + inlineMarkdown(item[1]) + "</li>";
          return;
        }
        if (list) {
          html += "</ul>";
          list = false;
        }
        if (line.trim()) html += "<p>" + inlineMarkdown(line) + "</p>";
      });
    return html + (list ? "</ul>" : "");
  }
  function dateInputValue(date) {
    return date.toISOString().slice(0, 10);
  }
  function optionDescription(option) {
    return String((option && (option.description || option.label)) || "");
  }
  function optionName(option) {
    var source = String(
      (option && (option.name || optionDescription(option))) || ""
    );
    var lines = source.replace(/\r\n?/g, "\n").split("\n");
    var first = "";
    lines.some(function (line) {
      first = line
        .replace(/^\s*[-*#]+\s*/, "")
        .replace(/\*\*/g, "")
        .replace(/\*/g, "")
        .trim();
      return !!first;
    });
    return first || "Untitled option";
  }
  function optionTextPrompt(option) {
    return String((option && option.textPrompt) || "Optional detail");
  }
  function surveyCopyMarkup(s) {
    var sections = [];
    if (s.description)
      sections.push(
        '<section class="survey-member-card__context">' +
          markdown(s.description) +
          "</section>"
      );
    if (s.question || s.callToAction)
      sections.push(
        '<section class="survey-member-card__question"><h3>' +
          (s.callToAction ? inlineMarkdown(s.callToAction) : "Your question") +
          "</h3>" +
          (s.question ? markdown(s.question) : "") +
          "</section>"
      );
    return sections.join("");
  }
  function setDescriptionExpanded(description, expanded) {
    var button = description.parentNode.querySelector(".survey-choice__more");
    if (!button) return;
    description.classList.toggle("is-expanded", expanded);
    description.classList.toggle("is-collapsed", !expanded);
    button.textContent = expanded ? "Show less" : "More";
    button.setAttribute("aria-expanded", String(expanded));
  }
  function expandSelectedOptionDescriptions(form) {
    Array.prototype.forEach.call(
      form.querySelectorAll('input[name="survey"]:checked'),
      function (input) {
        var description = input
          .closest(".survey-choice")
          .querySelector(".survey-choice__description");
        if (description) setDescriptionExpanded(description, true);
      }
    );
  }
  function prepareOptionDescriptions(root) {
    Array.prototype.forEach.call(
      root.querySelectorAll(".survey-choice__description"),
      function (description, index) {
        if (!description.textContent.trim()) {
          description.hidden = true;
          return;
        }
        description.classList.add("is-collapsed");
        description.id =
          "survey-option-description-" +
          index +
          "-" +
          Math.random().toString(36).slice(2);
        window.requestAnimationFrame(function () {
          var needsMore =
            description.scrollHeight > description.clientHeight + 1;
          if (!needsMore) {
            description.classList.remove("is-collapsed");
            return;
          }
          var button = document.createElement("button");
          button.type = "button";
          button.className = "survey-choice__more";
          button.textContent = "More";
          button.setAttribute("aria-expanded", "false");
          button.setAttribute("aria-controls", description.id);
          button.onclick = function (event) {
            event.preventDefault();
            event.stopPropagation();
            setDescriptionExpanded(
              description,
              !description.classList.contains("is-expanded")
            );
          };
          description.parentNode.appendChild(button);
        });
      }
    );
  }
  function choiceMarkup(
    option,
    index,
    multiple,
    allowsPreferred,
    selected,
    preferred,
    text,
    idPrefix
  ) {
    var textID = idPrefix + esc(option.id),
      description = optionDescription(option);
    return (
      '<div class="survey-choice"><label><input type="' +
      (multiple ? "checkbox" : "radio") +
      '" name="survey" value="' +
      esc(option.id) +
      '" ' +
      (selected ? "checked" : "") +
      '><span class="survey-choice__number" aria-hidden="true">' +
      (index + 1) +
      '</span><span class="survey-choice__copy"><span class="survey-choice__name">' +
      esc(optionName(option)) +
      '</span><span class="survey-markdown survey-choice__description">' +
      markdown(description) +
      "</span></span></label>" +
      (multiple && allowsPreferred
        ? '<div class="survey-choice__preferred" ' +
          (selected ? "" : "hidden") +
          '><label><input type="checkbox" data-preferred-option value="' +
          esc(option.id) +
          '" ' +
          (preferred ? "checked" : "") +
          "> Make this my preferred outcome</label></div>"
        : "") +
      (option.allowsText
        ? '<div class="survey-choice__text" ' +
          (selected ? "" : "hidden") +
          '><label for="' +
          textID +
          '"> ' +
          esc(optionTextPrompt(option)) +
          ' <span>(optional, up to 250 characters)</span></label><textarea id="' +
          textID +
          '" data-option-text-id="' +
          esc(option.id) +
          '" maxlength="250" rows="3" placeholder="enter your text here...">' +
          esc(text) +
          "</textarea></div>"
        : "") +
      "</div>"
    );
  }
  function resultMarkupOption(
    option,
    count,
    preferredCount,
    multiple,
    allowsPreferred,
    total,
    showDescription,
    textCount
  ) {
    var percentage = total ? Math.round((count / total) * 100) : 0;
    return (
      '<div class="survey-result' +
      (showDescription ? "" : " survey-result--summary") +
      '"><div><strong class="survey-result__name">' +
      esc(optionName(option)) +
      '</strong>' +
      (showDescription
        ? '<div class="survey-markdown survey-result__description">' +
          markdown(optionDescription(option)) +
          "</div>"
        : "") +
      '<div class="survey-result__bar" aria-hidden="true"><span style="width:' +
      percentage +
      '%"></span></div>' +
      '</div><strong class="survey-result__count">' +
      '<span class="survey-result__votes">' +
      count +
      " " +
      (count === 1 ? "vote" : "votes") +
      " · " +
      percentage +
      "%</span>" +
      (multiple && allowsPreferred
        ? '<span class="survey-result__preferred">' +
          preferredCount +
          " preferred</span>"
        : "") +
      (!showDescription && option.allowsText
        ? '<span class="survey-result__comments" aria-label="' +
          textCount +
          " optional " +
          (textCount === 1 ? "detail" : "details") +
          '"><span aria-hidden="true">💬</span> ' +
          textCount +
          "</span>"
        : "") +
      "</strong></div>"
    );
  }

  function preferredOutcomeLeader(survey, preferredCounts) {
    if (!survey.multiple) return null;
    var leader = null,
      highestCount = 0,
      tied = false;
    survey.options.forEach(function (option) {
      if (!option.allowsPreferred && survey.preferredEligibilityConfigured) return;
      var count = preferredCounts[option.id] || 0;
      if (count > highestCount) {
        leader = option;
        highestCount = count;
        tied = false;
      } else if (count > 0 && count === highestCount) {
        tied = true;
      }
    });
    return leader && highestCount > 0 && !tied
      ? { option: leader, count: highestCount }
      : null;
  }

  function setupAdmin(root) {
    var form = root.querySelector("[data-survey-editor]"),
      options = root.querySelector("[data-survey-options]"),
      status = root.querySelector("[data-survey-admin-status]"),
      list = root.querySelector("[data-survey-admin-list]"),
      legacyPreferredEligibility = false;
    function syncOptionTextPrompt(row) {
      var enabled = row.querySelector("[data-option-text]").checked,
        prompt = row.querySelector("[data-option-text-prompt]");
      prompt.disabled = !enabled;
      prompt
        .closest(".survey-option-editor__text-prompt")
        .classList.toggle("is-disabled", !enabled);
    }
    function add(o) {
      var row = document.createElement("div");
      row.className = "survey-option-editor";
      row.innerHTML =
        '<input data-option-id type="hidden" value="' +
        esc(o && o.id) +
        '"><label class="form-label">Option name<input class="survey-option-editor__name" data-option-name aria-label="Option name" maxlength="120" required type="text" placeholder="Replace all HV modules" value="' +
        esc(o ? optionName(o) : "") +
        '"></label><label class="form-label">Option description<textarea class="survey-option-editor__input" data-option-description aria-label="Option description" maxlength="2000" required rows="5" placeholder="Longer explanation for members. Markdown supported.">' +
        esc(optionDescription(o)) +
        '</textarea></label><div class="cluster survey-option-editor__preferred-toggle"><label><input data-option-preferred type="checkbox" ' +
        ((o && o.allowsPreferred) || legacyPreferredEligibility
          ? "checked"
          : "") +
        '> Allow members to mark this as their preferred option <span class="form-hint">(multiple-choice surveys only)</span></label></div><div class="cluster survey-option-editor__detail-toggle"><label><input data-option-text type="checkbox" ' +
        (o && o.allowsText ? "checked" : "") +
        '> Offer a short detail response</label></div><label class="form-label survey-option-editor__text-prompt">Detail-response prompt<input data-option-text-prompt aria-label="Detail-response prompt" maxlength="160" type="text" placeholder="Optional detail" value="' +
        esc(o ? optionTextPrompt(o) : "") +
        '"></label><button class="btn btn--danger btn--sm survey-option-editor__remove" type="button">Remove option</button>';
      row.querySelector("button").onclick = function () {
        row.remove();
      };
      row.querySelector("[data-option-text]").onchange = function () {
        syncOptionTextPrompt(row);
      };
      syncOptionTextPrompt(row);
      options.appendChild(row);
    }
    function clear() {
      var start = new Date(),
        end = new Date(start);
      end.setDate(end.getDate() + 6);
      form.reset();
      form.querySelector("[data-survey-id]").value = "";
      form.querySelector("[data-survey-status]").value = "draft";
      legacyPreferredEligibility = false;
      form.querySelector("[data-survey-start]").value = dateInputValue(start);
      form.querySelector("[data-survey-end]").value = dateInputValue(end);
      options.innerHTML = "";
      add();
      add();
      status.textContent = "";
    }
    function edit(s) {
      form.querySelector("[data-survey-id]").value = s.id;
      form.querySelector("[data-survey-title]").value = s.title;
      form.querySelector("[data-survey-description]").value =
        s.description || "";
      form.querySelector("[data-survey-question]").value = s.question || "";
      form.querySelector("[data-survey-cta]").value = s.callToAction || "";
      form.querySelector("[data-survey-status]").value =
        s.status || "published";
      form.querySelector("[data-survey-start]").value = s.startsOn;
      form.querySelector("[data-survey-end]").value = s.endsOn;
      form.querySelector("[data-survey-multiple]").checked = s.multiple;
      form.querySelector("[data-survey-results]").checked = s.showResults;
      legacyPreferredEligibility = !s.preferredEligibilityConfigured;
      options.innerHTML = "";
      s.options.forEach(add);
      window.scrollTo({ top: 0, behavior: "smooth" });
    }
    function render(surveys) {
      list.innerHTML = "";
      if (!surveys.length) {
        list.textContent = "No surveys yet.";
        return;
      }
      surveys.forEach(function (s) {
        var item = document.createElement("article");
        item.className = "card";
        item.innerHTML =
          "<h3>" +
          esc(s.title) +
          "</h3><p>" +
          esc(s.startsOn) +
          " to " +
          esc(s.endsOn) +
          " · " +
          esc(s.status || "published") +
          " · " +
          (s.multiple ? "Multiple choice" : "Single choice") +
          '</p><div class="cluster"><a class="btn btn--secondary btn--sm" href="/admin/survey-preview/?id=' +
          encodeURIComponent(s.id) +
          '">Preview / test</a><a class="btn btn--secondary btn--sm" href="/admin/survey-results/?id=' +
          encodeURIComponent(s.id) +
          '">View responses</a><button class="btn btn--secondary btn--sm" type="button">Edit</button><button class="btn btn--danger btn--sm" type="button">Delete</button></div>';
        var buttons = item.querySelectorAll("button");
        buttons[0].onclick = function () {
          edit(s);
        };
        buttons[1].onclick = function () {
          if (
            confirm(
              "Delete this survey? Existing responses will no longer be visible."
            )
          )
            request(
              "/api/admin/surveys?id=" + encodeURIComponent(s.id),
              "DELETE"
            )
              .then(load)
              .catch(function (e) {
                status.textContent = e.message;
              });
        };
        list.appendChild(item);
      });
    }
    var started = false;
    function load() {
      request("/api/admin/surveys", "GET")
        .then(function (d) {
          render(d.surveys);
        })
        .catch(function (e) {
          list.textContent = e.message;
        });
    }
    function start() {
      if (started) return;
      started = true;
      load();
      window.setInterval(load, 30000);
      window.addEventListener("focus", load);
    }
    root.querySelector("[data-survey-add-option]").onclick = function () {
      add();
    };
    root.querySelector("[data-survey-new]").onclick = clear;
    root.querySelector("[data-survey-refresh]").onclick = load;
    form.onsubmit = function (e) {
      e.preventDefault();
      var payload = {
        id: form.querySelector("[data-survey-id]").value,
        title: form.querySelector("[data-survey-title]").value,
        description: form.querySelector("[data-survey-description]").value,
        question: form.querySelector("[data-survey-question]").value,
        callToAction: form.querySelector("[data-survey-cta]").value,
        status: form.querySelector("[data-survey-status]").value,
        startsOn: form.querySelector("[data-survey-start]").value,
        endsOn: form.querySelector("[data-survey-end]").value,
        multiple: form.querySelector("[data-survey-multiple]").checked,
        showResults: form.querySelector("[data-survey-results]").checked,
        options: Array.prototype.map.call(options.children, function (x) {
          return {
            id: x.querySelector("[data-option-id]").value,
            name: x.querySelector("[data-option-name]").value,
            description: x.querySelector("[data-option-description]").value,
            allowsPreferred: x.querySelector("[data-option-preferred]").checked,
            allowsText: x.querySelector("[data-option-text]").checked,
            textPrompt: x.querySelector("[data-option-text-prompt]").value
          };
        })
      };
      request("/api/admin/surveys", payload.id ? "PUT" : "POST", payload)
        .then(function () {
          status.textContent = "Survey saved.";
          clear();
          load();
        })
        .catch(function (err) {
          status.textContent = err.message;
        });
    };
    clear();
    document.addEventListener("admin:data", start);
  }
  function setupAdminResults(root) {
    var started = false;
    function download(id) {
      token()
        .then(function (value) {
          return fetch(
            "/api/admin/survey-results?id=" +
              encodeURIComponent(id) +
              "&format=csv",
            { headers: authHeaders({ Authorization: "Bearer " + value }) }
          );
        })
        .then(function (response) {
          if (!response.ok) throw new Error("Could not download CSV");
          return response.blob();
        })
        .then(function (blob) {
          var link = document.createElement("a");
          link.href = URL.createObjectURL(blob);
          link.download = "survey-results.csv";
          link.click();
          URL.revokeObjectURL(link.href);
        })
        .catch(function (error) {
          root.querySelector("[role=status]").textContent = error.message;
        });
    }
    function render(analysis) {
      root.innerHTML =
        '<article class="dashboard-panel"><div class="dashboard-panel__header"><div><p class="dashboard-panel__eyebrow">Admin-only analysis</p><h2 class="dashboard-panel__title">' +
        esc(analysis.survey.title) +
        '</h2><p class="form-hint">Masked respondent emails and free-text are visible only to administrators. The CSV contains no full emails, names, or Firebase IDs.</p></div><button class="btn btn--primary" type="button" data-survey-download>Download CSV</button></div><div class="survey-results"><div class="survey-results__header"><h3>Aggregate results</h3><span>' +
        analysis.total +
        " response" +
        (analysis.total === 1 ? "" : "s") +
        "</span></div>" +
        analysis.survey.options
          .map(function (option) {
            var preferred =
              (analysis.preferredCounts &&
                analysis.preferredCounts[option.id]) ||
              0;
            return resultMarkupOption(
              option,
              (analysis.counts && analysis.counts[option.id]) || 0,
              preferred,
              analysis.survey.multiple,
              option.allowsPreferred || !analysis.survey.preferredEligibilityConfigured,
              analysis.total,
              true,
              0
            );
          })
          .join("") +
        '</div><h3 class="survey-analysis__title">Individual submissions</h3><div class="table-wrapper"><table class="data-table"><thead><tr><th>Masked respondent</th><th>Submitted</th><th>Selected option IDs</th><th>Preferred option ID</th><th>Text responses</th></tr></thead><tbody>' +
        analysis.responses
          .map(function (response) {
            return (
              "<tr><td>" +
              esc(response.respondent) +
              "</td><td>" +
              esc(response.updatedAt) +
              "</td><td>" +
              esc((response.optionIds || []).join(" · ")) +
              "</td><td>" +
              esc(response.preferredOptionId || "") +
              "</td><td>" +
              esc((response.textResponses || []).join(" · ")) +
              "</td></tr>"
            );
          })
          .join("") +
        '</tbody></table></div><p class="form-hint" role="status" aria-live="polite"></p></article>';
      root.querySelector("[data-survey-download]").onclick = function () {
        download(analysis.survey.id);
      };
    }
    function load() {
      var id = new URLSearchParams(window.location.search).get("id");
      if (!id) {
        root.innerHTML =
          '<section class="dashboard-panel"><h2 class="dashboard-panel__title">Choose a survey</h2><a class="btn btn--primary" href="/admin/surveys/">Back to surveys</a></section>';
        return;
      }
      request("/api/admin/survey-results?id=" + encodeURIComponent(id), "GET")
        .then(render)
        .catch(function (error) {
          root.textContent = error.message;
        });
    }
    function start() {
      if (started) return;
      started = true;
      load();
    }
    document.addEventListener("admin:data", start);
  }
  function setupAdminPreview(root) {
    var started = false;
    function render(s) {
      root.innerHTML =
        '<article class="dashboard-panel survey-member-card"><p class="dashboard-panel__eyebrow">Admin-only preview</p><h2 class="survey-member-card__title">' +
        esc(s.title) +
        '</h2><p class="form-hint">This uses the member response layout. Test responses are validated but never saved, counted, or visible to members.</p><div class="survey-markdown survey-member-card__copy">' +
        surveyCopyMarkup(s) +
        '</div><form class="survey-options"><fieldset><legend>' +
        (s.multiple
          ? "Select every outcome you would support"
          : "Select the outcome you would support") +
        '</legend><p class="form-hint">' +
        (s.multiple
          ? s.options.some(function (option) {
              return option.allowsPreferred || !s.preferredEligibilityConfigured;
            })
            ? "You can choose more than one option and may mark an eligible option as preferred."
            : "You can choose more than one option."
          : "Choose one option.") +
        '</p><div data-survey-choice-list></div></fieldset><button class="btn btn--primary survey-options__submit" type="submit">Test response</button><p role="status" aria-live="polite"></p></form></article>';
      var form = root.querySelector("form"),
        choices = form.querySelector("[data-survey-choice-list]");
      s.options.forEach(function (o, index) {
        choices.insertAdjacentHTML(
          "beforeend",
          choiceMarkup(o, index, s.multiple, o.allowsPreferred || !s.preferredEligibilityConfigured, false, false, "", "preview-text-")
        );
      });
      prepareOptionDescriptions(choices);
      form.onchange = function (event) {
        Array.prototype.forEach.call(
          form.querySelectorAll("[data-option-text-id]"),
          function (field) {
            field.closest(".survey-choice__text").hidden = !form.querySelector(
              'input[name="survey"][value="' +
                field.getAttribute("data-option-text-id") +
                '"]'
            ).checked;
          }
        );
        Array.prototype.forEach.call(
          form.querySelectorAll("[data-preferred-option]"),
          function (field) {
            var selected = form.querySelector(
              'input[name="survey"][value="' + field.value + '"]'
            ).checked;
            field.closest(".survey-choice__preferred").hidden = !selected;
            if (!selected) field.checked = false;
          }
        );
        if (
          event.target.matches("[data-preferred-option]") &&
          event.target.checked
        )
          Array.prototype.forEach.call(
            form.querySelectorAll("[data-preferred-option]"),
            function (field) {
              if (field !== event.target) field.checked = false;
            }
          );
        expandSelectedOptionDescriptions(form);
      };
      form.onsubmit = function (event) {
        event.preventDefault();
        var selected = Array.prototype.map.call(
            form.querySelectorAll('input[name="survey"]:checked'),
            function (input) {
              return input.value;
            }
          ),
          preferred = form.querySelector("[data-preferred-option]:checked"),
          textByOption = {};
        selected.forEach(function (id) {
          var field = form.querySelector('[data-option-text-id="' + id + '"]');
          if (field) textByOption[id] = field.value;
        });
        request(
          "/api/admin/survey-preview?id=" + encodeURIComponent(s.id),
          "POST",
          {
            surveyId: s.id,
            optionIds: selected,
            preferredOptionId: preferred ? preferred.value : "",
            textByOption: textByOption
          }
        )
          .then(function () {
            form.querySelector("[role=status]").textContent =
              "Test response is valid. Nothing was saved or counted.";
          })
          .catch(function (error) {
            form.querySelector("[role=status]").textContent = error.message;
          });
      };
    }
    function load() {
      var id = new URLSearchParams(window.location.search).get("id");
      if (!id) {
        root.innerHTML =
          '<section class="dashboard-panel"><h2 class="dashboard-panel__title">Choose a survey</h2><a class="btn btn--primary" href="/admin/surveys/">Back to surveys</a></section>';
        return;
      }
      request("/api/admin/survey-preview?id=" + encodeURIComponent(id), "GET")
        .then(render)
        .catch(function (error) {
          root.innerHTML =
            '<section class="dashboard-panel"><h2 class="dashboard-panel__title">Could not load survey preview</h2><p role="alert">' +
            esc(error.message) +
            '</p><button class="btn btn--secondary" type="button" data-survey-preview-retry>Try again</button></section>';
          root.querySelector("[data-survey-preview-retry]").onclick = load;
        });
    }
    function start() {
      if (started) return;
      started = true;
      load();
    }
    document.addEventListener("admin:data", start);
    var adminContainer = root.closest("[data-admin-container]");
    if (adminContainer && adminContainer.dataset.adminData) start();
  }
  function setupResponse(root) {
    function render(all) {
      var requestedID = new URLSearchParams(window.location.search).get("id");
      var active = all.filter(function (result) {
        return (
          result.canRespond &&
          (!requestedID || result.survey.id === requestedID)
        );
      });
      root.innerHTML = "";
      if (!active.length) {
        root.innerHTML =
          '<section class="dashboard-panel"><h2 class="dashboard-panel__title">This survey is not open</h2><p>You can browse all current, future, and closed surveys from the member survey list.</p><a class="btn btn--secondary" href="/member/surveys/">View member surveys</a></section>';
        return;
      }
      active.forEach(function (result) {
        var s = result.survey,
          article = document.createElement("article");
        article.className = "dashboard-panel survey-member-card";
        article.innerHTML =
          '<div class="survey-member-card__intro"><p class="dashboard-panel__eyebrow">Member survey</p><h2 class="survey-member-card__title">' +
          esc(s.title) +
          '</h2><div class="survey-markdown survey-member-card__copy">' +
          surveyCopyMarkup(s) +
          '</div><p class="survey-member-card__status">Open now · ' +
          esc(s.startsOn) +
          " to " +
          esc(s.endsOn) +
          '</p></div><form class="survey-options"></form><p role="status" aria-live="polite"></p>';
        var form = article.querySelector("form");
        form.insertAdjacentHTML(
          "beforeend",
          "<fieldset><legend>" +
            (s.multiple
              ? "Select every outcome you would support"
              : "Select the outcome you would support") +
            '</legend><p class="form-hint">' +
            (s.multiple
              ? "You can choose more than one option."
              : "Choose one option.") +
            "</p><div data-survey-choice-list></div></fieldset>"
        );
        var choices = form.querySelector("[data-survey-choice-list]");
        s.options.forEach(function (o, index) {
          var selected = (result.myOptionIds || []).indexOf(o.id) > -1,
            text = (result.myTextByOption && result.myTextByOption[o.id]) || "";
          var preferred = result.myPreferredOptionId === o.id;
          choices.insertAdjacentHTML(
            "beforeend",
            choiceMarkup(
              o,
              index,
              s.multiple,
              o.allowsPreferred || !s.preferredEligibilityConfigured,
              selected,
              preferred,
              text,
              "survey-text-"
            )
          );
        });
        form.insertAdjacentHTML(
          "beforeend",
          '<button class="btn btn--primary survey-options__submit" type="submit">' +
            ((result.myOptionIds || []).length
              ? "Update your response"
              : "Submit your response") +
            "</button>"
        );
        form.onchange = function (event) {
          Array.prototype.forEach.call(
            form.querySelectorAll("[data-option-text-id]"),
            function (field) {
              field.closest(".survey-choice__text").hidden =
                !form.querySelector(
                  'input[name="survey"][value="' +
                    field.getAttribute("data-option-text-id") +
                    '"]'
                ).checked;
            }
          );
          Array.prototype.forEach.call(
            form.querySelectorAll("[data-preferred-option]"),
            function (field) {
              var selectedOption = form.querySelector(
                'input[name="survey"][value="' + field.value + '"]'
              ).checked;
              field.closest(".survey-choice__preferred").hidden =
                !selectedOption;
              if (!selectedOption) field.checked = false;
            }
          );
          if (
            event.target.matches("[data-preferred-option]") &&
            event.target.checked
          )
            Array.prototype.forEach.call(
              form.querySelectorAll("[data-preferred-option]"),
              function (field) {
                if (field !== event.target) field.checked = false;
              }
            );
          expandSelectedOptionDescriptions(form);
        };
        form.onsubmit = function (e) {
          e.preventDefault();
          var selected = Array.prototype.map.call(
              form.querySelectorAll('input[name="survey"]:checked'),
              function (input) {
                return input.value;
              }
            ),
            preferred = form.querySelector("[data-preferred-option]:checked"),
            textByOption = {};
          selected.forEach(function (id) {
            var field = form.querySelector(
              '[data-option-text-id="' + id + '"]'
            );
            if (field) textByOption[id] = field.value;
          });
          request("/api/member/survey-response", "POST", {
            surveyId: s.id,
            optionIds: selected,
            preferredOptionId: preferred ? preferred.value : "",
            textByOption: textByOption
          })
            .then(function () {
              window.location.assign(
                "/member/survey-results/?id=" + encodeURIComponent(s.id)
              );
            })
            .catch(function (err) {
              article.querySelector("[role=status]").textContent = err.message;
            });
        };
        root.appendChild(article);
        prepareOptionDescriptions(choices);
      });
    }
    var started = false;
    function load() {
      request("/api/member/surveys", "GET")
        .then(function (d) {
          render(d.surveys);
        })
        .catch(function (e) {
          root.textContent = e.message;
        });
    }
    function start() {
      if (started) return;
      started = true;
      load();
    }
    document.addEventListener("member:data", start);
  }
  function setupMemberList(root) {
    var started = false;
    function render(all) {
      var filter =
        new URLSearchParams(window.location.search).get("filter") || "all";
      var today = dateInputValue(new Date());
      var surveys = all
        .slice()
        .sort(function (a, b) {
          return String(b.survey.startsOn).localeCompare(
            String(a.survey.startsOn)
          );
        })
        .filter(function (result) {
          var future = result.survey.startsOn > today,
            closed = !result.canRespond && !future;
          return filter === "open"
            ? result.canRespond
            : filter === "upcoming"
              ? future
              : filter === "closed"
                ? closed
                : true;
        });
      var filters = [
        ["all", "All surveys"],
        ["open", "Open"],
        ["upcoming", "Upcoming"],
        ["closed", "Closed"]
      ]
        .map(function (item) {
          return (
            '<a class="btn btn--sm ' +
            (filter === item[0] ? "btn--primary" : "btn--secondary") +
            '" href="/member/surveys/' +
            (item[0] === "all" ? "" : "?filter=" + item[0]) +
            '">' +
            item[1] +
            "</a>"
          );
        })
        .join("");
      root.innerHTML =
        '<div class="survey-list__header"><div><p class="dashboard-panel__eyebrow">Your voice matters</p><h2>All member surveys</h2><p>Browse surveys by date and see whether you have already responded.</p></div><div class="cluster">' +
        filters +
        "</div></div>";
      if (!surveys.length) {
        root.insertAdjacentHTML(
          "beforeend",
          '<section class="dashboard-panel"><p>No ' +
            esc(filter === "all" ? "" : filter + " ") +
            "surveys to show.</p></section>"
        );
        return;
      }
      surveys.forEach(function (result) {
        var s = result.survey,
          future = s.startsOn > today,
          closed = !result.canRespond && !future,
          submitted = (result.myOptionIds || []).length > 0;
        var status = result.canRespond
          ? "Open now"
          : future
            ? "Opens " + s.startsOn
            : "Closed " + s.endsOn;
        var actions = "";
        if (result.canRespond)
          actions +=
            '<a class="btn btn--primary" href="/member/survey-response/?id=' +
            encodeURIComponent(s.id) +
            '">' +
            (submitted ? "Edit response" : "Submit response") +
            "</a>";
        if (s.showResults && (submitted || closed))
          actions +=
            '<a class="btn btn--secondary" href="/member/survey-results/?id=' +
            encodeURIComponent(s.id) +
            '">View results</a>';
        if (!actions && future)
          actions =
            '<span class="form-hint">Available from ' +
            esc(s.startsOn) +
            "</span>";
        if (!actions && closed && !submitted)
          actions =
            '<span class="form-hint">Results are available after you respond, or where published when a survey closes.</span>';
        root.insertAdjacentHTML(
          "beforeend",
          '<article class="dashboard-panel survey-list-item"><div><p class="survey-member-card__status">' +
            esc(status) +
            '</p><h3 class="dashboard-panel__title">' +
            esc(s.title) +
            '</h3><p class="form-hint">' +
            esc(s.startsOn) +
            " to " +
            esc(s.endsOn) +
            " · " +
            (submitted ? "Response submitted" : "Not yet responded") +
            '</p></div><div class="cluster">' +
            actions +
            "</div></article>"
        );
      });
    }
    function load() {
      request("/api/member/surveys", "GET")
        .then(function (data) {
          render(data.surveys);
        })
        .catch(function (error) {
          root.textContent = error.message;
        });
    }
    function start() {
      if (started) return;
      started = true;
      load();
    }
    document.addEventListener("member:data", start);
  }
  function resultMarkup(result) {
    var s = result.survey,
      submitted = result.myOptionIds && result.myOptionIds.length,
      closed = s.endsOn < dateInputValue(new Date()),
      preferredLeader = preferredOutcomeLeader(s, result.preferredCounts || {});
    if (!submitted && !closed)
      return (
        '<section class="dashboard-panel"><h2 class="dashboard-panel__title">Submit your response first</h2><p>Aggregate results become available after you have submitted your own response.</p><a class="btn btn--primary" href="/member/survey-response/?id=' +
        encodeURIComponent(s.id) +
        '">Submit response</a></section>'
      );
    if (!s.showResults)
      return (
        '<section class="dashboard-panel"><h2 class="dashboard-panel__title">Results are not published for this survey</h2><p>Your response has been saved.</p><a class="btn btn--secondary" href="/member/survey-response/?id=' +
        encodeURIComponent(s.id) +
        '">Edit your response</a></section>'
      );
    return (
      '<article class="dashboard-panel survey-member-card"><p class="dashboard-panel__eyebrow">' +
      (submitted ? "Response saved" : "Survey closed") +
      '</p><h2 class="survey-member-card__title">' +
      esc(s.title) +
      '</h2><p class="survey-member-card__status">' +
      (submitted
        ? "We have added your submission to the overall results shown below."
        : "Results are now available to all members.") +
      '</p><div class="survey-results"><div class="survey-results__header"><h3>Overall results</h3><span>' +
      result.total +
      " response" +
      (result.total === 1 ? "" : "s") +
      "</span></div>" +
      (preferredLeader
        ? '<aside class="survey-results__leader"><p>Currently most preferred outcome</p><strong>' +
          esc(optionName(preferredLeader.option)) +
          '</strong><span>Selected as preferred by ' +
          preferredLeader.count +
          " member" +
          (preferredLeader.count === 1 ? "" : "s") +
          ".</span></aside>"
        : "") +
      s.options
        .map(function (o) {
          var preferred =
            (result.preferredCounts && result.preferredCounts[o.id]) || 0;
          return resultMarkupOption(
            o,
            (result.counts && result.counts[o.id]) || 0,
            preferred,
            s.multiple,
            o.allowsPreferred || !s.preferredEligibilityConfigured,
            result.total,
            false,
            (result.textCounts && result.textCounts[o.id]) || 0
          );
        })
        .join("") +
      '</div><div class="cluster survey-results__actions">' +
      '<a class="btn btn--primary" href="/member/account/">Return to member dashboard</a>' +
      (result.canRespond && submitted
        ? '<a class="btn btn--secondary" href="/member/survey-response/?id=' +
          encodeURIComponent(s.id) +
          '">Edit response</a>'
        : "") +
      '<a class="btn btn--secondary" href="/member/surveys/?filter=closed">Closed surveys</a></div></article>'
    );
  }
  function setupResults(root) {
    var started = false;
    function load() {
      var id = new URLSearchParams(window.location.search).get("id");
      request("/api/member/surveys", "GET")
        .then(function (data) {
          var result = data.surveys.filter(function (item) {
            return item.survey.id === id;
          })[0];
          root.innerHTML = result
            ? resultMarkup(result)
            : '<section class="dashboard-panel"><h2 class="dashboard-panel__title">Survey not found</h2><a class="btn btn--primary" href="/member/surveys/">View open surveys</a></section>';
        })
        .catch(function (error) {
          root.textContent = error.message;
        });
    }
    function start() {
      if (started) return;
      started = true;
      load();
    }
    document.addEventListener("member:data", start);
  }
  function setupMemberDashboard(root) {
    var started = false;
    function render(surveys) {
      var active = surveys.filter(function (result) {
          return result.canRespond;
        }),
        closed = surveys.filter(function (result) {
          return (
            !result.canRespond &&
            result.survey.startsOn <= dateInputValue(new Date())
          );
        }),
        historyAction = closed.length
          ? '<a class="btn btn--secondary" href="/member/surveys/?filter=closed">Past surveys</a>'
          : "";
      if (!active.length) {
        root.innerHTML =
          '<h2 class="survey-dashboard-callout__title">Member surveys</h2><p>There are no open surveys right now.</p>' +
          historyAction;
        return;
      }
      var names = active
        .map(function (result) {
          return "<li>" + esc(result.survey.title) + "</li>";
        })
        .join("");
      root.innerHTML =
        '<p class="dashboard-panel__eyebrow">Your voice matters</p><h2 class="survey-dashboard-callout__title">Help steer our discussions with JLR</h2><p>There ' +
        (active.length === 1
          ? "is an open member survey"
          : "are " + active.length + " open member surveys") +
        " waiting for your response.</p><ul>" +
        names +
        '</ul><div class="cluster"><a class="btn btn--primary" href="' +
        (active.length === 1
          ? "/member/survey-response/?id=" + encodeURIComponent(active[0].survey.id)
          : "/member/surveys/") +
        '">Take the survey' +
        (active.length === 1 ? "" : "s") +
        "</a>" +
        historyAction +
        "</div>";
    }
    function load() {
      request("/api/member/surveys", "GET")
        .then(function (data) {
          render(data.surveys);
        })
        .catch(function () {
          root.textContent = "Member surveys are currently unavailable.";
        });
    }
    function start() {
      if (started) return;
      started = true;
      load();
      window.setInterval(load, 60000);
      window.addEventListener("focus", load);
    }
    document.addEventListener("member:data", start);
  }
  document.querySelectorAll("[data-survey-admin]").forEach(setupAdmin);
  document
    .querySelectorAll("[data-admin-survey-results]")
    .forEach(setupAdminResults);
  document
    .querySelectorAll("[data-admin-survey-preview]")
    .forEach(setupAdminPreview);
  document
    .querySelectorAll("[data-member-survey-list]")
    .forEach(setupMemberList);
  document
    .querySelectorAll("[data-member-survey-response]")
    .forEach(setupResponse);
  document
    .querySelectorAll("[data-member-survey-results]")
    .forEach(setupResults);
  document
    .querySelectorAll("[data-member-survey-summary]")
    .forEach(setupMemberDashboard);
})();
