import { NextResponse } from "next/server";

import { apiFetch, ApiRequestError } from "@/lib/api-server";
import type { ApiError, Report } from "@/lib/api-types";

// Proxy BFF para POST /sites/{siteId}/reports (Fase 7) — mesmo padrão de
// api/sites/[siteId]/commands: o navegador nunca fala diretamente com o
// backend Go, o cookie de sessão é repassado por apiFetch (server-only).
export async function POST(
  request: Request,
  { params }: { params: Promise<{ siteId: string }> },
) {
  const { siteId } = await params;
  const body = await request.text();

  try {
    const report = await apiFetch<Report>(`/sites/${siteId}/reports`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: body || undefined,
    });
    return NextResponse.json(report, { status: 201 });
  } catch (err) {
    if (err instanceof ApiRequestError) {
      return NextResponse.json(err.body as ApiError, { status: err.status });
    }
    throw err;
  }
}
