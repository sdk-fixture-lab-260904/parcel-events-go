<!-- managed by Doctorine — edits will be overwritten -->
This client is generated FROM the API's spec-derived policy, not hand-tuned:
- Raw status, headers, and a captured `x-request-id` on every response.
- Retries and backoff match this API's declared policy exactly (no invented defaults).
- Pagination termination follows the wire-truth predicate for each bound scheme.
It is a starting point, not the only correct client — adapt it to your codebase.

```ts
// doctorine-skill-recipe-client v1 — generated; semantics mirror the SDK kernel

const DEFAULT_TIMEOUT_MS = 45000; // client.timeouts.defaultMs
const REQUEST_ID_HEADER = "x-request-id";
const IDEMPOTENCY_HEADER = "idempotency-key"; // client.idempotency.header
function randomIdempotencyKey(): string {
  return `idem_${Math.random().toString(36).slice(2)}${Date.now().toString(36)}`;
}

// --auth--
const API_KEY_ENV_VAR = "PARCEL_EVENTS_API_KEY"; // apiKey scheme "api_key", declared `in: header`
function authHeaders(apiKey: string): Record<string, string> {
  return { "Parcel-API-Key": apiKey };
}

// Fail-closed credential lookup for the "Using it" example: throws BEFORE any request is sent.
declare const process: { readonly env: Readonly<Record<string, string | undefined>> };
export function requireEnv(name: string): string {
  const value = process.env[name];
  if (value === undefined || value === "") {
    throw new Error(`missing credential env var ${name} — see SKILL.md "Authentication"`);
  }
  return value;
}
// --end-auth--

// Escape hatch: this API also declares bearer_token as alternative auth schemes.

// retries: traced to client.retries (max 3, exponent 2, no jitter — reproducible schedules)
const MAX_RETRIES = 3;
const INITIAL_DELAY_MS = 250;
const MAX_DELAY_MS = 8000;
const MAX_ELAPSED_MS = 60000;
const EXPONENT = 2;
const RETRY_STATUSES: readonly (number | "4XX" | "5XX")[] = [408,409,429,"5XX"];
const RETRY_CONNECTION_ERRORS = true;

/** A non-JSON (or JSON-lying) failure body — the response cannot be parsed as the API's error shape. */
export class TransportError extends Error {
  constructor(readonly status: number, readonly statusText: string, readonly bodyText: string, readonly requestId: string | undefined) {
    super(`transport error: HTTP ${status} ${statusText} (non-JSON body)`);
    this.name = "TransportError";
  }
}

/** A well-formed JSON API error — `body` is the parsed error payload verbatim. */
export class ApiError extends Error {
  constructor(readonly status: number, readonly body: unknown, readonly requestId: string | undefined) {
    super(`API error: HTTP ${status}`);
    this.name = "ApiError";
  }
}

function statusMatches(status: number, matcher: number | "4XX" | "5XX"): boolean {
  if (typeof matcher === "number") return status === matcher;
  return matcher === "4XX" ? status >= 400 && status < 500 : status >= 500 && status < 600;
}

// Retry-After: delta-seconds ("3") or an IMF-fixdate; unparseable -> undefined (fall back to backoff).
function parseRetryAfterMs(value: string | null, nowMs: number): number | undefined {
  if (value === null) return undefined;
  const trimmed = value.trim();
  if (/^\d+$/.test(trimmed)) return Number(trimmed) * 1000;
  const at = Date.parse(trimmed);
  return Number.isNaN(at) ? undefined : Math.max(0, at - nowMs);
}

function buildUrl(baseUrl: string, path: string, query: Readonly<Record<string, string | number | boolean>> | undefined): string {
  const base = /^https?:\/\//.test(path) ? path : `${baseUrl.replace(/\/+$/, "")}${path.startsWith("/") ? "" : "/"}${path}`;
  const entries = Object.entries(query ?? {});
  if (entries.length === 0) return base;
  const qs = entries.map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`).join("&");
  return `${base}${base.includes("?") ? "&" : "?"}${qs}`;
}

async function readBody<T>(response: Response): Promise<T> {
  if (response.status === 204) return null as T;
  const contentType = response.headers.get("content-type") ?? "";
  // MUST NOT: buffer a live stream into one string — honest refusal beats a silently wrong consume.
  if (contentType.includes("text/event-stream")) {
    throw new Error("apiRequest refuses text/event-stream — consume it per the Streaming section of recipes/http-client.md");
  }
  if (!contentType.includes("json")) return (await response.text()) as T;
  const text = await response.text();
  return (text === "" ? null : JSON.parse(text)) as T;
}

