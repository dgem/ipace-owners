'use strict';

function magicLinkFromMessage(message, previewURL) {
  const content = `${message.html || ''}\n${message.text || ''}`.replaceAll('&amp;', '&');
  const expectedOrigin = new URL(previewURL).origin;
  for (const candidate of content.match(/https:\/\/[^\s"'<>]+/gi) || []) {
    try {
      const action = new URL(candidate);
      const query = action.searchParams;
      const continuation = new URL(query.get('continueUrl'));
      const trace = continuation.searchParams.getAll('authTrace');
      if (action.origin === expectedOrigin && action.pathname === '/auth/action' && !action.username && !action.password && !action.hash &&
          query.getAll('mode').length === 1 && query.get('mode') === 'signIn' &&
          query.getAll('oobCode').length === 1 && Boolean(query.get('oobCode')) &&
          query.getAll('apiKey').length === 1 && Boolean(query.get('apiKey')) &&
          query.getAll('continueUrl').length === 1 && continuation.origin === expectedOrigin &&
          continuation.pathname === '/member/account/' && !continuation.username && !continuation.password && !continuation.hash &&
          [...continuation.searchParams.keys()].every(key => key === 'authTrace') &&
          trace.length <= 1 && (!trace.length || /^IP-[A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4}$/.test(trace[0]))) return candidate;
    } catch {
      // Ignore malformed links without revealing their contents.
    }
  }
  return undefined;
}
module.exports = { magicLinkFromMessage };
