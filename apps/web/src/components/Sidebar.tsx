"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";

const NAV_ITEMS = [
  { href: "/", label: "Visão geral" },
  { href: "/internet", label: "Internet" },
  { href: "/wifi", label: "Wi-Fi" },
  { href: "/devices", label: "Dispositivos" },
  { href: "/switches", label: "Switches" },
  { href: "/clients", label: "Clientes" },
  { href: "/map", label: "Mapa" },
  { href: "/diagnostics", label: "Diagnósticos" },
  { href: "/alerts", label: "Alertas" },
  { href: "/history", label: "Histórico" },
  { href: "/reports", label: "Relatórios" },
  { href: "/settings", label: "Configurações" },
] as const;

export function Sidebar() {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);

  return (
    <nav
      aria-label="Navegação principal"
      className={`shrink-0 border-r border-egg-border bg-egg-surface transition-[width] duration-150 ${
        collapsed ? "w-16" : "w-56"
      }`}
    >
      <div className="flex h-14 items-center justify-between px-3">
        {!collapsed && (
          <span className="truncate text-sm font-semibold text-egg-text-primary">
            Egger
          </span>
        )}
        <button
          type="button"
          onClick={() => setCollapsed((v) => !v)}
          aria-label={collapsed ? "Expandir menu" : "Recolher menu"}
          className="rounded-md p-1.5 text-egg-text-secondary hover:text-egg-text-primary"
        >
          {collapsed ? "»" : "«"}
        </button>
      </div>

      <ul className="space-y-0.5 px-2">
        {NAV_ITEMS.map((item) => {
          const active = pathname === item.href;
          return (
            <li key={item.href}>
              <Link
                href={item.href}
                aria-current={active ? "page" : undefined}
                title={collapsed ? item.label : undefined}
                className={`block rounded-md px-3 py-2 text-sm ${
                  active
                    ? "bg-egg-accent/10 font-medium text-egg-accent"
                    : "text-egg-text-secondary hover:bg-egg-background hover:text-egg-text-primary"
                }`}
              >
                {collapsed ? item.label.slice(0, 1) : item.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
