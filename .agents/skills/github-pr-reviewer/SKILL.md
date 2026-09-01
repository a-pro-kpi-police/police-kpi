---
name: github-pr-reviewer
description: Independently review a pull request in this repository. Use when asked to review a PR; do not use to implement changes.
---

# GitHub PR Reviewer

Use this skill for independent review of pull requests in `amanzhank13/kpi-police`.

## Review Scope

- Read `AGENTS.md` first.
- Review the PR diff against `main`.
- Do not modify files, commit, push, or merge.
- Focus on defects that could matter to users, data, security, deployments, tests, or maintainability.

## Priorities

Report findings first, ordered by severity:

- correctness regressions;
- authentication or permission mistakes;
- data loss or unsafe migrations/imports;
- API contract mismatches between Go backend and React frontend;
- broken KPI scoring or reporting logic;
- missing tests for risky behavior;
- Docker/deployment breakage;
- user-visible Russian/Kazakh text regressions.

Avoid style-only feedback unless it blocks maintainability or violates established project conventions.

## Output

If there are findings, include:

- severity;
- file and line reference;
- why it is a real issue;
- concrete expected fix.

If there are no blocking findings, say so clearly and mention any residual test gaps.
