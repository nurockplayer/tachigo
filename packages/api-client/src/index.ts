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

interface ApiClientErrorDetails<ResponseBody = unknown> {
  status: number;
  statusText: string;
  response: ResponseBody;
  headers?: Record<string, string>;
  cause?: unknown;
}

export class ApiClientError<ResponseBody = unknown> extends Error {
  readonly status: number;
  readonly statusText: string;
  readonly response: ResponseBody;
  readonly headers: Record<string, string>;
  readonly cause?: unknown;

  constructor(message: string, details: ApiClientErrorDetails<ResponseBody>) {
    super(message);
    this.name = "ApiClientError";
    this.status = details.status;
    this.statusText = details.statusText;
    this.response = details.response;
    this.headers = details.headers ?? {};
    if (Object.hasOwn(details, "cause")) {
      this.cause = details.cause;
    }
  }
}

export function createApiClient(options: ApiClientOptions = {}): ApiClient {
  const fetchImpl = options.fetch ?? (globalThis.fetch as ApiClientOptions["fetch"] | undefined);
  if (typeof fetchImpl !== "function") {
    throw new TypeError("createApiClient requires a fetch implementation.");
  }

  return {
    async request<Operation extends ApiOperation>(
      operation: Operation,
      init: ApiRequestInit<Operation> = {} as ApiRequestInit<Operation>,
    ): Promise<ApiResponse<Operation>> {
      const { method, pathTemplate } = parseOperation(operation);
      const url = buildUrl(
        options.baseUrl ?? "",
        pathTemplate,
        init.pathParams as Record<string, unknown> | undefined,
        init.queryParams as Record<string, unknown> | undefined,
      );
      const headers = await mergeHeaders(options.headers, init.headers);
      const requestInit: RequestInit = {
        method,
        headers,
        signal: init.signal,
      };

      if (Object.hasOwn(init, "requestBody") && init.requestBody !== undefined) {
        requestInit.body = JSON.stringify(init.requestBody);
        if (!hasHeader(headers, "content-type")) {
          headers["content-type"] = "application/json";
        }
      }

      const response = await fetchImpl(url, requestInit);
      const responseBody = await parseResponseBody(response);
      if (!response.ok) {
        throw new ApiClientError(`API request failed with status ${response.status}`, {
          status: response.status,
          statusText: response.statusText,
          response: responseBody,
          headers: headersToObject(response.headers),
        });
      }

      return responseBody as ApiResponse<Operation>;
    },
  };
}

function parseOperation(operation: string): { method: string; pathTemplate: string } {
  const match = /^([A-Z]+)\s+(.+)$/.exec(operation);
  if (!match) {
    throw new TypeError(`Invalid API operation: ${operation}`);
  }

  return {
    method: match[1],
    pathTemplate: match[2],
  };
}

function buildUrl(
  baseUrl: string | URL,
  pathTemplate: string,
  pathParams: Record<string, unknown> | undefined = {},
  queryParams: Record<string, unknown> | undefined = {},
): string {
  const resolvedPathParams = pathParams ?? {};
  const resolvedQueryParams = queryParams ?? {};
  const path = pathTemplate.replace(/\{([^}]+)\}/g, (_, key) => {
    const value = resolvedPathParams[key];
    if (value === undefined || value === null) {
      throw new TypeError(`Missing path param: ${key}`);
    }

    return encodeURIComponent(String(value));
  });

  const base = String(baseUrl).replace(/\/+$/, "");
  const pathWithSlash = path.startsWith("/") ? path : `/${path}`;
  const placeholderOrigin = "http://tachigo.local";
  const baseIsAbsolute = isAbsoluteUrl(base);
  const url = new URL(`${base}${pathWithSlash}`, baseIsAbsolute ? undefined : placeholderOrigin);

  for (const [key, value] of Object.entries(resolvedQueryParams)) {
    appendQueryParam(url, key, value);
  }

  if (baseIsAbsolute) {
    return url.toString();
  }

  return `${url.pathname}${url.search}`;
}

function isAbsoluteUrl(value: string): boolean {
  return /^[A-Za-z][A-Za-z\d+\-.]*:/.test(value);
}

function appendQueryParam(url: URL, key: string, value: unknown): void {
  if (value === undefined || value === null) {
    return;
  }

  if (Array.isArray(value)) {
    for (const item of value) {
      appendQueryParam(url, key, item);
    }
    return;
  }

  url.searchParams.append(key, String(value));
}

async function mergeHeaders(
  defaultHeaders: HeadersInit | HeaderFactory | undefined,
  requestHeaders: HeadersInit | undefined,
): Promise<Record<string, string>> {
  return {
    ...(await resolveHeaders(defaultHeaders)),
    ...(await resolveHeaders(requestHeaders)),
  };
}

async function resolveHeaders(headers: HeadersInit | HeaderFactory | undefined): Promise<Record<string, string>> {
  const resolved = typeof headers === "function" ? await headers() : headers;
  if (!resolved) {
    return {};
  }

  if (typeof Headers !== "undefined" && resolved instanceof Headers) {
    return Object.fromEntries(resolved.entries());
  }

  if (Array.isArray(resolved)) {
    return Object.fromEntries(resolved);
  }

  return { ...resolved };
}

function hasHeader(headers: Record<string, string>, name: string): boolean {
  const normalized = name.toLowerCase();
  return Object.keys(headers).some((key) => key.toLowerCase() === normalized);
}

async function parseResponseBody(response: Response): Promise<unknown> {
  if (response.status === 204 || response.status === 205) {
    return undefined;
  }

  const text = await response.text();
  if (!text) {
    return undefined;
  }

  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    try {
      return JSON.parse(text);
    } catch (error) {
      throw new ApiClientError("API response JSON parse failed", {
        status: response.status,
        statusText: response.statusText,
        response: {
          rawBody: text,
          parseError: formatError(error),
        },
        headers: headersToObject(response.headers),
        cause: error,
      });
    }
  }

  return text;
}

function headersToObject(headers: Headers): Record<string, string> {
  return Object.fromEntries(headers.entries());
}

function formatError(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}
