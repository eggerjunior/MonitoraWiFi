import path from "node:path";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Monorepo: apps/web importa packages/design-tokens (tokens.css) por
  // caminho relativo. Sem isso, o Turbopack detecta apps/web (onde está o
  // package-lock.json) como raiz do projeto e recusa resolver arquivos fora
  // dela ("leaves the filesystem root"). Ver
  // node_modules/next/dist/docs/01-app/03-api-reference/05-config/01-next-config-js/turbopack.md#root-directory
  turbopack: {
    root: path.join(__dirname, "../.."),
  },
};

export default nextConfig;
