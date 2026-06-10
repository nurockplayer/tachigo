# OpenSpec

Tachigo uses OpenSpec SDD artifacts as the source of truth for new feature and behavior-change implementation.

See `docs/ai/openspec-workflow.md` for the repo workflow and guardrails.

## Layout

```text
openspec/
├── config.yaml
├── specs/
│   └── README.md
└── changes/
    └── README.md
```

- `changes/` contains proposed changes before implementation or while implementation is in progress.
- `specs/` contains accepted living behavior after a change is archived / synced.

Do not commit local OpenSpec cache, private context, task packages, or dependency-generated output.
