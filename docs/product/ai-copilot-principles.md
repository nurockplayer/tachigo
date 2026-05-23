# AI Copilot Product Principles

## Purpose

This document defines the safety, privacy, execution, and provider principles for future AI-powered product features in tachigo.

It supports [`ai-native-roadmap.md`](./ai-native-roadmap.md), which describes Track B: AI-native product direction.

This document is a planning reference. It is not an implementation source of truth until converted into issues, specs, acceptance criteria, and small PRs.

## Core Rule

```txt
AI can suggest.
Backend decides.
Human confirms.
Ledger remains deterministic.
```

This matters because tachigo touches points, tokens, wallets, claims, spending, and agency ownership.

## Safety Boundary

AI must not directly:

```txt
- issue tokens
- modify points ledger
- modify wallet ownership
- approve claims
- bypass Twitch auth
- modify agency/channel ownership
- publish campaigns automatically
- issue high-value rewards automatically
- modify reward economy parameters automatically
- execute chain transactions automatically
```

AI may:

```txt
- generate campaign drafts
- explain data
- suggest reward parameters
- flag risks
- generate SQL / API proposals
- generate test cases
- generate migration drafts
- generate dashboard copy
```

## Human Confirmation

AI-generated output should enter the product as a draft or recommendation.

Recommended pattern:

```txt
AI generates draft
→ backend validates structure and permissions
→ human reviews and edits
→ backend executes deterministic behavior
→ system records feedback and outcome
```

AI should not be the final authority for behavior that affects value, identity, permissions, or blockchain activity.

## Deterministic Backend Execution

The backend owns final decisions for:

```txt
- points calculation
- ledger mutation
- spendable balance updates
- claim eligibility
- wallet ownership checks
- agency/channel permissions
- campaign publication
- reward issuance
- chain transaction execution
```

AI output should be validated as input data, not trusted as policy.

## Privacy / Data Minimization

AI Native does not mean sending all data to an LLM.

Allowed input candidates:

```txt
- aggregated watch stats
- anonymized viewer segments
- campaign metadata
- reward economy summaries
- non-sensitive dashboard context
```

Do not send directly:

```txt
- raw access tokens
- refresh tokens
- wallet private keys
- secrets
- full OAuth payloads
- raw personally identifiable data
- unnecessary Twitch identity details
```

Any future LLM integration should include a redaction layer before data leaves the backend boundary.

## Provider Strategy

Do not bind the first implementation directly to a specific LLM provider.

Before using a real LLM, build:

```txt
TemplateProvider / MockProvider
```

This validates the domain boundary, API contract, dashboard workflow, persistence, feedback loop, and permission checks.

Then introduce an interface such as:

```go
type LLMClient interface {
    GenerateJSON(ctx context.Context, req GenerateJSONRequest) (*GenerateJSONResponse, error)
}
```

The adapter should include timeout, retry policy, JSON schema validation, prompt versioning, model tracking, token usage tracking, redaction, audit logs, and fallback behavior.

## Recommendation Lifecycle

Future AI product work should treat AI output as a lifecycle, not one-off text generation.

The system should track:

```txt
- what AI recommended
- whether the user accepted it
- whether the user rejected it
- why it was rejected
- whether accepted recommendations improved product metrics
```

Candidate data areas:

```txt
ai_recommendations
ai_recommendation_feedback
ai_insight_snapshots
```

## Product UX Principle

AI should be embedded into concrete workflows instead of defaulting to a generic chatbot.

Recommended dashboard surfaces:

```txt
Campaign Creation → Create Campaign with AI
Viewer Insights → AI-generated segments and recommended actions
Economy Health → issuance / sink / balance risk recommendations
```

The product should make AI useful at the moment a streamer or agency is making a decision.

## Non-goals

This principles document does not require the current PR to:

- implement backend AI services
- add migrations
- add dashboard UI
- connect to an LLM provider
- change ledger logic
- change wallet logic
- change claim logic
- add dependencies
- permit autonomous product actions

## Summary

```txt
Suggestive AI, deterministic backend, human confirmation, auditable outcomes.
```
