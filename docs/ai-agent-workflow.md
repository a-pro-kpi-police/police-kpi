# AI Agent Workflow

This repository uses GitHub Issues as the backlog, Codex as the implementation agent, and Codex GitHub review as the independent review agent.

## Workflow

1. Create a GitHub Issue with the `AI task` template.
2. Add label `ai-ready`.
3. The automated issue runner selects one eligible issue.
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
