import { apiFetch } from "@/lib/api-server";
import { getCurrentSite } from "@/lib/current-site";
import type { UniFiDeviceList } from "@/lib/api-types";

// Primeiro dashboard real da Fase 3/4: inventário de dispositivos UniFi
// sincronizado pelo agente local — nunca simulado. Detalhe de rádio/porta
// ainda não está disponível (capability-matrix.md "a validar") — por isso
// só os campos confirmados aparecem aqui.
export default async function DevicesPage() {
  const current = await getCurrentSite();
  if ("error" in current) {
    return <EmptyState message={current.error} />;
  }
  const { site } = current;

  const devices = await apiFetch<UniFiDeviceList>(`/sites/${site.id}/unifi/devices`);

  return (
    <div className="max-w-4xl space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Dispositivos</h1>
        <p className="mt-1 text-sm text-egg-text-secondary">
          Site: {site.name} · inventário UniFi sincronizado pelo agente local
          ({devices.items.length} {devices.items.length === 1 ? "dispositivo" : "dispositivos"})
        </p>
      </div>

      {devices.items.length === 0 ? (
        <EmptyCard message="Nenhum dispositivo sincronizado ainda. Requer um agente local enrolado com UNIFI_BASE_URL/UNIFI_API_KEY/UNIFI_SITE_ID configurados." />
      ) : (
        <ul className="space-y-2">
          {devices.items.map((d) => (
            <li
              key={d.id}
              className="rounded-lg border border-egg-border bg-egg-surface p-4"
            >
              <div className="flex items-center justify-between">
                <span className="font-medium text-egg-text-primary">{d.name}</span>
                <StateBadge state={d.state} />
              </div>
              <div className="mt-2 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
                <Field label="Modelo" value={d.model} />
                <Field label="Firmware" value={d.firmware_version || "Indisponível"} />
                <Field label="IP" value={d.ip_address || "Indisponível"} />
                <Field label="MAC" value={d.mac_address} />
              </div>
              <div className="mt-2 flex flex-wrap gap-1">
                {d.features.map((f) => (
                  <span
                    key={f}
                    className="rounded-full bg-egg-accent/10 px-2 py-0.5 text-xs text-egg-accent"
                  >
                    {f}
                  </span>
                ))}
              </div>
            </li>
          ))}
        </ul>
      )}

      <p className="text-xs text-egg-text-disabled">
        Detalhe de rádio (canal, potência, utilização) e estatística por porta
        ainda não confirmados contra a instalação real — não exibidos até
        serem validados (ver docs/unifi/capability-matrix.md).
      </p>
    </div>
  );
}

function StateBadge({ state }: { state: string }) {
  const isOnline = state === "ONLINE";
  return (
    <span
      className={`text-xs font-medium ${isOnline ? "text-egg-success" : "text-egg-text-secondary"}`}
    >
      {state}
    </span>
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
      <h1 className="text-xl font-semibold text-egg-text-primary">Dispositivos</h1>
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
