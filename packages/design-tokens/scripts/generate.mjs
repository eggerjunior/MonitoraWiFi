#!/usr/bin/env node
// Gera tokens.ts (web) e DesignTokens.swift (iOS) a partir de tokens.json —
// fonte única de verdade. Nunca editar os arquivos gerados à mão.
import { readFileSync, writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const root = join(here, "..");
const tokens = JSON.parse(readFileSync(join(root, "tokens.json"), "utf8"));

const banner = "// GERADO AUTOMATICAMENTE por packages/design-tokens/scripts/generate.mjs\n// Não editar à mão — edite tokens.json e rode `node scripts/generate.mjs`.\n\n";

function tsColorObject(scheme) {
  const entries = Object.entries(tokens.color[scheme])
    .map(([k, v]) => `    ${k}: "${v}",`)
    .join("\n");
  return `{\n${entries}\n  }`;
}

function tsTypographyScale() {
  const entries = Object.entries(tokens.typography.scale)
    .map(([k, v]) => `    ${k}: { size: ${v.size}, lineHeight: ${v.lineHeight}, weight: ${v.weight} },`)
    .join("\n");
  return `{\n${entries}\n}`;
}

const ts = `${banner}export const colorLight = ${tsColorObject("light")} as const;

export const colorDark = ${tsColorObject("dark")} as const;

export const typography = {
  fontFamilySystem: "${tokens.typography.fontFamilySystem}",
  scale: ${tsTypographyScale()},
} as const;

export const space = ${JSON.stringify(tokens.space, null, 2)} as const;

export const radius = ${JSON.stringify(tokens.radius, null, 2)} as const;

export type ColorToken = keyof typeof colorLight;
export type MetricSource = ${Object.keys(tokens.provenance).map((k) => `"${k}"`).join(" | ")};

export const provenance: Record<MetricSource, { label: string; icon: string }> = ${JSON.stringify(tokens.provenance, null, 2)};
`;

writeFileSync(join(root, "tokens.ts"), ts);

function swiftColorCase(scheme) {
  return Object.entries(tokens.color[scheme])
    .map(([k, v]) => `        case .${k}: return Color(hex: "${v}")`)
    .join("\n");
}

const swiftCaseNames = Object.keys(tokens.color.light)
  .map((k) => `    case ${k}`)
  .join("\n");

const swiftProvenanceCases = Object.entries(tokens.provenance)
  .map(([k, v]) => `    case ${toSwiftEnumCase(k)} = "${k}"`)
  .join("\n");

function toSwiftEnumCase(snake) {
  const parts = snake.split("_");
  return parts[0] + parts.slice(1).map((p) => p[0].toUpperCase() + p.slice(1)).join("");
}

const swift = `${banner}import SwiftUI

/// Token de cor semântico. O valor concreto depende do esquema (claro/escuro),
/// resolvido em runtime via \`Color.egger(_:for:)\`.
public enum EggerColorToken: String, CaseIterable, Sendable {
${swiftCaseNames}
}

public extension Color {
    static func egger(_ token: EggerColorToken, scheme: ColorScheme) -> Color {
        switch scheme {
        case .dark:
            switch token {
${swiftColorCase("dark")}
            }
        default:
            switch token {
${swiftColorCase("light")}
            }
        }
    }

    init(hex: String) {
        var value: UInt64 = 0
        Scanner(string: hex.replacingOccurrences(of: "#", with: "")).scanHexInt64(&value)
        let r = Double((value >> 16) & 0xFF) / 255
        let g = Double((value >> 8) & 0xFF) / 255
        let b = Double(value & 0xFF) / 255
        self.init(.sRGB, red: r, green: g, blue: b, opacity: 1)
    }
}

/// Fonte de proveniência de uma métrica (Seção 2.1 do documento-fonte): toda
/// métrica exibida indica de onde veio, nunca é inventada.
public enum EggerMetricSource: String, CaseIterable, Sendable, Codable {
${swiftProvenanceCases}
}
`;

writeFileSync(join(root, "DesignTokens.swift"), swift);

function cssVars(scheme) {
  return Object.entries(tokens.color[scheme])
    .map(([k, v]) => `  --egger-color-${camelToKebab(k)}: ${v};`)
    .join("\n");
}

function camelToKebab(s) {
  return s.replace(/[A-Z]/g, (m) => "-" + m.toLowerCase());
}

const css = `/* GERADO AUTOMATICAMENTE por packages/design-tokens/scripts/generate.mjs — não editar à mão */

:root {
${cssVars("light")}
}

@media (prefers-color-scheme: dark) {
  :root {
${cssVars("dark")}
  }
}

:root[data-theme="dark"] {
${cssVars("dark")}
}

:root[data-theme="light"] {
${cssVars("light")}
}
`;

writeFileSync(join(root, "tokens.css"), css);

console.log("gerado: tokens.ts, DesignTokens.swift, tokens.css");
