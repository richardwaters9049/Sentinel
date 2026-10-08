import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import { Suspense } from "react";

import SentinelBrand from "@/components/sentinel/brand";
import { TooltipProvider } from "@/components/ui/tooltip";

import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const jetBrainsMono = JetBrains_Mono({
  variable: "--font-jetbrains-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Sentinel | Analyst Console",
  description:
    "Defensive threat hunting, detection engineering, and investigation console for the Northstar Energy synthetic lab.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className={`${inter.variable} ${jetBrainsMono.variable} dark`}
    >
      <body className="min-h-screen bg-background text-foreground antialiased">
        <a
          href="#main-content"
          className="fixed left-3 top-3 z-[100] -translate-y-24 rounded-lg bg-sky-300 px-3 py-2 text-sm font-semibold text-slate-950 transition focus:translate-y-0"
        >
          Skip to main content
        </a>
        <TooltipProvider>
          <div id="main-content" tabIndex={-1}>
            <Suspense
              fallback={
                <div className="flex min-h-dvh flex-col items-center justify-center gap-6 px-6">
                  <SentinelBrand variant="hero" />
                  <p className="text-sm text-slate-400" role="status">
                    Checking console access…
                  </p>
                </div>
              }
            >
              {children}
            </Suspense>
          </div>
        </TooltipProvider>
      </body>
    </html>
  );
}
