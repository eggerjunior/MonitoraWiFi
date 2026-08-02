"use client";

import { useState } from "react";

import type { LanScanCommandResult } from "@/lib/api-types";
import { statusLabel, useCommand } from "@/lib/useCommand";

// LAN scanner (Fase 5): o agente varre um bloco CIDR privado (RFC 1918, no
// máximo /22) por hosts respondendo em portas comuns — nunca simula quais
// hosts existem (Seção 2.1). O backend recusa qualquer CIDR público ou
// maior que o limite (threat-model.md §5).
export function LanScanTool({ siteId }: { siteId: string }) {
  const [cidr, setCidr] = useState("192.168.1.0/24");
  const { command, isSubmitting, error, run } = useCommand(siteId);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await run("lan_scan", { cidr });
  }

  const result = command?.result as LanScanCommandResult | null | undefined;

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">LAN scanner</h2>
      <p className="mt-1 text-xs text-egg-text-secondary">
        Bloco CIDR privado (RFC 1918), no máximo /22 (1024 endereços).
      </p>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="lan-scan-cidr" className="text-xs text-egg-text-secondary">
            CIDR
          </label>
          <input
            id="lan-scan-cidr"
            type="text"
            value={cidr}
            onChange={(e) => setCidr(e.target.value)}
            required
            className="w-64 rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <button
          type="submit"
          disabled={isSubmitting || !cidr}
          className="rounded-md bg-egg-accent px-4 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        >
          {isSubmitting ? "Varrendo…" : "Executar"}
        </button>
      </form>

      {error && <p className="mt-3 text-sm text-egg-critical">{error}</p>}

      {command && (
        <div className="mt-4 rounded-lg border border-egg-border bg-egg-surface p-4 text-sm">
          <div className="flex items-center justify-between">
            <span className="font-medium text-egg-text-primary">{cidr}</span>
            <span className="text-xs text-egg-text-secondary">{statusLabel(command.status)}</span>
          </div>

          {command.status === "completed" && result && (
            <div className="mt-3 text-xs text-egg-text-primary">
              {result.hosts.length === 0 ? (
                <p className="text-egg-text-secondary">Nenhum host respondeu.</p>
              ) : (
                <>
                  <p className="mb-1 text-egg-text-secondary">{result.hosts.length} host(s) encontrado(s):</p>
                  <ul className="grid grid-cols-2 gap-x-4 font-mono">
                    {result.hosts.map((h) => (
                      <li key={h}>{h}</li>
                    ))}
                  </ul>
                </>
              )}
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
