import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { pathToFileURL } from "node:url";

const templateNames = {
  resetPasswordTemplate: "reset-password.html",
  verifyEmailTemplate: "verify-email.html",
  changeEmailTemplate: "change-email.html",
  revertSecondFactorAdditionTemplate: "revert-second-factor.html",
};

export function normalizeDomain(value, name) {
  const domain = String(value || "").trim().toLowerCase();
  if (!domain) return "";
  if (!/^(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,63}$/.test(domain)) {
    throw new Error(`${name} must be a hostname without a scheme, path, or port`);
  }
  return domain;
}

export function buildEmailConfig(options, templates) {
  const emailTemplate = (body) => ({
    senderLocalPart: options.senderLocalPart,
    senderDisplayName: options.senderDisplayName,
    subject: body.subject,
    body: body.html,
    bodyFormat: "HTML",
    replyTo: options.replyTo,
  });

  const sendEmail = {
    method: "DEFAULT",
    callbackUri: `https://${options.actionDomain}/__/auth/action`,
    resetPasswordTemplate: emailTemplate({ subject: "Reset your I-PACE Owners password", html: templates.resetPasswordTemplate }),
    verifyEmailTemplate: emailTemplate({ subject: "Confirm your I-PACE Owners email address", html: templates.verifyEmailTemplate }),
    changeEmailTemplate: emailTemplate({ subject: "Your I-PACE Owners sign-in email changed", html: templates.changeEmailTemplate }),
    revertSecondFactorAdditionTemplate: emailTemplate({ subject: "Your I-PACE Owners account security changed", html: templates.revertSecondFactorAdditionTemplate }),
  };

  if (options.emailDomain) sendEmail.dnsInfo = { useCustomDomain: true };
  return { notification: { defaultLocale: "en-GB", sendEmail } };
}

export function emailConfigUpdateMask(includeDomain) {
  const fields = [
    "notification.defaultLocale",
    "notification.sendEmail.method",
    "notification.sendEmail.callbackUri",
    ...Object.keys(templateNames).map((name) => `notification.sendEmail.${name}`),
  ];
  if (includeDomain) fields.push("notification.sendEmail.dnsInfo.useCustomDomain");
  return fields.join(",");
}

export function emailDomainVerificationEndpoint(configEndpoint) {
  return `${configEndpoint.replace(/\/config$/, "")}/domain:verify`;
}

function loadTemplates(directory) {
  return Object.fromEntries(
    Object.entries(templateNames).map(([name, filename]) => [name, readFileSync(`${directory}/${filename}`, "utf8")]),
  );
}

async function request(url, options, operation) {
  const response = await fetch(url, options);
  const text = await response.text();
  if (!response.ok) {
    throw new Error(`${operation} failed (${response.status}): ${text.slice(0, 1000)}`);
  }
  return text ? JSON.parse(text) : {};
}

async function reconcileEmailDomain(endpoint, headers, config, emailDomain) {
  if (!emailDomain) return;
  const dnsInfo = config.notification?.sendEmail?.dnsInfo || {};
  if (dnsInfo.customDomain === emailDomain && dnsInfo.useCustomDomain) {
    console.log(`Firebase Auth email sender domain is active: ${emailDomain}`);
    return;
  }

  let state = dnsInfo.customDomainState;
  if (dnsInfo.pendingCustomDomain !== emailDomain || state === "NOT_STARTED" || state === "FAILED") {
    const result = await request(
      emailDomainVerificationEndpoint(endpoint),
      { method: "POST", headers, body: JSON.stringify({ domain: emailDomain, action: "VERIFY" }) },
      "Firebase Auth email-domain verification",
    );
    state = result.verificationState;
  }

  if (state === "SUCCEEDED") {
    await request(
      emailDomainVerificationEndpoint(endpoint),
      { method: "POST", headers, body: JSON.stringify({ domain: emailDomain, action: "APPLY" }) },
      "Firebase Auth email-domain activation",
    );
    console.log(`Firebase Auth email sender domain activated: ${emailDomain}`);
    return;
  }

  console.log(`Firebase Auth email sender domain verification is ${state || "IN_PROGRESS"}: ${emailDomain}`);
  console.log("Add the TXT and CNAME records shown under Firebase Authentication > Templates, then run make infra-email-domain ENV=<environment>.");
}

async function main() {
  const projectId = process.env.GCP_PROJECT_ID;
  const templateDirectory = process.env.FIREBASE_AUTH_EMAIL_TEMPLATE_DIR;
  if (!projectId) throw new Error("GCP_PROJECT_ID is required");
  if (!templateDirectory) throw new Error("FIREBASE_AUTH_EMAIL_TEMPLATE_DIR is required");

  const actionDomain = normalizeDomain(process.env.FIREBASE_AUTH_EMAIL_ACTION_DOMAIN, "FIREBASE_AUTH_EMAIL_ACTION_DOMAIN");
  if (!actionDomain) throw new Error("FIREBASE_AUTH_EMAIL_ACTION_DOMAIN is required");
  const emailDomain = normalizeDomain(process.env.FIREBASE_AUTH_EMAIL_DOMAIN, "FIREBASE_AUTH_EMAIL_DOMAIN");
  const options = {
    actionDomain,
    emailDomain,
    senderLocalPart: process.env.FIREBASE_AUTH_EMAIL_SENDER_LOCAL_PART || "members",
    senderDisplayName: process.env.FIREBASE_AUTH_EMAIL_SENDER_DISPLAY_NAME || "I-PACE Owners Advocacy Group",
    replyTo: process.env.FIREBASE_AUTH_EMAIL_REPLY_TO || "contact@ipace-owners.org",
  };

  const token = execFileSync("gcloud", ["auth", "print-access-token"], { encoding: "utf8" }).trim();
  const endpoint = `https://identitytoolkit.googleapis.com/admin/v2/projects/${encodeURIComponent(projectId)}/config`;
  const headers = { Authorization: `Bearer ${token}`, "Content-Type": "application/json", "X-Goog-User-Project": projectId };
  const currentConfig = await request(endpoint, { headers }, "Firebase Auth config read");
  const payload = buildEmailConfig(options, loadTemplates(templateDirectory));
  const updateMask = emailConfigUpdateMask(Boolean(emailDomain));

  await request(
    `${endpoint}?updateMask=${encodeURIComponent(updateMask)}`,
    { method: "PATCH", headers, body: JSON.stringify(payload) },
    "Firebase Auth email-template update",
  );
  console.log(`Firebase Auth email templates configured for ${projectId}`);
  await reconcileEmailDomain(endpoint, headers, currentConfig, emailDomain);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  await main();
}
