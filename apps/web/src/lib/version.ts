import "server-only";

import packageJson from "../../package.json";

// Fonte única de versão (skill ildemar_app-versioning, "outros-stacks.md"):
// package.json version + GIT_COMMIT/BUILD_DATE injetados no build da imagem
// (ver Dockerfile, --build-arg) — nunca hardcoded aqui. Fallback "dev" para
// builds locais sem os build args (ex.: `npm run dev`).
export const APP_VERSION: string = packageJson.version;
export const GIT_COMMIT: string = process.env.GIT_COMMIT || "dev";
export const BUILD_DATE: string | null = process.env.BUILD_DATE || null;

export function commitUrl(commit: string): string | null {
  if (commit === "dev") {
    return null;
  }
  return `https://github.com/eggerjunior/MonitoraWiFi/commit/${commit}`;
}
