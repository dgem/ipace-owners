'use strict';

var assert = require('node:assert/strict');
var test = require('node:test');

test('Node audit baseline accepts only its listed advisory at or below the approved severity', async function () {
  var module = await import('../scripts/audit-node.mjs');
  var audit = {
    vulnerabilities: {
      parent: { via: ['accepted', 'newHigh'] },
      accepted: {
        via: [{
          url: 'https://example.test/accepted',
          severity: 'high'
        }]
      },
      newHigh: {
        via: [{
          url: 'https://example.test/new-high',
          severity: 'high'
        }]
      }
    }
  };
  var baseline = {
    advisories: {
      'https://example.test/accepted': {
        severity: 'high',
        reason: 'Upstream-only dependency path.'
      }
    }
  };

  var result = module.evaluateAudit(audit, baseline);

  assert.equal(result.accepted.length, 1);
  assert.equal(result.failures.length, 1);
  assert.equal(result.failures[0].url, 'https://example.test/new-high');
});

test('Node audit baseline does not accept a severity increase', async function () {
  var module = await import('../scripts/audit-node.mjs');
  var audit = {
    vulnerabilities: {
      package: {
        via: [{
          url: 'https://example.test/increased',
          severity: 'critical'
        }]
      }
    }
  };
  var baseline = {
    advisories: {
      'https://example.test/increased': {
        severity: 'high',
        reason: 'Previously high.'
      }
    }
  };

  var result = module.evaluateAudit(audit, baseline);

  assert.equal(result.accepted.length, 0);
  assert.equal(result.failures.length, 1);
});
