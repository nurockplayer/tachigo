import type { ApiOperations } from "@tachigo/shared-types";

export type ApiOperation = keyof ApiOperations & string;

type OperationSpec<Operation extends ApiOperation> = ApiOperations[Operation];
type PropertyValue<Spec, Key extends PropertyKey> = Key extends keyof Spec ? Spec[Key] : never;
type PropertyOption<Key extends string, Value> = undefined extends Value
  ? { [Property in Key]?: Exclude<Value, undefined> }
  : { [Property in Key]: Value };

type PathParamsOption<Operation extends ApiOperation> = "pathParams" extends keyof OperationSpec<Operation>
  ? PropertyOption<"pathParams", PropertyValue<OperationSpec<Operation>, "pathParams">>
  : { pathParams?: never };

type QueryParamsOption<Operation extends ApiOperation> = "queryParams" extends keyof OperationSpec<Operation>
  ? PropertyOption<"queryParams", PropertyValue<OperationSpec<Operation>, "queryParams">>
  : { queryParams?: never };

type RequestBodyOption<Operation extends ApiOperation> = "requestBody" extends keyof OperationSpec<Operation>
  ? PropertyOption<"requestBody", PropertyValue<OperationSpec<Operation>, "requestBody">>
  : { requestBody?: never };

type RequiredKeys<Type> = {
  [Key in keyof Type]-?: {} extends Pick<Type, Key> ? never : Key;
}[keyof Type];

export type ApiResponse<Operation extends ApiOperation> = PropertyValue<
  OperationSpec<Operation>,
  "response"
>;

export type ApiRequestInit<Operation extends ApiOperation> = PathParamsOption<Operation> &
  QueryParamsOption<Operation> &
  RequestBodyOption<Operation> & {
    headers?: HeadersInit;
    signal?: AbortSignal;
  };

export type HeaderFactory = () => HeadersInit | Promise<HeadersInit>;

export interface ApiClientOptions {
  baseUrl?: string | URL;
  fetch?: (input: string, init?: RequestInit) => Promise<Response>;
  headers?: HeadersInit | HeaderFactory;
}

export interface ApiClient {
  request<Operation extends ApiOperation>(
    operation: Operation,
    ...args: RequiredKeys<ApiRequestInit<Operation>> extends never
      ? [init?: ApiRequestInit<Operation>]
      : [init: ApiRequestInit<Operation>]
  ): Promise<ApiResponse<Operation>>;
}

export class ApiClientError<ResponseBody = unknown> extends Error {
  readonly status: number;
  readonly statusText: string;
  readonly response: ResponseBody;
  readonly headers: Record<string, string>;
  readonly cause?: unknown;
}

export function createApiClient(options?: ApiClientOptions): ApiClient;
