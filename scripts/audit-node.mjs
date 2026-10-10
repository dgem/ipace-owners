#!/usr/bin/env node

import { readFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const severityRank = { low: 1, moderate: 2, high: 3, critical: 4 };

export function evaluateAudit(audit, baseline) {
  const vulnerabilities = audit.vulnerabilities || {};
  const advisoryByURL = new Map();

  function visit(name, seen) {
    if (seen.has(name)) {
      return;
    }
    seen.add(name);
    const vulnerability = vulnerabilities[name];
    if (!vulnerability) {
      return;
    }
    for (const item of vulnerability.via || []) {
      if (typeof item === 'string') {
        visit(item, seen);
      } else if (item && item.url) {
        advisoryByURL.set(item.url, item);
      }
    }
  }

  Object.keys(vulnerabilities).forEach(function (name) {
    visit(name, new Set());
  });

  const accepted = [];
  const failures = [];
  const warnings = [];
  for (const advisory of advisoryByURL.values()) {
    const acceptedAdvisory = baseline.advisories[advisory.url];
    const actualRank = severityRank[advisory.severity] || 0;
    const allowedRank = severityRank[acceptedAdvisory && acceptedAdvisory.severity] || 0;
    if (acceptedAdvisory && actualRank <= allowedRank) {
      accepted.push({ advisory, reason: acceptedAdvisory.reason });
    } else if (actualRank >= severityRank.high) {
      failures.push(advisory);
    } else {
      warnings.push(advisory);
    }
  }

  return { accepted, failures, warnings };
}

function printAnnotation(level, message) {
  process.stdout.write(`::${level}::${message}\n`);
}

function main() {
  const baseline = JSON.parse(readFileSync(new URL('../security/npm-audit-baseline.json', import.meta.url), 'utf8'));
  const result = spawnSync('npm', ['audit', '--json'], { encoding: 'utf8' });
  let audit;
  try {
    audit = JSON.parse(result.stdout);
  } catch (error) {
    process.stderr.write(result.stderr || 'npm audit did not return JSON.\n');
    throw new Error(`npm audit could not be evaluated: ${error.message}`, { cause: error });
  }
  if (result.error || (!audit.vulnerabilities && result.status !== 0)) {
    throw new Error(`npm audit failed before reporting vulnerabilities: ${result.stderr || result.error?.message || 'unknown error'}`);
  }

  const evaluation = evaluateAudit(audit, baseline);
  for (const entry of evaluation.accepted) {
    printAnnotation('warning', `Accepted npm advisory ${entry.advisory.url} (${entry.advisory.severity}): ${entry.reason}`);
  }
  for (const advisory of evaluation.warnings) {
    printAnnotation('warning', `npm advisory below the blocking threshold: ${advisory.url} (${advisory.severity}).`);
  }
  if (evaluation.failures.length > 0) {
    for (const advisory of evaluation.failures) {
      printAnnotation('error', `New unaccepted high or critical npm advisory: ${advisory.url} (${advisory.severity}).`);
    }
    process.exitCode = 1;
  }

  if (evaluation.accepted.length === 0 && evaluation.failures.length === 0 && evaluation.warnings.length === 0) {
    process.stdout.write('npm audit found no advisories.\n');
  }
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  main();
}
