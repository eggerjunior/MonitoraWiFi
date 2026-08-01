import { getCurrentSite } from "@/lib/current-site";
import { PingTool } from "@/components/PingTool";
import { BatchPingTool } from "@/components/BatchPingTool";
import { DnsLookupTool } from "@/components/DnsLookupTool";
import { TracerouteTool } from "@/components/TracerouteTool";
import { SslCheckTool } from "@/components/SslCheckTool";
import { RdapTool } from "@/components/RdapTool";
import { SubnetCalculator } from "@/components/SubnetCalculator";

// Ferramentas de diagnóstico sob demanda (Fase 5): ping, DNS lookup,
// traceroute e SSL/TLS checker disparam um comando de verdade executado pelo
// agente do site — nunca simulam um resultado antes de o agente responder
// (Seção 2.1). RDAP/WHOIS roda direto no backend (informação pública da
// internet, não da LAN). A calculadora de sub-rede é cálculo puro.
export default async function DiagnosticsPage() {
  const current = await getCurrentSite();

  return (
    <div className="max-w-3xl space-y-8">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Diagnósticos</h1>
        {"site" in current ? (
          <p className="mt-1 text-sm text-egg-text-secondary">
            Site: {current.site.name} · executado pelo agente local do site (requer
            um agente ativo).
          </p>
        ) : (
          <p className="mt-1 text-sm text-egg-text-secondary">{current.error}</p>
        )}
      </div>

      {"site" in current && (
        <>
          <PingTool siteId={current.site.id} />
          <BatchPingTool siteId={current.site.id} />
          <DnsLookupTool siteId={current.site.id} />
          <TracerouteTool siteId={current.site.id} />
          <SslCheckTool siteId={current.site.id} />
        </>
      )}

      <RdapTool />
      <SubnetCalculator />
    </div>
  );
}
