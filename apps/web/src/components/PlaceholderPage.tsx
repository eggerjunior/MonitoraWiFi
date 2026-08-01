export function PlaceholderPage({
  title,
  phase,
}: {
  title: string;
  phase: string;
}) {
  return (
    <div className="max-w-2xl">
      <h1 className="text-xl font-semibold text-egg-text-primary">{title}</h1>
      <p className="mt-2 text-sm text-egg-text-secondary">
        Em construção — esta tela chega na {phase} (ver{" "}
        <code className="rounded bg-egg-background px-1 py-0.5">
          docs/architecture/06-roadmap.md
        </code>
        ). Nenhum dado exibido aqui é simulado: quando implementada, esta tela
        mostrará apenas dados reais com sua fonte identificada.
      </p>
    </div>
  );
}
