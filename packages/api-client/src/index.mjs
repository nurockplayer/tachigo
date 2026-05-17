export class ApiClientError extends Error {
  constructor(message, details) {
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

export function createApiClient(options = {}) {
  const fetchImpl = options.fetch ?? globalThis.fetch;
  if (typeof fetchImpl !== "function") {
    throw new TypeError("createApiClient requires a fetch implementation.");
  }

  return {
    async request(operation, init = {}) {
      const { method, pathTemplate } = parseOperation(operation);
      const url = buildUrl(options.baseUrl ?? "", pathTemplate, init.pathParams, init.queryParams);
      const headers = await mergeHeaders(options.headers, init.headers);
      const requestInit = {
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

      return responseBody;
    },
  };
}

function parseOperation(operation) {
  const match = /^([A-Z]+)\s+(.+)$/.exec(operation);
  if (!match) {
    throw new TypeError(`Invalid API operation: ${operation}`);
  }

  return {
    method: match[1],
    pathTemplate: match[2],
  };
}

function buildUrl(baseUrl, pathTemplate, pathParams = {}, queryParams = {}) {
  const path = pathTemplate.replace(/\{([^}]+)\}/g, (_, key) => {
    const value = pathParams[key];
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

  for (const [key, value] of Object.entries(queryParams)) {
    appendQueryParam(url, key, value);
  }

  if (baseIsAbsolute) {
    return url.toString();
  }

  return `${url.pathname}${url.search}`;
}

function isAbsoluteUrl(value) {
  return /^[A-Za-z][A-Za-z\d+\-.]*:/.test(value);
}

function appendQueryParam(url, key, value) {
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

async function mergeHeaders(defaultHeaders, requestHeaders) {
  return {
    ...(await resolveHeaders(defaultHeaders)),
    ...(await resolveHeaders(requestHeaders)),
  };
}

async function resolveHeaders(headers) {
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

function hasHeader(headers, name) {
  const normalized = name.toLowerCase();
  return Object.keys(headers).some((key) => key.toLowerCase() === normalized);
}

async function parseResponseBody(response) {
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

function headersToObject(headers) {
  return Object.fromEntries(headers.entries());
}

function formatError(error) {
  return error instanceof Error ? error.message : String(error);
}
