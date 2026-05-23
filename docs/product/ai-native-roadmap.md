# tachigo AI Native Product Roadmap

## Purpose

This document captures **Track B: AI-native product direction** for tachigo.

It complements Discussion #877, which remains the short-term source of truth for **Track A: AI-native repo / workflow foundation**.

This is a planning reference, not an implementation source of truth. Before implementation, each section must be converted into issues, specs, acceptance criteria, and small PRs.

## Relationship to Discussion #877

Recommended split:

```txt
Track A: AI-native repo / workflow foundation
- Source of truth: Discussion #877
- Focus: Dev Portal, Wiki, AI Agent Start Here, domain cards, issue templates, PR gates, review / merge guardrails
- Timing: short term

Track B: AI-native product roadmap
- Source of truth: this document, after scope is accepted
- Focus: Campaign Copilot, viewer insights, economy advisor, recommendation feedback loop
- Timing: medium term
```

Short-term priority:

```txt
Make tachigo AI-readable and AI-safe before making tachigo product features AI-powered.
```

## Product Positioning

Recommended positioning:

> tachigo is an AI-native creator rewards OS for Twitch streamers, agencies, and viewers.

AI should help streamers and agencies:

- understand viewer behavior
- design reward campaigns
- manage token economy health
- detect suspicious reward farming
- turn dashboard data into operational actions

## AI Native Definition

For tachigo, AI Native means:

```txt
AI is embedded into creator operations, reward design, viewer insight, and economy optimization.
```

The product should evolve from:

```txt
viewer watches stream → heartbeat → points increase → token claim / spend
```

Into:

```txt
viewer activity → AI insight / recommendation → streamer review → deterministic backend execution → feedback loop
```

Core principle:

```txt
AI is not the authority.
AI is the operating layer for recommendations, insights, and optimization.
```

## Recommended First Product Feature

Do not start with a generic chatbot.

The first AI-native product feature should be embedded into a concrete dashboard workflow:

```txt
Dashboard Campaign Copilot
```

Recommended workflow:

```txt
Create Campaign
  → streamer inputs goal
  → AI generates campaign draft
  → backend calculates cost / risk
  → streamer edits
  → streamer confirms
  → campaign is saved / published
```

The first version should generate a campaign draft, not publish it automatically.

Detailed safety, privacy, and execution rules live in [`ai-copilot-principles.md`](./ai-copilot-principles.md).

## Product Feature Phases

### Phase 1: Dashboard Campaign Copilot

Help streamers and agencies create reward campaigns from goals.

Example draft components:

```txt
- Watch Streak Mission
- Chat Burst Mission
- TACHI Sink
- New Viewer Bonus
- Risk controls
```

### Phase 2: Viewer Segment Insight

Help streamers and agencies understand viewer behavior and convert analytics into actions.

Example segments:

```txt
- Core fans
- Silent loyal viewers
- Reward hunters
- New viewers
- Dormant viewers
```

### Phase 3: Token Economy Advisor

Help streamers and agencies manage reward economy health.

Metrics to analyze:

```txt
- weekly point issuance
- weekly point spending
- sink ratio
- spendable balance growth
- whale concentration
- reward farming pattern
- inactive token accumulation
```

Differentiation:

```txt
Generic tool: helps issue points.
tachigo: helps design, monitor, and adjust the creator reward economy.
```

### Phase 4: Recommendation Feedback Loop

The product should track:

```txt
What did AI recommend?
Did the streamer accept it?
Why was it rejected?
Did accepted recommendations improve metrics?
```

Candidate data areas:

```txt
ai_recommendations
ai_recommendation_feedback
ai_insight_snapshots
```

## Backend Direction

Add an independent AI bounded context only after Track A has made the repo AI-readable and AI-safe.

Boundary rule:

```txt
AI reads multiple domains and produces recommendations.
It does not own ledger, auth, wallet, or campaign execution.
```

Recommended ownership:

```txt
points service owns points.
agency service owns agency/channel permissions.
campaign service owns campaign lifecycle.
AI service owns recommendation lifecycle.
```

## Suggested Roadmap

```txt
Phase 0: AI-native repo / workflow foundation from #877
Phase 1: Product roadmap and AI copilot principles docs
Phase 2: Campaign Copilot MVP with deterministic TemplateProvider
Phase 3: LLM-backed Campaign Copilot with redaction and schema validation
Phase 4: Viewer Insight
Phase 5: Economy Advisor
Phase 6: Semi-autonomous creator operations with human approval
```

## Suggested PR Breakdown

```txt
PR 1: Track B product roadmap and copilot principles docs
PR 2: AI recommendation domain skeleton
PR 3: AI recommendation persistence
PR 4: Campaign Copilot API with deterministic TemplateProvider
PR 5: Dashboard Campaign Copilot UI
PR 6: LLM provider adapter
PR 7: Viewer Segment Insight
PR 8: Economy Advisor
```

Each implementation PR should have its own source issue, explicit non-goals, acceptance criteria, and test plan.

## North Star Metrics

Measure product impact, not whether AI output looks smart.

```txt
Streamer: campaign creation time decreases, accepted AI recommendation ratio increases.
Viewer: repeat viewer ratio, watch retention, reward redemption, and chat participation improve.
Economy: sink ratio improves, token inflation risk and suspicious farming decrease.
AI loop: accepted recommendation → measurable product metric improvement.
```

## Non-goals

This roadmap does not require the current PR to:

- implement backend AI services
- add migrations
- add dashboard UI
- connect to an LLM provider
- modify points or token ledger behavior
- change wallet or claim logic
- add dependencies
- add autonomous campaign publishing
- replace Discussion #877 as the Track A source of truth

## Conclusion

The short-term priority remains Track A from Discussion #877:

```txt
Make the repo AI-readable and AI-safe first.
```

Then Track B can proceed as an AI-native creator rewards OS, starting with Dashboard Campaign Copilot.
