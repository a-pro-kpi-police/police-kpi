# AGENTS.md

## Purpose

This repository uses an AI-agent workflow:

1. GitHub Issues describe backlog work.
2. One implementation agent makes a small, isolated change in a branch.
3. Codex review checks the pull request independently.
4. A human decides whether to merge.

## Project Context

- Product: KPI web app for police department staff and leadership.
- Backend: Go 1.24 in `backend/`, using `net/http` and PostgreSQL.
- Frontend: React 18 and Vite in `frontend/`.
- Primary docs live in `README.md` and `docs/`.

## Working Rules

- Work on one GitHub Issue at a time.
- Read the issue, acceptance criteria, and relevant files before editing.
- Prefer a dedicated branch named `codex/issue-<number>-<short-description>`.
- Keep changes scoped to the requested behavior.
- Do not push directly to `main`.
- Do not merge pull requests.
- After the user confirms a PR was merged, verify its merged state, switch to its base branch, fast-forward that branch from the remote, and delete the PR's local task branch. Preserve uncommitted work and branches used by other worktrees; report any blocker instead of forcing deletion. Do not delete remote branches unless requested.
- Do not make unrelated refactors, formatting sweeps, dependency upgrades, or metadata churn.
- Do not commit secrets, credentials, local config, generated caches, uploads, or personal data.
- For user-visible text, preserve the existing Russian/Kazakh product tone and terminology.
- If the task is ambiguous, missing acceptance criteria, or requires production credentials, stop and ask for human clarification.

## Verification

Choose checks based on touched areas:

- Backend Go changes:
  - `cd backend`
  - `go test ./...`
- Frontend changes:
  - `cd frontend`
  - `npm test` when tests are relevant or touched.
  - `npm run build`
- Docker or integration changes:
  - Prefer a focused config validation first.
  - Run full Docker Compose only when the issue requires it or the change cannot be checked otherwise.
- Documentation-only changes:
  - Read the changed docs back and verify links/commands are coherent.

If a check cannot run, explain the exact reason in the PR.

## Pull Request Expectations

Every pull request should include:

- `Closes #<issue-number>`.
- A short summary of behavior or documentation changes.
- Changed files.
- Verification evidence with exact commands where possible.
- Known risks, skipped checks, or follow-ups.

## Code Review Rules

Review agents must act independently from the implementation agent.

- Review the diff against the intended base branch.
- Prioritize correctness, regressions, data loss, security, permissions, API contracts, database changes, and missing tests.
- Findings must lead the review, ordered by severity.
- Include file and line references for actionable issues.
- Avoid style-only comments unless they block maintainability or violate local conventions.
- Do not modify files during review.
- If there are no blocking findings, say that clearly and mention any residual test gaps.

## Done Definition

A task is done only when:

- The acceptance criteria are satisfied.
- Relevant checks pass or skipped checks are explained.
- Codex review has no blocking findings.
- A human approves the final result for merge.
