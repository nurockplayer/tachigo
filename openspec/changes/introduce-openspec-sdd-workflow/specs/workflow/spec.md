# Workflow Delta Spec

## ADDED Requirements

### Requirement: Feature work uses OpenSpec artifacts

New feature or behavior-change implementation MUST use an OpenSpec change as the proposal, specs, design, and tasks source of truth unless the PR documents an explicit exception.

#### Scenario: Agent starts feature implementation
- Given a GitHub source issue exists
- When an agent starts implementation
- Then the agent references `openspec/changes/<change-id>/` artifacts before editing runtime code

### Requirement: OpenSpec complements existing gates

OpenSpec MUST NOT replace spec-injector local-only gates, PR Scope Police, or human review.

#### Scenario: Autonomous PR uses OpenSpec
- Given an autonomous PR has an OpenSpec change
- When the PR reaches commit or merge gate
- Then the PR still follows `spec workflow-check` evidence requirements when applicable
