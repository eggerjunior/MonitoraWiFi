// GERADO AUTOMATICAMENTE por packages/design-tokens/scripts/generate.mjs
// Não editar à mão — edite tokens.json e rode `node scripts/generate.mjs`.

export const colorLight = {
    background: "#F5F6F8",
    surface: "#FFFFFF",
    surfaceRaised: "#FFFFFF",
    border: "#D8DBE0",
    textPrimary: "#12151A",
    textSecondary: "#4B5563",
    textDisabled: "#6D7787",
    accent: "#0A6CFF",
    accentPressed: "#0854CC",
    success: "#0F893E",
    warning: "#A3690A",
    critical: "#C22C2C",
    info: "#0A6CFF",
    unavailable: "#8A8F98",
  } as const;

export const colorDark = {
    background: "#0E1116",
    surface: "#161B22",
    surfaceRaised: "#1E242D",
    border: "#2A313C",
    textPrimary: "#F2F4F7",
    textSecondary: "#AEB4BF",
    textDisabled: "#7B8394",
    accent: "#4C9AFF",
    accentPressed: "#7AB6FF",
    success: "#3FCB78",
    warning: "#E0A526",
    critical: "#F26D6D",
    info: "#4C9AFF",
    unavailable: "#6B7280",
  } as const;

export const typography = {
  fontFamilySystem: "system-ui, -apple-system, 'SF Pro Text', 'Segoe UI', sans-serif",
  scale: {
    caption: { size: 12, lineHeight: 16, weight: 400 },
    body: { size: 15, lineHeight: 22, weight: 400 },
    bodyStrong: { size: 15, lineHeight: 22, weight: 600 },
    title3: { size: 18, lineHeight: 24, weight: 600 },
    title2: { size: 22, lineHeight: 28, weight: 700 },
    title1: { size: 28, lineHeight: 34, weight: 700 },
    largeTitle: { size: 34, lineHeight: 41, weight: 800 },
},
} as const;

export const space = {
  "xxs": 2,
  "xs": 4,
  "sm": 8,
  "md": 12,
  "lg": 16,
  "xl": 24,
  "xxl": 32,
  "xxxl": 48
} as const;

export const radius = {
  "sm": 6,
  "md": 10,
  "lg": 16,
  "pill": 999
} as const;

export type ColorToken = keyof typeof colorLight;
export type MetricSource = "unifi_local_api" | "unifi_site_manager" | "agent_icmp" | "agent_tcp" | "agent_udp" | "agent_dns" | "agent_http" | "snmp" | "arkit" | "estimated" | "user_declared" | "unavailable";

export const provenance: Record<MetricSource, { label: string; icon: string }> = {
  "unifi_local_api": {
    "label": "UniFi — API local",
    "icon": "wifi.router"
  },
  "unifi_site_manager": {
    "label": "UniFi — Site Manager",
    "icon": "cloud"
  },
  "agent_icmp": {
    "label": "Agente — ICMP",
    "icon": "bolt.horizontal"
  },
  "agent_tcp": {
    "label": "Agente — TCP",
    "icon": "bolt.horizontal"
  },
  "agent_udp": {
    "label": "Agente — UDP",
    "icon": "bolt.horizontal"
  },
  "agent_dns": {
    "label": "Agente — DNS",
    "icon": "network"
  },
  "agent_http": {
    "label": "Agente — HTTP",
    "icon": "globe"
  },
  "snmp": {
    "label": "SNMP",
    "icon": "gearshape"
  },
  "arkit": {
    "label": "ARKit (geometria, não rádio)",
    "icon": "arkit"
  },
  "estimated": {
    "label": "Estimativa matemática",
    "icon": "function"
  },
  "user_declared": {
    "label": "Informado pelo usuário",
    "icon": "person"
  },
  "unavailable": {
    "label": "Indisponível",
    "icon": "questionmark.circle"
  }
};
