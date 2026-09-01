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

export function requestID(prefix: string): string {
  if (typeof crypto.randomUUID === "function") return `${prefix}-${crypto.randomUUID()}`;
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  const value = [...bytes].map((byte) => byte.toString(16).padStart(2, "0")).join("");
  return `${prefix}-${value}`;
}

export function streamURL(): string {
  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${location.host}/api/v1/stream`;
}