// Non-JSON (or JSON-lying) error bodies degrade to TransportError; a well-formed JSON error body throws the typed ApiError.
async function throwForFailure(response: Response, requestId: string | undefined): Promise<never> {
  const contentType = response.headers.get("content-type") ?? "";
  const text = await response.text();
  if (contentType.includes("json")) {
    try {
      const body = text === "" ? null : JSON.parse(text);
      throw new ApiError(response.status, body, requestId);
    } catch (error) {
      if (error instanceof ApiError) throw error; // else: the body LIES about its content-type — fall through
    }
  }
  throw new TransportError(response.status, response.statusText, text, requestId);
}

export interface ClientOptions {
  readonly baseUrl: string;
  readonly apiKey: string;
  readonly fetch?: typeof fetch;
  readonly timeoutMs?: number;
  readonly sleep?: (ms: number) => Promise<void>;
  readonly now?: () => number;
}

export interface RequestOptions {
  readonly query?: Readonly<Record<string, string | number | boolean>>;
  readonly headers?: Readonly<Record<string, string>>;
  /** JSON-serialized, content-type application/json — for JSON bodies ONLY. */
  readonly body?: unknown;
  /** Sent AS-IS — the non-JSON switch (multipart/form-data via FormData: fetch sets the boundary,
   *  leave content-type unset; application/octet-stream: pass the bytes and set content-type
   *  yourself). Retries re-send it — pass re-readable bytes/FormData, never a one-shot stream. */
  readonly rawBody?: BodyInit;
  readonly idempotencyKey?: string;
}

export interface ApiResponse<T> {
  readonly status: number;
  readonly requestId: string | undefined;
  readonly data: T;
}

export async function apiRequest<T = unknown>(
  client: ClientOptions,
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD" | "OPTIONS" | "TRACE",
  path: string,
  options: RequestOptions = {},
): Promise<ApiResponse<T>> {
  const fetchFn = client.fetch ?? fetch;
  const sleepFn = client.sleep ?? ((ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms)));
  const nowFn = client.now ?? (() => Date.now());
  const timeoutMs = client.timeoutMs ?? DEFAULT_TIMEOUT_MS;
  const query: Record<string, string | number | boolean> = { ...options.query };
  const headers: Record<string, string> = { accept: "application/json" };
  for (const [key, value] of Object.entries(options.headers ?? {})) headers[key.toLowerCase()] = value;
  if (!client.apiKey) throw new Error(`missing credential: ClientOptions.apiKey (env ${API_KEY_ENV_VAR}) — refusing to send the request unauthenticated`);
  for (const [key, value] of Object.entries(authHeaders(client.apiKey))) headers[key.toLowerCase()] = value;
  headers[IDEMPOTENCY_HEADER] = options.idempotencyKey ?? randomIdempotencyKey(); // ONE key per logical request, reused across every retry attempt
  if (options.body !== undefined && options.rawBody !== undefined) {
    throw new Error("pass body (JSON) OR rawBody (non-JSON), never both");
  }
  const body = options.rawBody ?? (options.body === undefined ? undefined : JSON.stringify(options.body));
  if (options.body !== undefined) headers["content-type"] ??= "application/json"; // rawBody callers own content-type (FormData: leave unset)
  const url = buildUrl(client.baseUrl, path, query);
  const startMs = nowFn();
  for (let attempt = 0; ; attempt++) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);
    let response: Response;
    try {
      response = await fetchFn(url, { method, headers, body, signal: controller.signal });
    } catch (networkError) {
      clearTimeout(timer);
      const elapsedMs = nowFn() - startMs;
      const delayMs = Math.min(INITIAL_DELAY_MS * EXPONENT ** attempt, MAX_DELAY_MS);
      if (!RETRY_CONNECTION_ERRORS || attempt >= MAX_RETRIES || elapsedMs + delayMs > MAX_ELAPSED_MS) throw networkError;
      await sleepFn(delayMs);
      continue;
    }
    clearTimeout(timer);
    const requestId = response.headers.get(REQUEST_ID_HEADER) ?? undefined;
    if (response.ok) return { status: response.status, requestId, data: await readBody<T>(response) };
    const isRetryableStatus = RETRY_STATUSES.some((matcher) => statusMatches(response.status, matcher));
    const retryAfterMs = parseRetryAfterMs(response.headers.get("retry-after"), nowFn());
    const delayMs =
      retryAfterMs !== undefined
        ? Math.min(Math.max(retryAfterMs, 0), MAX_ELAPSED_MS)
        : Math.min(INITIAL_DELAY_MS * EXPONENT ** attempt, MAX_DELAY_MS);
    const elapsedMs = nowFn() - startMs;
    if (attempt >= MAX_RETRIES || !isRetryableStatus || elapsedMs + delayMs > MAX_ELAPSED_MS) {
      return await throwForFailure(response, requestId);
    }
    await sleepFn(delayMs);
  }
}

