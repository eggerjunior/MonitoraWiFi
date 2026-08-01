"use client";

import { useRouter } from "next/navigation";

export function LogoutButton() {
  const router = useRouter();

  async function handleLogout() {
    await fetch("/api/auth/logout", { method: "POST" });
    router.push("/login");
    router.refresh();
  }

  return (
    <button
      type="button"
      onClick={handleLogout}
      className="rounded-md border border-egg-border px-3 py-1.5 text-sm text-egg-text-secondary hover:text-egg-text-primary"
    >
      Sair
    </button>
  );
}
