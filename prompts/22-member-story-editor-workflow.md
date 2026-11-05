# Member story list and editorial workflow

Extend the member experiences feature so managing stories remains clear as members create and revise several submissions.

## Member routes

- Keep `/member/stories/` as the member's private, feature-gated story list. Show each story's title, status, feedback, sentiment where available, published date/link, and an action to continue editing.
- Put new and existing story forms on `/member/stories/edit/?id=<uuid>`. The edit page contains the editor only and links back to the list.
- Generate a UUID v4 in the browser before creating a story. Keep it in the editor URL and send it with every save and media upload. If the first save response is lost, a retry must update the same Firestore document instead of creating a second story. The API accepts a previously unseen client ID only when it is a valid UUID v4; existing owned stories retain their current edit rules.
- Prevent duplicate submissions while a save is in progress. Keep the form's stable ID on screen after errors.

## Admin routes

- `/admin/member-stories/` is a filterable, sortable, paged list with a small media preview, member, status, and updated time. Keep bulk approve, publish, and archive actions here.
- Link each item to `/admin/member-stories/edit/?id=<story-id>`. The editorial page uses the existing admin authorization and review API to display the full Markdown source, media, consent, analysis and impact, and supports save, analysis, clarification, approval, publication and archive actions.
- Keep API reads and edits behind the existing administrator claim check. The detail page may locate an item through the existing paginated admin list API; it must not expose member stories through a public route.

## Media and action controls

- Display pending and uploaded media with the same thumbnail card, with visible button controls for inserting an image/video and removing it.
- Style destructive actions such as remove, withdraw permission and archive with the red danger button treatment. Keep ordinary insert, save and navigation actions on the existing standard button styles.
- Do not override danger-button colours inside story media cards.

## Verification

- Test the two member routes, admin list/editor routes, UUID validation, retry-stable IDs, button classes and admin authorization boundaries.
- Run `make test`, `make build`, `make lint`, `make audit`, plus responsive visual checks for the member list, member editor, admin list and admin editorial page.
