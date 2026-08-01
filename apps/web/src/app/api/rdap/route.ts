import { NextResponse } from "next/server";

import { apiFetch, ApiRequestError } from "@/lib/api-server";
import type { ApiError, RdapResult } from "@/lib/api-types";

// Proxy do BFF pra consulta RDAP/WHOIS (Fase 5) — roda direto no backend Go
// (sem envolver o agente do site, ver docs/architecture/06-roadmap.md).
export async function GET(request: Request) {
  const query = new URL(request.url).searchParams.get("query") ?? "";

  try {
    const result = await apiFetch<RdapResult>(`/rdap/lookup?query=${encodeURIComponent(query)}`);
    return NextResponse.json(result, { status: 200 });
  } catch (err) {
    if (err instanceof ApiRequestError) {
      return NextResponse.json(err.body as ApiError, { status: err.status });
    }
    throw err;
  }
}
