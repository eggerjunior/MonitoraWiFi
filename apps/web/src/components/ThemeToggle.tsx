"use client";

import { useState } from "react";

type Theme = "light" | "dark";

function applyTheme(theme: Theme) {
  document.documentElement.setAttribute("data-theme", theme);
  try {
    localStorage.setItem("egger-theme", theme);
  } catch {
    // localStorage pode estar indisponível (modo privado); tema simplesmente
    // não persiste entre sessões nesse caso, o que é aceitável.
  }
}

// Lê o data-theme atual do <html> (já definido antes do primeiro paint pelo
// script inline em layout.tsx) via inicializador preguiçoso do useState —
// nunca em um efeito, para não disparar uma renderização em cascata só para
// sincronizar um valor que o DOM já tem.
function readInitialTheme(): Theme {
  if (typeof document === "undefined") {
    return "light";
  }
  const current = document.documentElement.getAttribute("data-theme");
  return current === "dark" ? "dark" : "light";
}

export function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>(readInitialTheme);

  function toggle() {
    const next: Theme = theme === "light" ? "dark" : "light";
    setTheme(next);
    applyTheme(next);
  }

  return (
    <button
      type="button"
      onClick={toggle}
      aria-label={theme === "light" ? "Ativar tema escuro" : "Ativar tema claro"}
      className="rounded-md border border-egg-border px-3 py-1.5 text-sm text-egg-text-secondary hover:text-egg-text-primary"
    >
      {theme === "light" ? "Escuro" : "Claro"}
    </button>
  );
}
