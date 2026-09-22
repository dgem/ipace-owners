const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const JSZip = require('jszip');

test('browser deck creates an editable PowerPoint with grounded counts and reviewed quotes', async () => {
  const temp = fs.mkdtempSync(path.join(os.tmpdir(), 'ipace-survey-deck-'));
  try {
    const generator = path.resolve('src/assets/js/survey-deck.js');
    const packageRoot = path.resolve('node_modules/pptxgenjs');
    const zipRoot = path.resolve('node_modules/jszip');
    const script = `
      const fs = require('node:fs');
      const vm = require('node:vm');
      const browser = { PptxGenJS: require(${JSON.stringify(packageRoot)}), JSZip: require(${JSON.stringify(zipRoot)}) };
      const heroData = 'data:image/jpeg;base64,' + fs.readFileSync(${JSON.stringify(path.resolve('public/images/september-survey-reminder-2026-hero.jpg'))}).toString('base64');
      const wreathData = 'data:image/png;base64,' + fs.readFileSync(${JSON.stringify(path.resolve('public/images/racing-laurel-email.png'))}).toString('base64');
      vm.runInNewContext(fs.readFileSync(${JSON.stringify(generator)}, 'utf8'), { window: browser, Date, Promise });
      browser.ipaceMakeSurveyDeck({
        survey: { question: 'Which outcome would you support?', options: [
          { id: 'repair', name: 'Full HV Replacement', description: 'Reliable replacement without repeat visits.' },
          { id: 'buyback', name: 'A Fair Buy Back', description: 'A fair price despite affected market value.' },
          { id: 'neither', name: 'None of the above', description: 'Neither primary route meets my needs.' },
          { id: 'compensation', name: 'Fair Compensation', description: 'For disruption and repeat visits.' },
          { id: 'concerns', name: 'Additional Concerns', description: 'Other faults also need rectifying.' }
        ] },
        counts: { repair: 8, buyback: 5, neither: 1, compensation: 3, concerns: 2 }, preferredCounts: { repair: 6, buyback: 2 }, textCounts: { repair: 4, buyback: 3, neither: 1, compensation: 2, concerns: 1 },
        totalResponses: 10, items: [
          { optionId: 'repair', sentiment: 'negative', themes: ['battery', 'service'], signals: [{ name: 'repair-delays', sentiment: 'negative' }] },
          { optionId: 'repair', sentiment: 'positive', themes: ['battery'], signals: [{ name: 'replacement-reliability', sentiment: 'positive' }] },
          { optionId: 'buyback', sentiment: 'negative', themes: ['value'], signals: [{ name: 'resale-value', sentiment: 'negative' }] },
          { optionId: 'concerns', sentiment: 'unclassified', themes: [] }
        ], overview: 'Battery faults and parts delays were raised.',
        actions: ['Explain the battery plan', 'Set repair time targets', 'Clarify warranty cover'],
        quotes: ['good', 'bad', 'ugly'].flatMap((kind) => [1, 2, 3].map((i) => ({ kind, optionId: kind === 'good' ? 'repair' : kind === 'bad' ? 'buyback' : 'concerns', sentiment: kind === 'good' ? 'positive' : 'negative', text: kind + ' owner quote number ' + i + ' about their experience.' }))),
        themeCounts: [{ name: 'battery', count: 4 }, { name: 'air-conditioning', count: 2 }],
        sentiment: { repair: { positive: 1, mixed: 2, negative: 3 }, buyback: { positive: 0, mixed: 1, negative: 2 }, neither: { positive: 0, mixed: 1, negative: 0 }, compensation: { positive: 0, mixed: 1, negative: 1 }, concerns: { positive: 0, mixed: 0, negative: 1 } }
      }, { serviceLocations: { known: 6, unknown: 2, areas: [{ area: 'SW', count: 5 }, { area: 'Other areas', count: 1 }] }, consentedJoinTimeline: [{ label: '2026-07-01', count: 2 }, { label: '2026-08-01', count: 4 }, { label: '2026-09-01', count: 4 }], consentedMemberCountries: [{ label: 'United Kingdom', count: 9 }, { label: 'Other / unknown', count: 1 }] },
      { joinedOwners: 1477, vehiclesRegistered: 721, modelYearDistribution: [{ label: '2020', count: 4 }], generatedAt: '2026-09-19' },
      (bytes) => fs.writeFileSync('ipace-owner-survey-jlr-24-september-2026.pptx', bytes), { heroData, wreathData })
        .catch((error) => { console.error(error); process.exitCode = 1; });
    `;
    execFileSync(process.execPath, ['-e', script], { cwd: temp, stdio: 'pipe' });
    const deck = fs.readFileSync(path.join(temp, 'ipace-owner-survey-jlr-24-september-2026.pptx'));
    const zip = await JSZip.loadAsync(deck);
    const contentTypes = await zip.file('[Content_Types].xml').async('string');
    assert.ok(!contentTypes.includes('slideMaster2.xml'), 'orphan slide master entries must be removed');
    const layout = await zip.file('ppt/slideLayouts/slideLayout2.xml').async('string');
    assert.ok(layout.includes('type="title"') && layout.includes('0F766E') && layout.includes('Sept JLR Meeting'), 'editable content layout must carry title, divider and footer');
    assert.ok((await zip.file('ppt/theme/theme1.xml').async('string')).includes('Arial'), 'added slides must inherit the deck font theme');
    for (const match of contentTypes.matchAll(/<Override PartName="\/([^"]+)"/g)) {
      assert.ok(zip.file(match[1]), `package declares missing part ${match[1]}`);
    }
    const slides = Object.keys(zip.files).filter((name) => /^ppt\/slides\/slide\d+\.xml$/.test(name));
    assert.equal(slides.length, 17);
    const orderedSlides = await Promise.all(Array.from({ length: 17 }, (_, index) => zip.file(`ppt/slides/slide${index + 1}.xml`).async('string')));
    const contents = orderedSlides.join('\n');
    for (const phrase of ['i-Pace Owners Advocacy Group', 'Member survey findings for discussion with JLR', 'UK I-PACE Forum', '17 July 2026', 'Members joined', 'Cars registered', '721', 'good owner quote number 3', 'ugly owner quote number 3', 'Full HV Replacement', 'Fair Compensation', 'fictional', 'MEMBER SURVEY', 'Make this my preferred outcome', 'DEMONSTRATION ONLY', 'Battery work has needed repeat visits.', 'Vehicle model years recorded', 'Overall consented vehicle records', 'Repair delays', 'Resale value', 'Sept JLR Meeting', 'Explain the battery plan', '8 votes', '6 preferred', '4 comments', 'WINNER']) {
      assert.ok(contents.includes(phrase), `PowerPoint omitted ${phrase}`);
    }
    assert.ok(!contents.includes('How to read these findings') && !contents.includes('Member and service locations recorded'), 'removed slides should stay out of the meeting deck');
    assert.ok(orderedSlides[1].includes('What we need from this meeting') && orderedSlides[2].includes('How the group formed') && orderedSlides[3].includes('Vehicle model years recorded') && orderedSlides[4].includes('What members were asked') && orderedSlides[5].includes('A completed member response') && orderedSlides[6].includes('Survey participation'), 'introductory slides should follow the meeting and member journey');
    assert.ok(orderedSlides[8].includes('Survey findings at a glance') && orderedSlides[9].includes('Recurring themes') && orderedSlides[10].includes('Owner voices') && orderedSlides[13].includes('Sentiment in the optional comments') && orderedSlides[14].includes('Experience themes by survey option') && orderedSlides[16].includes('Decisions and dates'), 'key findings, themes and quotes should precede sentiment and decisions');
    assert.ok(!contents.includes('Owner voices — the good') && !contents.includes('Owner voices — the bad') && !contents.includes('Owner voices — the ugly') && !contents.includes('Phrases shaping customer confidence'), 'meeting deck should use plain editorial headings');
    const growthSlide = orderedSlides[2];
    assert.ok(!growthSlide.includes('1,477'), 'pre-launch test members must not enter the growth total');
    assert.equal((growthSlide.match(/<p:pic>/g) || []).length, 2, 'both group growth counters should include racing laurels');
    assert.ok(growthSlide.includes('type="title"'), 'generated content headings should remain editable title placeholders');
    assert.equal((orderedSlides[6].match(/<p:pic>/g) || []).length, 2, 'response and comment counters should include racing laurels');
    const choicesSlide = orderedSlides[7];
    assert.ok(choicesSlide.indexOf('WINNER') < choicesSlide.indexOf('Full HV Replacement'), 'selected route should win');
    assert.ok(choicesSlide.includes('Fair Compensation') && choicesSlide.includes('Additional Concerns'), 'additional requests should use the same vote rows');
    assert.ok(orderedSlides[10].includes('positive experience') && orderedSlides[11].includes('A Fair Buy Back') && orderedSlides[11].includes('negative experience'), 'quote slides should identify option and sentiment');
    assert.ok(orderedSlides[14].includes('Reliable replacement') && orderedSlides[14].includes('(1)') && !orderedSlides[14].includes('Reliable replacement  1'), 'theme names and smaller counts should use separate single-line text boxes');
    execFileSync(process.execPath, ['-e', script.replace('counts: { repair: 8, buyback: 5', 'counts: { repair: 5, buyback: 5').replace('preferredCounts: { repair: 6, buyback: 2 }', 'preferredCounts: { repair: 4, buyback: 5 }')], { cwd: temp, stdio: 'pipe' });
    const tiedZip = await JSZip.loadAsync(fs.readFileSync(path.join(temp, 'ipace-owner-survey-jlr-24-september-2026.pptx')));
    const tiedChoices = await tiedZip.file('ppt/slides/slide8.xml').async('string');
    assert.ok(tiedChoices.indexOf('WINNER') < tiedChoices.indexOf('A Fair Buy Back') && tiedChoices.indexOf('A Fair Buy Back') < tiedChoices.indexOf('Full HV Replacement'), 'preferred votes should break a selected-vote tie');
    execFileSync(process.execPath, ['-e', script.replace('counts: { repair: 8, buyback: 5', 'counts: { repair: 5, buyback: 5').replace('preferredCounts: { repair: 6, buyback: 2 }', 'preferredCounts: { repair: 4, buyback: 4 }')], { cwd: temp, stdio: 'pipe' });
    const exactTieZip = await JSZip.loadAsync(fs.readFileSync(path.join(temp, 'ipace-owner-survey-jlr-24-september-2026.pptx')));
    assert.ok((await exactTieZip.file('ppt/slides/slide8.xml').async('string')).includes('No single winner'), 'exact selected and preferred tie must not claim a winner');
    assert.ok(Object.keys(zip.files).some((name) => name.startsWith('ppt/media/')), 'cover hero image omitted');
    assert.ok(!contents.includes('sam@example.org'), 'no respondent identity should enter the deck');
    if (process.env.SURVEY_DECK_PREVIEW_DIR) {
      fs.mkdirSync(process.env.SURVEY_DECK_PREVIEW_DIR, { recursive: true });
      fs.copyFileSync(path.join(temp, 'ipace-owner-survey-jlr-24-september-2026.pptx'), path.join(process.env.SURVEY_DECK_PREVIEW_DIR, 'survey-deck-test.pptx'));
    }
  } finally {
    fs.rmSync(temp, { recursive: true, force: true });
  }
});
