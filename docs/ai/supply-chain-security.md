# Supply-chain Security Guardrails

本文件記錄 tachigo 對 Mini Shai-Hulud-style 供應鏈事件採取的 repo-level 與 AI-agent guardrails。這些規則的目標不是保證永遠不會下載到被污染的官方套件，而是降低安裝期自動執行、憑證外洩、持久化與橫向擴散的機率。

## Agent Package-install Rules

AI agent 不得自行執行下列動作：

- 新增 npm / pnpm / Go / system dependency，除非 issue 或使用者明確授權。
- 執行 `npx`、`pnpm dlx`、`npm exec`、`curl | bash`、`wget | sh` 或等價的動態下載執行。
- 修改 `package.json`、lockfile 或 Docker install path 後跳過人工可讀說明。

若確實需要新增或升級依賴，PR 必須說明：

- 套件名稱、版本、用途。
- 是否會新增 lifecycle script。
- lockfile 變更是否符合 issue scope。
- 已跑過哪些 guardrail 與 workspace 測試。

## Repo Guardrails

- `make supply-chain-check` 跑 `infra/scripts/check-supply-chain-guardrails.mjs`。
- CI 的 `Supply-chain guardrails` job 會拒絕新的 `preinstall`、`install`、`postinstall`、`prepare` scripts。
- 同一個 guardrail 會拒絕 package scripts 內的 `npx`、`pnpm dlx`、`npm exec`、`curl | bash`、`wget | sh`。
- Frontend Docker install 使用 `pnpm install --frozen-lockfile --ignore-scripts`。
- GitHub dependency review 已在 CI 中啟用；runtime high-severity vulnerability 會 fail PR。

## Developer-machine Persistence Check

`make developer-persistence-check` 是本機人工檢查，不應放進 CI。它只回報可疑路徑或檔名，不印出設定檔內容。

目前檢查：

- `~/.claude/settings.json`
- `~/.claude/settings.local.json`
- `~/.vscode/tasks.json`
- `~/Library/LaunchAgents`
- `${XDG_CONFIG_HOME:-~/.config}/systemd/user`

## TanStack Check On 2026-05-13

tachigo 的 dashboard lockfile 解析到 `@tanstack/react-query@5.100.6`。截至 2026-05-13 的公開 Mini Shai-Hulud / TanStack advisories，受影響清單聚焦於 TanStack Router / Start 相關套件；目前未看到 `@tanstack/react-query@5.100.6` 被列為受影響版本。

這個結論只代表本次 hardening PR 的判斷基準。若未來 advisories 更新，應重新跑公開清單比對並更新本文件。

參考：

- https://www.wiz.io/blog/mini-shai-hulud-strikes-again-tanstack-more-npm-packages-compromised
- https://digital.nhs.uk/cyber-alerts/2026/cc-4781
- https://www.bleepingcomputer.com/news/security/shai-hulud-attack-ships-signed-malicious-tanstack-mistral-npm-packages/
