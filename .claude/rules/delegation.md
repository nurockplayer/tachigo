# Delegation Rules

Claude / Codex are the planners, decision makers, and final reviewers.
Gemini CLI is the low-cost worker for broad, repetitive, or high-volume tasks.

For PR review, default to metadata-first + Gemini first-pass + Claude/Codex verification
unless the PR is small or the user explicitly says not to use Gemini.

For issue matching, first gather GitHub issue metadata with `gh`. If there are
many candidates, use Gemini to rank the top 3 likely issues with reasons, then
Claude / Codex verifies the final issue with `gh issue view`.

## Delegate to Gemini for:

- codebase-wide summarization
- searching patterns across many files
- repetitive code generation
- drafting boilerplate
- drafting tests
- log scanning
- grouping related files by topic
- first-pass PR review before Claude / Codex review
- scope pollution scanning against linked issue, PR title, and repo rules
- ranking many candidate GitHub issues before Claude / Codex chooses the final reference

## Do NOT delegate to Gemini for:

- architecture decisions
- security-critical review
- final merge decisions
- complex refactor planning
- production incident decisions
- nuanced product judgment

## Delegation Process

1. Write a precise prompt.
2. Ask for concise structured output.
3. Verify important findings before changes.
4. Claude / Codex keeps final judgment.

## Preferred Output Format

- summary
- findings
- file paths
- risks
- next steps
