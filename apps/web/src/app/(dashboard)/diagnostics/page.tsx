import { apiFetch } from "@/lib/api-server";
import type { Organization, Page as ApiPage, Site } from "@/lib/api-types";
import { PingTool } from "@/components/PingTool";

// Primeira ferramenta real da Fase 5 (diagnósticos sob demanda): dispara um
// comando de ping executado de verdade pelo agente do site — nunca simula
// um resultado antes de o agente responder (Seção 2.1).
export default async function DiagnosticsPage() {
  const orgs = await apiFetch<ApiPage<Organization>>("/organizations?page=1&page_size=1");
  const org = orgs.items[0];

  if (!org) {
    return <EmptyState message="Nenhuma organização cadastrada ainda — nada para testar." />;
  }

  const sites = await apiFetch<ApiPage<Site>>(
    `/sites?organization_id=${org.id}&page=1&page_size=1`,
  );
  const site = sites.items[0];

  if (!site) {
    return <EmptyState message="Nenhum site cadastrado nesta organização ainda." />;
  }

  return (
    <div className="max-w-3xl space-y-8">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Diagnósticos</h1>
        <p className="mt-1 text-sm text-egg-text-secondary">
          Site: {site.name} · executado pelo agente local do site (requer um agente
          ativo). Outras ferramentas (traceroute, DNS lookup, port scanner etc.) chegam
          ao longo da Fase 5.
        </p>
      </div>

      <PingTool siteId={site.id} />
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="max-w-3xl">
      <h1 className="text-xl font-semibold text-egg-text-primary">Diagnósticos</h1>
      <div className="mt-4 rounded-lg border border-dashed border-egg-border p-6 text-sm text-egg-text-secondary">
        {message}
      </div>
    </div>
  );
}
