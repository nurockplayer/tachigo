const http = require('http');
const crypto = require('crypto');
const { execFile } = require('child_process');

const PORT = Number(process.env.PORT || 3000);
const WEBHOOK_SECRET = process.env.WEBHOOK_SECRET || '';
const CLAUDE_TIMEOUT_MS = 10 * 60 * 1000;

function sendJson(res, statusCode, body) {
  res.writeHead(statusCode, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(body));
}

function timingSafeSecretMatch(headerValue) {
  if (!WEBHOOK_SECRET || typeof headerValue !== 'string') {
    return false;
  }

  const provided = Buffer.from(headerValue, 'utf8');
  const expected = Buffer.from(WEBHOOK_SECRET, 'utf8');

  if (provided.length !== expected.length) {
    return false;
  }

  return crypto.timingSafeEqual(provided, expected);
}

function buildPrompt(prNumber) {
  return `審查 PR #${prNumber} in nurockplayer/tachigo。

步驟：
1. gh pr view ${prNumber} --repo nurockplayer/tachigo --json body,reviews 取得 CR 內容
2. gh pr diff ${prNumber} --repo nurockplayer/tachigo 取得 diff
3. 判斷 CR 問題是否已解決，有無新的 bug/regression
4. 如果 CR 已修好 → gh pr review ${prNumber} --repo nurockplayer/tachigo --approve -b "LGTM"
5. 如果 CR 未修好 → gh pr review ${prNumber} --repo nurockplayer/tachigo --request-changes -b "<問題說明>"

reviewer: nurockplayer`;
}

function runClaudeReview(prNumber) {
  const prompt = buildPrompt(prNumber);

  console.log(`Starting Claude review for PR #${prNumber}`);

  execFile(
    'claude',
    ['-p', prompt, '--allowedTools', 'Bash'],
    { timeout: CLAUDE_TIMEOUT_MS },
    (error, stdout, stderr) => {
      if (error) {
        console.log(`Claude review failed for PR #${prNumber}`);
        console.error(error);
        if (stdout) {
          console.log(stdout);
        }
        if (stderr) {
          console.error(stderr);
        }
        return;
      }

      console.log(`Claude review completed for PR #${prNumber}`);
      if (stdout) {
        console.log(stdout);
      }
      if (stderr) {
        console.error(stderr);
      }
    }
  );
}

const server = http.createServer((req, res) => {
  if (req.method !== 'POST' || req.url !== '/review') {
    sendJson(res, 404, { error: 'Not found' });
    return;
  }

  if (!timingSafeSecretMatch(req.headers['x-webhook-secret'])) {
    sendJson(res, 401, { error: 'Unauthorized' });
    return;
  }

  let body = '';

  req.on('data', (chunk) => {
    body += chunk;
  });

  req.on('end', () => {
    let payload;

    try {
      payload = body ? JSON.parse(body) : {};
    } catch (error) {
      sendJson(res, 400, { error: 'Invalid JSON' });
      return;
    }

    const prNumber = payload.pr;

    if (typeof prNumber !== 'number' && typeof prNumber !== 'string') {
      sendJson(res, 400, { error: 'Missing pr' });
      return;
    }

    sendJson(res, 202, { status: 'accepted' });
    runClaudeReview(prNumber);
  });

  req.on('error', (error) => {
    console.log('Request error');
    console.error(error);
    if (!res.headersSent) {
      sendJson(res, 400, { error: 'Request error' });
    }
  });
});

server.listen(PORT, () => {
  console.log(`NAS webhook listening on port ${PORT}`);
});
