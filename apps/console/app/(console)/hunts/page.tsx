import { Suspense } from "react";

import HuntCanvas from "@/components/sentinel/hunt-canvas";

export default function HuntsPage() {
  return (
    <Suspense
      fallback={
        <main className="sentinel-grid sentinel-glow min-h-screen bg-[#070a0f] text-slate-100">
          <div className="grid min-h-screen place-items-center text-sm text-slate-500">
            Loading threat hunt canvas…
          </div>
        </main>
      }
    >
      <HuntCanvas />
    </Suspense>
  );
}
