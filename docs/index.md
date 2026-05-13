---
title: tachigo Dev Portal
hide_title: true
sidebar_position: 1
status: active
owner: engineering
last_reviewed: 2026-05-13
source_of_truth: true
code_areas:
  - services/api
  - apps/extension
  - apps/dashboard
related_repos:
  - tachigo
  - tachiya
---

<div className="tachigo-home">
  <section className="tachigo-hero" aria-labelledby="tachigo-home-title">
    <div className="tachigo-hero__copy">
      <div>
        <div className="tachigo-kicker">Project navigation console</div>
        <h1 id="tachigo-home-title">tachigo / tachiya project map</h1>
        <div className="tachigo-lede"><span>給新同事與日常開發者使用的專案導覽入口。先建立系統 mental model， 再追 domain、flow、source、tests 與跨 repo 邊界。</span></div>
      </div>

      <div className="tachigo-status-rail" aria-label="portal status">
        <span>source of truth</span>
        <span>reviewed 2026-05-13</span>
        <span>base develop</span>
        <span>P0 onboarding</span>
      </div>

      <div className="tachigo-actions">
        <a className="tachigo-action tachigo-action--primary" href="/tachigo/dev-portal/start-here">Start here</a>
        <a className="tachigo-action" href="/tachigo/dev-portal/domain-maps">Domain maps</a>
        <a className="tachigo-action" href="/tachigo/dev-portal/source-index">Source index</a>
      </div>
    </div>

    <div className="tachigo-topology" aria-label="tachigo and tachiya topology">
      <div className="tachigo-topology__header">
        <span>runtime path</span>
        <strong>viewer action to commerce boundary</strong>
      </div>
      <div className="tachigo-topology__grid">
        <div className="tachigo-node tachigo-node--viewer">
          <span className="tachigo-node__tag">entry</span>
          <strong>Twitch</strong>
          <span>Viewer identity, channel context, extension session</span>
        </div>
        <div className="tachigo-node tachigo-node--client">
          <span className="tachigo-node__tag">frontend</span>
          <strong>Extension + Dashboard</strong>
          <span>Sidepanel UX, streamer operations, agency workflows</span>
        </div>
        <div className="tachigo-node tachigo-node--api">
          <span className="tachigo-node__tag">core</span>
          <strong>tachigo API</strong>
          <span>Go boundary for auth, watch, points, spend, claims</span>
        </div>
        <div className="tachigo-node tachigo-node--ledger">
          <span className="tachigo-node__tag">state</span>
          <strong>Ledger + PostgreSQL</strong>
          <span>Users, balances, sessions, redemptions, receipts</span>
        </div>
        <div className="tachigo-node tachigo-node--tachiya">
          <span className="tachigo-node__tag">commerce</span>
          <strong>tachiya</strong>
          <span>FastAPI integration and Saleor protection layer</span>
        </div>
        <div className="tachigo-node tachigo-node--external">
          <span className="tachigo-node__tag">edges</span>
          <strong>Saleor + Chain</strong>
          <span>Checkout, orders, discount application, future claims</span>
        </div>
      </div>
    </div>
  </section>

  <section className="tachigo-route-board" aria-label="primary guide routes">
    <a className="tachigo-card tachigo-card--wide" href="/tachigo/dev-portal/start-here">
      <span className="tachigo-card__eyebrow">01 / first hour</span>
      <h2>Onboarding path</h2>
      <span className="tachigo-card__body">先看系統地圖、開發環境、第一個 PR 的安全路徑。</span>
    </a>
    <a className="tachigo-card" href="/tachigo/dev-portal/domain-maps">
      <span className="tachigo-card__eyebrow">02 / domains</span>
      <h2>Domain maps</h2>
      <span className="tachigo-card__body">Points、Auth、Extension 的 source、routes、tests 與風險。</span>
    </a>
    <a className="tachigo-card" href="/tachigo/dev-portal/flows">
      <span className="tachigo-card__eyebrow">03 / cross repo</span>
      <h2>Flow traces</h2>
      <span className="tachigo-card__body">watch-to-points、spend-to-tachiya、claim flow 的邊界。</span>
    </a>
    <a className="tachigo-card" href="/tachigo/dev-portal/daily-dev-guide">
      <span className="tachigo-card__eyebrow">04 / daily work</span>
      <h2>Daily dev</h2>
      <span className="tachigo-card__body">改功能前先定位 owner、測試、PR scope 與 rollback path。</span>
    </a>
    <a className="tachigo-card" href="/tachigo/dev-portal/graph-explorer">
      <span className="tachigo-card__eyebrow">05 / radar</span>
      <h2>Graph explorer</h2>
      <span className="tachigo-card__body">把 graphify 當成影響範圍雷達，而不是架構真相本身。</span>
    </a>
  </section>

  <section className="tachigo-flow-strip" aria-label="frequent development flows">
    <div className="tachigo-flow"><span>watch</span><strong>Extension heartbeat</strong><span>API session</span><span>points ledger</span></div>
    <div className="tachigo-flow"><span>spend</span><strong>balance check</strong><span>tachiya handoff</span><span>Saleor order</span></div>
    <div className="tachigo-flow"><span>review</span><strong>domain map</strong><span>source index</span><span>targeted tests</span></div>
  </section>

  <section className="tachigo-checklist" aria-label="entry points by role">
    <section>
      <h2>新同事</h2>
      <div className="tachigo-checklist__body"><span>從 </span><a href="/tachigo/dev-portal/start-here">Start Here</a><span> 進入，先跑完 first-hour route。</span></div>
    </section>
    <section>
      <h2>日常開發</h2>
      <div className="tachigo-checklist__body"><span>從 </span><a href="/tachigo/dev-portal/daily-dev-guide">Daily Dev Guide</a><span> 找改動入口，再回 domain map。</span></div>
    </section>
    <section>
      <h2>架構 review</h2>
      <div className="tachigo-checklist__body"><span>從 </span><a href="/tachigo/dev-portal/flows">Cross-Repo Flows</a><span> 判斷是否碰到 repo、權限或 ledger 邊界。</span></div>
    </section>
  </section>

  <div className="tachigo-band">
    <span>Markdown 仍是 source of truth。既有 docs taxonomy、文件狀態說明與 source links 已搬到 </span><a href="/tachigo/dev-portal/source-index">Source Index</a><span>；首頁只負責導覽決策。</span>
  </div>
</div>
