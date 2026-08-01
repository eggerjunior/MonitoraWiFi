// Changelog em código (skill ildemar_app-versioning) — mesma estrutura do
// VersionHistory.swift do app iOS. Nova entrada sempre no topo com
// isCurrent: true; a anterior vira false.
export type VersionEntry = {
  version: string;
  date: string;
  changes: string[];
  isCurrent: boolean;
};

export const versionHistory: VersionEntry[] = [
  {
    version: "0.3.0",
    date: "2026-08-01",
    changes: [
      "Dispositivos, Wi-Fi e Clientes: inventário UniFi real (sincronizado pelo agente local) — deixam de ser placeholders (Fase 3/4)",
    ],
    isCurrent: true,
  },
  {
    version: "0.2.0",
    date: "2026-08-01",
    changes: [
      "Diagnósticos: ferramenta de ping sob demanda — dispara um comando real executado pelo agente do site e acompanha o resultado (Fase 5, início)",
    ],
    isCurrent: false,
  },
  {
    version: "0.1.0",
    date: "2026-07-31",
    changes: [
      "Shell inicial: login via BFF (Route Handlers) contra o backend Go, navegação lateral, tema claro/escuro sem FOUC",
      "Visão geral com organizações/sites reais do backend (Fase 1)",
      "Internet: ping/speed tests reais do agente local, com badge de proveniência por métrica (Fase 2)",
    ],
    isCurrent: false,
  },
];
