import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Egger Network Intelligence",
  description:
    "Monitoramento de Internet, LAN, Wi-Fi e equipamentos UniFi — Egger Network Intelligence.",
};

// Tema claro/escuro: o servidor sempre renderiza data-theme="light" como
// padrão; o script inline (técnica documentada em
// node_modules/next/dist/docs/01-app/02-guides/preventing-flash-before-hydration.md)
// aplica a preferência salva em localStorage antes do primeiro paint, evitando
// flash de tema errado. suppressHydrationWarning é necessário porque o script
// altera o atributo antes da hidratação do React.
const THEME_INIT_SCRIPT = `(function(){try{var t=localStorage.getItem("egger-theme");if(t)document.documentElement.setAttribute("data-theme",t)}catch(e){}})()`;

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="pt-BR" data-theme="light" suppressHydrationWarning className="h-full">
      <head>
        <script dangerouslySetInnerHTML={{ __html: THEME_INIT_SCRIPT }} />
      </head>
      <body className="min-h-full antialiased">{children}</body>
    </html>
  );
}