interface PageParams {
  readonly query: Record<string, string | number | boolean>;
  readonly headers: Record<string, string>;
}

const NO_PARAMS: PageParams = { query: {}, headers: {} };

function bodyAt(body: unknown, pointer: readonly string[]): unknown {
  let current: unknown = body;
  for (const segment of pointer) {
    if (current === null || typeof current !== "object") return undefined;
    current = (current as Record<string, unknown>)[segment];
  }
  return current;
}

function itemsAt<T>(body: unknown, pointer: readonly string[]): T[] {
  const value = bodyAt(body, pointer);
  return Array.isArray(value) ? (value as T[]) : [];
}

// A crisp throw beats a silently wrong request: body/path pagination locators are
// not modeled by this recipe — see reference/pagination.md for that operation's binding.
function withRoleValue(current: PageParams, location: string, wireName: string, value: string | number): PageParams {
  if (location === "header") return { query: current.query, headers: { ...current.headers, [wireName]: String(value) } };
  if (location === "query") return { query: { ...current.query, [wireName]: value }, headers: current.headers };
  throw new Error(`pagination role location "${location}" is not covered by this recipe`);
}

/** cursor scheme "cursor_page": request.cursor -> query.cursor; response.items -> $.data; response.next_cursor -> $.next_cursor. */
export async function* listAllCursor<T>(client: ClientOptions, path: string, params: PageParams = NO_PARAMS): AsyncGenerator<T> {
  let current = params;
  for (;;) {
    const { data } = await apiRequest(client, "GET", path, { query: current.query, headers: current.headers });
    const items = itemsAt<T>(data, ["data"]);
    if (items.length === 0) return; // MUST: an empty page stops iteration
    for (const item of items) yield item;
    const nextCursor = bodyAt(data, ["next_cursor"]);
    if (nextCursor === null || nextCursor === undefined || nextCursor === "") return; // MUST: null/''/undefined stops
    if (typeof nextCursor !== "string" && typeof nextCursor !== "number") return;
    current = withRoleValue(current, "query", "cursor", nextCursor); // MUST NOT: mutate the opaque cursor value
  }
}
```

## Using it

```ts
const client: ClientOptions = {
  baseUrl: "https://api.parcel-events.example", // the "production" environment — SKILL.md "Base URL" decides among all 2
  // Credentials are REQUIRED — requireEnv throws BEFORE any request when a var is unset (fail-closed).
  apiKey: requireEnv(API_KEY_ENV_VAR),
};

// GET /v1/labels/{label_id}
const label_id = "REPLACE_label_id"; // required path parameter
const { data } = await apiRequest(client, "GET", `/v1/labels/${label_id}`);

// A call that sends a REQUIRED request body — every operation marked `body required` in the operation index needs one:
// POST /v1/labels
const payload = { /* required application/json body (CreateLabelRequest) — full shape: the operation's reference page + openapi/openapi.json */ };
const { data: data_2 } = await apiRequest(client, "POST", "/v1/labels", {
  body: payload,
});

// GET /v1/pickups — paginated (cursor)
for await (const item of listAllCursor(client, "/v1/pickups")) {
  console.log(item);
}
```

## What this deliberately does not do

- No automatic re-auth on 401 — out of scope for this recipe; the shipped SDK kernel
  performs exactly one transparent re-auth when the auth provider can invalidate a cached credential.
- No connection pooling or keep-alive tuning — it relies on the runtime's default fetch agent.
- No jitter in backoff — schedules stay reproducible, the same determinism discipline as the SDK kernel;
  add jitter yourself if you operate at very high concurrency.
