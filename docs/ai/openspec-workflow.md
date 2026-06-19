# OpenSpec SDD Workflow

本文件定義 tachigo 的 OpenSpec spec-driven development（SDD）標準工作流。OpenSpec 用來讓新 feature / behavior change 在實作前先有可 review 的 proposal、specs、design 與 tasks；它不取代既有 GitHub issue、PR Scope Police、spec-injector local-only gate 或 human review。

## 何時使用

必須使用 OpenSpec change：

- 新 feature 或現有功能的可觀察行為變更。
- API contract、資料流、權限、帳務、wallet、points ledger、Twitch / dashboard / extension 整合行為變更。
- 需要拆多個 PR 或跨 backend / frontend / docs 的 product work。
- AI agent 需要把需求轉成實作 checklist 的工作。

可不使用 OpenSpec change：

- 純 typo、連結修正、註解調整。
- 單一測試或 CI metadata 修補，且沒有 runtime 行為變更。
- 依 reviewer 要求補充 PR body / docs wording 的小修。

若不確定，先開 OpenSpec change；不要把模糊需求直接變成 patch。

## Source of Truth 分工

| 來源 | 責任 |
|---|---|
| GitHub issue | 使用者需求、scope、acceptance criteria、公開討論與追蹤 |
| `openspec/changes/<change-id>/proposal.md` | 為什麼做、做什麼、不做什麼 |
| `openspec/changes/<change-id>/design.md` | 技術方案、影響範圍、風險、rollback / migration considerations |
| `openspec/changes/<change-id>/tasks.md` | 實作 checklist；PR 的 Acceptance Criteria 必須對齊這裡 |
| `openspec/changes/<change-id>/specs/**/spec.md` | Delta specs；描述本 change 要新增或修改的行為 |
| `openspec/specs/**/spec.md` | 已採納的 living specs；archive / sync 後才更新 |
| `docs/ai/autonomous-pr-gates.md` | autonomous / spec-injector gate；保留 local-only evidence discipline |

OpenSpec output 是 implementation source of truth，不是擴 scope 授權。若 proposal / design / delta specs 發現額外工作，先拆 follow-up issue / change。

## 標準流程

1. **Issue first**：先確認 GitHub source issue。沒有合適 issue 時，依 repo 規則建立新 issue。
2. **Explore if needed**：需求不清楚時先用 `/opsx:explore` 或手動探索，不直接寫 code。
3. **Create change**：使用 `/opsx:propose <change-id>`，或 OpenSpec CLI 可用時使用 `openspec new change <change-id>` 後補齊 artifacts。
4. **Review artifacts**：確認 `proposal.md`、`design.md`、`tasks.md`、delta specs 與 issue scope 一致。
5. **Apply by tasks**：實作只跟著 `tasks.md`；每個 PR 應能對應其中一組 task。
6. **Sync while working**：若實作揭露新限制，先更新 OpenSpec artifacts，再改 code。
7. **Archive after acceptance**：change 完成且 PR merge 後，用 `/opsx:archive` 或人工流程把採納的 delta specs sync 到 `openspec/specs/`，再保留 archive evidence。

預設 OpenSpec command path 依官方 OPSX core profile：

```text
/opsx:propose → /opsx:apply → /opsx:sync → /opsx:archive
```

擴充命令如 `/opsx:new`、`/opsx:ff`、`/opsx:continue`、`/opsx:verify` 可在工具支援時使用，但不是 tachigo PR 的硬性門檻。

## Artifact 規範

每個 change 使用 kebab-case：

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

### `.openspec.yaml`

```yaml
schema: spec-driven
issue: "https://github.com/nurockplayer/tachigo/issues/<number>"
status: proposed
```

### `proposal.md`

必須包含：

- 背景與 source issue。
- 本 change 要改的可觀察行為。
- 明確不做。
- 影響範圍與相依 PR / issue。

### `design.md`

必須包含：

- 方案與取捨。
- 受影響的 code areas。
- 風險、rollback、migration / deploy order（若適用）。
- 對既有 spec-injector / PR gate 的影響；通常應為 `n/a`。

### `tasks.md`

必須包含可驗證 checklist：

```markdown
- [ ] 1. 更新 backend contract
- [ ] 2. 補 handler / service 測試
- [ ] 3. 更新 frontend consumer
- [ ] 4. 跑相關驗證
```

每一項 task 都要能對應 PR body 的 Acceptance Criteria 或「明確不做」。

### Delta specs

Delta spec 應描述行為，不描述實作細節。建議使用 Given / When / Then：

```markdown
## ADDED Requirements

### Requirement: Streamer can configure raffle visibility

#### Scenario: Disabled raffle is hidden from viewers
- Given raffle visibility is disabled
- When a viewer opens the extension
- Then the raffle panel is not shown
```

## 與既有 tachigo workflow 的關係

- GitHub issue 仍是公開需求入口；OpenSpec change 是實作前規格與 task source of truth。
- `spec-injector` 仍是 autonomous workflow 的 local-only guardrail；OpenSpec 不取代 `spec workflow-check`。
- PR template 的 `Source of truth` 應同時填 issue 與 OpenSpec change path。
- PR template 的 `Acceptance Criteria` 應對齊 `openspec/changes/<change-id>/tasks.md`。
- `openspec/changes/**` 是 reviewable artifacts；private context、CLI cache、task package 或未 review generated output 不得 commit。
- `openspec/specs/**` 只代表已採納的 living behavior，不得把 proposal 直接搬進 specs。

## CLI / 供應鏈限制

- 本 repo 不要求 AI agent 自行安裝 OpenSpec CLI。
- 不得使用 `npx`、`pnpm dlx`、`npm exec`、`curl | bash`、`wget | sh` 取得 OpenSpec。
- 若本機已有 OpenSpec CLI / OPSX skills，可使用；若沒有，人工建立 artifacts 也可以，但必須保留 `.openspec.yaml` 與完整 artifacts。
- 若要新增 OpenSpec dependency、lockfile 或 generated command files，必須另開 PR 並接受人工 review。

## PR Checklist

開 PR 前確認：

- `Source of truth` 指向 GitHub issue 與 `openspec/changes/<change-id>/`。
- `proposal.md` / `design.md` / `tasks.md` / delta specs 均已 review。
- 本 PR 的修改範圍完全對齊 `tasks.md`。
- 若不使用 OpenSpec，PR body 說明例外原因。
- 沒有 commit `.openspec/`、cache、private context、task package 或未 review generated output。
- 若 change 完成，已在 PR body 說明 archive / sync 狀態；若尚未完成，填 `pending with reason`。
