import "server-only";

import { apiFetch, ApiRequestError } from "./api-server";
import type { User } from "./api-types";

/** Retorna o usuário autenticado ou `null` (sem sessão válida) — nunca lança
 * para o caso "não autenticado", que é uma condição esperada, não um erro. */
export async function getCurrentUser(): Promise<User | null> {
  try {
    return await apiFetch<User>("/auth/me");
  } catch (err) {
    if (err instanceof ApiRequestError && err.status === 401) {
      return null;
    }
    throw err;
  }
}
