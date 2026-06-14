# Identity, Member, and Admin Gating Prompt

Implement or refine Netlify Identity integration and placeholder gated pages.

## Goal

Provide frontend Identity UX for sign in, sign out, registration, member-only placeholder pages, and admin-only placeholder pages while making clear that real data access requires server-side JWT verification later.

## Requirements

- Load `netlify-identity-widget` from `https://identity.netlify.com/v1/netlify-identity-widget.js` in the base layout.
- Use `src/assets/js/identity.js` for Identity behavior.
- Keep JavaScript plain and unbundled.
- Initialize `window.netlifyIdentity` defensively.
- Update header and mobile controls based on current user state.
- Support buttons with `data-identity-open`.
- Support member UI attributes:
  - `data-auth-gate`
  - `data-auth-content`
  - `data-requires-auth`
  - `data-requires-guest`
- Support admin UI attributes:
  - `data-admin-gate`
  - `data-admin-content`
  - `data-requires-admin`
- Treat admin users as users with `admin` in `user.app_metadata.roles`.
- Support optional body redirects:
  - `data-auth-redirect-on-login`
  - `data-auth-redirect-on-logout`

## Pages

- Member dashboard at `/member/dashboard/`.
- Account page at `/account/`.
- Admin review queue at `/admin/review-queue/`.
- Header controls for login, signup/register where relevant, and logout.
- Sign-in buttons should open the Identity modal without custom password storage.
- Join or registration completion uses a **magic link flow** implemented in `identity.js`:
  - On form submit, call `POST https://ipace-owners.netlify.app/.netlify/identity/signup`
    with the user's email and a randomly generated password (user never sees this).
  - If the response is `422` (email already registered), call
    `POST /.netlify/identity/recover` to send a sign-in link instead.
  - On success, show a "check your inbox" message — no signup modal is opened.
  - Always use the `*.netlify.app` subdomain for these API calls (not the custom domain)
    so that the CSP `connect-src` policy works from deploy previews as well as production.
- Do not open `identity.open('signup')` or `identity.open('login')` on join form completion.
- Netlify Identity must have **autoconfirm disabled** for the confirmation email
  (magic link) to be sent. Verify under Site Settings → Identity → Registration.
- `identity.init()` must be called with `{ APIUrl: 'https://ipace-owners.org/.netlify/identity' }`
  so that the widget resolves settings correctly on the custom domain.

## Security copy

Every placeholder member/admin area that refers to future private data must say that frontend gating is not access control. Future Netlify Functions must verify Identity JWTs server-side and check roles before returning sensitive data.

## Validation

- Run `npm run build`.
- Test logged-out states.
- Confirm gated content is hidden until Identity initializes.
- Confirm logged-in/member placeholders respond to Identity state.
- Confirm admin placeholders check for an admin role where possible.
- Confirm placeholder pages do not expose real private data.
