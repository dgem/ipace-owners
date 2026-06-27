'use strict';

const { createRequire } = require('node:module');

const accessToken = process.env.FIREBASE_CLI_ACCESS_TOKEN;

if (accessToken) {
  const firebaseRequire = createRequire(require.resolve('firebase-tools/package.json'));
  const { GoogleAuth } = firebaseRequire('google-auth-library');

  // GitHub OIDC has already been exchanged for this short-lived token.
  GoogleAuth.prototype.getAccessToken = async function getAccessToken() {
    return accessToken;
  };

  GoogleAuth.prototype.getCredentials = async function getCredentials() {
    return {};
  };
}
