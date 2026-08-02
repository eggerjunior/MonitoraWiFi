import { apiFetch } from "@/lib/api-server";
import { getCurrentSite } from "@/lib/current-site";
import type { UniFiClientList, UniFiDeviceList } from "@/lib/api-types";
import { provenance } from "@egger/tokens";

// Dashboard Switches (Fase 4): antes só apareciam misturados na tela de
// Dispositivos — mesmo padrão de filtragem por `features` já usado em
// wifi/page.tsx (accessPoint), aqui com "switching". Estatística por porta
// (PoE, VLAN, velocidade negociada) ainda não sincronizada nem armazenada
// — não exibida até existir (docs/architecture/05-modelo-dados.md).
export default async function SwitchesPage() {
  const current = await getCurrentSite();
  if ("error" in current) {
    return <EmptyState message={current.error} />;
  }
  const { site } = current;

  const [devices, clients] = await Promise.all([
    apiFetch<UniFiDeviceList>(`/sites/${site.id}/unifi/devices`),
    apiFetch<UniFiClientList>(`/sites/${site.id}/unifi/clients`),
  ]);

  const switches = devices.items.filter((d) => d.features.includes("switching"));

  const wiredCountByDevice = new Map<string, number>();
  for (const c of clients.items) {
    if (c.type !== "WIRED") continue;
    wiredCountByDevice.set(
      c.uplink_device_id,
      (wiredCountByDevice.get(c.uplink_device_id) ?? 0) + 1,
    );
  }

  return (
    <div className="max-w-4xl space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Switches</h1>
        <p className="mt-1 text-sm text-egg-text-secondary">
          Site: {site.name} · {switches.length}{" "}
          {switches.length === 1 ? "switch" : "switches"}
        </p>
      </div>

      {switches.length === 0 ? (
        <EmptyCard message="Nenhum switch sincronizado ainda. Requer um agente local com a integração UniFi configurada." />
      ) : (
        <ul className="space-y-2">
          {switches.map((sw) => (
            <li
              key={sw.id}
              className="rounded-lg border border-egg-border bg-egg-surface p-4"
            >
              <div className="flex items-center justify-between">
                <span className="font-medium text-egg-text-primary">{sw.name}</span>
                <span
                  className={`text-xs font-medium ${sw.state === "ONLINE" ? "text-egg-success" : "text-egg-text-secondary"}`}
                >
                  {sw.state}
                </span>
              </div>
              <div className="mt-2 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
                <Field label="Modelo" value={sw.model} />
                <Field label="Firmware" value={sw.firmware_version || "Indisponível"} />
                <Field
                  label="Clientes cabeados"
                  value={String(wiredCountByDevice.get(sw.external_id) ?? 0)}
                />
                <Field label="IP" value={sw.ip_address || "Indisponível"} />
              </div>
              <p className="mt-2 text-xs text-egg-text-disabled">
                Fonte: {provenance.unifi_local_api.label}
              </p>
            </li>
          ))}
        </ul>
      )}

      <p className="text-xs text-egg-text-disabled">
        Estatística por porta (PoE, VLAN, velocidade negociada) ainda não
        sincronizada — não exibida até existir (ver
        docs/architecture/05-modelo-dados.md).
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
      <h1 className="text-xl font-semibold text-egg-text-primary">Switches</h1>
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
