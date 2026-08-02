import type { UniFiDevice } from "./api-types";

export interface DeviceNode extends UniFiDevice {
  children: DeviceNode[];
}

// buildDeviceTree monta a hierarquia gateway → switch → AP a partir de
// uplink_device_id (Fase 3, confirmado em 2026-08-02 contra a instalação
// real: GET .../devices/{id} traz uplink.deviceId). Um dispositivo cujo
// uplink não existe na lista atual (ex.: dado transitório durante uma
// sincronização) vira raiz também — nunca descartado silenciosamente.
export function buildDeviceTree(devices: UniFiDevice[]): DeviceNode[] {
  const byExternalId = new Map<string, DeviceNode>();
  for (const d of devices) {
    byExternalId.set(d.external_id, { ...d, children: [] });
  }

  const roots: DeviceNode[] = [];
  for (const node of byExternalId.values()) {
    const parent = node.uplink_device_id ? byExternalId.get(node.uplink_device_id) : undefined;
    if (parent) {
      parent.children.push(node);
    } else {
      roots.push(node);
    }
  }
  return roots;
}
