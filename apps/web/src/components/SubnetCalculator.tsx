"use client";

import { useMemo, useState } from "react";

// Calculadora de sub-rede IPv4 (Fase 5): cálculo matemático puro, sem
// chamada de rede/agente — por isso não tem noção de "proveniência de
// dado" como as outras ferramentas (não há medição envolvida).
export function SubnetCalculator() {
  const [input, setInput] = useState("192.168.1.0/24");

  const parsed = useMemo(() => parseCIDR(input), [input]);

  return (
    <div className="max-w-xl">
      <h2 className="text-sm font-semibold text-egg-text-primary">Calculadora de sub-rede</h2>
      <div className="mt-3">
        <label htmlFor="subnet-input" className="text-xs text-egg-text-secondary">
          Endereço/prefixo (CIDR)
        </label>
        <input
          id="subnet-input"
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          className="mt-1 block w-full rounded-md border border-egg-border bg-egg-surface px-3 py-1.5 text-sm text-egg-text-primary"
          placeholder="192.168.1.0/24"
        />
      </div>

      {"error" in parsed ? (
        <p className="mt-3 text-sm text-egg-critical">{parsed.error}</p>
      ) : (
        <div className="mt-4 grid grid-cols-2 gap-3 rounded-lg border border-egg-border bg-egg-surface p-4 text-sm sm:grid-cols-3">
          <Field label="Máscara" value={parsed.mask} />
          <Field label="Endereço de rede" value={parsed.network} />
          <Field label="Broadcast" value={parsed.broadcast} />
          <Field label="Primeiro host" value={parsed.firstHost} />
          <Field label="Último host" value={parsed.lastHost} />
          <Field label="Hosts utilizáveis" value={String(parsed.usableHosts)} />
        </div>
      )}
    </div>
  );
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs text-egg-text-secondary">{label}</div>
      <div className="font-medium text-egg-text-primary">{value}</div>
    </div>
  );
}

type SubnetInfo = {
  mask: string;
  network: string;
  broadcast: string;
  firstHost: string;
  lastHost: string;
  usableHosts: number;
};

function parseCIDR(input: string): SubnetInfo | { error: string } {
  const match = input.trim().match(/^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})\/(\d{1,2})$/);
  if (!match) {
    return { error: "Formato esperado: A.B.C.D/prefixo (ex.: 192.168.1.0/24)" };
  }
  const octets = [match[1], match[2], match[3], match[4]].map(Number);
  const prefix = Number(match[5]);

  if (octets.some((o) => o < 0 || o > 255) || prefix < 0 || prefix > 32) {
    return { error: "Endereço ou prefixo fora do intervalo válido" };
  }

  const ipInt = (octets[0] << 24) | (octets[1] << 16) | (octets[2] << 8) | octets[3];
  const maskInt = prefix === 0 ? 0 : (0xffffffff << (32 - prefix)) >>> 0;
  const networkInt = (ipInt & maskInt) >>> 0;
  const broadcastInt = (networkInt | (~maskInt >>> 0)) >>> 0;

  const totalHosts = 2 ** (32 - prefix);
  const usableHosts = prefix >= 31 ? 0 : totalHosts - 2;

  return {
    mask: intToIP(maskInt),
    network: intToIP(networkInt),
    broadcast: intToIP(broadcastInt),
    firstHost: usableHosts > 0 ? intToIP(networkInt + 1) : intToIP(networkInt),
    lastHost: usableHosts > 0 ? intToIP(broadcastInt - 1) : intToIP(broadcastInt),
    usableHosts,
  };
}

function intToIP(value: number): string {
  return [(value >>> 24) & 255, (value >>> 16) & 255, (value >>> 8) & 255, value & 255].join(".");
}
