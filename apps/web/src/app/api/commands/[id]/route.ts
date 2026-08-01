import { NextResponse } from "next/server";

import { apiFetch, ApiRequestError } from "@/lib/api-server";
import type { ApiError, Command } from "@/lib/api-types";

export async function GET(
  _request: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  const { id } = await params;

  try {
    const command = await apiFetch<Command>(`/commands/${id}`);
    return NextResponse.json(command, { status: 200 });
  } catch (err) {
    if (err instanceof ApiRequestError) {
      return NextResponse.json(err.body as ApiError, { status: err.status });
    }
    throw err;
  }
}
