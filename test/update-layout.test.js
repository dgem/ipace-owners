'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var root = path.join(__dirname, '..');
var updatesDirectory = path.join(root, 'src', 'updates');

test('update posts use the dedicated editorial layout', function () {
  fs.readdirSync(updatesDirectory)
    .filter(function (file) { return file.endsWith('.md'); })
    .forEach(function (file) {
      var source = fs.readFileSync(path.join(updatesDirectory, file), 'utf8');
      assert.match(source, /layout: update\.njk/, file + ' should use update.njk');
    });
});

test('the update layout keeps article context and optional heroes together', function () {
  var layout = fs.readFileSync(path.join(root, 'src', '_includes', 'layouts', 'update.njk'), 'utf8');
  var styles = fs.readFileSync(path.join(root, 'src', 'assets', 'css', 'site.css'), 'utf8');

  assert.match(layout, /date \| readableDate/);
  assert.match(layout, /I-PACE Owners' Advocacy Group/);
  assert.match(layout, /update-header__summary/);
  assert.match(layout, /update-article/);
  assert.match(layout, /{% if heroImage %}/);
  assert.match(styles, /\.update-article__hero img\s*{[^}]*aspect-ratio:\s*16 \/ 7/s);
  assert.match(styles, /@media \(max-width: 40rem\)[\s\S]*\.update-article__hero img\s*{[^}]*aspect-ratio:\s*16 \/ 9/);
});

test('the first member survey update has a purposeful hero image', function () {
  var surveyUpdate = fs.readFileSync(
    path.join(updatesDirectory, 'first-member-survey.md'),
    'utf8'
  );

  assert.match(surveyUpdate, /heroImage: \/images\/september-survey-2026-hero\.jpg/);
  assert.match(surveyUpdate, /heroImageAlt:\s*["'][^"']+['"]/);
});

test('the closing reminder update carries current counters and social shares', function () {
  var closingUpdate = fs.readFileSync(
    path.join(updatesDirectory, 'survey-closing-tomorrow.njk'),
    'utf8'
  );

  assert.match(closingUpdate, /Less than 24 hours/);
  assert.match(closingUpdate, /midnight tonight/);
  assert.match(closingUpdate, /738/);
  assert.match(closingUpdate, /data-survey-participation/);
  assert.match(closingUpdate, /data-public-stats/);
  assert.equal(closingUpdate.split('social-share__link').length - 1, 4);
  for (const network of ['Facebook', 'WhatsApp', 'X', 'LinkedIn']) {
    assert.ok(closingUpdate.includes(`>${network}</a>`));
  }
});

test('the final survey update publishes selections, comments and the winner', function () {
  var resultsUpdate = fs.readFileSync(
    path.join(updatesDirectory, 'september-survey-results.njk'),
    'utf8'
  );

  assert.match(resultsUpdate, /847 owners/);
  assert.match(resultsUpdate, /501 optional comment entries/);
  assert.match(resultsUpdate, /<p>Winner<\/p>\s*<strong>Full HV Replacement<\/strong>/);
  assert.match(resultsUpdate, /681 selections · 420 preferred votes/);
  assert.match(resultsUpdate, /563 selected/);
  assert.match(resultsUpdate, /636 marked a preferred main route/);
  assert.match(resultsUpdate, /preferred votes used to break an exact tie/);
  assert.match(resultsUpdate, /Full HV Replacement leads both measures/);
  assert.equal(resultsUpdate.split('social-share__link').length - 1, 4);
});

test('the post-meeting update records the asks and protects member data', function () {
  var meetingUpdate = fs.readFileSync(
    path.join(updatesDirectory, 'after-our-first-jlr-meeting.md'),
    'utf8'
  );

  assert.match(meetingUpdate, /UK Director of Client Care/);
  assert.match(meetingUpdate, /particularly keen to improve the customer experience/);
  assert.match(meetingUpdate, /inconsistent and, too often, poor/);
  assert.match(meetingUpdate, /brand has not moved on from them/);
  assert.match(meetingUpdate, /H570, H571 or H572/);
  assert.match(meetingUpdate, /JLR representatives were very interested in our evidence and customer experiences/);
  assert.match(meetingUpdate, /direct customer support channels\s+went unanswered or were dismissed/);
  assert.match(meetingUpdate, /substantive ask is one clear commitment/);
  assert.match(meetingUpdate, /board-level statement to I-PACE owners setting\s+out how\s+JLR will resolve the high-voltage battery issue/i);
  assert.match(meetingUpdate, /by the end of\s+next\s+week/);
  assert.match(meetingUpdate, /do not currently have members’ permission/);
  assert.match(meetingUpdate, /ask each person for explicit consent/);
  assert.match(meetingUpdate, /including\s+legal\s+avenues/);
  assert.match(meetingUpdate, /Thank you to everyone who has recently volunteered to help/);
  assert.match(meetingUpdate, /sufficient\s+legal\s+and\s+litigation\s+experience\s+for\s+the\s+current\s+stage/);
  assert.match(meetingUpdate, /particularly\s+interested\s+to\s+hear\s+from\s+members\s+with\s+media\s+or\s+public\s+relations\s+experience/);
  assert.match(meetingUpdate, /relevant\s+press\s+and\s+media\s+contacts/);
  assert.match(meetingUpdate, /href="\/contact\/"/);
  assert.match(meetingUpdate, /heroImage: \/images\/post-jlr-meeting-2026-hero\.jpg/);
  assert.equal(meetingUpdate.split('social-share__link').length - 1, 4);
});

test('the next-steps update explains the private follow-up and grows the group', function () {
  var nextStepsUpdate = fs.readFileSync(
    path.join(updatesDirectory, 'what-we-have-asked-jlr-to-do-next.md'),
    'utf8'
  );

  assert.match(nextStepsUpdate, /don't plan to publish the correspondence in full/i);
  assert.match(nextStepsUpdate, /transparent as possible without prejudicing the\s+group's position/i);
  assert.match(nextStepsUpdate, /hope members can\s+understand our caution/i);
  assert.match(nextStepsUpdate, /As soon as we hear anything from JLR, we will update the\s+group/);
  assert.match(nextStepsUpdate, /Response deadline: Friday 2 October 2026/);
  assert.match(nextStepsUpdate, /board-level commitment/);
  assert.match(nextStepsUpdate, /Full HV Replacement was the winning\s+outcome/);
  assert.match(nextStepsUpdate, /reviewing cases one by one cannot replace\s+a consistent approach/);
  assert.match(nextStepsUpdate, /do not have permission to share them/);
  assert.match(nextStepsUpdate, /data-public-stat="joinedOwners"/);
  assert.match(nextStepsUpdate, /3,000/);
  assert.match(nextStepsUpdate, /about 10% of the UK I-PACE parc/);
  assert.match(nextStepsUpdate, /href="\/join\/"/);
  assert.match(nextStepsUpdate, /href="\/member\/dashboard\/"/);
  assert.match(nextStepsUpdate, /heroImage: \/images\/jlr-next-steps-2026-hero\.jpg/);
  assert.equal(nextStepsUpdate.split('social-share__link').length - 1, 4);
});
