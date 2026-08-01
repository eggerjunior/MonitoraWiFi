import { NextResponse } from "next/server";

import { API_BASE_URL, SESSION_COOKIE_NAME } from "@/lib/session";
import type { ApiError, AuthSession } from "@/lib/api-types";

// Rota BFF (Seção "Backend for Frontend" do Next.js): o navegador nunca fala
// diretamente com o backend Go. Esta rota repassa e-mail/senha para
// /auth/login no backend e re-emite o cookie de sessão sob o domínio do
// próprio Next.js — evita qualquer complexidade de cookie cross-site em
// desenvolvimento (backend e web em portas diferentes) e casa com o desenho
// de produção (nginx atrás do mesmo domínio).
export async function POST(request: Request) {
  const body = await request.json();

  const upstream = await fetch(`${API_BASE_URL}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });

  const payload = await upstream.json().catch(() => null);

  if (!upstream.ok) {
    return NextResponse.json(payload as ApiError, { status: upstream.status });
  }

  const setCookie = upstream.headers.get("set-cookie");
  const tokenMatch = setCookie?.match(new RegExp(`${SESSION_COOKIE_NAME}=([^;]+)`));

  const response = NextResponse.json(payload as AuthSession, { status: 200 });

  if (tokenMatch) {
    const session = payload as AuthSession;
    response.cookies.set({
      name: SESSION_COOKIE_NAME,
      value: tokenMatch[1],
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      expires: new Date(session.expires_at),
    });
  }

  return response;
}
