import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ApiError, request } from "./api";

// ---------------------------------------------------------------------------
// ApiError
// ---------------------------------------------------------------------------

describe("ApiError", () => {
  it("has the correct name and properties", () => {
    const err = new ApiError(404, "NOT_FOUND", "User not found", {
      id: "123",
    });
    expect(err.name).toBe("ApiError");
    expect(err.message).toBe("User not found");
    expect(err.status).toBe(404);
    expect(err.code).toBe("NOT_FOUND");
    expect(err.details).toEqual({ id: "123" });
  });

  it("is instanceof Error", () => {
    const err = new ApiError(500, "INTERNAL_ERROR", "Oops");
    expect(err).toBeInstanceOf(Error);
    expect(err).toBeInstanceOf(ApiError);
  });
});

// ---------------------------------------------------------------------------
// request() — happy path and error parsing
// ---------------------------------------------------------------------------

describe("request()", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    fetchMock.mockReset();
  });

  it("returns parsed JSON on 200", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ ok: true }), { status: 200 })
    );
    const result = await request<{ ok: boolean }>("/api/v1/test");
    expect(result).toEqual({ ok: true });
  });

  it("returns undefined on 204", async () => {
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }));
    const result = await request<void>("/api/v1/test");
    expect(result).toBeUndefined();
  });

  it("throws ApiError with parsed code on 400", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          error: { code: "VALIDATION_ERROR", message: "Bad input" },
        }),
        { status: 400 }
      )
    );
    await expect(request("/api/v1/test")).rejects.toMatchObject({
      status: 400,
      code: "VALIDATION_ERROR",
      message: "Bad input",
    });
  });

  it("throws ApiError with INTERNAL_ERROR when body is not JSON", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response("not json", { status: 500 })
    );
    await expect(request("/api/v1/test")).rejects.toMatchObject({
      status: 500,
      code: "INTERNAL_ERROR",
    });
  });

  it("sends credentials: include and Content-Type: application/json", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({}), { status: 200 })
    );
    await request("/api/v1/test");
    const [, init] = fetchMock.mock.calls[0]!;
    expect(init?.credentials).toBe("include");
    expect((init?.headers as Record<string, string>)?.["Content-Type"]).toBe(
      "application/json"
    );
  });

  it("retries after a successful refresh on 401", async () => {
    // First call → 401, refresh → 200, retry → 200
    fetchMock
      .mockResolvedValueOnce(new Response(null, { status: 401 })) // original call
      .mockResolvedValueOnce(new Response(null, { status: 200 })) // refresh
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ retried: true }), { status: 200 })
      ); // retry

    const result = await request<{ retried: boolean }>("/api/v1/test");
    expect(result).toEqual({ retried: true });
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("dispatches auth:unauthenticated and throws when refresh fails on 401", async () => {
    const events: Event[] = [];
    window.addEventListener("auth:unauthenticated", (e) => events.push(e));

    fetchMock
      .mockResolvedValueOnce(new Response(null, { status: 401 })) // original call
      .mockResolvedValueOnce(new Response(null, { status: 401 })); // refresh fails

    await expect(request("/api/v1/test")).rejects.toMatchObject({
      status: 401,
      code: "UNAUTHORIZED",
    });

    expect(events).toHaveLength(1);
    expect(events[0]!.type).toBe("auth:unauthenticated");

    window.removeEventListener("auth:unauthenticated", (e) => events.push(e));
  });
});
