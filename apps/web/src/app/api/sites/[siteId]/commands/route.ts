import { NextResponse } from "next/server";

import { apiFetch, ApiRequestError } from "@/lib/api-server";
import type { ApiError, Command } from "@/lib/api-types";

// Proxy BFF para POST /sites/{siteId}/commands — o navegador nunca fala
// diretamente com o backend Go (mesmo padrão de auth/login). O cookie de
// sessão é repassado por apiFetch (server-only), lido do contexto de
// requisição do Next.js.
export async function POST(
  request: Request,
  { params }: { params: Promise<{ siteId: string }> },
) {
  const { siteId } = await params;
  const body = await request.json();

  try {
    const command = await apiFetch<Command>(`/sites/${siteId}/commands`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    return NextResponse.json(command, { status: 202 });
  } catch (err) {
    if (err instanceof ApiRequestError) {
      return NextResponse.json(err.body as ApiError, { status: err.status });
    }
    throw err;
  }
}
