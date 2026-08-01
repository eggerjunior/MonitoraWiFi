"use client";

import { useRef, useState } from "react";

import type { Command } from "@/lib/api-types";

// Hook compartilhado pelas ferramentas de diagnóstico sob demanda (Fase 5):
// dispara POST /api/sites/{id}/commands via BFF e faz polling do status até
// completed/failed — nunca mostra um resultado antes de o agente realmente
// ter executado (Seção 2.1, "nunca simular dado").
export function useCommand(siteId: string) {
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

  async function run(type: string, params: Record<string, unknown>) {
    stopPolling();
    setError(null);
    setCommand(null);
    setIsSubmitting(true);
    try {
      const res = await fetch(`/api/sites/${siteId}/commands`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ type, params }),
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

  return { command, isSubmitting, error, run };
}

export function statusLabel(status: Command["status"]): string {
  const labels: Record<Command["status"], string> = {
    pending: "Aguardando agente…",
    claimed: "Executando…",
    completed: "Concluído",
    failed: "Falhou",
  };
  return labels[status];
}
