'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var repoRoot = path.join(__dirname, '..');
var migrationPath = path.join(repoRoot, 'netlify/database/migrations/20260616120000_create_owner_data_model.sql');

test('Postgres migration defines canonical owner data tables', function () {
  var sql = fs.readFileSync(migrationPath, 'utf8');

  [
    'members',
    'join_submissions',
    'vehicles',
    'vehicle_battery_readings',
    'evidence_files',
    'member_static_snapshots',
    'public_stats_snapshots',
    'audit_events',
  ].forEach(function (table) {
    assert.match(sql, new RegExp('CREATE TABLE ' + table + ' \\('));
  });
});

test('vehicle schema supports multiple vehicles per member without raw VIN storage', function () {
  var sql = fs.readFileSync(migrationPath, 'utf8');

  assert.match(sql, /member_id TEXT NOT NULL REFERENCES members\(id\) ON DELETE CASCADE/);
  assert.match(sql, /vin_hmac TEXT/);
  assert.match(sql, /vin_last6 TEXT/);
  assert.match(sql, /CREATE INDEX idx_vehicles_member_id ON vehicles\(member_id\)/);
  assert.doesNotMatch(sql, /\bvin_full\b/i);
  assert.doesNotMatch(sql, /\braw_vin\b/i);
});

test('schema separates private member snapshots from public aggregate snapshots', function () {
  var sql = fs.readFileSync(migrationPath, 'utf8');

  assert.match(sql, /CREATE TABLE member_static_snapshots \(/);
  assert.match(sql, /identity_user_id TEXT NOT NULL UNIQUE/);
  assert.match(sql, /CREATE TABLE public_stats_snapshots \(/);
  assert.match(sql, /published_by_identity_user_id TEXT/);
});
