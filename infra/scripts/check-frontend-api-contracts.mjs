#!/usr/bin/env node
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const files = {
  router: readFile('services/api/internal/router/router.go'),
  swagger: JSON.parse(readFile('services/api/docs/swagger.json')),
  dashboardChannels: readFile('apps/dashboard/src/services/channels.ts'),
  dashboardRaffles: readFile('apps/dashboard/src/services/raffles.ts'),
  dashboardDataProvider: readFile('apps/dashboard/src/providers/dataProvider.ts'),
  extensionApi: readFile('apps/extension/src/services/api.ts'),
}

const checks = [
  {
    label: 'dashboard streamers list',
    frontend: [files.dashboardChannels, '/api/v1/dashboard/streamers'],
    router: 'dashboard.GET("/streamers"',
  },
  {
    label: 'dashboard streamer channels',
    frontend: [files.dashboardChannels, '/api/v1/dashboard/streamers/channels'],
    router: 'dashboard.GET("/streamers/channels"',
  },
  {
    label: 'dashboard channel config',
    frontend: [files.dashboardChannels, '/api/v1/dashboard/channels/${encodedChannelId}/config'],
    router: 'dashboard.GET("/channels/:channel_id/config"',
  },
  {
    label: 'dashboard raffle list',
    frontend: [files.dashboardRaffles, '/api/v1/dashboard/raffles'],
    router: 'dashboard.GET("/raffles"',
    swagger: '/dashboard/raffles',
  },
  {
    label: 'dashboard raffle draws',
    frontend: [files.dashboardRaffles, '/api/v1/dashboard/raffles/${raffleId}/draws'],
    router: 'dashboard.GET("/raffles/:id/draws"',
    swagger: '/dashboard/raffles/{id}/draws',
  },
  {
    label: 'dashboard transactions history',
    frontend: [files.dashboardDataProvider, "transactions: '/users/me/points/history'"],
    router: 'protected.GET("users/me/points/history"',
    swagger: '/users/me/points/history',
  },
  {
    label: 'extension login',
    frontend: [files.extensionApi, '/api/v1/extension/auth/login'],
    router: 'ext.POST("/auth/login"',
    swagger: '/extension/auth/login',
  },
  {
    label: 'extension heartbeat',
    frontend: [files.extensionApi, '/api/v1/extension/watch/heartbeat'],
    router: 'watch.POST("/heartbeat"',
  },
  {
    label: 'extension raffle result',
    frontend: [files.extensionApi, '/api/v1/extension/raffles/${raffleId}/result'],
    router: 'ext.GET("/raffles/:id/result"',
    swagger: '/extension/raffles/{id}/result',
  },
]

function readFile(path) {
  return readFileSync(path, 'utf8')
}

for (const check of checks) {
  assert.ok(
    check.frontend[0].includes(check.frontend[1]),
    `${check.label}: frontend source no longer references ${check.frontend[1]}`,
  )
  assert.ok(
    files.router.includes(check.router),
    `${check.label}: backend router no longer contains ${check.router}`,
  )

  if (check.swagger) {
    assert.ok(
      Object.prototype.hasOwnProperty.call(files.swagger.paths, check.swagger),
      `${check.label}: swagger.json no longer documents ${check.swagger}`,
    )
  }
}

console.log(`Cross-surface API contract smoke passed: ${checks.length} checks.`)
