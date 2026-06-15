# OpenSpec Changes

Use this directory for proposed feature or behavior changes.

Preferred command path when OpenSpec / OPSX is available:

```text
/opsx:propose <change-id>
```

Manual fallback structure:

```text
openspec/changes/<change-id>/
├── .openspec.yaml
├── proposal.md
├── design.md
├── tasks.md
└── specs/
    └── <domain>/
        └── spec.md
```

Each change must reference a GitHub issue and stay within that issue's scope. Do not commit local cache, private context, task packages, or unreviewed generated output.
