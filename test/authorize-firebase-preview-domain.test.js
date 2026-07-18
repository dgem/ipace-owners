const assert = require("node:assert/strict");
const { test } = require("node:test");

test("accepts a preview URL belonging to the staging Firebase project", async () => {
  const { previewHostname } = await import("../scripts/authorize-firebase-preview-domain.mjs");

  assert.equal(
    previewHostname(
      "ipace-owners-staging",
      "https://ipace-owners-staging--pr-20-ef2wibc5.web.app/member/account/",
    ),
    "ipace-owners-staging--pr-20-ef2wibc5.web.app",
  );
});

test("rejects domains outside the current Firebase project", async () => {
  const { previewHostname } = await import("../scripts/authorize-firebase-preview-domain.mjs");

  assert.throws(
    () => previewHostname("ipace-owners-staging", "https://attacker--pr-20.web.app"),
    /Firebase Hosting preview URL/,
  );
  assert.throws(
    () => previewHostname("ipace-owners-staging", "https://stage.ipace-owners.org"),
    /Firebase Hosting preview URL/,
  );
});

test("replaces stale PR domains without dropping permanent domains", async () => {
  const { mergeAuthorizedDomains } = await import("../scripts/authorize-firebase-preview-domain.mjs");
  const hostname = "ipace-owners-staging--pr-20-ef2wibc5.web.app";

  assert.deepEqual(
    mergeAuthorizedDomains(
      [
        "localhost",
        "ipace-owners-staging.web.app",
        "ipace-owners-staging--pr-19-old.web.app",
        hostname,
      ],
      hostname,
      "ipace-owners-staging",
    ),
    ["ipace-owners-staging--pr-20-ef2wibc5.web.app", "ipace-owners-staging.web.app", "localhost"],
  );
});
