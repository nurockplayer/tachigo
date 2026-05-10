# GitHub SSH 443 Push Playbook

當 Codex / Claude Code / 本機 shell 在 `git push` 時反覆遇到 GitHub DNS 或 SSH 連線問題，優先使用 repo-local 的 GitHub SSH over 443 設定，不要把 GitHub connector 當成長期替代方案。

GitHub connector 適合查 issue、PR、review metadata；本機 branch、commit、push 仍應以 git transport 為主。

## 適用症狀

- `git push` 偶發或反覆出現 DNS 解析失敗。
- 一般 SSH 路徑 `git@github.com:owner/repo.git` 不穩，但 GitHub 本身可連。
- Codex 新 session 可以讀 repo，但 push 收尾常卡在 network / SSH transport。

## Tachigo Repo 建議設定

只改 `origin` 的 push URL，保留 fetch URL 不變：

```bash
rtk git --no-optional-locks remote set-url --push origin ssh://git@ssh.github.com:443/nurockplayer/tachigo.git
rtk git --no-optional-locks remote -v
```

預期結果：

```text
origin  git@github.com:nurockplayer/tachigo.git (fetch)
origin  ssh://git@ssh.github.com:443/nurockplayer/tachigo.git (push)
```

這是 repo-local 設定，會寫進 `.git/config`。同一個 clone 的新 Codex session 會自動沿用；不同 clone 或新 worktree 可能需要重設一次。

## Host Key 驗證

第一次改走 `ssh.github.com:443` 時，SSH 可能失敗：

```text
Host key verification failed.
```

先查本機是否已信任 GitHub 443 host key：

```bash
rtk ssh-keygen -F '[ssh.github.com]:443' -f ~/.ssh/known_hosts
```

若沒有紀錄，先掃描並計算 fingerprint：

```bash
rtk ssh-keyscan -p 443 -T 10 -t ed25519 ssh.github.com 2>/dev/null | rtk ssh-keygen -lf - -E sha256
```

GitHub 官方 ED25519 fingerprint 目前應為：

```text
SHA256:+DiY3wvvV6TuJJhbpZisF/zLDA0zPMSvHdkr4UvCOqU
```

以 GitHub 官方文件為準，不要盲目信任 `ssh-keyscan` 結果：

<https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/githubs-ssh-key-fingerprints>

確認 fingerprint 相符後，再新增 host key：

```bash
rtk ssh-keyscan -p 443 -T 10 -t ed25519 ssh.github.com >> ~/.ssh/known_hosts
rtk chmod 600 ~/.ssh/known_hosts
rtk ssh-keygen -F '[ssh.github.com]:443' -f ~/.ssh/known_hosts
```

## 驗證 Push 路徑

使用 dry-run 驗證，不會真的寫入遠端：

```bash
rtk env GIT_TERMINAL_PROMPT=0 GIT_SSH_COMMAND='ssh -o BatchMode=yes -o ConnectTimeout=10' git --no-optional-locks push --dry-run origin HEAD
```

成功時會看到類似：

```text
To ssh://ssh.github.com:443/nurockplayer/tachigo.git
 * [new branch]      HEAD -> <branch>
```

## 何時改全域 SSH Config

repo-local push URL 是保守預設。只有在多個 GitHub repo 都反覆遇到同樣問題時，才考慮改 `~/.ssh/config`：

```sshconfig
Host github.com
  HostName ssh.github.com
  Port 443
  User git
```

全域設定會影響所有 GitHub SSH 連線，修改前需先確認團隊或使用者接受這個行為。

## 回復方式

若要把 tachigo 的 push URL 改回一般 SSH：

```bash
rtk git --no-optional-locks remote set-url --push origin git@github.com:nurockplayer/tachigo.git
```

若要移除 repo-local push URL，讓 push 回到 fetch URL：

```bash
rtk git --no-optional-locks config --unset remote.origin.pushurl
```
