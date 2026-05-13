import assert from "node:assert/strict";
import { describe, it } from "node:test";

import { ApiClientError, createApiClient } from "./index.mjs";

function createFetchRecorder(response) {
  const calls = [];
  const fetchImpl = async (url, init) => {
    calls.push({ url: String(url), init });
    return response;
  };

  return { calls, fetchImpl };
}

function jsonResponse(body, init = {}) {
  return new Response(JSON.stringify(body), {
    status: init.status ?? 200,
    headers: {
      "content-type": "application/json",
      ...(init.headers ?? {}),
    },
  });
}

describe("createApiClient", () => {
  it("serializes typed operations with path params, query params, JSON body, and headers", async () => {
    const recorder = createFetchRecorder(
      jsonResponse({ success: true, data: { message: "ok" } }),
    );
    const client = createApiClient({
      baseUrl: "https://api.example.test/api/v1/",
      fetch: recorder.fetchImpl,
      headers: async () => ({ authorization: "Bearer token-a" }),
    });

    await client.request("PUT /users/me/addresses/{id}", {
      pathParams: { id: "addr/1" },
      queryParams: { preview: true, empty: undefined },
      requestBody: {
        address_line1: "Road 1",
        city: "Taipei",
        recipient_name: "Tachi",
      },
      headers: { "x-request-id": "req-1" },
    });

    assert.equal(
      recorder.calls[0].url,
      "https://api.example.test/api/v1/users/me/addresses/addr%2F1?preview=true",
    );
    assert.equal(recorder.calls[0].init.method, "PUT");
    assert.equal(recorder.calls[0].init.headers.authorization, "Bearer token-a");
    assert.equal(recorder.calls[0].init.headers["x-request-id"], "req-1");
    assert.equal(recorder.calls[0].init.headers["content-type"], "application/json");
    assert.deepEqual(JSON.parse(recorder.calls[0].init.body), {
      address_line1: "Road 1",
      city: "Taipei",
      recipient_name: "Tachi",
    });
  });

  it("returns parsed JSON responses for successful requests", async () => {
    const recorder = createFetchRecorder(
      jsonResponse({ success: true, data: { user: { id: "u1" } } }),
    );
    const client = createApiClient({
      baseUrl: "https://api.example.test/api/v1",
      fetch: recorder.fetchImpl,
    });

    const result = await client.request("GET /users/me");

    assert.deepEqual(result, { success: true, data: { user: { id: "u1" } } });
    assert.equal(recorder.calls[0].init.body, undefined);
    assert.equal(recorder.calls[0].init.headers["content-type"], undefined);
  });

  it("throws ApiClientError with parsed response details when the API returns an error status", async () => {
    const client = createApiClient({
      baseUrl: "https://api.example.test/api/v1",
      fetch: async () => jsonResponse({ success: false, error: "unauthorized" }, { status: 401 }),
    });

    await assert.rejects(
      () => client.request("GET /users/me"),
      (error) => {
        assert.ok(error instanceof ApiClientError);
        assert.equal(error.status, 401);
        assert.deepEqual(error.response, { success: false, error: "unauthorized" });
        return true;
      },
    );
  });

  it("supports relative base URLs, repeated query params, and text responses", async () => {
    const recorder = createFetchRecorder(
      new Response("ok", {
        status: 200,
        headers: { "content-type": "text/plain" },
      }),
    );
    const client = createApiClient({
      baseUrl: "/api/v1/",
      fetch: recorder.fetchImpl,
    });

    const result = await client.request("GET /dashboard/raffles/{id}/draws", {
      pathParams: { id: "raffle 1" },
      queryParams: { tag: ["alpha", "beta"], empty: null },
    });

    assert.equal(
      recorder.calls[0].url,
      "/api/v1/dashboard/raffles/raffle%201/draws?tag=alpha&tag=beta",
    );
    assert.equal(result, "ok");
  });

  it("throws a TypeError when a path param is missing", async () => {
    const client = createApiClient({
      baseUrl: "https://api.example.test/api/v1",
      fetch: async () => jsonResponse({ success: true }),
    });

    await assert.rejects(
      () => client.request("GET /dashboard/raffles/{id}/draws", { pathParams: {} }),
      (error) => {
        assert.ok(error instanceof TypeError);
        assert.match(error.message, /Missing path param: id/);
        return true;
      },
    );
  });

  it("returns undefined for empty success responses", async () => {
    const client = createApiClient({
      baseUrl: "https://api.example.test/api/v1",
      fetch: async () => new Response(null, { status: 204 }),
    });

    assert.equal(await client.request("POST /auth/verify-email/send"), undefined);
  });

  it("returns undefined for 205 reset-content responses", async () => {
    const client = createApiClient({
      baseUrl: "https://api.example.test/api/v1",
      fetch: async () => new Response(null, { status: 205 }),
    });

    assert.equal(await client.request("POST /auth/verify-email/send"), undefined);
  });

  it("wraps invalid JSON responses in ApiClientError with raw response details", async () => {
    const client = createApiClient({
      baseUrl: "https://api.example.test/api/v1",
      fetch: async () =>
        new Response("{", {
          status: 200,
          statusText: "OK",
          headers: {
            "content-type": "application/json",
            "x-trace-id": "trace-1",
          },
        }),
    });

    await assert.rejects(
      () => client.request("GET /users/me"),
      (error) => {
        assert.ok(error instanceof ApiClientError);
        assert.equal(error.status, 200);
        assert.equal(error.statusText, "OK");
        assert.equal(error.response.rawBody, "{");
        assert.match(error.response.parseError, /JSON|Unexpected|Expected/);
        assert.equal(error.headers["content-type"], "application/json");
        assert.equal(error.headers["x-trace-id"], "trace-1");
        assert.ok(error.cause instanceof SyntaxError);
        return true;
      },
    );
  });
});
