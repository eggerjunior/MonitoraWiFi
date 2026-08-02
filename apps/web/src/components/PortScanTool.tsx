"use client";

import { useState } from "react";

import type { PortScanCommandResult } from "@/lib/api-types";
import { statusLabel, useCommand } from "@/lib/useCommand";

// Port scanner (Fase 5): o agente tenta conectar em cada porta do
// intervalo e só reporta como aberta a que aceitou conexão de verdade
// (Seção 2.1). Mitigação completa do threat model (§5): o backend só
// aceita um IPv4 privado literal como alvo (nunca hostname, nunca IP
// público) e no máximo 1024 portas por execução.
export function PortScanTool({ siteId }: { siteId: string }) {
  const [target, setTarget] = useState("192.168.1.1");
  const [startPort, setStartPort] = useState(1);
  const [endPort, setEndPort] = useState(1024);
  const { command, isSubmitting, error, run } = useCommand(siteId);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await run("port_scan", { target, start_port: startPort, end_port: endPort });
  }

  const result = command?.result as PortScanCommandResult | null | undefined;

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">Port scanner</h2>
      <p className="mt-1 text-xs text-egg-text-secondary">
        Alvo precisa ser um IP privado (RFC 1918), no máximo 1024 portas por execução.
      </p>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="port-scan-target" className="text-xs text-egg-text-secondary">
            Alvo (IP)
          </label>
          <input
            id="port-scan-target"
            type="text"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            required
            className="w-40 rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor="port-scan-start" className="text-xs text-egg-text-secondary">
            Porta inicial
          </label>
          <input
            id="port-scan-start"
            type="number"
            min={1}
            max={65535}
            value={startPort}
            onChange={(e) => setStartPort(Number(e.target.value))}
            className="w-24 rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor="port-scan-end" className="text-xs text-egg-text-secondary">
            Porta final
          </label>
          <input
            id="port-scan-end"
            type="number"
            min={1}
            max={65535}
            value={endPort}
            onChange={(e) => setEndPort(Number(e.target.value))}
            className="w-24 rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <button
          type="submit"
          disabled={isSubmitting || !target}
          className="rounded-md bg-egg-accent px-4 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        >
          {isSubmitting ? "Varrendo…" : "Executar"}
        </button>
      </form>

      {error && <p className="mt-3 text-sm text-egg-critical">{error}</p>}

      {command && (
        <div className="mt-4 rounded-lg border border-egg-border bg-egg-surface p-4 text-sm">
          <div className="flex items-center justify-between">
            <span className="font-medium text-egg-text-primary">
              {target}:{startPort}-{endPort}
            </span>
            <span className="text-xs text-egg-text-secondary">{statusLabel(command.status)}</span>
          </div>

          {command.status === "completed" && result && (
            <div className="mt-3 text-xs text-egg-text-primary">
              {result.open_ports.length === 0 ? (
                <p className="text-egg-text-secondary">Nenhuma porta aberta encontrada.</p>
              ) : (
                <p className="font-mono">{result.open_ports.join(", ")}</p>
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
