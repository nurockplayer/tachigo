# Introduce OpenSpec SDD Workflow

## Why

Tachigo already has issue-first development, PR Scope Police, spec-injector local gates, and Superpowers specs / plans. New feature specifications are still split across issues, docs, plans, and agent chat. OpenSpec gives feature work one reviewable proposal/spec/design/tasks bundle before implementation.

Source issue: https://github.com/nurockplayer/tachigo/issues/1039

## What Changes

- Add repo-level OpenSpec SDD guidance for new feature and behavior-change work.
- Add an `openspec/` scaffold with config, change docs, and specs docs.
- Update agent entrypoints, PR template, docs index, and CI regression tests.

## Non-Goals

- Do not install OpenSpec CLI.
- Do not add package dependencies or lockfile changes.
- Do not replace spec-injector autonomous gates.
