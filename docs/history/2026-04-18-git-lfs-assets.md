# Git LFS Asset Handling Plan

> 用途：記錄 `tachigo` 對大型前端 binary assets 採用 Git LFS 的決策、設定方式與 PR #265 後續修復流程。
> 狀態：歷史決策與操作紀錄；本分支已落地 `apps/extension/src/assets/` 的 Git LFS tracking 與 frontend CI checkout 設定。
> 最後更新：2026-04-18
> 最後校正：2026-05-05（#490 docs root audit）

---

## 1. 背景

PR #265 (`[frontend] migrate demo app-shell foundations into tachimint (1/4)`) 將 demo app-shell 的第一批基礎資產搬入 `apps/extension/`，包含：

- logo PNG
- pixel font `PressStart2P-Latin.woff2`
- pixel font `Zpix.ttf`
- base styles、i18n、coupon catalog、demo state foundation

在嘗試用 Gemini CLI 做 PR first-pass review 時，`gh pr diff 265 --patch` 產生約 42k 行 diff。這不是正常的 source diff 大小，而是由兩個因素疊加造成：

1. `--patch` 會輸出類似 `git format-patch` 的 commit series，同一批檔案可能在多個 commit 中反覆出現。
2. binary asset 被展開成 `GIT binary patch`，其中 `apps/extension/src/assets/fonts/Zpix.ttf` 約 7MB，會讓 diff、review 與 clone 成本失真。

因此，這類前端 binary assets 不應直接以一般 Git blob 進入長期 history，應使用 Git LFS。

## 2. 目標

- 避免大型 PNG / font binary 直接污染 `develop` history
- 讓 GitHub PR diff、Codex / Gemini review、local clone 成本維持可控
- 讓 CI build 能正確 checkout LFS assets
- 保持 PR scope 乾淨：先落地 repo-level LFS 設定，再修正 PR #265 branch

## 3. 不做事項

這份文件只處理 Git LFS 與大型 frontend binary assets，不包含：

- 不重新設計 tachimint app-shell migration scope
- 不調整 PR #265 的 UI / i18n / storage 實作內容
- 不把所有 repo binary 檔一次性遷移到 LFS
- 不重寫已 merge 到 `develop` 的既有 history

## 4. 建議 repo 設定

應從 `develop` 拉一條獨立 branch，例如：

```bash
git switch develop
git pull
git switch -c chore/git-lfs-assets
```

新增 `.gitattributes`，先針對 `apps/extension/src/assets/` 中常見 binary asset 類型啟用 LFS：

```gitattributes
# Tachimint binary assets
apps/extension/src/assets/**/*.png filter=lfs diff=lfs merge=lfs -text
apps/extension/src/assets/**/*.jpg filter=lfs diff=lfs merge=lfs -text
apps/extension/src/assets/**/*.jpeg filter=lfs diff=lfs merge=lfs -text
apps/extension/src/assets/**/*.webp filter=lfs diff=lfs merge=lfs -text
apps/extension/src/assets/**/*.gif filter=lfs diff=lfs merge=lfs -text

# Tachimint fonts
apps/extension/src/assets/**/*.ttf filter=lfs diff=lfs merge=lfs -text
apps/extension/src/assets/**/*.otf filter=lfs diff=lfs merge=lfs -text
apps/extension/src/assets/**/*.woff filter=lfs diff=lfs merge=lfs -text
apps/extension/src/assets/**/*.woff2 filter=lfs diff=lfs merge=lfs -text
```

先不要一口氣套到所有 product surface，原因是：

- `apps/dashboard/`、`extensions/` 可能有不同資產策略
- 小型 SVG 或 source-like assets 不一定需要 LFS
- 避免單一 infra PR 造成不必要的大範圍 renormalize

若後續其他 product surface 也需要，可另開 PR 擴充 `.gitattributes`。

## 5. CI checkout 設定

若 frontend build 會 import LFS-managed assets，CI 的 `actions/checkout` 必須啟用 LFS。

至少 `frontend` job 應改成：

```yaml
- uses: actions/checkout@v4
  with:
    lfs: true
```

是否要在 `dashboard`、`contracts`、`backend` jobs 也加 `lfs: true`，取決於它們是否需要讀取 LFS assets。以目前需求來看，先只改 `frontend` job 較符合最小 scope。

## 6. 為什麼只 merge LFS 設定還不夠

`.gitattributes` 只會影響後續被 Git 重新 add / normalize 的檔案。

如果 PR #265 branch 已經在較早 commit 中加入一般 Git blob，例如：

- `apps/extension/src/assets/242a2b8162b4542ca6839e84ad45ad4a36c0257c.png`
- `apps/extension/src/assets/fonts/PressStart2P-Latin.woff2`
- `apps/extension/src/assets/fonts/Zpix.ttf`

