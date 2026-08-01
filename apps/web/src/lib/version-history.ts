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
    version: "0.1.0",
    date: "2026-07-31",
    changes: [
      "Shell inicial: login via BFF (Route Handlers) contra o backend Go, navegação lateral, tema claro/escuro sem FOUC",
      "Visão geral com organizações/sites reais do backend (Fase 1)",
      "Internet: ping/speed tests reais do agente local, com badge de proveniência por métrica (Fase 2)",
    ],
    isCurrent: true,
  },
];
