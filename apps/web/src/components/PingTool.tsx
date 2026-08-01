"use client";

import { useRef, useState } from "react";

import type { Command, PingProtocol } from "@/lib/api-types";

const PROTOCOLS: PingProtocol[] = ["icmp", "tcp", "http", "dns"];

// Ferramenta de "ping sob demanda" (Fase 5, início): dispara um comando real
// no agente do site e faz polling do status até completed/failed — nunca
// mostra um resultado antes de o agente realmente ter executado (Seção 2.1,
// "nunca simular dado").
export function PingTool({ siteId }: { siteId: string }) {
  const [target, setTarget] = useState("1.1.1.1");
  const [protocol, setProtocol] = useState<PingProtocol>("icmp");
  const [command, setCommand] = useState<Command | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const pollTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  function stopPolling() {
    if (pollTimer.current) {
      clearTimeout(pollTimer.current);
      pollTimer.current = null;
    }
  }

  function pollStatus(commandId: string) {
    pollTimer.current = setTimeout(async () => {
      const res = await fetch(`/api/commands/${commandId}`);
      if (!res.ok) {
        setError("Não foi possível consultar o status do comando.");
        return;
      }
      const updated: Command = await res.json();
      setCommand(updated);
      if (updated.status === "pending" || updated.status === "claimed") {
        pollStatus(commandId);
      }
    }, 1000);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    stopPolling();
    setError(null);
    setCommand(null);
    setIsSubmitting(true);
    try {
      const res = await fetch(`/api/sites/${siteId}/commands`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ type: "ping", params: { target, protocol } }),
      });
      const body = await res.json();
      if (!res.ok) {
        setError(body.message ?? "Erro ao criar o comando.");
        return;
      }
      setCommand(body as Command);
      pollStatus(body.id);
    } catch {
      setError("Erro de rede ao criar o comando.");
    } finally {
      setIsSubmitting(false);
    }
  }

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

      {error && (
        <p className="mt-3 text-sm text-egg-critical">{error}</p>
      )}

      {command && (
        <div className="mt-4 rounded-lg border border-egg-border bg-egg-surface p-4 text-sm">
          <div className="flex items-center justify-between">
            <span className="font-medium text-egg-text-primary">
              {command.params.target as string} ({(command.params.protocol as string)?.toUpperCase()})
            </span>
            <StatusBadge status={command.status} />
          </div>

          {command.status === "completed" && command.result && (
            <div className="mt-3 grid grid-cols-3 gap-3">
              <Metric label="p50" value={formatMs(command.result.latency_ms_p50)} />
              <Metric label="perda" value={formatPct(command.result.packet_loss_pct)} />
              <Metric label="jitter" value={formatMs(command.result.jitter_ms)} />
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

function StatusBadge({ status }: { status: Command["status"] }) {
  const labels: Record<Command["status"], string> = {
    pending: "Aguardando agente…",
    claimed: "Executando…",
    completed: "Concluído",
    failed: "Falhou",
  };
  return <span className="text-xs text-egg-text-secondary">{labels[status]}</span>;
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
