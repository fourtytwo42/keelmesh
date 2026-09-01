import { afterEach, describe, expect, it, vi } from "vitest";
import { api, KeelMeshError, requestID, streamURL } from "./api";

describe("API client contracts", () => {
  afterEach(() => vi.restoreAllMocks());

  it("preserves stable server error codes", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "STALE_STATE", message: "refresh" }), { status: 409 })));
    await expect(api("/api/v1/plans")).rejects.toEqual(new KeelMeshError("STALE_STATE", "refresh"));
  });

  it("creates traceable request IDs and same-origin stream URLs", () => {
    expect(requestID("compile")).toMatch(/^compile-[0-9a-f-]+$/);
    expect(streamURL()).toBe("ws://localhost:3000/api/v1/stream");
  });
});

