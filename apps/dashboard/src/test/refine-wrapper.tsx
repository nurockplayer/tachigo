import { act } from 'react'
import type { BaseRecord, DataProvider } from '@refinedev/core'
import { Refine } from '@refinedev/core'
import type { ReactNode } from 'react'

export type MockGetListFn = () => Promise<BaseRecord[]>
export type MockGetOneFn = (id: string | number) => Promise<BaseRecord>
export type MockCreateFn = (variables: unknown) => Promise<BaseRecord>

export interface MockDataConfig {
  getList?: Record<string, MockGetListFn>
  getOne?: Record<string, MockGetOneFn>
  create?: Record<string, MockCreateFn>
}

export function createMockDataProvider(config: MockDataConfig): DataProvider {
  return {
    getList: async ({ resource }) => {
      const handler = config.getList?.[resource]
      if (!handler) return { data: [], total: 0 }
      const data = await handler()
      return { data, total: data.length }
    },
    getOne: async ({ resource, id }) => {
      const handler = config.getOne?.[resource]
      if (!handler) return { data: {} as BaseRecord }
      const data = await handler(id)
      return { data }
    },
    create: async ({ resource, variables }) => {
      const handler = config.create?.[resource]
      if (!handler) return { data: {} as BaseRecord }
      const data = await handler(variables)
      return { data }
    },
    update: async () => ({ data: {} as BaseRecord }),
    deleteOne: async () => ({ data: {} as BaseRecord }),
    getApiUrl: () => 'http://localhost:8080/api/v1',
  }
}

export function RefineWrapper({
  children,
  dataProvider,
}: {
  children: ReactNode
  dataProvider: DataProvider
}) {
  return (
    <Refine dataProvider={dataProvider}>
      {children}
    </Refine>
  )
}

/**
 * Flush pending React state updates and TanStack Query async chains.
 * Uses setTimeout(0) rather than Promise.resolve() so that macro-tasks
 * scheduled by React 19 concurrent rendering also get a chance to run.
 */
export async function flushAsync() {
  await act(async () => {
    await new Promise<void>(resolve => setTimeout(resolve, 0))
  })
}

/**
 * Repeatedly flush until `assertion` passes or `maxMs` expires.
 * Replaces bare flush() calls in tests that need data to appear.
 */
export async function waitFor(
  assertion: () => void,
  maxMs = 2000,
): Promise<void> {
  const deadline = Date.now() + maxMs
  let lastError: unknown
  while (Date.now() < deadline) {
    await flushAsync()
    try {
      assertion()
      return
    } catch (e) {
      lastError = e
    }
  }
  throw lastError
}
