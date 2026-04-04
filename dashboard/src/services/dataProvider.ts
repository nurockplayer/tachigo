import type { DataProvider } from '@refinedev/core'
import client from '@/services/api'

type ApiResponse<T> = { success: boolean; data: T }

export const dataProvider: DataProvider = {
  getList: async ({ resource, pagination }) => {
    const { data: body } = await client.get<ApiResponse<unknown>>(`/api/v1/${resource}`, {
      params:
        pagination?.currentPage && pagination?.pageSize
          ? { page: pagination.currentPage, page_size: pagination.pageSize }
          : undefined,
    })
    const payload = body.data
    if (Array.isArray(payload)) {
      return { data: payload, total: payload.length }
    }
    const p = payload as { items?: unknown[]; total?: number; data?: unknown[] }
    const items = p.items ?? p.data ?? []
    return { data: items as never[], total: p.total ?? (items as unknown[]).length }
  },

  getOne: async ({ resource, id }) => {
    const { data: body } = await client.get<ApiResponse<unknown>>(`/api/v1/${resource}/${id}`)
    return { data: body.data as never }
  },

  create: async ({ resource, variables }) => {
    const { data: body } = await client.post<ApiResponse<unknown>>(`/api/v1/${resource}`, variables)
    return { data: body.data as never }
  },

  update: async ({ resource, id, variables }) => {
    const { data: body } = await client.put<ApiResponse<unknown>>(
      `/api/v1/${resource}/${id}`,
      variables,
    )
    return { data: body.data as never }
  },

  deleteOne: async ({ resource, id }) => {
    const { data: body } = await client.delete<ApiResponse<unknown>>(`/api/v1/${resource}/${id}`)
    return { data: (body.data ?? {}) as never }
  },

  getApiUrl: () => client.defaults.baseURL ?? '',
}
