"use client";

import { useState } from "react";

import type { TracerouteCommandResult } from "@/lib/api-types";
import { statusLabel, useCommand } from "@/lib/useCommand";

// Ferramenta de "traceroute sob demanda" (Fase 5): roda um traceroute ICMP
// real a partir do agente do site — cada salto reportado é uma resposta
// real (ou "sem resposta"), nunca inventado.
export function TracerouteTool({ siteId }: { siteId: string }) {
  const [target, setTarget] = useState("1.1.1.1");
  const { command, isSubmitting, error, run } = useCommand(siteId);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await run("traceroute", { target });
  }

  const result = command?.result as TracerouteCommandResult | null | undefined;

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">Traceroute sob demanda</h2>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="traceroute-target" className="text-xs text-egg-text-secondary">
            Alvo
          </label>
          <input
            id="traceroute-target"
            type="text"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            required
            className="rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
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
              {command.params.target as string}
            </span>
            <span className="text-xs text-egg-text-secondary">{statusLabel(command.status)}</span>
          </div>

          {command.status === "completed" && result && (
            <div className="mt-3 space-y-1">
              <div className="text-xs text-egg-text-secondary">
                {result.reached ? "Destino alcançado" : "Destino não alcançado dentro do limite de saltos"}
              </div>
              <ol className="space-y-1">
                {result.hops.map((hop) => (
                  <li key={hop.hop} className="flex justify-between font-mono text-xs">
                    <span>{hop.hop}.</span>
                    <span>{hop.address || "* sem resposta *"}</span>
                    <span>{hop.rtt_ms !== null ? `${hop.rtt_ms.toFixed(1)} ms` : "—"}</span>
                  </li>
                ))}
              </ol>
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
