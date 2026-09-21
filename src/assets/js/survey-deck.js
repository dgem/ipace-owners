(function () {
  "use strict";

  var navy = "12324A";
  var teal = "0F766E";
  var ink = "111827";
  var muted = "4B5563";
  var pale = "F7F8FB";
  var gold = "B58A31";
  var red = "A83B42";
  var sentimentColors = { positive: teal, mixed: gold, negative: red };
  var themeLabels = { battery: "HV battery", charging: "Charging", "air-conditioning": "Air conditioning", service: "Service visits", parts: "Parts", warranty: "Warranty", safety: "Safety", value: "Market value", other: "Other" };
  var signalLabels = { "repair-delays": "Repair delays", "repeat-visits": "Repeat visits", "replacement-reliability": "Reliable replacement", "battery-confidence": "Battery confidence", "parts-availability": "Parts availability", "resale-value": "Resale value", "buyback-fairness": "Fair buy-back", "air-conditioning": "Air conditioning", charging: "Charging", warranty: "Warranty clarity", communication: "Communication", "customer-care": "Customer care" };

  function label(value, limit) {
    var text = String(value == null ? "" : value).replace(/\s+/g, " ").trim();
    return text.length > limit ? text.slice(0, limit - 1).trimEnd() + "…" : text;
  }

  function number(value) { return Number(value || 0).toLocaleString("en-GB"); }
  function percent(value, total) { return total ? Math.round(100 * value / total) + "%" : "0%"; }

  function text(slide, value, x, y, w, h, opts) {
    slide.addText(String(value), Object.assign({ x: x, y: y, w: w, h: h, margin: 0, fontFace: "Arial", color: ink, fontSize: 18, breakLine: false, valign: "mid", fit: "shrink" }, opts || {}));
  }

  function baseSlide(pptx, title, index, date) {
    var slide = pptx.addSlide({ masterName: "IPACE_CONTENT" });
    text(slide, title, 0.75, 0.42, 11.9, 0.68, { color: navy, fontSize: 28, bold: true, placeholder: "title" });
    text(slide, date, 10.55, 7.06, 1.45, 0.2, { color: muted, fontSize: 9, align: "right" });
    text(slide, String(index), 12.2, 7.06, 0.35, 0.2, { color: muted, fontSize: 9, align: "right" });
    return slide;
  }

  function chartRow(slide, pptx, caption, value, max, x, y, width, color, suffix) {
    text(slide, label(caption, 48), x, y, 4.55, 0.4, { fontSize: 15 });
    slide.addShape(pptx.ShapeType.rect, { x: x + 4.65, y: y + 0.1, w: width, h: 0.2, line: { color: "D8E0E5", transparency: 100 }, fill: { color: "D8E0E5" } });
    if (value > 0 && max > 0) slide.addShape(pptx.ShapeType.rect, { x: x + 4.65, y: y + 0.1, w: width * value / max, h: 0.2, line: { color: color, transparency: 100 }, fill: { color: color } });
    text(slide, number(value) + (suffix || ""), x + 4.75 + width, y, 1.3, 0.4, { fontSize: 14, bold: true, color: navy, align: "right" });
  }

  function wreathCounter(slide, imageData, value, caption, x, y, size) {
    slide.addImage({ data: imageData, altText: "Gold racing laurel framing the " + caption.toLowerCase() + " count", x: x, y: y, w: size, h: size });
    var digits = number(value).length;
    text(slide, number(value), x + size * 0.19, y + size * 0.35, size * 0.62, size * 0.32, { fontSize: digits > 5 ? 26 : digits > 4 ? 30 : digits > 3 ? 35 : 40, bold: true, color: navy, align: "center" });
    text(slide, caption, x - 0.76, y + size + 0.02, size + 1.52, 0.42, { fontSize: 17, color: navy, align: "center", bold: true });
  }

  function signalCounts(items) {
    var rows = {};
    (items || []).forEach(function (item) {
      (item.signals || []).forEach(function (signal) {
        if (!signalLabels[signal.name] || !sentimentColors[signal.sentiment]) return;
        var key = item.optionId + "|" + signal.name;
        if (!rows[key]) rows[key] = { optionId: item.optionId, name: signal.name, positive: 0, mixed: 0, negative: 0, count: 0 };
        rows[key][signal.sentiment] += 1;
        rows[key].count += 1;
      });
    });
    return Object.keys(rows).map(function (key) { return rows[key]; });
  }

  function optionGroups(survey) {
    var groups = { replacement: null, buyback: null, neither: null, compensation: null, concerns: null };
    (survey.options || []).forEach(function (option) {
      var name = (option.name || option.label || option.id || "").toLowerCase();
      if (/replacement|full hv/.test(name)) groups.replacement = option;
      else if (/buy.?back/.test(name)) groups.buyback = option;
      else if (/none|neither/.test(name)) groups.neither = option;
      else if (/compensation/.test(name)) groups.compensation = option;
      else if (/additional|concern/.test(name)) groups.concerns = option;
    });
    return groups;
  }

  function box(slide, pptx, x, y, w, h, color, line) {
    slide.addShape(pptx.ShapeType.roundRect, { x: x, y: y, w: w, h: h, rectRadius: 0.12, radius: 0.12, line: { color: line || color, width: 1 }, fill: { color: color } });
  }

  function callout(slide, pptx, heading, body, x, y, w, h, tone) {
    box(slide, pptx, x, y, w, h, tone || "E8F4F2", tone || "E8F4F2");
    text(slide, heading, x + 0.22, y + 0.16, w - 0.44, 0.36, { bold: true, fontSize: 17, color: teal });
    text(slide, body, x + 0.22, y + 0.58, w - 0.44, h - 0.73, { fontSize: 16, color: navy, valign: "top" });
  }

  function optionText(option, fallback) { return option ? (option.name || option.label || fallback) : fallback; }

  function memberChoice(slide, pptx, option, index, selected, preferred, detail, x, y, w, h) {
    box(slide, pptx, x, y, w, h, selected ? "F0F9F7" : "FFFFFF", selected ? teal : "D8E0E5");
    text(slide, selected ? "☑" : "□", x + 0.13, y + 0.12, 0.3, 0.32, { fontSize: 20, color: teal });
    slide.addShape(pptx.ShapeType.ellipse, { x: x + 0.53, y: y + 0.12, w: 0.32, h: 0.32, line: { color: navy, transparency: 100 }, fill: { color: navy } });
    text(slide, String(index), x + 0.53, y + 0.13, 0.32, 0.29, { fontSize: 12, bold: true, color: "FFFFFF", align: "center" });
    text(slide, optionText(option, "Choice"), x + 0.98, y + 0.11, w - 1.18, 0.32, { fontSize: 15, bold: true, color: navy });
    if (h > 0.6) text(slide, label(option && option.description || "", 105), x + 0.98, y + 0.46, w - 1.2, 0.3, { fontSize: 10, color: muted });
    if (preferred) text(slide, "☑  Make this my preferred outcome", x + 0.98, y + 0.79, w - 1.2, 0.27, { fontSize: 11, bold: true, color: navy });
    if (detail) {
      var detailY = preferred ? y + 1.09 : y + 0.76;
      text(slide, label(option && option.textPrompt || "Optional detail", 60) + "  (optional)", x + 0.98, detailY, w - 1.2, 0.21, { fontSize: 9, color: navy });
      box(slide, pptx, x + 0.98, detailY + 0.23, w - 1.23, 0.35, "FFFFFF", "D8E0E5");
      text(slide, detail, x + 1.08, detailY + 0.27, w - 1.43, 0.23, { fontSize: 10, color: ink });
    }
  }

  function checkInput(report, adminStats, publicStats) {
    if (!report || !report.survey || !Array.isArray(report.survey.options) || !Array.isArray(report.items) || !Array.isArray(report.quotes) || !Array.isArray(report.actions) || report.actions.length !== 3 || !adminStats || !publicStats) throw new Error("Incomplete report");
    if (report.quotes.length > 9) throw new Error("Too many quotes");
  }

  window.ipaceMakeSurveyDeck = function (report, adminStats, publicStats, saveForTest, assets) {
    checkInput(report, adminStats, publicStats);
    if (!window.PptxGenJS) return Promise.reject(new Error("PowerPoint library unavailable"));
    if (!window.JSZip) return Promise.reject(new Error("PowerPoint package validator unavailable"));
    function loadImage(source, supplied, message) {
      if (supplied) return Promise.resolve(supplied);
      return fetch(source).then(function (response) {
        if (!response.ok) throw new Error(message);
        return response.blob();
      }).then(function (blob) {
        return new Promise(function (resolve, reject) {
          var reader = new FileReader();
          reader.onload = function () { resolve(reader.result); };
          reader.onerror = reject;
          reader.readAsDataURL(blob);
        });
      });
    }
    return Promise.all([
      loadImage("/images/september-survey-reminder-2026-hero.jpg", assets && assets.heroData, "Meeting hero image unavailable"),
      loadImage("/images/racing-laurel-email.png", assets && assets.wreathData, "Racing laurel image unavailable")
    ]).then(function (images) {
    var heroData = images[0];
    var wreathData = images[1];
    var pptx = new window.PptxGenJS();
    pptx.layout = "LAYOUT_WIDE";
    pptx.theme = { headFontFace: "Arial", bodyFontFace: "Arial" };
    pptx.defineSlideMaster({
      title: "IPACE_CONTENT",
      background: { color: pale },
      objects: [
        { placeholder: { options: { name: "title", type: "title", x: 0.75, y: 0.42, w: 11.9, h: 0.68, fontFace: "Arial", fontSize: 28, bold: true, color: navy, margin: 0 }, text: "" } },
        { line: { x: 0.75, y: 1.2, w: 11.85, h: 0, line: { color: teal, width: 2 } } },
        { text: { text: "I-PACE Owners’ Advocacy Group  •  Sept JLR Meeting", options: { x: 0.75, y: 7.06, w: 8.9, h: 0.2, margin: 0, fontFace: "Arial", fontSize: 9, color: muted } } }
      ]
    });
    pptx.author = "I-PACE Owners’ Advocacy Group";
    pptx.subject = "Anonymous member survey analysis for the 24 September 2026 JLR meeting";
    pptx.title = "i-Pace Owners Advocacy Group";
    pptx.lang = "en-GB";
    var date = new Date().toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });
    var index = 1;
    var slide = pptx.addSlide();
    slide.addImage({ data: heroData, altText: "Two right-hand-drive I-PACE-style cars on a British country road", x: 0, y: 0, w: 13.333, h: 7.5 });
    slide.addShape(pptx.ShapeType.rect, { x: 0, y: 4.3, w: 13.333, h: 3.2, line: { color: navy, transparency: 100 }, fill: { color: navy, transparency: 4 } });
    text(slide, "i-Pace Owners Advocacy Group", 0.75, 4.72, 11.8, 0.69, { color: "FFFFFF", fontSize: 35, bold: true });
    text(slide, "Member survey findings for discussion with JLR", 0.78, 5.48, 11.8, 0.44, { color: "FFFFFF", fontSize: 23 });
    text(slide, "UK Director for Client Care  •  24 September 2026  •  Sept JLR Meeting", 0.78, 6.74, 11.8, 0.31, { color: "DCE9E9", fontSize: 14 });

    slide = baseSlide(pptx, "What we need from this meeting", ++index, date);
    text(slide, "An opportunity to agree what good looks like — and a plan to deliver it.", 0.86, 1.48, 11.3, 0.74, { fontSize: 25, bold: true, color: navy });
    [
      ["01", "Start with a shared picture", "Confirm everyone has seen the group's letter; hear what JLR has done and plans next."],
      ["02", "Agree the principles", "Discuss a reliable fix, a fair alternative where needed, and disruption separately."],
      ["03", "Leave with owners and dates", "Identify decisions, workstreams and a route to an update for members."]
    ].forEach(function (row, i) {
      var y = 2.48 + i * 1.27;
      box(slide, pptx, 0.86, y, 11.55, 1.08, "FFFFFF", "D8E0E5");
      text(slide, row[0], 1.12, y + 0.28, 0.72, 0.45, { fontSize: 26, bold: true, color: teal });
      text(slide, row[1], 2.08, y + 0.12, 9.8, 0.36, { fontSize: 19, bold: true, color: navy });
      text(slide, row[2], 2.08, y + 0.55, 9.85, 0.32, { fontSize: 14, color: muted });
    });
    text(slide, "A constructive channel for owner evidence can help JLR rebuild trust as future models arrive.", 0.86, 6.57, 11.3, 0.26, { fontSize: 12, color: muted });

    slide = baseSlide(pptx, "How the group formed and grew", ++index, date);
    text(slide, "A call to action in the UK I-PACE Forum brought owners together around battery, recall and support concerns.", 0.85, 1.55, 11.6, 1.0, { fontSize: 24, color: navy, bold: true });
    text(slide, "Official launch: 17 July 2026. The independent, UK-led group gives JLR a direct route to hear owners’ priorities and shape a fair, consistent response.", 0.85, 2.70, 11.2, 0.80, { fontSize: 19 });
    var timeline = (adminStats.consentedJoinTimeline || []).filter(function (point) { return point.label >= "2026-07-17"; }).sort(function (a, b) { return a.label.localeCompare(b.label); });
    var joined = 0;
    var milestones = [];
    timeline.forEach(function (point) { joined += Number(point.count || 0); milestones.push({ date: point.label, count: joined }); });
    if (joined > 0) {
      [0.25, 0.5, 0.75, 1].forEach(function (fraction, i) {
        var target = Math.ceil(joined * fraction);
        var reached = milestones.find(function (point) { return point.count >= target; });
        text(slide, number(target) + " by " + reached.date, 0.86 + i * 3.0, 3.75, 2.8, 0.26, { fontSize: 12, color: muted, bold: true });
      });
    }
    wreathCounter(slide, wreathData, joined, "Members joined", 2.17, 4.18, 1.82);
    wreathCounter(slide, wreathData, publicStats.vehiclesRegistered, "Cars registered", 8.30, 4.18, 1.82);
    text(slide, "Member figure: first contact-consenting Join from 17 July; pre-launch test members excluded. Cars: published consented total (" + label(publicStats.generatedAt || date, 24) + ").", 0.85, 6.53, 11.6, 0.28, { fontSize: 10, color: muted });

    slide = baseSlide(pptx, "Vehicle model years recorded", ++index, date);
    var years = (publicStats.modelYearDistribution || []).filter(function (row) { return row.count > 0; });
    var yearMax = Math.max.apply(null, years.map(function (row) { return row.count; }).concat([1]));
    years.slice(0, 8).forEach(function (row, i) { chartRow(slide, pptx, row.label, row.count, yearMax, 0.86, 1.5 + i * 0.59, 5.45, teal); });
    text(slide, "Overall consented vehicle records, not a model-year breakdown of survey respondents. Model year is shown where known.", 0.86, 6.5, 11.4, 0.35, { fontSize: 12, color: muted });

    var groups = optionGroups(report.survey);
    var primary = [groups.replacement, groups.buyback, groups.neither].filter(Boolean);
    var extras = [groups.compensation, groups.concerns].filter(Boolean);
    slide = baseSlide(pptx, "What members were asked", ++index, date);
    text(slide, label(report.survey.question || "Which outcomes would you support?", 180), 0.85, 1.42, 11.5, 0.55, { fontSize: 21, bold: true, color: navy });
    text(slide, "PRIMARY OUTCOMES", 0.85, 2.1, 5.7, 0.3, { fontSize: 12, bold: true, color: teal });
    primary.forEach(function (option, i) {
      var y = 2.48 + i * 0.93;
      box(slide, pptx, 0.84, y, 7.55, 0.8, "FFFFFF", "D8E0E5");
      text(slide, "□", 1.04, y + 0.13, 0.35, 0.4, { fontSize: 21, color: teal });
      text(slide, optionText(option, "Outcome"), 1.47, y + 0.1, 6.55, 0.28, { fontSize: 15, bold: true, color: navy });
      text(slide, label(option.description || "", 155), 1.47, y + 0.4, 6.55, 0.3, { fontSize: 11, color: muted });
    });
    text(slide, "IN ADDITION", 8.7, 2.1, 3.7, 0.3, { fontSize: 12, bold: true, color: teal });
    extras.forEach(function (option, i) {
      var y = 2.48 + i * 1.4;
      box(slide, pptx, 8.68, y, 3.8, 1.24, "E8F4F2", "E8F4F2");
      text(slide, "□ " + optionText(option, "Additional choice"), 8.9, y + 0.16, 3.36, 0.38, { fontSize: 15, bold: true, color: navy });
      text(slide, label(option.description || "", 160), 8.9, y + 0.56, 3.34, 0.51, { fontSize: 11, color: muted, valign: "top" });
    });
    callout(slide, pptx, "How the form worked", "Members could select more than one choice and mark an eligible primary outcome as preferred. Compensation and other concerns could accompany HV replacement or buy-back.", 0.85, 5.39, 11.62, 1.23);

    slide = baseSlide(pptx, "A completed member response — fictional", ++index, date);
    text(slide, "DEMONSTRATION ONLY  •  Invented choices and comments; never submitted", 0.86, 1.36, 11.6, 0.3, { fontSize: 13, bold: true, color: red });
    box(slide, pptx, 0.85, 1.76, 5.67, 4.96, "FFFFFF", "D8E0E5");
    box(slide, pptx, 6.8, 1.76, 5.67, 4.96, "FFFFFF", "D8E0E5");
    text(slide, "MEMBER SURVEY", 1.08, 1.96, 4.8, 0.2, { fontSize: 10, bold: true, color: teal });
    text(slide, label(report.survey.title || "Preferred outcomes", 58), 1.08, 2.19, 5.16, 0.39, { fontSize: 19, bold: true, color: navy });
    box(slide, pptx, 1.08, 2.66, 5.16, 0.57, "F7F8FB", "F7F8FB");
    slide.addShape(pptx.ShapeType.rect, { x: 1.08, y: 2.66, w: 0.06, h: 0.57, line: { color: navy, transparency: 100 }, fill: { color: navy } });
    text(slide, label(report.survey.question || "Which outcomes would you support?", 145), 1.28, 2.75, 4.72, 0.38, { fontSize: 12, color: navy });
    text(slide, "Select every outcome you would support", 1.08, 3.35, 5.15, 0.28, { fontSize: 15, bold: true, color: navy });
    text(slide, "You can choose more than one option.", 1.08, 3.65, 5.15, 0.2, { fontSize: 10, color: muted });
    memberChoice(slide, pptx, groups.replacement, 1, true, true, "Battery work has needed repeat visits.", 1.08, 3.94, 5.16, 1.69);
    memberChoice(slide, pptx, groups.buyback, 2, false, false, "", 1.08, 5.68, 5.16, 0.46);
    memberChoice(slide, pptx, groups.neither, 3, false, false, "", 1.08, 6.22, 5.16, 0.46);
    text(slide, "FORM CONTINUES ↓", 7.03, 1.96, 5.0, 0.2, { fontSize: 10, bold: true, color: teal });
    text(slide, "The same member form, scrolled down", 7.03, 2.2, 5.05, 0.35, { fontSize: 17, bold: true, color: navy });
    memberChoice(slide, pptx, groups.compensation, 4, true, false, "Time without the car deserves recognition.", 7.03, 2.66, 5.16, 1.39);
    memberChoice(slide, pptx, groups.concerns, 5, true, false, "Please also fix the air-conditioning fault.", 7.03, 4.16, 5.16, 1.39);
    box(slide, pptx, 7.03, 5.7, 2.45, 0.47, teal, teal);
    text(slide, "Update your response", 7.18, 5.8, 2.17, 0.25, { fontSize: 12, bold: true, color: "FFFFFF", align: "center" });
    text(slide, "Selected cards, numbered options, preferred checkbox and option-level text mirror the member response layout.", 7.03, 6.31, 5.08, 0.29, { fontSize: 10, color: muted });

    slide = baseSlide(pptx, "Survey participation", ++index, date);
    var classifiedComments = (report.items || []).filter(function (item) { return item.sentiment !== "unclassified"; }).length;
    wreathCounter(slide, wreathData, report.totalResponses, "Survey responses", 2.05, 1.67, 2.55);
    wreathCounter(slide, wreathData, (report.items || []).length, "Optional comments", 8.72, 1.67, 2.55);
    box(slide, pptx, 1.08, 5.12, 11.1, 0.83, "E8F4F2", "E8F4F2");
    text(slide, number(classifiedComments) + " classified comments", 1.42, 5.36, 5.1, 0.34, { fontSize: 19, color: navy, bold: true });
    text(slide, number((report.items || []).length - classifiedComments) + " unclassified", 7.15, 5.36, 4.5, 0.34, { fontSize: 19, color: navy, bold: true, align: "right" });
    text(slide, "Results describe members who chose to answer this survey; optional comments are counted per selected option.", 1.08, 6.32, 11.1, 0.43, { fontSize: 14, color: muted });

    slide = baseSlide(pptx, "Three primary routes owners considered", ++index, date);
    var allOptions = primary.concat(extras);
    var primaryMax = Math.max.apply(null, allOptions.map(function (option) { return report.counts[option.id] || 0; }).concat([1]));
    var leading = primary.slice().sort(function (left, right) {
      return ((report.counts || {})[right.id] || 0) - ((report.counts || {})[left.id] || 0) ||
        ((report.preferredCounts || {})[right.id] || 0) - ((report.preferredCounts || {})[left.id] || 0);
    });
    var leader = leading.length && (report.counts[leading[0].id] || 0) > 0 && (!leading[1] ||
      (report.counts[leading[0].id] || 0) !== (report.counts[leading[1].id] || 0) ||
      ((report.preferredCounts || {})[leading[0].id] || 0) !== ((report.preferredCounts || {})[leading[1].id] || 0)) ? leading[0] : null;
    box(slide, pptx, 0.86, 1.38, 11.6, 0.68, "E8F4F2", "E8F4F2");
    text(slide, "WINNER", 1.08, 1.56, 1.8, 0.25, { fontSize: 13, bold: true, color: teal });
    text(slide, leader ? optionText(leader, "Outcome") : "No single winner", 3.0, 1.51, 8.8, 0.35, { fontSize: 19, bold: true, color: navy });
    allOptions.forEach(function (option, i) {
      var count = report.counts[option.id] || 0;
      var y = 2.17 + i * 0.86;
      box(slide, pptx, 0.86, y, 11.6, 0.77, "FFFFFF", "D8E0E5");
      text(slide, optionText(option, "Outcome"), 1.05, y + 0.10, 5.25, 0.28, { fontSize: 16, bold: true, color: navy });
      if (i >= primary.length) text(slide, "ADDITIONAL REQUEST", 5.9, y + 0.13, 2.2, 0.22, { fontSize: 9, color: muted, bold: true });
      slide.addShape(pptx.ShapeType.rect, { x: 1.05, y: y + 0.48, w: 6.2, h: 0.13, line: { color: "D8E0E5", transparency: 100 }, fill: { color: "D8E0E5" } });
      if (count) slide.addShape(pptx.ShapeType.rect, { x: 1.05, y: y + 0.48, w: 6.2 * count / primaryMax, h: 0.13, line: { color: teal, transparency: 100 }, fill: { color: teal } });
      text(slide, number(count) + " votes · " + percent(count, report.totalResponses), 7.45, y + 0.11, 2.0, 0.27, { fontSize: 14, bold: true, color: teal, align: "right" });
      text(slide, (i < primary.length ? number((report.preferredCounts || {})[option.id]) + " preferred · " : "") + number((report.textCounts || {})[option.id]) + " comments", 9.55, y + 0.11, 2.58, 0.27, { fontSize: 12, color: muted, align: "right" });
    });
    text(slide, "Winner uses most votes; preferred votes break a primary-route tie. Additional requests can accompany a primary route.", 0.86, 6.55, 11.4, 0.25, { fontSize: 11, color: muted });

    slide = baseSlide(pptx, "Survey findings at a glance", ++index, date);
    var alternative = leading.length > 1 ? leading[1] : null;
    var topTheme = (report.themeCounts || [])[0];
    [
      ["MOST SELECTED ROUTE", leader ? optionText(leader, "Outcome") : "No single winner", leader ? number(report.counts[leader.id]) + " selections · " + number((report.preferredCounts || {})[leader.id]) + " preferred" : "Primary routes tied on votes and preference"],
      ["STRONGEST ALTERNATIVE", alternative ? optionText(alternative, "Outcome") : "No alternative recorded", alternative ? number(report.counts[alternative.id]) + " selections" : ""],
      ["ADDITIONAL REQUEST", optionText(groups.compensation, "Fair compensation"), number(groups.compensation && report.counts[groups.compensation.id]) + " selections"],
      ["TOP COMMENT CONCERN", topTheme ? themeLabels[topTheme.name] || topTheme.name : "No classified theme", topTheme ? number(topTheme.count) + " comment entries" : "No classified comments"]
    ].forEach(function (row, i) {
      var x = 0.86 + (i % 2) * 5.91;
      var y = 1.50 + Math.floor(i / 2) * 2.18;
      box(slide, pptx, x, y, 5.55, 1.86, i === 0 ? "E8F4F2" : "FFFFFF", "D8E0E5");
      text(slide, row[0], x + 0.24, y + 0.20, 4.95, 0.25, { fontSize: 11, bold: true, color: teal });
      text(slide, row[1], x + 0.24, y + 0.61, 5.05, 0.51, { fontSize: 21, bold: true, color: navy });
      text(slide, row[2], x + 0.24, y + 1.30, 5.05, 0.29, { fontSize: 15, color: muted });
    });
    text(slide, "Members could select several routes and additional requests; these counts are not a head-to-head ballot. Comment concerns are AI-assisted labels, not fleet prevalence.", 0.86, 6.35, 11.4, 0.47, { fontSize: 12, color: muted });

    slide = baseSlide(pptx, "Recurring themes in owners’ words", ++index, date);
    text(slide, "KEY CONCERN", 1.08, 1.53, 3.2, 0.25, { fontSize: 12, bold: true, color: muted });
    text(slide, "COMMENTS", 5.1, 1.53, 1.4, 0.25, { fontSize: 12, bold: true, color: muted });
    text(slide, "OPTION(S) WHERE RAISED", 6.65, 1.53, 5.3, 0.25, { fontSize: 12, bold: true, color: muted });
    report.themeCounts.slice(0, 7).forEach(function (row, i) {
      var y = 1.94 + i * 0.64;
      box(slide, pptx, 0.86, y, 11.6, 0.56, i % 2 ? "F0F9F7" : "FFFFFF", "D8E0E5");
      var byOption = {};
      (report.items || []).forEach(function (item) { if ((item.themes || []).indexOf(row.name) !== -1) byOption[item.optionId] = (byOption[item.optionId] || 0) + 1; });
      var names = allOptions.filter(function (option) { return byOption[option.id]; }).map(function (option) { return optionText(option, "Outcome"); });
      text(slide, themeLabels[row.name] || row.name, 1.08, y + 0.11, 3.9, 0.31, { fontSize: 16, bold: true, color: navy });
      text(slide, number(row.count), 5.12, y + 0.11, 1.2, 0.31, { fontSize: 17, bold: true, color: teal });
      text(slide, label(names.join(", ") || "Not specified", 100), 6.65, y + 0.10, 5.55, 0.35, { fontSize: 13, color: navy });
    });
    text(slide, "AI-assisted labels from optional comments. A comment can raise several themes; these counts are comments, not vehicles or votes.", 0.86, 6.62, 11.45, 0.25, { fontSize: 11, color: muted });

    ["good", "bad", "ugly"].forEach(function (kind) {
      var accent = kind === "good" ? teal : kind === "bad" ? gold : red;
      slide = baseSlide(pptx, "Owner voices — the " + kind, ++index, date);
      var selected = report.quotes.filter(function (quote) { return quote.kind === kind; }).slice(0, 3);
      selected.forEach(function (quote, i) {
        var y = 1.42 + i * 1.62;
        box(slide, pptx, 0.85, y, 11.62, 1.43, "FFFFFF", "D8E0E5");
        slide.addShape(pptx.ShapeType.rect, { x: 0.85, y: y, w: 0.12, h: 1.43, line: { color: accent, transparency: 100 }, fill: { color: accent } });
        text(slide, optionText((report.survey.options || []).find(function (option) { return option.id === quote.optionId; }), "Survey option") + "  •  " + (quote.sentiment || "unlabelled") + " experience", 1.19, y + 0.14, 10.8, 0.26, { fontSize: 12, bold: true, color: accent });
        text(slide, "“", 1.15, y + 0.39, 0.45, 0.55, { fontSize: 40, bold: true, color: accent });
        text(slide, label(quote.text, 400), 1.72, y + 0.47, 9.85, 0.8, { fontSize: 18, color: navy, valign: "mid" });
      });
      if (!selected.length) text(slide, "No permission-cleared quote was selected for this category.", 0.86, 2.14, 11.5, 0.5, { fontSize: 20, color: muted });
      text(slide, "Anonymous, permission-checked survey comments chosen by an administrator. Quote category and experience sentiment are separate AI labels.", 0.86, 6.43, 11.4, 0.34, { fontSize: 12, color: muted });
    });

    var phrases = signalCounts(report.items);
    slide = baseSlide(pptx, "Sentiment in the optional comments", ++index, date);
    allOptions.forEach(function (option, i) {
      var row = report.sentiment[option.id] || { positive: 0, mixed: 0, negative: 0, unclassified: 0 };
      var total = row.positive + row.mixed + row.negative;
      var y = 1.48 + i * 0.98;
      text(slide, label(optionText(option, "Outcome"), 35), 0.86, y, 3.55, 0.32, { fontSize: 15, bold: true, color: navy });
      var x = 4.45;
      ["positive", "mixed", "negative"].forEach(function (kind) {
        var width = total ? 5.1 * row[kind] / total : 0;
        if (width) slide.addShape(pptx.ShapeType.rect, { x: x, y: y + 0.07, w: width, h: 0.25, line: { color: sentimentColors[kind], transparency: 100 }, fill: { color: sentimentColors[kind] } });
        x += width;
      });
      text(slide, "+" + row.positive + " / ±" + row.mixed + " / −" + row.negative + "  (U " + (row.unclassified || 0) + ")", 9.75, y, 2.5, 0.34, { fontSize: 12, color: navy, align: "right" });
      var optionPhrases = phrases.filter(function (phrase) { return phrase.optionId === option.id; });
      var positivePhrase = optionPhrases.slice().sort(function (a, b) { return b.positive - a.positive; })[0];
      var negativePhrase = optionPhrases.slice().sort(function (a, b) { return b.negative - a.negative; })[0];
      text(slide, positivePhrase && positivePhrase.positive ? "+ " + signalLabels[positivePhrase.name] + " " + positivePhrase.positive : "No positive phrase tag", 0.86, y + 0.46, 5.42, 0.28, { fontSize: 12, color: teal });
      text(slide, negativePhrase && negativePhrase.negative ? "− " + signalLabels[negativePhrase.name] + " " + negativePhrase.negative : "No negative phrase tag", 6.64, y + 0.46, 5.42, 0.28, { fontSize: 12, color: red });
    });
    text(slide, "Bars count classified comments per option. Phrase tags are controlled labels from those comments; U is unclassified. A member may comment on several options.", 0.86, 6.61, 11.4, 0.27, { fontSize: 11, color: muted });

    slide = baseSlide(pptx, "Phrases shaping customer confidence", ++index, date);
    allOptions.forEach(function (option, i) {
      var x = 0.86 + (i % 3) * 3.94;
      var y = 1.47 + Math.floor(i / 3) * 2.52;
      box(slide, pptx, x, y, 3.67, 2.31, "FFFFFF", "D8E0E5");
      text(slide, label(optionText(option, "Outcome"), 32), x + 0.18, y + 0.17, 3.28, 0.37, { fontSize: 16, bold: true, color: navy });
      var ranked = phrases.filter(function (phrase) { return phrase.optionId === option.id; }).sort(function (a, b) { return b.count - a.count || a.name.localeCompare(b.name); }).slice(0, 4);
      if (!ranked.length) text(slide, "No specific phrase tags", x + 0.19, y + 0.89, 3.25, 0.36, { fontSize: 14, color: muted });
      ranked.forEach(function (phrase, j) {
        var tone = phrase.positive > phrase.negative ? teal : phrase.negative > phrase.positive ? red : gold;
        text(slide, signalLabels[phrase.name] + "  " + number(phrase.count), x + 0.19 + (j % 2) * 0.12, y + 0.68 + j * 0.37, 3.18, 0.34, { fontSize: 14 + Math.min(7, phrase.count * 0.6), bold: true, color: tone });
      });
    });
    text(slide, "Colour shows the dominant experience: positive, mixed or negative; size follows comment frequency. For JLR discussion: negative language can affect confidence in future launches, while positive language shows what to reinforce. This is not a sales forecast.", 8.75, 4.27, 3.58, 1.8, { fontSize: 13, color: muted, valign: "top" });

    slide = baseSlide(pptx, "Next steps for JLR Client Care", ++index, date);
    box(slide, pptx, 0.86, 1.42, 11.52, 0.73, "E8F4F2", "E8F4F2");
    text(slide, "Survey winner: " + (leader ? optionText(leader, "Outcome") : "no single route"), 1.12, 1.59, 10.9, 0.4, { fontSize: 21, bold: true, color: navy });
    text(slide, "Three actions to agree", 0.86, 2.36, 8.0, 0.34, { fontSize: 17, bold: true, color: teal });
    report.actions.forEach(function (action, i) {
      var y = 2.83 + i * 1.14;
      box(slide, pptx, 0.86, y, 11.52, 1.03, "FFFFFF", "D8E0E5");
      text(slide, String(i + 1).padStart(2, "0"), 1.10, y + 0.25, 0.7, 0.45, { fontSize: 26, bold: true, color: teal });
      text(slide, String(action), 1.95, y + 0.13, 10.05, 0.76, { fontSize: 17, color: navy, valign: "mid" });
    });
    text(slide, "Reliable resolution and clear communication are an opportunity to rebuild owner confidence. Proposed discussion points; no JLR commitment is implied.", 0.86, 6.54, 11.4, 0.34, { fontSize: 12, color: muted });

    slide = baseSlide(pptx, "Decisions and dates", ++index, date);
    [
      ["Outcome", "Agree what a dependable repair means and when a fair buy-back choice is appropriate."],
      ["JLR plan", "Understand work completed, planned work, constraints, owners and realistic dates."],
      ["Confidentiality", "Agree what needs protection; discuss a narrow arrangement for sensitive workstream detail if necessary."],
      ["Member update", "Ask whether a board-level written statement can be provided before 5 October; agree an owner and date."]
    ].forEach(function (row, i) {
      var y = 1.47 + i * 1.24;
      box(slide, pptx, 0.86, y, 11.55, 1.08, i % 2 ? "F0F9F7" : "FFFFFF", "D8E0E5");
      text(slide, row[0], 1.12, y + 0.18, 2.25, 0.42, { fontSize: 20, bold: true, color: teal });
      text(slide, row[1], 3.48, y + 0.16, 8.5, 0.74, { fontSize: 17, color: navy });
    });
    text(slide, "Proposed discussion outcomes; no JLR agreement or board statement is assumed.", 0.86, 6.64, 11.4, 0.25, { fontSize: 11, color: muted });
    // PptxGenJS 4 emits content-type entries for slide masters that do not
    // exist in the archive. Remove only those orphan entries before download.
    return pptx.write({ outputType: "uint8array", compression: true }).then(function (bytes) {
      return window.JSZip.loadAsync(bytes);
    }).then(function (zip) {
      return zip.file("[Content_Types].xml").async("string").then(function (contentTypes) {
        var repaired = contentTypes.replace(/<Override PartName="\/(ppt\/slideMasters\/slideMaster\d+\.xml)"[^>]*\/>/g, function (entry, part) {
          return zip.file(part) ? entry : "";
        });
        zip.file("[Content_Types].xml", repaired);
        return zip.generateAsync({ type: "uint8array", compression: "DEFLATE" });
      });
    }).then(function (bytes) {
      if (saveForTest) return saveForTest(bytes);
      var blob = new Blob([bytes], { type: "application/vnd.openxmlformats-officedocument.presentationml.presentation" });
      var url = window.URL.createObjectURL(blob);
      var link = document.createElement("a");
      link.href = url;
      link.download = "ipace-owner-survey-jlr-24-september-2026.pptx";
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(function () { window.URL.revokeObjectURL(url); }, 60000);
    });
    });
  };
})();
