import type {
  BaseRecord,
  CreateParams,
  CreateResponse,
  DataProvider,
  DeleteOneParams,
  DeleteOneResponse,
  GetListParams,
  GetListResponse,
  GetOneParams,
  GetOneResponse,
  UpdateParams,
  UpdateResponse,
} from '@refinedev/core'
import client from '@/services/api'

// Backend wraps every success response in { success: true, data: <payload> }.
// For single-resource endpoints the payload is often a named object:
//   { data: { user: {...} } }  or  { data: { config: {...} } }
// This helper unwraps both layers so Refine always receives the inner record.
function unwrapItem<TData extends BaseRecord>(envelope: unknown): TData {
  const outer = (envelope as { data?: unknown })?.data

  // If the payload is already a plain record (no extra named key), return it.
  if (!outer || typeof outer !== 'object' || Array.isArray(outer)) {
    return (outer ?? {}) as TData
  }

  const values = Object.values(outer as Record<string, unknown>)
  // Single-key named wrapper → unwrap it.
  if (values.length === 1 && values[0] !== null && typeof values[0] === 'object') {
    return values[0] as TData
  }

  return outer as TData
}

// For list endpoints the payload may be:
//   { data: [...] }  or  { data: { items: [...], total: N } }  or  a raw array
function unwrapList<TData extends BaseRecord>(
  envelope: unknown,
): { data: TData[]; total: number } {
  const outer = (envelope as { data?: unknown })?.data ?? envelope

  if (Array.isArray(outer)) {
    return { data: outer as TData[], total: outer.length }
  }

  if (outer && typeof outer === 'object') {
    const obj = outer as Record<string, unknown>

    // Look for the first array-valued key (the items list).
    for (const val of Object.values(obj)) {
      if (Array.isArray(val)) {
        const total =
          typeof obj['total'] === 'number'
            ? obj['total']
            : typeof obj['count'] === 'number'
              ? obj['count']
              : val.length
        return { data: val as TData[], total }
      }
    }
  }

  return { data: [], total: 0 }
}

export const dataProvider: DataProvider = {
  getList: async <TData extends BaseRecord = BaseRecord>(
    params: GetListParams,
  ): Promise<GetListResponse<TData>> => {
    const { resource, pagination } = params
    const current = pagination?.currentPage
    const pageSize = pagination?.pageSize
    const queryParams =
      current && pageSize ? { page: current, page_size: pageSize } : undefined

    const response = await client.get(`/api/v1/${resource}`, { params: queryParams })
    return unwrapList<TData>(response.data)
  },

  getOne: async <TData extends BaseRecord = BaseRecord>(
    params: GetOneParams,
  ): Promise<GetOneResponse<TData>> => {
    const { resource, id } = params
    const response = await client.get(`/api/v1/${resource}/${id}`)
    return { data: unwrapItem<TData>(response.data) }
  },

  create: async <TData extends BaseRecord = BaseRecord, TVariables = object>(
    params: CreateParams<TVariables>,
  ): Promise<CreateResponse<TData>> => {
    const { resource, variables } = params
    const response = await client.post(`/api/v1/${resource}`, variables)
    return { data: unwrapItem<TData>(response.data) }
  },

  update: async <TData extends BaseRecord = BaseRecord, TVariables = object>(
    params: UpdateParams<TVariables>,
  ): Promise<UpdateResponse<TData>> => {
    const { resource, id, variables } = params
    const response = await client.put(`/api/v1/${resource}/${id}`, variables)
    return { data: unwrapItem<TData>(response.data) }
  },

  deleteOne: async <TData extends BaseRecord = BaseRecord, TVariables = object>(
    params: DeleteOneParams<TVariables>,
  ): Promise<DeleteOneResponse<TData>> => {
    const { resource, id } = params
    const response = await client.delete(`/api/v1/${resource}/${id}`)
    return { data: unwrapItem<TData>(response.data) }
  },

  getApiUrl: () => client.defaults.baseURL ?? '',
}
