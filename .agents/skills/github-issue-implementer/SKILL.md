---
name: github-issue-implementer
description: Implement a GitHub Issue in this repository through a branch and pull request. Use when asked to take an AI-ready backlog issue and produce a PR; do not use for review-only tasks.
---

# GitHub Issue Implementer

Use this skill to turn one GitHub Issue into one scoped pull request.

## Required Context

- Repository: `amanzhank13/kpi-police`
- Base branch: `main`
- General rules: read and follow `AGENTS.md` before editing.

## Issue Selection

When no issue number is given:

- Search open GitHub Issues in `amanzhank13/kpi-police`.
- Prefer label `ai-ready`.
- If labels are absent, accept issues whose title starts with `[AI Task]`.
- Ignore pull requests.
- Ignore issues that already have an open PR referencing them.
- Work on at most one issue per run, oldest eligible issue first.

Do not start implementation if the selected issue lacks a clear goal or acceptance criteria.

## Implementation Workflow

1. Read the full issue body, acceptance criteria, and expected verification.
2. Read `AGENTS.md`.
3. Create branch `codex/issue-<number>-<short-slug>`.
4. Implement only the requested change.
5. Run relevant checks:
   - backend: `cd backend-golang && go test ./...`
   - frontend: `cd frontend && npm test` when relevant, and `cd frontend && npm run build`
   - docs: read back changed docs for coherence
6. Commit and push the branch.
7. Open a pull request into `main`.
8. Include `Closes #<issue-number>` in the PR body.
9. Add a top-level PR comment exactly: `@codex review`.
10. Stop. Do not merge.

## Handoff

Report:

- issue number;
- branch name;
- PR number/link;
- changed files;
- summary;
- verification commands and results;
- skipped checks or known risks;
- confirmation that `@codex review` was requested.
