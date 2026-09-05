const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const { resolve } = require("node:path");
const { test } = require("node:test");

const repoRoot = resolve(__dirname, "..");

function read(path) {
  return readFileSync(resolve(repoRoot, path), "utf8");
}

test("production monitoring independently checks the public site and API", function () {
  const main = read("infra/opentofu/modules/ipace-owners/main.tf");
  const monitoring = read("infra/opentofu/modules/ipace-owners/monitoring.tf");
  const variables = read("infra/opentofu/modules/ipace-owners/variables.tf");
  const production = read("infra/opentofu/env/production.tfvars.example");
  const staging = read("infra/opentofu/env/staging.tfvars.example");

  assert.match(main, /"monitoring\.googleapis\.com"/);
  assert.match(monitoring, /resource "google_monitoring_uptime_check_config" "endpoint"/);
  assert.match(monitoring, /path\s*=\s*"\/"/);
  assert.match(monitoring, /path\s*=\s*"\/api\/public-stats"/);
  assert.match(monitoring, /selected_regions\s*=\s*\["EUROPE"\]/);
  assert.match(monitoring, /resource "google_monitoring_dashboard" "operations"/);
  assert.match(monitoring, /run\.googleapis\.com\/request_count/);
  assert.match(monitoring, /local\.email_continue_host/);
  assert.doesNotMatch(monitoring, /monitoring_host/);
  assert.match(monitoring, /\$\{var\.environment\} operations/);
  assert.match(variables, /variable "monitoring_enabled"/);
  assert.match(variables, /variable "monitoring_alert_email"/);
  assert.match(production, /monitoring_enabled\s*=\s*true/);
  assert.match(production, /monitoring_alert_email\s*=\s*"contact@ipace-owners\.org"/);
  assert.match(staging, /monitoring_enabled\s*=\s*false/);
});

test("monitoring alerts are durable and only send email when a recipient is configured", function () {
  const monitoring = read("infra/opentofu/modules/ipace-owners/monitoring.tf");
  const outputs = read("infra/opentofu/modules/ipace-owners/outputs.tf");
  const envOutputs = read("infra/opentofu/env/outputs.tf");

  assert.match(monitoring, /resource "google_monitoring_notification_channel" "operator_email"/);
  assert.match(monitoring, /var\.monitoring_alert_email != ""/);
  assert.match(monitoring, /resource "google_monitoring_alert_policy" "uptime"/);
  assert.match(monitoring, /duration\s*=\s*"600s"/);
  assert.match(monitoring, /auto_close\s*=\s*"1800s"/);
  assert.match(outputs, /output "monitoring"/);
  assert.doesNotMatch(outputs, /Production operations/);
  assert.match(envOutputs, /output "monitoring"/);
});
