import { apiFetch } from "@/lib/api-server";
import { getCurrentSite } from "@/lib/current-site";
import type { UniFiClientList, UniFiDeviceList } from "@/lib/api-types";

export default async function ClientsPage() {
  const current = await getCurrentSite();
  if ("error" in current) {
    return <EmptyState message={current.error} />;
  }
  const { site } = current;

  const [clients, devices] = await Promise.all([
    apiFetch<UniFiClientList>(`/sites/${site.id}/unifi/clients`),
    apiFetch<UniFiDeviceList>(`/sites/${site.id}/unifi/devices`),
  ]);

  const deviceNameByExternalId = new Map(devices.items.map((d) => [d.external_id, d.name]));

  return (
    <div className="max-w-4xl space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Clientes</h1>
        <p className="mt-1 text-sm text-egg-text-secondary">
          Site: {site.name} · clientes conhecidos pelo UniFi ({clients.items.length}{" "}
          {clients.items.length === 1 ? "cliente" : "clientes"})
        </p>
      </div>

      {clients.items.length === 0 ? (
        <EmptyCard message="Nenhum cliente sincronizado ainda. Requer um agente local enrolado com a integração UniFi configurada." />
      ) : (
        <ul className="space-y-2">
          {clients.items.map((c) => (
            <li
              key={c.id}
              className="rounded-lg border border-egg-border bg-egg-surface p-4"
            >
              <div className="flex items-center justify-between">
                <span className="font-medium text-egg-text-primary">
                  {c.name || "Sem nome"}
                </span>
                <span className="text-xs uppercase text-egg-text-secondary">{c.type}</span>
              </div>
              <div className="mt-2 grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
                <Field label="IP" value={c.ip_address || "Indisponível"} />
                <Field label="MAC" value={c.mac_address} />
                <Field
                  label="Conectado via"
                  value={deviceNameByExternalId.get(c.uplink_device_id) ?? "Indisponível"}
                />
              </div>
            </li>
          ))}
        </ul>
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

function EmptyState({ message }: { message: string }) {
  return (
    <div className="max-w-4xl">
      <h1 className="text-xl font-semibold text-egg-text-primary">Clientes</h1>
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
