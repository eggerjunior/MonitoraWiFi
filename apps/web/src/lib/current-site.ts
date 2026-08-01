import { apiFetch } from "@/lib/api-server";
import type { Organization, Page, Site } from "@/lib/api-types";

// Assunção de single-tenant desta fase (mesma usada em internet/page.tsx e
// diagnostics/page.tsx): primeira organização, primeiro site. Multi-tenant
// real fica para quando houver mais de uma organização/site de verdade.
export async function getCurrentSite(): Promise<
  { site: Site } | { error: string }
> {
  const orgs = await apiFetch<Page<Organization>>("/organizations?page=1&page_size=1");
  const org = orgs.items[0];
  if (!org) {
    return { error: "Nenhuma organização cadastrada ainda." };
  }

  const sites = await apiFetch<Page<Site>>(
    `/sites?organization_id=${org.id}&page=1&page_size=1`,
  );
  const site = sites.items[0];
  if (!site) {
    return { error: "Nenhum site cadastrado nesta organização ainda." };
  }

  return { site };
}
