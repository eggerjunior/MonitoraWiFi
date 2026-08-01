"use client";

import { useState } from "react";

import type { SslCheckCommandResult } from "@/lib/api-types";
import { statusLabel, useCommand } from "@/lib/useCommand";

// Verificador de certificado SSL/TLS sob demanda (Fase 5): faz um handshake
// TLS real contra host:porta a partir do agente do site e reporta o
// certificado apresentado de verdade — nunca simula validade/expiração
// (Seção 2.1).
export function SslCheckTool({ siteId }: { siteId: string }) {
  const [target, setTarget] = useState("example.com");
  const [port, setPort] = useState(443);
  const { command, isSubmitting, error, run } = useCommand(siteId);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await run("ssl_check", { target, port });
  }

  const result = command?.result as SslCheckCommandResult | null | undefined;

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">SSL/TLS checker</h2>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="ssl-check-target" className="text-xs text-egg-text-secondary">
            Host
          </label>
          <input
            id="ssl-check-target"
            type="text"
            value={target}
            onChange={(e) => setTarget(e.target.value)}
            required
            className="rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor="ssl-check-port" className="text-xs text-egg-text-secondary">
            Porta
          </label>
          <input
            id="ssl-check-port"
            type="number"
            min={1}
            max={65535}
            value={port}
            onChange={(e) => setPort(Number(e.target.value))}
            className="w-24 rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
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
              {command.params.target as string}:{command.params.port as number}
            </span>
            <span className="text-xs text-egg-text-secondary">{statusLabel(command.status)}</span>
          </div>

          {command.status === "completed" && result && (
            <div className="mt-3 space-y-1 text-xs text-egg-text-primary">
              <div className={result.valid_now ? "text-egg-success" : "text-egg-critical"}>
                {result.valid_now ? "Cadeia de certificado válida" : `Cadeia inválida: ${result.verify_error}`}
              </div>
              <div>Emissor: {result.issuer}</div>
              <div>Assunto: {result.subject}</div>
              <div>
                Validade: {new Date(result.not_before).toLocaleDateString()} –{" "}
                {new Date(result.not_after).toLocaleDateString()} ({result.days_until_expiry} dia(s) até expirar)
              </div>
              {result.dns_names.length > 0 && <div>DNS names: {result.dns_names.join(", ")}</div>}
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
