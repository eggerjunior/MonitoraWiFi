import "server-only";
import { cookies } from "next/headers";

import { API_BASE_URL, SESSION_COOKIE_NAME } from "./session";
import type { ApiError } from "./api-types";

export class ApiRequestError extends Error {
  constructor(
    public status: number,
    public body: ApiError | null,
  ) {
    super(body?.message ?? `erro na API (status ${status})`);
  }
}

/**
 * Faz uma requisição server-side ao backend Go, repassando o cookie de sessão
 * do usuário atual (lido via `cookies()`). Usado por Server Components e Route
 * Handlers desta aplicação — nunca pelo navegador diretamente, que fala apenas
 * com o próprio domínio Next.js (padrão "backend for frontend").
 */
export async function apiFetch<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const cookieStore = await cookies();
  const session = cookieStore.get(SESSION_COOKIE_NAME);

  const headers = new Headers(init?.headers);
  headers.set("Accept", "application/json");
  if (session) {
    headers.set("Cookie", `${SESSION_COOKIE_NAME}=${session.value}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers,
    cache: "no-store",
  });

  if (response.status === 204) {
    return undefined as T;
  }

  const contentType = response.headers.get("content-type") ?? "";
  const payload = contentType.includes("application/json")
    ? await response.json()
    : null;

  if (!response.ok) {
    throw new ApiRequestError(response.status, payload as ApiError | null);
  }

  return payload as T;
}
