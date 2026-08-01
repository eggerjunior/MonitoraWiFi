"use client";

import { useState } from "react";

import type { PingCommandResult, PingProtocol } from "@/lib/api-types";
import { statusLabel, useCommand } from "@/lib/useCommand";

const PROTOCOLS: PingProtocol[] = ["icmp", "tcp", "http", "dns"];

// Ferramenta de "ping sob demanda" (Fase 5, início): dispara um comando real
// no agente do site e faz polling do status até completed/failed — nunca
// mostra um resultado antes de o agente realmente ter executado (Seção 2.1,
// "nunca simular dado").
export function PingTool({ siteId }: { siteId: string }) {
  const [target, setTarget] = useState("1.1.1.1");
  const [protocol, setProtocol] = useState<PingProtocol>("icmp");
  const { command, isSubmitting, error, run } = useCommand(siteId);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await run("ping", { target, protocol });
  }

  const result = command?.result as PingCommandResult | null | undefined;

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">Ping sob demanda</h2>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="ping-target" className="text-xs text-egg-text-secondary">
            Alvo
          </label>
          <input
            id="ping-target"
            type="text"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            required
            className="rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor="ping-protocol" className="text-xs text-egg-text-secondary">
            Protocolo
          </label>
          <select
            id="ping-protocol"
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
          disabled={isSubmitting || !target}
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
              {command.params.target as string} ({(command.params.protocol as string)?.toUpperCase()})
            </span>
            <span className="text-xs text-egg-text-secondary">{statusLabel(command.status)}</span>
          </div>

          {command.status === "completed" && result && (
            <div className="mt-3 grid grid-cols-3 gap-3">
              <Metric label="p50" value={formatMs(result.latency_ms_p50)} />
              <Metric label="perda" value={formatPct(result.packet_loss_pct)} />
              <Metric label="jitter" value={formatMs(result.jitter_ms)} />
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

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs text-egg-text-secondary">{label}</div>
      <div className="text-sm font-medium text-egg-text-primary">{value}</div>
    </div>
  );
}

function formatMs(value: number | null): string {
  return value === null ? "Indisponível" : `${value.toFixed(1)} ms`;
}

function formatPct(value: number | null): string {
  return value === null ? "Indisponível" : `${value.toFixed(1)}%`;
}
