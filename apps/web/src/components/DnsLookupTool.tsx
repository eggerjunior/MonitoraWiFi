"use client";

import { useState } from "react";

import type { DnsLookupCommandResult } from "@/lib/api-types";
import { statusLabel, useCommand } from "@/lib/useCommand";

// Ferramenta de "DNS lookup sob demanda" (Fase 5): resolve um hostname de
// verdade a partir do agente do site — nunca simula endereços.
export function DnsLookupTool({ siteId }: { siteId: string }) {
  const [hostname, setHostname] = useState("example.com");
  const { command, isSubmitting, error, run } = useCommand(siteId);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await run("dns_lookup", { hostname });
  }

  const result = command?.result as DnsLookupCommandResult | null | undefined;

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">DNS lookup sob demanda</h2>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="dns-hostname" className="text-xs text-egg-text-secondary">
            Hostname
          </label>
          <input
            id="dns-hostname"
            type="text"
            value={hostname}
            onChange={(e) => setHostname(e.target.value)}
            required
            className="rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <button
          type="submit"
          disabled={isSubmitting || !hostname}
          className="rounded-md bg-egg-accent px-4 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        >
          {isSubmitting ? "Enviando…" : "Executar"}
        </button>
      </form>

      {error && <p className="mt-3 text-sm text-egg-critical">{error}</p>}

      {command && (
        <div className="mt-4 rounded-lg border border-egg-border bg-egg-surface p-4 text-sm">
          <div className="flex items-center justify-between">
            <span className="font-medium text-egg-text-primary">
              {command.params.hostname as string}
            </span>
            <span className="text-xs text-egg-text-secondary">{statusLabel(command.status)}</span>
          </div>

          {command.status === "completed" && result && (
            <div className="mt-3 space-y-1">
              <div className="text-xs text-egg-text-secondary">
                Resolvido em {result.duration_ms.toFixed(1)} ms
              </div>
              <ul className="list-disc pl-5">
                {result.addresses.map((addr) => (
                  <li key={addr} className="font-medium text-egg-text-primary">
                    {addr}
                  </li>
                ))}
              </ul>
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
