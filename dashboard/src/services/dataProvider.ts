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

type ApiListResponse<TData extends BaseRecord> = {
  data?: TData[]
  total?: number
  meta?: {
    total?: number
  }
}

type ApiItemResponse<TData extends BaseRecord> = {
  data: TData
}

function getCollectionFromResponse<TData extends BaseRecord>(
  payload: ApiListResponse<TData> | TData[],
): {
  data: TData[]
  total: number
} {
  if (Array.isArray(payload)) {
    return {
      data: payload,
      total: payload.length,
    }
  }

  const data = Array.isArray(payload.data) ? payload.data : []
  const total = payload.total ?? payload.meta?.total ?? data.length

  return { data, total }
}

export const dataProvider: DataProvider = {
  getList: async <TData extends BaseRecord = BaseRecord>(
    params: GetListParams,
  ): Promise<GetListResponse<TData>> => {
    const { resource, pagination } = params
    const current = pagination?.currentPage
    const pageSize = pagination?.pageSize
    const queryParams =
      current && pageSize
        ? {
            page: current,
            page_size: pageSize,
          }
        : undefined

    const response = await client.get<ApiListResponse<TData> | TData[]>(`/api/v1/${resource}`, {
      params: queryParams,
    })
    const { data, total } = getCollectionFromResponse(response.data)

    return { data, total }
  },
  getOne: async <TData extends BaseRecord = BaseRecord>(
    params: GetOneParams,
  ): Promise<GetOneResponse<TData>> => {
    const { resource, id } = params
    const response = await client.get<ApiItemResponse<TData>>(`/api/v1/${resource}/${id}`)

    return { data: response.data.data }
  },
  create: async <TData extends BaseRecord = BaseRecord, TVariables = {}>(
    params: CreateParams<TVariables>,
  ): Promise<CreateResponse<TData>> => {
    const { resource, variables } = params
    const response = await client.post<ApiItemResponse<TData>>(`/api/v1/${resource}`, variables)

    return { data: response.data.data }
  },
  update: async <TData extends BaseRecord = BaseRecord, TVariables = {}>(
    params: UpdateParams<TVariables>,
  ): Promise<UpdateResponse<TData>> => {
    const { resource, id, variables } = params
    const response = await client.put<ApiItemResponse<TData>>(
      `/api/v1/${resource}/${id}`,
      variables,
    )

    return { data: response.data.data }
  },
  deleteOne: async <TData extends BaseRecord = BaseRecord, TVariables = {}>(
    params: DeleteOneParams<TVariables>,
  ): Promise<DeleteOneResponse<TData>> => {
    const { resource, id } = params
    const response = await client.delete<ApiItemResponse<TData>>(`/api/v1/${resource}/${id}`)

    return { data: response.data.data }
  },
  getApiUrl: () => client.defaults.baseURL ?? '',
}
