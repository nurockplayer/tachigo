import type {
  BaseRecord,
  CreateResponse,
  DataProvider,
  DeleteOneResponse,
  GetListResponse,
  GetOneResponse,
  UpdateResponse,
  CreateParams,
  DeleteOneParams,
  GetListParams,
  GetOneParams,
  UpdateParams,
} from '@refinedev/core'
import simpleRestDataProvider from '@refinedev/simple-rest'
import client from '@/services/api'

const API_ORIGIN =
  import.meta.env.VITE_TACHIGO_API_URL
  ?? import.meta.env.VITE_API_URL
  ?? 'http://localhost:8080'

const API_URL = `${API_ORIGIN.replace(/\/$/, '')}/api/v1`

const resourcePaths: Record<string, string> = {
  streamers: '/dashboard/streamers',
  'streamer-channels': '/dashboard/streamers/channels',
  'streamer-stats': '/dashboard/streamers',
  raffles: '/dashboard/raffles',
  transactions: '/users/me/points/history',
  'channel-configs': '/dashboard/channels',
}

const unsupportedResources: Record<string, string> = {
  settings: 'Dashboard settings resource is not wired to a backend endpoint yet',
}

function assertResourceSupported(resource: string) {
  const message = unsupportedResources[resource]
  if (message) {
    throw new Error(message)
  }
}

function resourcePath(resource: string) {
  return resourcePaths[resource] ?? `/${resource}`
}

function apiUrl(path: string) {
  return `${API_URL}${path}`
}

function pathWithId(resource: string, id: string | number) {
  const encodedId = encodeURIComponent(String(id))

  if (resource === 'streamer-stats') {
    return `${resourcePath(resource)}/${encodedId}/stats`
  }

  if (resource === 'channel-configs') {
    return `${resourcePath(resource)}/${encodedId}/config`
  }

  return `${resourcePath(resource)}/${encodedId}`
}

function updateMethod(resource: string) {
  if (resource === 'channel-configs') {
    return client.put.bind(client)
  }

  return client.patch.bind(client)
}

function unwrapPayload<T>(payload: unknown): T {
  if (payload && typeof payload === 'object' && 'data' in payload) {
    return (payload as { data: T }).data
  }

  return payload as T
}

function unwrapList<T extends BaseRecord>(payload: unknown, resource: string): T[] {
  const data = unwrapPayload<unknown>(payload)

  if (Array.isArray(data)) return data as T[]
  if (!data || typeof data !== 'object') return []

  const keyed = data as Record<string, unknown>
  const candidates = [
    resource,
    resource.replace(/^streamer-/, ''),
    'channels',
    'streamers',
    'raffles',
    'transactions',
    'items',
  ]

  for (const key of candidates) {
    const value = keyed[key]
    if (Array.isArray(value)) return value as T[]
  }

  return []
}

function unwrapOne<T extends BaseRecord>(payload: unknown, resource: string): T {
  const data = unwrapPayload<unknown>(payload)

  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    return data as T
  }

  const keyed = data as Record<string, unknown>
  const candidates = [
    resource.replace(/s$/, ''),
    'raffle',
    'streamer',
    'stats',
    'config',
  ]

  for (const key of candidates) {
    const value = keyed[key]
    if (value && typeof value === 'object') {
      if (resource === 'streamer-stats' && key === 'stats') {
        return { ...(value as object), channel_id: keyed.channel_id } as unknown as T
      }

      return value as T
    }
  }

  return data as T
}

export const dataProvider: DataProvider = {
  ...simpleRestDataProvider(API_URL, client),

  getList: async <TData extends BaseRecord = BaseRecord>({
    resource,
    pagination,
    sorters,
    filters,
    meta,
  }: GetListParams): Promise<GetListResponse<TData>> => {
    assertResourceSupported(resource)

    const { data } = await client.get(apiUrl(resourcePath(resource)), {
      params: {
        ...(meta?.params as object | undefined),
        pagination,
        sorters,
        filters,
      },
    })
    const list = unwrapList<TData>(data, resource)

    return {
      data: list,
      total: list.length,
    }
  },

  getOne: async <TData extends BaseRecord = BaseRecord>({
    resource,
    id,
  }: GetOneParams): Promise<GetOneResponse<TData>> => {
    assertResourceSupported(resource)

    const { data } = await client.get(apiUrl(pathWithId(resource, id)))

    return {
      data: unwrapOne<TData>(data, resource),
    }
  },

  create: async <TData extends BaseRecord = BaseRecord, TVariables = Record<string, unknown>>({
    resource,
    variables,
  }: CreateParams<TVariables>): Promise<CreateResponse<TData>> => {
    assertResourceSupported(resource)

    const { data } = await client.post(apiUrl(resourcePath(resource)), variables)

    return {
      data: unwrapOne<TData>(data, resource),
    }
  },

  update: async <TData extends BaseRecord = BaseRecord, TVariables = Record<string, unknown>>({
    resource,
    id,
    variables,
  }: UpdateParams<TVariables>): Promise<UpdateResponse<TData>> => {
    assertResourceSupported(resource)

    const request = updateMethod(resource)
    const { data } = await request(apiUrl(pathWithId(resource, id)), variables)

    return {
      data: unwrapOne<TData>(data, resource),
    }
  },

  deleteOne: async <TData extends BaseRecord = BaseRecord, TVariables = Record<string, unknown>>({
    resource,
    id,
  }: DeleteOneParams<TVariables>): Promise<DeleteOneResponse<TData>> => {
    assertResourceSupported(resource)

    const { data } = await client.delete(apiUrl(pathWithId(resource, id)))

    return {
      data: unwrapOne<TData>(data, resource),
    }
  },

  getApiUrl: () => API_URL,
}