那麼只把 `.gitattributes` merge 到 `develop`，再讓 PR #265 merge `develop`，不會自動把這些既有 binary blob 轉成 LFS pointer。

換句話說：

- `develop` 尚未 merge PR #265：仍可避免污染正式 history。
- PR #265 branch 已有 binary blob：需要在該 branch 額外處理。

## 7. 建議處理順序

### Step 1: 先做 LFS setup PR

從 `develop` 開獨立 branch，內容只包含：

- `.gitattributes`
- 必要的 CI checkout `lfs: true`
- 既有且已符合新規則的 `apps/extension/src/assets/hero.png` LFS pointer 轉換
- 這份文件或對應決策文件

PR title 可使用：

```text
[infra] add Git LFS tracking for tachimint binary assets
```

這張 PR merge 到 `develop` 後，repo 才有正式 LFS 規則。

### Step 2: 更新 PR #265 branch

切回 PR #265 branch：

```bash
git switch feat/tachimint-app-shell-2a
git fetch origin
git merge origin/develop
```

這會把 `.gitattributes` 與 CI checkout 設定帶進 PR #265 branch。

### Step 3: 將 PR #265 assets 轉成 LFS

若只想讓 final tree 符合 LFS 規則，可執行：

```bash
git add --renormalize apps/extension/src/assets
git commit -m "chore: store tachimint assets with Git LFS"
```

但這只會讓最後一個 commit 的 tree 使用 LFS pointer，不會移除 PR branch 早期 commits 裡的 binary blob。

若 PR #265 最後會用 merge commit 合併，為了避免 binary blob 隨 branch history 進入 `develop`，應在 PR #265 branch 上做 history rewrite：

```bash
git lfs migrate import \
  --include="apps/extension/src/assets/**/*.png,apps/extension/src/assets/**/*.jpg,apps/extension/src/assets/**/*.jpeg,apps/extension/src/assets/**/*.webp,apps/extension/src/assets/**/*.gif,apps/extension/src/assets/**/*.ttf,apps/extension/src/assets/**/*.otf,apps/extension/src/assets/**/*.woff,apps/extension/src/assets/**/*.woff2"
```

接著確認結果後 force push：

```bash
git push --force-with-lease origin feat/tachimint-app-shell-2a
```

這會改寫 PR #265 branch history，因此執行前要確認沒有其他人基於該 branch 繼續開分支。

## 8. 驗證指令

確認本機 Git LFS 可用：

```bash
git lfs version
```

確認 LFS tracking 規則：

```bash
git lfs track
cat .gitattributes
```

確認 assets 已由 LFS 管理：

```bash
git lfs ls-files
```

確認指定 asset 在 Git 裡是 pointer：

```bash
git show HEAD:apps/extension/src/assets/fonts/Zpix.ttf | sed -n '1,5p'
```

正常應看到類似：

```text
version https://git-lfs.github.com/spec/v1
oid sha256:<hash>
size <bytes>
```

確認 PR diff 不再展開巨大 binary patch：

```bash
gh pr diff 265 --color=never
```

review 工具不應使用：

```bash
gh pr diff 265 --patch
```

原因是 `--patch` 會輸出 commit series，容易讓多 commit PR 的 diff 重複膨脹，不適合拿來餵 Codex / Gemini 做 final PR review。

## 9. Review 注意事項

PR review 時應把 binary assets 分開處理：

- binary 檔只檢查檔名、大小、引用位置與 LFS pointer 狀態
- source diff 只看 `.ts`、`.json`、`.css`、`.yml`、`.md` 等文字檔
- 不把 `GIT binary patch` 直接餵給 Gemini / Codex
- 若使用 Gemini CLI 做 first-pass review，應餵 combined text diff，不要餵 format-patch series

## 10. 已知風險

- `git lfs migrate import` 會重寫 branch history，需要 force push。
- 若 PR #265 branch 已被其他人 checkout 或基於其開分支，force push 會造成協作成本。
- 若 CI checkout 未啟用 `lfs: true`，frontend build 可能拿到 pointer file 而非實際 binary asset。
- GitHub LFS 有 storage / bandwidth quota，需要避免把不必要的大型檔案都納入 repo。

## 11. 決策

採用以下流程：

1. 先從 `develop` 開獨立 infra PR，加入 tachimint asset LFS 設定與必要 CI checkout。
2. 該 PR merge 到 `develop` 後，再讓 PR #265 merge 最新 `develop`。
3. 在 PR #265 branch 將已加入的 binary assets 轉成 LFS pointer。
4. 若要避免 binary blob 進入 `develop` history，PR #265 branch 需使用 `git lfs migrate import` 重寫 history 後 force push。

這樣可以在 `develop` 尚未被 PR #265 merge 前，避免大型 binary blob 進入正式長期 history，同時保持 LFS 設定 PR 的 scope 清楚。
