import { Suspense } from "react";

import InvestigationWorkspace from "@/components/sentinel/investigation-workspace";

export default function InvestigationsPage() {
  return (
    <Suspense
      fallback={
        <main className="sentinel-grid sentinel-glow min-h-screen bg-[#070a0f] text-slate-100">
          <div className="grid min-h-screen place-items-center text-sm text-slate-500">
            Loading investigation workspace…
          </div>
        </main>
      }
    >
      <InvestigationWorkspace />
    </Suspense>
  );
}
