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
    var slide = pptx.addSlide();
    slide.background = { color: pale };
    text(slide, title, 0.75, 0.42, 11.9, 0.68, { color: navy, fontSize: 28, bold: true });
    slide.addShape(pptx.ShapeType.line, { x: 0.75, y: 1.2, w: 11.85, h: 0, line: { color: teal, width: 2 } });
    text(slide, "I-PACE Owners’ Advocacy Group  •  Private JLR meeting  •  " + date, 0.75, 7.06, 11.5, 0.2, { color: muted, fontSize: 9 });
    text(slide, String(index), 12.2, 7.06, 0.35, 0.2, { color: muted, fontSize: 9, align: "right" });
    return slide;
  }

  function chartRow(slide, pptx, caption, value, max, x, y, width, color, suffix) {
    text(slide, label(caption, 48), x, y, 4.55, 0.4, { fontSize: 15 });
    slide.addShape(pptx.ShapeType.rect, { x: x + 4.65, y: y + 0.1, w: width, h: 0.2, line: { color: "D8E0E5", transparency: 100 }, fill: { color: "D8E0E5" } });
    if (value > 0 && max > 0) slide.addShape(pptx.ShapeType.rect, { x: x + 4.65, y: y + 0.1, w: width * value / max, h: 0.2, line: { color: color, transparency: 100 }, fill: { color: color } });
    text(slide, number(value) + (suffix || ""), x + 4.75 + width, y, 1.3, 0.4, { fontSize: 14, bold: true, color: navy, align: "right" });
  }

  function compactChartRow(slide, pptx, caption, value, max, x, y) {
    text(slide, label(caption, 22), x, y, 2.4, 0.38, { fontSize: 13 });
    slide.addShape(pptx.ShapeType.rect, { x: x + 2.45, y: y + 0.1, w: 2.15, h: 0.19, line: { color: "D8E0E5", transparency: 100 }, fill: { color: "D8E0E5" } });
    if (value > 0 && max > 0) slide.addShape(pptx.ShapeType.rect, { x: x + 2.45, y: y + 0.1, w: 2.15 * value / max, h: 0.19, line: { color: teal, transparency: 100 }, fill: { color: teal } });
    text(slide, number(value), x + 4.7, y, 0.7, 0.38, { fontSize: 13, bold: true, color: navy, align: "right" });
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

  function checkInput(report, adminStats, publicStats) {
    if (!report || !report.survey || !Array.isArray(report.survey.options) || !Array.isArray(report.items) || !Array.isArray(report.quotes) || !Array.isArray(report.actions) || report.actions.length !== 3 || !adminStats || !publicStats) throw new Error("Incomplete report");
    if (report.quotes.length > 9) throw new Error("Too many quotes");
  }

  window.ipaceMakeSurveyDeck = function (report, adminStats, publicStats, saveForTest, assets) {
    checkInput(report, adminStats, publicStats);
    if (!window.PptxGenJS) return Promise.reject(new Error("PowerPoint library unavailable"));
    if (!window.JSZip) return Promise.reject(new Error("PowerPoint package validator unavailable"));
    function loadHero() {
      if (assets && assets.heroData) return Promise.resolve(assets.heroData);
      return fetch("/images/september-survey-reminder-2026-hero.jpg").then(function (response) {
        if (!response.ok) throw new Error("Meeting hero image unavailable");
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
    return loadHero().then(function (heroData) {
    var pptx = new window.PptxGenJS();
    pptx.layout = "LAYOUT_WIDE";
    pptx.author = "I-PACE Owners’ Advocacy Group";
    pptx.subject = "Anonymous member survey analysis for the 24 September 2026 JLR meeting";
    pptx.title = "I-PACE owners: member survey findings";
    pptx.lang = "en-GB";
    var date = new Date().toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });
    var index = 1;
    var slide = pptx.addSlide();
    slide.addImage({ data: heroData, x: 0, y: 0, w: 13.333, h: 7.5 });
    slide.addShape(pptx.ShapeType.rect, { x: 0, y: 4.3, w: 13.333, h: 3.2, line: { color: navy, transparency: 100 }, fill: { color: navy, transparency: 4 } });
    text(slide, "I-PACE owners: a route to resolution", 0.75, 4.72, 11.8, 0.69, { color: "FFFFFF", fontSize: 35, bold: true });
    text(slide, "Member survey findings and owner priorities", 0.78, 5.48, 11.8, 0.44, { color: "FFFFFF", fontSize: 23 });
    text(slide, "UK Director for Client Care  •  24 September 2026  •  Private meeting", 0.78, 6.74, 11.8, 0.31, { color: "DCE9E9", fontSize: 14 });

    slide = baseSlide(pptx, "How the group formed and grew", ++index, date);
    text(slide, "A call to action in the UK I-PACE Forum brought owners together around battery, recall and support concerns.", 0.85, 1.55, 11.6, 1.0, { fontSize: 24, color: navy, bold: true });
    text(slide, "The group became an independent, consent-based way to collect owners’ experiences and priorities for constructive discussion with JLR.", 0.85, 2.78, 11.2, 1.0, { fontSize: 21 });
    var timeline = (adminStats.consentedJoinTimeline || []).slice().sort(function (a, b) { return a.label.localeCompare(b.label); });
    var joined = 0;
    var milestones = [];
    timeline.forEach(function (point) { joined += Number(point.count || 0); milestones.push({ date: point.label, count: joined }); });
    if (joined > 0) {
      [0.25, 0.5, 0.75, 1].forEach(function (fraction, i) {
        var target = Math.ceil(joined * fraction);
        var reached = milestones.find(function (point) { return point.count >= target; });
        text(slide, number(target) + " by " + reached.date, 0.86 + i * 3.0, 3.93, 2.8, 0.43, { fontSize: 13, color: muted, bold: true });
      });
    }
    text(slide, number(publicStats.joinedOwners), 0.85, 4.85, 4.2, 0.85, { fontSize: 43, bold: true, color: teal });
    text(slide, "owners joined at latest published count", 0.85, 5.68, 5.5, 0.42, { fontSize: 17, color: muted });
    text(slide, number(publicStats.vehiclesRegistered), 7.15, 4.85, 4.2, 0.85, { fontSize: 43, bold: true, color: teal });
    text(slide, "consented vehicles recorded", 7.15, 5.68, 4.7, 0.42, { fontSize: 17, color: muted });
    text(slide, "Growth milestones: consented member joins. Current totals: published site statistics (" + label(publicStats.generatedAt || date, 24) + ").", 0.85, 6.5, 11.6, 0.32, { fontSize: 11, color: muted });

    slide = baseSlide(pptx, "Survey participation and main finding", ++index, date);
    text(slide, number(report.totalResponses), 0.85, 1.58, 4.5, 1.1, { fontSize: 62, bold: true, color: teal });
    text(slide, "member responses", 0.86, 2.66, 5.3, 0.45, { fontSize: 21, color: navy });
    text(slide, label(report.overview, 650), 6.1, 1.64, 6.15, 3.6, { fontSize: 23, color: navy, valign: "top" });
    text(slide, "Respondents could select more than one outcome and optionally identify a preferred one. Comments were optional. Results describe this self-selected group.", 0.86, 5.75, 11.5, 0.85, { fontSize: 16, color: muted });

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

    slide = baseSlide(pptx, "A completed response — fictional example", ++index, date);
    text(slide, "ILLUSTRATION ONLY  •  No member data", 0.86, 1.43, 11.6, 0.35, { fontSize: 14, bold: true, color: red });
    box(slide, pptx, 0.85, 1.93, 7.5, 3.77, "FFFFFF", "D8E0E5");
    var exampleRows = [
      [groups.replacement, true, true], [groups.buyback, false, false], [groups.neither, false, false],
      [groups.compensation, true, false], [groups.concerns, true, false]
    ].filter(function (row) { return row[0]; });
    exampleRows.forEach(function (row, i) {
      var y = 2.13 + i * 0.67;
      text(slide, (row[1] ? "☑" : "□") + "  " + optionText(row[0], "Choice"), 1.13, y, 5.9, 0.34, { fontSize: 17, color: navy, bold: row[1] });
      if (row[2]) text(slide, "◎ Preferred", 6.42, y, 1.55, 0.3, { fontSize: 12, color: teal, bold: true });
    });
    callout(slide, pptx, "Example comment", "“I want a reliable full battery solution without repeat visits. Please also address the air-conditioning fault and the time without my car.”", 8.67, 1.93, 3.78, 2.6);
    text(slide, "The example demonstrates how an option-level comment is read with the member’s other selections and preferred outcome. It is invented for this slide.", 0.86, 6.06, 11.45, 0.56, { fontSize: 15, color: muted });

    slide = baseSlide(pptx, "Three primary routes owners considered", ++index, date);
    var primaryMax = Math.max.apply(null, primary.map(function (option) { return report.counts[option.id] || 0; }).concat([1]));
    primary.forEach(function (option, i) {
      var count = report.counts[option.id] || 0;
      var y = 1.6 + i * 1.32;
      text(slide, optionText(option, "Outcome"), 0.86, y, 6.5, 0.38, { fontSize: 21, bold: true, color: navy });
      slide.addShape(pptx.ShapeType.rect, { x: 0.86, y: y + 0.57, w: 8.55, h: 0.28, line: { color: "D8E0E5", transparency: 100 }, fill: { color: "D8E0E5" } });
      if (count) slide.addShape(pptx.ShapeType.rect, { x: 0.86, y: y + 0.57, w: 8.55 * count / primaryMax, h: 0.28, line: { color: teal, transparency: 100 }, fill: { color: teal } });
      text(slide, number(count) + "  ·  " + percent(count, report.totalResponses), 9.68, y + 0.36, 2.6, 0.45, { fontSize: 22, bold: true, color: teal, align: "right" });
    });
    callout(slide, pptx, "Additional requests", extras.map(function (option) { return optionText(option, "Choice") + ": " + number(report.counts[option.id] || 0); }).join("   •   "), 0.86, 5.42, 11.6, 1.08);
    text(slide, "Share of " + number(report.totalResponses) + " respondents. Choices could overlap; add-on selections are not alternative primary routes.", 0.86, 6.62, 11.4, 0.28, { fontSize: 12, color: muted });

    slide = baseSlide(pptx, "Sentiment in the optional comments", ++index, date);
    var sentimentOptions = primary.concat(extras);
    sentimentOptions.forEach(function (option, i) {
      var row = report.sentiment[option.id] || { positive: 0, mixed: 0, negative: 0 };
      var total = row.positive + row.mixed + row.negative;
      var y = 1.52 + i * 0.71;
      text(slide, label(optionText(option, "Outcome"), 37), 0.86, y, 3.6, 0.38, { fontSize: 14, bold: i < primary.length });
      var x = 4.55;
      ["positive", "mixed", "negative"].forEach(function (kind) {
        var width = total ? 5.7 * row[kind] / total : 0;
        if (width) slide.addShape(pptx.ShapeType.rect, { x: x, y: y + 0.06, w: width, h: 0.28, line: { color: sentimentColors[kind], transparency: 100 }, fill: { color: sentimentColors[kind] } });
        x += width;
      });
      text(slide, row.positive + " / " + row.mixed + " / " + row.negative + "  (n=" + total + ")", 10.42, y, 1.89, 0.4, { fontSize: 11, color: navy, align: "right" });
    });
    text(slide, "Positive", 0.86, 5.42, 1.2, 0.3, { fontSize: 13, color: teal, bold: true });
    text(slide, "Mixed", 2.1, 5.42, 1.1, 0.3, { fontSize: 13, color: gold, bold: true });
    text(slide, "Negative", 3.24, 5.42, 1.4, 0.3, { fontSize: 13, color: red, bold: true });
    callout(slide, pptx, "How we arrived at these labels", "Each optional comment was labelled in the context of its option, all selected choices and preferred choice. Figures are positive / mixed / negative comment entries; n is the sum. Unanswered options carry no sentiment.", 0.85, 5.68, 11.62, 1.2);

    slide = baseSlide(pptx, "Recurring themes in owners’ words", ++index, date);
    var themes = report.themeCounts.slice(0, 7);
    var themeMax = Math.max.apply(null, themes.map(function (row) { return row.count; }).concat([1]));
    themes.forEach(function (row, i) {
      var size = 16 + 10 * row.count / themeMax;
      var x = 0.86 + (i % 3) * 3.93;
      var y = 1.6 + Math.floor(i / 3) * 1.38;
      box(slide, pptx, x, y, 3.58, 1.12, i % 2 ? "E8F4F2" : "FFFFFF", "D8E0E5");
      text(slide, themeLabels[row.name] || row.name, x + 0.2, y + 0.18, 3.13, 0.38, { fontSize: size, bold: true, color: navy });
      text(slide, number(row.count) + " comment entries", x + 0.2, y + 0.69, 3.13, 0.24, { fontSize: 12, color: teal });
    });
    text(slide, "Size and count use the same frequency. AI-assisted theme labels use a fixed vocabulary; one comment can mention several themes. Counts are comment entries, not vehicles or owners.", 0.86, 6.32, 11.45, 0.54, { fontSize: 13, color: muted });

    ["good", "bad", "ugly"].forEach(function (kind) {
      var accent = kind === "good" ? teal : kind === "bad" ? gold : red;
      slide = baseSlide(pptx, "Owner voices — the " + kind, ++index, date);
      var selected = report.quotes.filter(function (quote) { return quote.kind === kind; }).slice(0, 3);
      selected.forEach(function (quote, i) {
        var y = 1.42 + i * 1.62;
        box(slide, pptx, 0.85, y, 11.62, 1.43, "FFFFFF", "D8E0E5");
        slide.addShape(pptx.ShapeType.rect, { x: 0.85, y: y, w: 0.12, h: 1.43, line: { color: accent, transparency: 100 }, fill: { color: accent } });
        text(slide, "“", 1.14, y + 0.03, 0.45, 0.58, { fontSize: 41, bold: true, color: accent });
        text(slide, label(quote.text, 400), 1.72, y + 0.19, 9.85, 1.0, { fontSize: 19, color: navy, valign: "mid" });
      });
      if (!selected.length) text(slide, "No permission-cleared quote was selected for this category.", 0.86, 2.14, 11.5, 0.5, { fontSize: 20, color: muted });
      text(slide, "Anonymous, permission-checked survey comments chosen by an administrator. " + selected.length + " of 3 possible quotes shown; no quotation has been invented.", 0.86, 6.43, 11.4, 0.34, { fontSize: 12, color: muted });
    });

    slide = baseSlide(pptx, "Vehicle model years recorded", ++index, date);
    var years = (publicStats.modelYearDistribution || []).filter(function (row) { return row.count > 0; });
    var yearMax = Math.max.apply(null, years.map(function (row) { return row.count; }).concat([1]));
    years.slice(0, 8).forEach(function (row, i) { chartRow(slide, pptx, row.label, row.count, yearMax, 0.86, 1.5 + i * 0.59, 5.45, teal); });
    text(slide, "Group vehicle records with anonymised-analysis consent; model year where known. These are not necessarily vehicles of survey respondents.", 0.86, 6.5, 11.4, 0.35, { fontSize: 12, color: muted });

    slide = baseSlide(pptx, "Member and service locations recorded", ++index, date);
    text(slide, "Member country", 0.86, 1.45, 5.6, 0.4, { fontSize: 20, bold: true, color: navy });
    text(slide, "Service provider postcode area", 6.78, 1.45, 5.6, 0.4, { fontSize: 20, bold: true, color: navy });
    var countries = adminStats.consentedMemberCountries || [];
    var locations = (adminStats.serviceLocations && adminStats.serviceLocations.areas) || [];
    var countryMax = Math.max.apply(null, countries.map(function (row) { return row.count; }).concat([1]));
    var locationMax = Math.max.apply(null, locations.map(function (row) { return row.count; }).concat([1]));
    countries.slice(0, 7).forEach(function (row, i) { compactChartRow(slide, pptx, row.label, row.count, countryMax, 0.86, 2.0 + i * 0.57); });
    locations.slice(0, 7).forEach(function (row, i) { compactChartRow(slide, pptx, row.area, row.count, locationMax, 6.78, 2.0 + i * 0.57); });
    if (!locations.length) text(slide, "No area breakdown available.", 6.78, 2.1, 5.45, 0.5, { fontSize: 18, color: muted });
    text(slide, "Member country: first Join record with anonymised-analysis consent, where known. Service areas: consent-eligible records with a provider postcode; areas with fewer than five records are combined. Known service locations: " + number((adminStats.serviceLocations || {}).known) + "; unknown: " + number((adminStats.serviceLocations || {}).unknown) + ". Neither distribution describes survey respondents.", 0.86, 6.2, 11.4, 0.73, { fontSize: 11, color: muted });

    slide = baseSlide(pptx, "An opportunity for JLR Client Care", ++index, date);
    text(slide, "A growing group of owners wants dependable resolution to battery, recall and related service concerns, including experiences around H441.", 0.86, 1.42, 11.52, 0.83, { fontSize: 21, bold: true, color: navy });
    report.actions.forEach(function (action, i) {
      var y = 2.46 + i * 1.08;
      box(slide, pptx, 0.86, y, 11.52, 0.92, "FFFFFF", "D8E0E5");
      text(slide, String(i + 1).padStart(2, "0"), 1.08, y + 0.18, 0.7, 0.52, { fontSize: 27, bold: true, color: teal });
      text(slide, label(action, 200), 1.94, y + 0.15, 10.06, 0.58, { fontSize: 19, color: navy });
    });
    text(slide, "Proposed actions for discussion, reviewed by the group. No JLR commitment or recall causality is implied.", 0.86, 6.45, 11.4, 0.32, { fontSize: 13, color: muted });

    slide = baseSlide(pptx, "How to read these findings", ++index, date);
    [
      "Responses and vehicle/service information are self-reported. The survey is self-selected and cannot estimate prevalence across all I-PACE owners.",
      "Survey choice counts use stored aggregates. Comment themes and sentiment are AI-assisted classifications of optional text, reviewed by an administrator.",
      "Comments may cover several themes; sentiment is tied to the option where the comment was entered, with all selected and preferred outcomes supplied as context. Figures show positive / mixed / negative comment entries.",
      "Quoted comments were redacted and selected for this private meeting with confirmed permission. Group demographics are not linked to survey respondents."
    ].forEach(function (item, i) { text(slide, "•  " + item, 0.86, 1.48 + i * 1.21, 11.4, 0.9, { fontSize: 18, color: navy }); });
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
