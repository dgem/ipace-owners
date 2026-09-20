export async function checkAuthLinkRedirect(baseUrl, fetcher = fetch) {
  const url = (path) => new URL(path, baseUrl).toString();
  // Dummy values only; never generate or follow a real one-time sign-in code.
  const query = '?apiKey=smoke-public-key&mode=signIn&oobCode=smoke%2Bdummy%2Fcode&continueUrl=' + encodeURIComponent(url('/member/account/?authTrace=smoke&next=survey'));
  for (const path of ['/auth/action', '/auth/action/']) {
    const response = await fetcher(url(path + query), { redirect: 'manual' });
    const location = response.headers.get('location');
    if (response.status !== 302 || !location || new URL(location, baseUrl).href !== url('/__/auth/action' + query)) {
      throw new Error('Branded sign-in redirect must preserve the query and use the same-site Firebase handler');
    }
  }
}
