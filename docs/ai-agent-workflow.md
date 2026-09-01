# AI Agent Workflow

This repository uses GitHub Issues as the backlog, Codex as the implementation agent, and Codex GitHub review as the independent review agent.

The repository contains the instructions, issue template, PR template, and skills for the workflow. Automatic issue pickup requires an external Codex scheduled automation configured for this repository; the repo files alone do not start the runner.

## Required External Setup

Before relying on automatic issue pickup, configure a Codex scheduled automation for `amanzhank13/kpi-police` that:

- searches open GitHub Issues with label `ai-ready` or title prefix `[AI Task]`;
- selects at most one eligible issue per run;
- invokes the issue implementation workflow;
- opens a branch and pull request into `main`;
- comments `@codex review` on the pull request.

For this repository, the expected automation name is `KPI-POLICE Auto Issue Runner`.

## Workflow

1. Create a GitHub Issue with the `AI task` template.
2. Add label `ai-ready`.
3. The external Codex scheduled automation selects one eligible issue.
4. The implementation agent creates a branch named `codex/issue-<number>-<short-slug>`.
5. The agent implements the issue and opens a pull request into `main`.
6. The agent requests review by commenting `@codex review` on the pull request.
7. Codex review checks the PR.
8. A human merges only after review and checks are acceptable.

## Labels

- `ai-ready`: ready for an agent to pick up.
- `ai-working`: currently being implemented.
- `ai-review`: pull request opened and awaiting review.
- `ai-needs-human`: blocked on product, design, credentials, data, or permissions.
- `ai-done`: implemented, reviewed, and accepted.

## Good AI Issues

Good issues are small and concrete:

- one behavior change;
- clear acceptance criteria;
- expected verification commands;
- screenshots or links for UI work;
- no hidden production credentials.

Large or vague issues should be split before assigning them to the agent.

## Manual Fallback Prompt

Use this only if the scheduled runner is paused:

```text
Use $github-issue-implementer for GitHub issue #<number>.
Do not push directly to main. Do not merge.
```

## Review Fallback Prompt

Use this only if GitHub `@codex review` does not trigger:

```text
Use $github-pr-reviewer for PR #<number>.
Do not modify files.
```
