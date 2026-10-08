"use client";
export default function ConsoleError({ reset }: { reset: () => void }) {
  return <main className="mx-auto max-w-lg px-6 py-20"><h1 className="text-xl font-semibold">Session verification unavailable</h1><p className="my-4 text-slate-400">Your session could not be checked. Try again when the gateway is available.</p><button onClick={reset} className="rounded border border-slate-600 px-4 py-2 focus-visible:outline-2 focus-visible:outline-sky-400">Try again</button></main>;
}
