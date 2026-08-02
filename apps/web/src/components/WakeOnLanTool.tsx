"use client";

import { useState } from "react";

import { statusLabel, useCommand } from "@/lib/useCommand";

// Wake-on-LAN (Fase 5, ADR-008): o agente envia o magic packet real via
// UDP — nunca simula o envio (Seção 2.1). Executado exclusivamente pelo
// agente (não pelo app iOS, que tem restrições de plataforma documentadas
// no ADR-008 pra broadcast/multicast).
export function WakeOnLanTool({ siteId }: { siteId: string }) {
  const [macAddress, setMacAddress] = useState("");
  const [broadcastIp, setBroadcastIp] = useState("255.255.255.255");
  const { command, isSubmitting, error, run } = useCommand(siteId);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await run("wake_on_lan", { mac_address: macAddress, broadcast_ip: broadcastIp });
  }

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">Wake-on-LAN</h2>
      <form onSubmit={handleSubmit} className="mt-3 flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="wol-mac" className="text-xs text-egg-text-secondary">
            Endereço MAC
          </label>
          <input
            id="wol-mac"
            type="text"
            value={macAddress}
            onChange={(e) => setMacAddress(e.target.value)}
            placeholder="aa:bb:cc:dd:ee:ff"
            required
            className="w-48 rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor="wol-broadcast" className="text-xs text-egg-text-secondary">
            Broadcast IP
          </label>
          <input
            id="wol-broadcast"
            type="text"
            value={broadcastIp}
            onChange={(e) => setBroadcastIp(e.target.value)}
            className="w-40 rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          />
        </div>
        <button
          type="submit"
          disabled={isSubmitting || !macAddress}
          className="rounded-md bg-egg-accent px-4 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        >
          {isSubmitting ? "Enviando…" : "Ligar"}
        </button>
      </form>

      {error && <p className="mt-3 text-sm text-egg-critical">{error}</p>}

      {command && (
        <div className="mt-4 rounded-lg border border-egg-border bg-egg-surface p-4 text-sm">
          <div className="flex items-center justify-between">
            <span className="font-medium text-egg-text-primary">{macAddress}</span>
            <span className="text-xs text-egg-text-secondary">{statusLabel(command.status)}</span>
          </div>
          {command.status === "completed" && (
            <p className="mt-2 text-xs text-egg-success">Magic packet enviado.</p>
          )}
          {command.status === "failed" && (
            <p className="mt-2 text-egg-critical">{command.error ?? "Falha não especificada."}</p>
          )}
        </div>
      )}
    </div>
  );
}
