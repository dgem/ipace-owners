## Summary

Describe what changed in this PR.

## Why

Describe why this change is needed and what problem it solves.

## Scope of changes

List the key files and areas changed.

## Verification steps

Provide exact local verification steps for reviewers.

1. 
2. 
3. 

## Pressure testing performed

Document the key happy-path, error-path, and access-control checks you ran.

- Happy path:
- Error paths:
- Access control and auth paths (if relevant):
- Manual UX and accessibility checks (if relevant):

## Tests

- Tests added or updated:
- If no tests were added, explain why:

### Required checks

- [ ] I ran `make build` and it passed.
- [ ] I ran `make test` for behavioural changes and all tests passed.
- [ ] I added or updated tests for behavioural changes, or explained why tests were not needed.
- [ ] I pressure-tested this change locally (happy path, error path, and auth/role path where relevant).
- [ ] This PR includes enough detail for a reviewer to reproduce and validate the change.

## Prompt and documentation alignment

- [ ] I updated prompts in prompts/ when behaviour or architecture changed.
- [ ] I kept README.md and prompts aligned so the project can be recreated from prompts plus README.
- [ ] I updated prompts/09-architecture-overview.md and
      prompts/20-clean-room-reconstruction-contract.md when architecture changed.

## Review and release readiness

- [ ] This PR is ready for Copilot review and human review.
- [ ] I confirmed no secrets, raw VINs, personal data, or sensitive tokens are committed.
- [ ] The final commit history uses semantic commit messages in type(scope): description format.

## Notes for reviewers

Add anything reviewers should pay special attention to.
