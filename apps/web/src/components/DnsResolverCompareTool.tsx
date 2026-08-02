"use client";

import { useState } from "react";

import type { DnsResolverCompareCommandResult } from "@/lib/api-types";
import { statusLabel, useCommand } from "@/lib/useCommand";

// Comparação entre resolvedores DNS (Fase 2, item pendente há tempo):
// resolve o mesmo hostname contra o resolvedor padrão da rede e três
// resolvedores públicos conhecidos (Cloudflare/Google/Quad9) — lista fixa no
// agente, nunca configurável por aqui (ver comentário em
// apps/local-agent/internal/probes/probes.go, KnownResolvers). Ajuda a
// perceber DNS de operadora divergindo/filtrando.
export function DnsResolverCompareTool({ siteId }: { siteId: string }) {
  const [hostname, setHostname] = useState("example.com");
  const { command, isSubmitting, error, run } = useCommand(siteId);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await run("dns_resolver_compare", { hostname });
  }

  const result = command?.result as DnsResolverCompareCommandResult | null | undefined;

  const successfulAddressSets = (result?.resolvers ?? [])
    .filter((r) => !r.error)
    .map((r) => [...r.addresses].sort().join(","));
  const divergent = new Set(successfulAddressSets).size > 1;

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">Comparação entre resolvedores DNS</h2>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="dns-compare-hostname" className="text-xs text-egg-text-secondary">
            Hostname
          </label>
          <input
            id="dns-compare-hostname"
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
            <div className="mt-3 space-y-3">
              {divergent && (
                <p className="rounded-md border border-egg-warning px-3 py-2 text-xs font-medium text-egg-warning">
                  Os resolvedores devolveram endereços diferentes para este
                  hostname — pode indicar DNS da operadora
                  redirecionando/filtrando, ou apenas balanceamento de carga
                  legítimo (CDN).
                </p>
              )}
              <ul className="space-y-2">
                {result.resolvers.map((r) => (
                  <li key={r.resolver} className="rounded-md border border-egg-border p-2">
                    <div className="flex items-center justify-between">
                      <span className="font-medium text-egg-text-primary">{r.resolver}</span>
                      <span className="text-xs text-egg-text-secondary">
                        {r.error ? "falhou" : `${r.duration_ms.toFixed(1)} ms`}
                      </span>
                    </div>
                    {r.error ? (
                      <p className="mt-1 text-egg-critical">{r.error}</p>
                    ) : (
                      <ul className="mt-1 list-disc pl-5">
                        {r.addresses.map((addr) => (
                          <li key={addr} className="text-egg-text-primary">
                            {addr}
                          </li>
                        ))}
                      </ul>
                    )}
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
