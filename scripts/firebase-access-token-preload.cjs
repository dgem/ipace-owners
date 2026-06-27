'use strict';

const { createRequire } = require('node:module');

const accessToken = process.env.FIREBASE_CLI_ACCESS_TOKEN;

if (accessToken) {
  const firebaseRequire = createRequire(require.resolve('firebase-tools/package.json'));
  const { GoogleAuth } = firebaseRequire('google-auth-library');

  // The GitHub auth action already exchanged OIDC for this short-lived token.
  // Reuse it so Firebase CLI does not perform a second, flaky STS exchange.
  GoogleAuth.prototype.getAccessToken = async function getAccessToken() {
    return accessToken;
  };

  GoogleAuth.prototype.getCredentials = async function getCredentials() {
    return {};
  };
}
