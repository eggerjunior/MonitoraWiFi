import { apiFetch } from "@/lib/api-server";
import { getCurrentSite } from "@/lib/current-site";
import type { UniFiDeviceList } from "@/lib/api-types";
import { buildDeviceTree, type DeviceNode } from "@/lib/unifi-topology";

// Primeiro dashboard real da Fase 3/4: inventário de dispositivos UniFi
// sincronizado pelo agente local — nunca simulado. Detalhe de rádio/porta
// (capability-matrix.md, itens confirmados como indisponíveis nesta versão
// da Network API) segue fora — só os campos confirmados aparecem aqui.
// Topologia dispositivo→dispositivo (uplink_device_id, confirmado em
// 2026-08-02) organiza a lista em árvore gateway → switch → AP em vez de
// lista plana.
export default async function DevicesPage() {
  const current = await getCurrentSite();
  if ("error" in current) {
    return <EmptyState message={current.error} />;
  }
  const { site } = current;

  const devices = await apiFetch<UniFiDeviceList>(`/sites/${site.id}/unifi/devices`);
  const tree = buildDeviceTree(devices.items);

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
          {tree.map((node) => (
            <DeviceTreeItem key={node.id} node={node} depth={0} />
          ))}
        </ul>
      )}

      <p className="text-xs text-egg-text-disabled">
        Potência de transmissão, utilização de canal, clientes/airtime/retries
        por rádio, e contadores RX/TX/erros/PoE por porta confirmados
        indisponíveis nesta versão da Network API local — não exibidos (ver
        docs/unifi/capability-matrix.md).
      </p>
    </div>
  );
}

function DeviceTreeItem({ node, depth }: { node: DeviceNode; depth: number }) {
  return (
    <li style={{ marginLeft: depth * 24 }}>
      <div className="rounded-lg border border-egg-border bg-egg-surface p-4">
        <div className="flex items-center justify-between">
          <span className="font-medium text-egg-text-primary">
            {depth > 0 && <span className="mr-1 text-egg-text-disabled">└</span>}
            {node.name}
          </span>
          <StateBadge state={node.state} />
        </div>
        <div className="mt-2 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
          <Field label="Modelo" value={node.model} />
          <Field label="Firmware" value={node.firmware_version || "Indisponível"} />
          <Field label="IP" value={node.ip_address || "Indisponível"} />
          <Field label="MAC" value={node.mac_address} />
        </div>
        <div className="mt-2 flex flex-wrap gap-1">
          {node.features.map((f) => (
            <span
              key={f}
              className="rounded-full bg-egg-accent/10 px-2 py-0.5 text-xs text-egg-accent"
            >
              {f}
            </span>
          ))}
        </div>
      </div>
      {node.children.length > 0 && (
        <ul className="mt-2 space-y-2">
          {node.children.map((child) => (
            <DeviceTreeItem key={child.id} node={child} depth={depth + 1} />
          ))}
        </ul>
      )}
    </li>
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
