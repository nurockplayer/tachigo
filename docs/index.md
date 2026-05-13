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
  <section className="tachigo-hero">
    <div className="tachigo-hero__copy">
      <div>
        <div className="tachigo-kicker">Repo-first guide</div>
        <h1>tachigo Dev Portal</h1>
        <div className="tachigo-lede">
          給新同事與日常開發者使用的專案導覽入口：先看系統地圖，再進 domain map，
          最後一路點回實際 source、tests、API route 與跨 repo flow。
        </div>
      </div>
      <div className="tachigo-actions">
        <a className="tachigo-action tachigo-action--primary" href="/tachigo/dev-portal/start-here">Start onboarding path</a>
        <a className="tachigo-action" href="/tachigo/dev-portal/domain-maps">Find a feature</a>
        <a className="tachigo-action" href="/tachigo/dev-portal/source-index">Open source index</a>
      </div>
    </div>
    <div className="tachigo-map" aria-label="tachigo system map">
      <div className="tachigo-node">
        <strong>Twitch</strong>
        <span>Viewer identity, channel context, extension session</span>
      </div>
      <div className="tachigo-node">
        <strong>Extension</strong>
        <span>Sidepanel UI, heartbeat, point balance, coupon shop</span>
      </div>
      <div className="tachigo-node">
        <strong>Dashboard</strong>
        <span>Agency and streamer operations surface</span>
      </div>
      <div className="tachigo-node tachigo-node--api">
        <strong>tachigo API</strong>
        <span>Go + Gin boundary for auth, watch, points, spend, claim</span>
      </div>
      <div className="tachigo-node tachigo-node--data">
        <strong>PostgreSQL</strong>
        <span>Users, providers, watch sessions, ledger, balances, redemptions</span>
      </div>
      <div className="tachigo-node tachigo-node--tachiya">
        <strong>tachiya</strong>
        <span>FastAPI commerce integration and Saleor protection layer</span>
      </div>
      <div className="tachigo-node">
        <strong>Saleor</strong>
        <span>Checkout, orders, discount application</span>
      </div>
      <div className="tachigo-node tachigo-node--chain">
        <strong>Chain</strong>
        <span>Future claim / soulbound token surface</span>
      </div>
    </div>
  </section>

  <section className="tachigo-grid" aria-label="primary guide cards">
    <a className="tachigo-card" href="/tachigo/dev-portal/start-here">
      <div className="tachigo-card__eyebrow">01</div>
      <h2>Onboarding</h2>
      <div className="tachigo-card__body">First hour、first day、first PR：照著走，先建立正確 mental model。</div>
    </a>
    <a className="tachigo-card" href="/tachigo/dev-portal/domain-maps">
      <div className="tachigo-card__eyebrow">02</div>
      <h2>Domain Maps</h2>
      <div className="tachigo-card__body">Points、Auth、Extension 三個 P0 domain 的 source、tests、routes 與踩雷點。</div>
    </a>
    <a className="tachigo-card" href="/tachigo/dev-portal/daily-dev-guide">
      <div className="tachigo-card__eyebrow">03</div>
      <h2>Daily Dev</h2>
      <div className="tachigo-card__body">「我要改 X」時該從哪裡打開、跑哪些測試、PR scope 要怎麼守住。</div>
    </a>
    <a className="tachigo-card" href="/tachigo/dev-portal/graph-explorer">
      <div className="tachigo-card__eyebrow">04</div>
      <h2>Graph Explorer</h2>
      <div className="tachigo-card__body">把 graphify 當成影響範圍雷達，而不是架構真相本身。</div>
    </a>
  </section>
</div>

## 這個入口怎麼用

<div className="tachigo-checklist">
  <section>
    <h2>新同事</h2>
    <div className="tachigo-checklist__body">從 <a href="/tachigo/dev-portal/start-here">Start Here</a> 開始，先理解主要系統，再挑一條 flow 追 source。</div>
  </section>
  <section>
    <h2>日常開發</h2>
    <div className="tachigo-checklist__body">從 <a href="/tachigo/dev-portal/daily-dev-guide">Daily Dev Guide</a> 找改動入口，再用 Domain Maps 確認測試。</div>
  </section>
  <section>
    <h2>架構 review</h2>
    <div className="tachigo-checklist__body">從 <a href="/tachigo/dev-portal/flows">Cross-Repo Flows</a> 判斷改動是否跨 repo、跨權限或跨 ledger。</div>
  </section>
</div>

<div className="tachigo-band">
  Markdown 仍是 source of truth。既有 docs taxonomy、文件狀態說明與 source links 已搬到
  <a href="/tachigo/dev-portal/source-index"> Source Index</a>；這個首頁只保留導覽任務。
</div>
