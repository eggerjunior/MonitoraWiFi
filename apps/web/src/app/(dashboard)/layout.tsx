import { redirect } from "next/navigation";

import { getCurrentUser } from "@/lib/auth";
import { Sidebar } from "@/components/Sidebar";
import { ThemeToggle } from "@/components/ThemeToggle";
import { LogoutButton } from "@/components/LogoutButton";

// Gate de autenticação real do shell: roda no servidor, contra a sessão
// validada pelo backend Go (não uma checagem só de UI). `proxy.ts` poderia
// fazer uma checagem otimista adicional no futuro, mas nunca substitui esta.
export default async function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const user = await getCurrentUser();
  if (!user) {
    redirect("/login");
  }

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <header className="flex h-14 items-center justify-between border-b border-egg-border bg-egg-surface px-6">
          <div className="text-sm text-egg-text-secondary">
            {user.email} · <span className="capitalize">{user.role}</span>
          </div>
          <div className="flex items-center gap-2">
            <ThemeToggle />
            <LogoutButton />
          </div>
        </header>
        <main className="flex-1 overflow-y-auto p-6">{children}</main>
      </div>
    </div>
  );
}
