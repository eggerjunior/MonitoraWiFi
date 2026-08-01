"use client";

import { useState } from "react";

import type { BatchPingCommandResult, PingProtocol } from "@/lib/api-types";
import { statusLabel, useCommand } from "@/lib/useCommand";

const PROTOCOLS: PingProtocol[] = ["icmp", "tcp", "http", "dns"];
const MAX_TARGETS = 20;

// Ping em lote (Fase 5): reaproveita a mesma fila de comando sob demanda do
// PingTool, só que testa vários alvos numa única execução do agente — cada
// alvo é uma sonda real e independente, nunca um valor inventado a partir de
// uma única medição (Seção 2.1, "nunca simular dado").
export function BatchPingTool({ siteId }: { siteId: string }) {
  const [targetsRaw, setTargetsRaw] = useState("1.1.1.1\n8.8.8.8");
  const [protocol, setProtocol] = useState<PingProtocol>("icmp");
  const { command, isSubmitting, error, run } = useCommand(siteId);

  const targets = targetsRaw
    .split("\n")
    .map((t) => t.trim())
    .filter(Boolean);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await run("batch_ping", { targets, protocol });
  }

  const result = command?.result as BatchPingCommandResult | null | undefined;

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">Ping em lote</h2>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-wrap items-start gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="batch-ping-targets" className="text-xs text-egg-text-secondary">
            Alvos (um por linha, até {MAX_TARGETS})
          </label>
          <textarea
            id="batch-ping-targets"
            value={targetsRaw}
            onChange={(e) => setTargetsRaw(e.target.value)}
            rows={4}
            className="w-64 rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor="batch-ping-protocol" className="text-xs text-egg-text-secondary">
            Protocolo
          </label>
          <select
            id="batch-ping-protocol"
            value={protocol}
            onChange={(e) => setProtocol(e.target.value as PingProtocol)}
            className="rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary uppercase"
          >
            {PROTOCOLS.map((p) => (
              <option key={p} value={p}>
                {p}
              </option>
            ))}
          </select>
        </div>
        <button
          type="submit"
          disabled={isSubmitting || targets.length === 0 || targets.length > MAX_TARGETS}
          className="rounded-md bg-egg-accent px-4 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        >
          {isSubmitting ? "Enviando…" : "Executar"}
        </button>
      </form>

      {targets.length > MAX_TARGETS && (
        <p className="mt-2 text-xs text-egg-critical">
          Máximo de {MAX_TARGETS} alvos por execução.
        </p>
      )}
      {error && <p className="mt-3 text-sm text-egg-critical">{error}</p>}

      {command && (
        <div className="mt-4 rounded-lg border border-egg-border bg-egg-surface p-4 text-sm">
          <div className="flex items-center justify-between">
            <span className="font-medium text-egg-text-primary">
              {targets.length} alvo(s) · {protocol.toUpperCase()}
            </span>
            <span className="text-xs text-egg-text-secondary">{statusLabel(command.status)}</span>
          </div>

          {command.status === "completed" && result && (
            <table className="mt-3 w-full text-left text-xs">
              <thead className="text-egg-text-secondary">
                <tr>
                  <th className="pb-1 pr-2">Alvo</th>
                  <th className="pb-1 pr-2">p50</th>
                  <th className="pb-1 pr-2">Perda</th>
                  <th className="pb-1">Jitter</th>
                </tr>
              </thead>
              <tbody>
                {result.results.map((r) => (
                  <tr key={r.target} className="text-egg-text-primary">
                    <td className="py-1 pr-2">{r.target}</td>
                    <td className="py-1 pr-2">{formatMs(r.latency_ms_p50)}</td>
                    <td className="py-1 pr-2">{formatPct(r.packet_loss_pct)}</td>
                    <td className="py-1">{formatMs(r.jitter_ms)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {command.status === "failed" && (
            <p className="mt-2 text-egg-critical">{command.error ?? "Falha não especificada."}</p>
          )}
        </div>
      )}
    </div>
  );
}

function formatMs(value: number | null): string {
  return value === null ? "Indisponível" : `${value.toFixed(1)} ms`;
}

function formatPct(value: number | null): string {
  return value === null ? "Indisponível" : `${value.toFixed(1)}%`;
}
