import { NextResponse } from "next/server";
import { cookies } from "next/headers";

import { API_BASE_URL, SESSION_COOKIE_NAME } from "@/lib/session";

export async function POST() {
  const cookieStore = await cookies();
  const session = cookieStore.get(SESSION_COOKIE_NAME);

  if (session) {
    await fetch(`${API_BASE_URL}/auth/logout`, {
      method: "POST",
      headers: { Cookie: `${SESSION_COOKIE_NAME}=${session.value}` },
      cache: "no-store",
    }).catch(() => {
      // Encerrar a sessão localmente mesmo se o backend estiver inalcançável
      // é preferível a deixar o usuário preso numa sessão "zumbi" no navegador.
    });
  }

  const response = NextResponse.json({ ok: true });
  response.cookies.delete(SESSION_COOKIE_NAME);
  return response;
}
