import type { APIError } from "./types";

export class KeelMeshError extends Error {
  constructor(public code: string, message: string) { super(message); }
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) },
  });
  const value = await response.json() as T | APIError;
  if (!response.ok) {
    const problem = value as APIError;
    throw new KeelMeshError(problem.code ?? "REQUEST_FAILED", problem.message ?? "Request failed");
  }
  return value as T;
}

export function clientUUID(): string {
  if (typeof crypto.randomUUID === "function") return crypto.randomUUID();
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  // RFC 4122 version 4 and variant bits. getRandomValues remains available
  // on LAN HTTP even though randomUUID is restricted to secure contexts.
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;
  const value = [...bytes].map((byte) => byte.toString(16).padStart(2, "0")).join("");
  return `${value.slice(0, 8)}-${value.slice(8, 12)}-${value.slice(12, 16)}-${value.slice(16, 20)}-${value.slice(20)}`;
}

export function requestID(prefix: string): string {
  return `${prefix}-${clientUUID()}`;
}

export function streamURL(): string {
  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${location.host}/api/v1/stream`;
}
