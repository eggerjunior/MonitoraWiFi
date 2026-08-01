"use client";

import { useState } from "react";

import type { RdapResult } from "@/lib/api-types";

// RDAP/WHOIS (Fase 5): consulta pública sobre domínio/IP, resolvida via
// bootstrap real da IANA — roda direto no backend, sem depender do agente do
// site (a informação é da internet, não da LAN do usuário).
export function RdapTool() {
  const [query, setQuery] = useState("example.com");
  const [result, setResult] = useState<RdapResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);
    setResult(null);
    try {
      const res = await fetch(`/api/rdap?query=${encodeURIComponent(query)}`);
      const body = await res.json();
      if (!res.ok) {
        setError(body.message ?? "Erro ao consultar RDAP.");
        return;
      }
      setResult(body as RdapResult);
    } catch {
      setError("Erro de rede ao consultar RDAP.");
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">RDAP / WHOIS</h2>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="rdap-query" className="text-xs text-egg-text-secondary">
            Domínio ou IP
          </label>
          <input
            id="rdap-query"
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            required
            className="w-64 rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <button
          type="submit"
          disabled={isSubmitting || !query}
          className="rounded-md bg-egg-accent px-4 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        >
          {isSubmitting ? "Consultando…" : "Consultar"}
        </button>
      </form>

      {error && <p className="mt-3 text-sm text-egg-critical">{error}</p>}

      {result && (
        <div className="mt-4 space-y-1 rounded-lg border border-egg-border bg-egg-surface p-4 text-sm text-egg-text-primary">
          <div className="font-medium">{result.name || result.query}</div>
          <div className="text-xs text-egg-text-secondary">Servidor: {result.server}</div>
          {result.handle && <div>Handle: {result.handle}</div>}
          {result.status.length > 0 && <div>Status: {result.status.join(", ")}</div>}
          {result.events.length > 0 && (
            <ul className="text-xs">
              {result.events.map((e) => (
                <li key={e.action}>
                  {e.action}: {new Date(e.date).toLocaleDateString()}
                </li>
              ))}
            </ul>
          )}
          {result.nameservers && result.nameservers.length > 0 && (
            <div className="text-xs">Nameservers: {result.nameservers.join(", ")}</div>
          )}
        </div>
      )}
    </div>
  );
}
