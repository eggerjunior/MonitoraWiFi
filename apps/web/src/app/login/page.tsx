"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const response = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });

      if (!response.ok) {
        const payload = await response.json().catch(() => null);
        setError(payload?.message ?? "Não foi possível entrar. Tente novamente.");
        return;
      }

      router.push("/");
      router.refresh();
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-4">
      <div className="w-full max-w-sm rounded-lg border border-egg-border bg-egg-surface p-8 shadow-sm">
        <h1 className="text-xl font-semibold text-egg-text-primary">
          Egger Network Intelligence
        </h1>
        <p className="mt-1 text-sm text-egg-text-secondary">
          Entre com seu e-mail e senha.
        </p>

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-egg-text-primary">
              E-mail
            </label>
            <input
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-1 w-full rounded-md border border-egg-border bg-egg-background px-3 py-2 text-sm text-egg-text-primary outline-none focus:border-egg-accent"
            />
          </div>

          <div>
            <label htmlFor="password" className="block text-sm font-medium text-egg-text-primary">
              Senha
            </label>
            <input
              id="password"
              name="password"
              type="password"
              autoComplete="current-password"
              required
              minLength={8}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="mt-1 w-full rounded-md border border-egg-border bg-egg-background px-3 py-2 text-sm text-egg-text-primary outline-none focus:border-egg-accent"
            />
          </div>

          {error && (
            <p role="alert" className="text-sm text-egg-critical">
              {error}
            </p>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full rounded-md bg-egg-accent px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-egg-accent-pressed disabled:opacity-60"
          >
            {loading ? "Entrando…" : "Entrar"}
          </button>
        </form>
      </div>
    </main>
  );
}
