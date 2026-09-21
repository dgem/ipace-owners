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
      vm.runInNewContext(fs.readFileSync(${JSON.stringify(generator)}, 'utf8'), { window: browser, Date, Promise });
      browser.ipaceMakeSurveyDeck({
        survey: { question: 'Which outcome would you support?', options: [
          { id: 'repair', name: 'Full HV Replacement', description: 'Reliable replacement without repeat visits.' },
          { id: 'buyback', name: 'A Fair Buy Back', description: 'A fair price despite affected market value.' },
          { id: 'neither', name: 'None of the above', description: 'Neither primary route meets my needs.' },
          { id: 'compensation', name: 'Fair Compensation', description: 'For disruption and repeat visits.' },
          { id: 'concerns', name: 'Additional Concerns', description: 'Other faults also need rectifying.' }
        ] },
        counts: { repair: 8, buyback: 5, neither: 1, compensation: 3, concerns: 2 }, preferredCounts: { repair: 6, buyback: 2 },
        totalResponses: 10, items: [], overview: 'Battery faults and parts delays were raised.',
        actions: ['Explain the battery plan', 'Set repair time targets', 'Clarify warranty cover'],
        quotes: ['good', 'bad', 'ugly'].flatMap((kind) => [1, 2, 3].map((i) => ({ kind, text: kind + ' owner quote number ' + i + ' about their experience.' }))),
        themeCounts: [{ name: 'battery', count: 4 }, { name: 'air-conditioning', count: 2 }],
        sentiment: { repair: { positive: 1, mixed: 2, negative: 3 }, buyback: { positive: 0, mixed: 1, negative: 2 }, neither: { positive: 0, mixed: 1, negative: 0 }, compensation: { positive: 0, mixed: 1, negative: 1 }, concerns: { positive: 0, mixed: 0, negative: 1 } }
      }, { serviceLocations: { known: 6, unknown: 2, areas: [{ area: 'SW', count: 5 }, { area: 'Other areas', count: 1 }] }, consentedJoinTimeline: [{ label: '2026-07-01', count: 2 }, { label: '2026-08-01', count: 4 }, { label: '2026-09-01', count: 4 }], consentedMemberCountries: [{ label: 'United Kingdom', count: 9 }, { label: 'Other / unknown', count: 1 }] },
      { joinedOwners: 1477, vehiclesRegistered: 721, modelYearDistribution: [{ label: '2020', count: 4 }], generatedAt: '2026-09-19' },
      (bytes) => fs.writeFileSync('ipace-owner-survey-jlr-24-september-2026.pptx', bytes), { heroData })
        .catch((error) => { console.error(error); process.exitCode = 1; });
    `;
    execFileSync(process.execPath, ['-e', script], { cwd: temp, stdio: 'pipe' });
    const deck = fs.readFileSync(path.join(temp, 'ipace-owner-survey-jlr-24-september-2026.pptx'));
    const zip = await JSZip.loadAsync(deck);
    const contentTypes = await zip.file('[Content_Types].xml').async('string');
    assert.ok(!contentTypes.includes('slideMaster2.xml'), 'orphan slide master entries must be removed');
    for (const match of contentTypes.matchAll(/<Override PartName="\/([^"]+)"/g)) {
      assert.ok(zip.file(match[1]), `package declares missing part ${match[1]}`);
    }
    const slides = Object.keys(zip.files).filter((name) => /^ppt\/slides\/slide\d+\.xml$/.test(name));
    assert.equal(slides.length, 15);
    const contents = (await Promise.all(slides.map((name) => zip.files[name].async('string')))).join('\n');
    for (const phrase of ['UK I-PACE Forum', '1,477', '721', 'Battery faults and parts delays', 'good owner quote number 3', 'ugly owner quote number 3', 'Full HV Replacement', 'Fair Compensation', 'fictional example', 'How we arrived at these labels', 'service locations recorded', 'United Kingdom', 'Explain the battery plan']) {
      assert.ok(contents.includes(phrase), `PowerPoint omitted ${phrase}`);
    }
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
