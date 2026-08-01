import { apiFetch } from "@/lib/api-server";
import type { Organization, Page as ApiPage, Site } from "@/lib/api-types";

// Visão geral real da Fase 1: organizações e sites cadastrados, vindos do
// backend (nunca dado simulado). Os cards de saúde de rede (Seção 7) chegam
// na Fase 4, quando houver telemetria real do agente/UniFi para preenchê-los
// honestamente — até lá, mostrar um card vazio de "0%" seria inventar dado.
export default async function OverviewPage() {
  const orgs = await apiFetch<ApiPage<Organization>>(
    "/organizations?page=1&page_size=20",
  );

  return (
    <div className="max-w-3xl space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Visão geral</h1>
        <p className="mt-1 text-sm text-egg-text-secondary">
          Organizações e sites cadastrados nesta instância.
        </p>
      </div>

      {orgs.total === 0 ? (
        <div className="rounded-lg border border-dashed border-egg-border p-6 text-sm text-egg-text-secondary">
          Nenhuma organização cadastrada ainda.
        </div>
      ) : (
        <ul className="space-y-3">
          {orgs.items.map((org) => (
            <OrganizationCard key={org.id} organization={org} />
          ))}
        </ul>
      )}
    </div>
  );
}

async function OrganizationCard({ organization }: { organization: Organization }) {
  const sites = await apiFetch<ApiPage<Site>>(
    `/sites?organization_id=${organization.id}&page=1&page_size=50`,
  );

  return (
    <li className="rounded-lg border border-egg-border bg-egg-surface p-4">
      <div className="flex items-center justify-between">
        <h2 className="font-medium text-egg-text-primary">{organization.name}</h2>
        <span className="text-xs text-egg-text-secondary">{organization.plan_tier}</span>
      </div>
      {sites.total === 0 ? (
        <p className="mt-2 text-sm text-egg-text-secondary">Nenhum site cadastrado.</p>
      ) : (
        <ul className="mt-2 space-y-1">
          {sites.items.map((site) => (
            <li key={site.id} className="text-sm text-egg-text-secondary">
              {site.name} · <span className="text-xs">{site.timezone}</span>
            </li>
          ))}
        </ul>
      )}
    </li>
  );
}
