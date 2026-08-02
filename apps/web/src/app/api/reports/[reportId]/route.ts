import { NextResponse } from "next/server";

import { apiFetch, ApiRequestError } from "@/lib/api-server";
import type { ApiError, Report } from "@/lib/api-types";

// Proxy BFF para GET /reports/{reportId} (Fase 7) — usado pelo painel de
// relatórios pra carregar o conteúdo completo de um relatório sob demanda
// (a listagem não inclui content, ver handlers_reports.go).
export async function GET(
  request: Request,
  { params }: { params: Promise<{ reportId: string }> },
) {
  const { reportId } = await params;

  try {
    const report = await apiFetch<Report>(`/reports/${reportId}`);
    return NextResponse.json(report, { status: 200 });
  } catch (err) {
    if (err instanceof ApiRequestError) {
      return NextResponse.json(err.body as ApiError, { status: err.status });
    }
    throw err;
  }
}
