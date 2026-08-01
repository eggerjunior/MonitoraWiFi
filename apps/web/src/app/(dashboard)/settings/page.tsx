import Link from "next/link";

import { APP_VERSION, BUILD_DATE, GIT_COMMIT, commitUrl } from "@/lib/version";
import { versionHistory } from "@/lib/version-history";

function formatBuildDate(iso: string | null): string | null {
  if (!iso) return null;
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return null;
  return date.toLocaleString("pt-BR", { dateStyle: "medium", timeStyle: "short" });
}

export default function Page() {
  const url = commitUrl(GIT_COMMIT);
  const buildDate = formatBuildDate(BUILD_DATE);

  return (
    <div className="max-w-2xl space-y-8">
      <div>
        <h1 className="text-xl font-semibold text-egg-text-primary">Configurações</h1>
        <p className="mt-2 text-sm text-egg-text-secondary">
          Gestão de integrações, usuários e organização chega na Fase 3 (ver{" "}
          <code className="rounded bg-egg-background px-1 py-0.5">
            docs/architecture/06-roadmap.md
          </code>
          ). Informações de versão abaixo já refletem o build real.
        </p>
      </div>

      <section>
        <h2 className="text-sm font-semibold text-egg-text-primary">Versão</h2>
        <div className="mt-2 rounded-lg border border-egg-border bg-egg-surface p-4 text-sm">
          <div className="font-medium text-egg-text-primary">
            v{APP_VERSION}
            {buildDate ? ` — ${buildDate}` : ""}
          </div>
          <div className="mt-1 text-egg-text-secondary">
            commit:{" "}
            {url ? (
              <Link href={url} className="text-egg-accent underline">
                {GIT_COMMIT}
              </Link>
            ) : (
              <span className="text-egg-text-disabled">{GIT_COMMIT}</span>
            )}
          </div>
        </div>
      </section>

      <section>
        <h2 className="text-sm font-semibold text-egg-text-primary">Histórico de versões</h2>
        <ul className="mt-2 space-y-3">
          {versionHistory.map((entry) => (
            <li
              key={entry.version}
              className="rounded-lg border border-egg-border bg-egg-surface p-4"
            >
              <div className="flex items-center justify-between">
                <span className="font-medium text-egg-text-primary">
                  {entry.version}
                  {entry.isCurrent && (
                    <span className="ml-2 rounded-full bg-egg-accent/10 px-2 py-0.5 text-xs font-medium text-egg-accent">
                      Atual
                    </span>
                  )}
                </span>
                <span className="text-xs text-egg-text-secondary">
                  {entry.isCurrent && buildDate ? buildDate : entry.date}
                </span>
              </div>
              <ul className="mt-2 list-disc space-y-1 pl-5 text-sm text-egg-text-secondary">
                {entry.changes.map((change) => (
                  <li key={change}>{change}</li>
                ))}
              </ul>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
