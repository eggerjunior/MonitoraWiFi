"use client";

import { useState } from "react";

import type { HttpRequestCommandResult } from "@/lib/api-types";
import { statusLabel, useCommand } from "@/lib/useCommand";

const METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"];

// HTTP client sob demanda (Fase 5): faz uma requisição HTTP real a partir do
// agente do site e reporta status/headers/corpo reais da resposta — nunca
// simula (Seção 2.1). Útil pra depurar serviços internos da LAN.
export function HttpRequestTool({ siteId }: { siteId: string }) {
  const [url, setUrl] = useState("http://localhost");
  const [method, setMethod] = useState("GET");
  const [body, setBody] = useState("");
  const { command, isSubmitting, error, run } = useCommand(siteId);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await run("http_request", { url, method, body: body || undefined });
  }

  const result = command?.result as HttpRequestCommandResult | null | undefined;

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">HTTP client sob demanda</h2>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="http-request-method" className="text-xs text-egg-text-secondary">
            Método
          </label>
          <select
            id="http-request-method"
            value={method}
            onChange={(e) => setMethod(e.target.value)}
            className="rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          >
            {METHODS.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor="http-request-url" className="text-xs text-egg-text-secondary">
            URL
          </label>
          <input
            id="http-request-url"
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            required
            className="w-64 rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <button
          type="submit"
          disabled={isSubmitting || !url}
          className="rounded-md bg-egg-accent px-4 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        >
          {isSubmitting ? "Enviando…" : "Executar"}
        </button>
      </form>
      {(method === "POST" || method === "PUT" || method === "PATCH") && (
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="Corpo da requisição (opcional)"
          rows={3}
          className="mt-2 w-full rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
        />
      )}

      {error && <p className="mt-3 text-sm text-egg-critical">{error}</p>}

      {command && (
        <div className="mt-4 rounded-lg border border-egg-border bg-egg-surface p-4 text-sm">
          <div className="flex items-center justify-between">
            <span className="font-medium text-egg-text-primary">
              {method} {url}
            </span>
            <span className="text-xs text-egg-text-secondary">{statusLabel(command.status)}</span>
          </div>

          {command.status === "completed" && result && (
            <div className="mt-3 space-y-1 text-xs text-egg-text-primary">
              <div>
                {result.status_code} {result.status_text} · {result.duration_ms.toFixed(1)} ms
              </div>
              <div className="text-egg-text-secondary">
                {Object.entries(result.headers).map(([k, v]) => (
                  <div key={k}>
                    {k}: {v}
                  </div>
                ))}
              </div>
              <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap rounded bg-black/5 p-2 font-mono text-xs dark:bg-white/5">
                {result.body_snippet}
                {result.body_truncated ? "\n… (truncado)" : ""}
              </pre>
            </div>
          )}

          {command.status === "failed" && (
            <p className="mt-2 text-egg-critical">{command.error ?? "Falha não especificada."}</p>
          )}
        </div>
      )}
    </div>
  );
}
