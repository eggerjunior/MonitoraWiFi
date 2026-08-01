import { apiFetch } from "@/lib/api-server";
import { getCurrentSite } from "@/lib/current-site";
import type { UniFiClientList, UniFiDeviceList } from "@/lib/api-types";

// Dashboard Wi-Fi (Fase 4): inventário de APs + contagem de clientes sem fio
// por AP. Métricas de rádio (canal, RSSI, utilização) ainda não estão
// confirmadas contra a instalação real (capability-matrix.md "a
// validar") — não exibidas até serem validadas, nunca inventadas.
export default async function WiFiPage() {
  const current = await getCurrentSite();
  if ("error" in current) {
    return <EmptyState message={current.error} />;
  }
  const { site } = current;

  const [devices, clients] = await Promise.all([
    apiFetch<UniFiDeviceList>(`/sites/${site.id}/unifi/devices`),
    apiFetch<UniFiClientList>(`/sites/${site.id}/unifi/clients`),
  ]);

  const accessPoints = devices.items.filter((d) => d.features.includes("accessPoint"));

  const wirelessCountByDevice = new Map<string, number>();
  for (const c of clients.items) {
    if (c.type !== "WIRELESS") continue;
    wirelessCountByDevice.set(
      c.uplink_device_id,
      (wirelessCountByDevice.get(c.uplink_device_id) ?? 0) + 1,
    );
  }

  return (
    <div className="max-w-4xl space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Wi-Fi</h1>
        <p className="mt-1 text-sm text-egg-text-secondary">
          Site: {site.name} · {accessPoints.length}{" "}
          {accessPoints.length === 1 ? "access point" : "access points"}
        </p>
      </div>

      {accessPoints.length === 0 ? (
        <EmptyCard message="Nenhum access point sincronizado ainda. Requer um agente local com a integração UniFi configurada." />
      ) : (
        <ul className="space-y-2">
          {accessPoints.map((ap) => (
            <li
              key={ap.id}
              className="rounded-lg border border-egg-border bg-egg-surface p-4"
            >
              <div className="flex items-center justify-between">
                <span className="font-medium text-egg-text-primary">{ap.name}</span>
                <span
                  className={`text-xs font-medium ${ap.state === "ONLINE" ? "text-egg-success" : "text-egg-text-secondary"}`}
                >
                  {ap.state}
                </span>
              </div>
              <div className="mt-2 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
                <Field label="Modelo" value={ap.model} />
                <Field label="Firmware" value={ap.firmware_version || "Indisponível"} />
                <Field
                  label="Clientes sem fio"
                  value={String(wirelessCountByDevice.get(ap.external_id) ?? 0)}
                />
                <Field label="IP" value={ap.ip_address || "Indisponível"} />
              </div>
            </li>
          ))}
        </ul>
      )}

      <p className="text-xs text-egg-text-disabled">
        Canal, largura de canal, potência de transmissão e utilização por
        rádio ainda não confirmados contra a instalação real — não exibidos
        até serem validados (ver docs/unifi/capability-matrix.md item 6).
      </p>
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

function EmptyState({ message }: { message: string }) {
  return (
    <div className="max-w-4xl">
      <h1 className="text-xl font-semibold text-egg-text-primary">Wi-Fi</h1>
      <div className="mt-4 rounded-lg border border-dashed border-egg-border p-6 text-sm text-egg-text-secondary">
        {message}
      </div>
    </div>
  );
}

function EmptyCard({ message }: { message: string }) {
  return (
    <div className="rounded-lg border border-dashed border-egg-border p-4 text-sm text-egg-text-secondary">
      {message}
    </div>
  );
}
