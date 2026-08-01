"use client";

// Erro acionável (Seção 15): se o backend estiver inalcançável, dizemos isso
// explicitamente em vez de mostrar a página de erro genérica do Next.js.
export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <div className="max-w-md rounded-lg border border-egg-border bg-egg-surface p-6 text-center">
        <h1 className="text-lg font-semibold text-egg-text-primary">
          Não foi possível conectar ao backend
        </h1>
        <p className="mt-2 text-sm text-egg-text-secondary">
          Verifique se <code className="rounded bg-egg-background px-1">apps/api</code>{" "}
          está em execução e acessível. Detalhe técnico: {error.message}
        </p>
        <button
          type="button"
          onClick={reset}
          className="mt-4 rounded-md bg-egg-accent px-3 py-2 text-sm font-semibold text-white hover:bg-egg-accent-pressed"
        >
          Tentar novamente
        </button>
      </div>
    </div>
  );
}
