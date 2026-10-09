"use client";

import Image from "next/image";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";

const SEEN_KEY = "sentinel:opening:v1";
const DURATION = 2600;
const ease = [0.22, 1, 0.36, 1] as const;

function shouldPlay() {
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return false;
  try { return sessionStorage.getItem(SEEN_KEY) !== "seen"; }
  catch { return true; }
}

function subscribe(callback: () => void) {
  const preference = window.matchMedia("(prefers-reduced-motion: reduce)");
  preference.addEventListener("change", callback);
  return () => preference.removeEventListener("change", callback);
}

const serverSnapshot = () => false;

export default function OpeningSequence({ children }: { children: React.ReactNode }) {
  const eligible = useSyncExternalStore(subscribe, shouldPlay, serverSnapshot);
  const [running, setRunning] = useState(true);
  const visible = eligible && running;
  const reducedMotion = useReducedMotion();
  const skip = useRef<HTMLButtonElement>(null);
  const content = useRef<HTMLDivElement>(null);
  const restoreFocus = useRef(false);

  const finish = useCallback(() => {
    restoreFocus.current = document.activeElement === skip.current;
    try { sessionStorage.setItem(SEEN_KEY, "seen"); } catch { /* Storage is optional. */ }
    setRunning(false);
  }, []);

  useEffect(() => {
    if (!visible) return;
    const previousOverflow = document.documentElement.style.overflow;
    document.documentElement.style.overflow = "hidden";
    const focusFrame = requestAnimationFrame(() => skip.current?.focus({ preventScroll: true }));
    const timer = window.setTimeout(finish, DURATION);
    const escape = (event: KeyboardEvent) => {
      if (event.key === "Escape") finish();
      if (event.key === "Tab") {
        event.preventDefault();
        skip.current?.focus({ preventScroll: true });
      }
    };
    window.addEventListener("keydown", escape);
    return () => {
      cancelAnimationFrame(focusFrame);
      clearTimeout(timer);
      window.removeEventListener("keydown", escape);
      document.documentElement.style.overflow = previousOverflow;
    };
  }, [visible, finish]);

  return (
    <>
      <div ref={content} inert={visible} tabIndex={-1} className="outline-none">
        {children}
      </div>
      <AnimatePresence onExitComplete={() => {
        if (restoreFocus.current) content.current?.focus({ preventScroll: true });
      }}>
        {visible && (
          <motion.section
            key="sentinel-opening"
            data-sentinel-opening
            role="dialog"
            aria-modal="true"
            aria-label="Welcome to Sentinel"
            className="fixed inset-0 z-[110] flex min-h-dvh flex-col items-center justify-center overflow-hidden bg-[#060910]"
            initial={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: reducedMotion ? 0 : 0.45, ease }}
          >
            <div aria-hidden="true" className="pointer-events-none absolute inset-0 sentinel-opening-atmosphere" />
            <motion.div
              aria-hidden="true"
              className="pointer-events-none absolute h-[32rem] w-[32rem] rounded-full bg-cyan-400/[0.07] blur-[90px]"
              initial={{ scale: 0.3, opacity: 0 }}
              animate={{ scale: [0.3, 1.2, 1], opacity: [0, 1, 0.6] }}
              transition={{ duration: 2.2, ease }}
            />
            <div className="relative flex w-full flex-col items-center px-5">
              <div aria-hidden="true" className="relative flex h-[min(68vw,22rem)] w-[min(68vw,22rem)] items-center justify-center">
                <motion.svg
                  className="absolute inset-0 h-full w-full overflow-visible"
                  viewBox="0 0 360 360"
                  fill="none"
                  initial={{ rotate: -35 }}
                  animate={{ rotate: 0 }}
                  transition={{ duration: 2.1, ease }}
                >
                  <circle cx="180" cy="180" r="150" stroke="#94a3b8" strokeOpacity=".07" />
                  <motion.circle cx="180" cy="180" r="150" stroke="#67e8f9" strokeWidth="1"
                    initial={{ pathLength: 0, opacity: 0 }}
                    animate={{ pathLength: 0.74, opacity: [0, 0.65, 0.2] }}
                    transition={{ duration: 1.8, delay: 0.1, ease }} />
                  <motion.circle cx="180" cy="180" r="126" stroke="#a78bfa" strokeWidth="1"
                    strokeDasharray="3 12"
                    initial={{ opacity: 0, rotate: 75 }}
                    animate={{ opacity: [0, 0.6, 0.15], rotate: 0 }}
                    transition={{ duration: 1.7, delay: 0.2, ease }} />
                  {[[48, 104], [305, 90], [70, 292], [302, 267]].map(([x, y], index) => (
                    <motion.g key={index}
                      initial={{ opacity: 0, x: (x - 180) * 0.2, y: (y - 180) * 0.2 }}
                      animate={{ opacity: [0, 0.8, 0.3], x: 0, y: 0 }}
                      transition={{ duration: 1.8, delay: index * 0.08, ease }}>
                      <path d={`M${x} ${y}L180 180`} stroke={index % 2 ? "#a78bfa" : "#67e8f9"} strokeOpacity=".15" strokeDasharray="2 8" />
                      <circle cx={x} cy={y} r="3" fill={index % 2 ? "#a78bfa" : "#67e8f9"} />
                      <circle cx={x} cy={y} r="8" stroke={index % 2 ? "#a78bfa" : "#67e8f9"} strokeOpacity=".3" />
                    </motion.g>
                  ))}
                  <motion.circle cx="180" cy="180" r="65" stroke="#7dd3fc" strokeWidth="1"
                    initial={{ scale: 0.6, opacity: 0 }}
                    animate={{ scale: [0.6, 2.5], opacity: [0, 0.7, 0] }}
                    transition={{ duration: 1.25, delay: 0.55, ease }} />
                </motion.svg>
                {[false, true].map((violet) => (
                  <motion.div key={String(violet)}
                    className={`absolute h-px w-36 ${violet ? "bg-gradient-to-l from-transparent to-violet-400" : "bg-gradient-to-r from-transparent to-cyan-300"}`}
                    initial={{ x: violet ? 230 : -230, y: violet ? 70 : -70, rotate: -18, opacity: 0 }}
                    animate={{ x: 0, y: 0, opacity: [0, 1, 0], scaleX: [1, 1.5, 0.2] }}
                    transition={{ duration: 0.9, delay: violet ? 0.12 : 0, ease }} />
                ))}
                <motion.div
                  className="relative h-[42%] w-[42%] drop-shadow-[0_0_30px_rgba(56,189,248,0.25)]"
                  initial={{ scale: 0.35, rotate: -25, opacity: 0 }}
                  animate={{ scale: [0.35, 1.09, 1], rotate: [-25, 4, 0], opacity: [0, 1, 1] }}
                  transition={{ duration: 1.25, delay: 0.3, ease }}
                >
                  <Image src="/brand/sentinel-logo.svg" fill sizes="150px" alt="" unoptimized loading="eager" />
                </motion.div>
              </div>
              <h1 className="-mt-2 flex text-[clamp(2.75rem,8vw,4.5rem)] font-semibold leading-tight tracking-[-0.055em] text-slate-100" aria-label="Sentinel">
                {"Sentinel".split("").map((letter, index) => (
                  <motion.span key={index} aria-hidden="true" className="inline-block"
                    initial={{ opacity: 0, y: 22, rotateX: -50 }}
                    animate={{ opacity: 1, y: 0, rotateX: 0 }}
                    transition={{ duration: 0.6, delay: 0.72 + index * 0.055, ease }}>
                    {letter}
                  </motion.span>
                ))}
              </h1>
              <motion.p className="mt-4 text-sm tracking-wide text-slate-400 sm:text-base"
                initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.6, delay: 1.1, ease }}>
                Find the signal beneath the noise.
              </motion.p>
              <motion.div aria-hidden="true" className="mt-8 flex items-center gap-3 font-mono text-[9px] uppercase tracking-[0.18em] text-slate-500 sm:text-[10px]"
                initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.5, delay: 1.3 }}>
                <span>Threat hunting</span>
                <span className="h-1 w-1 rounded-full bg-violet-400" />
                <span>Security analytics</span>
              </motion.div>
            </div>
            <button ref={skip} type="button" onClick={finish}
              className="absolute bottom-6 right-6 rounded-full border border-white/10 bg-white/[0.025] px-4 py-2 text-xs text-slate-400 transition-colors hover:border-cyan-300/30 hover:text-cyan-200 focus-visible:outline-2 focus-visible:outline-cyan-300 sm:bottom-8 sm:right-8">
              Skip intro <span aria-hidden="true" className="ml-2 text-slate-600">Esc</span>
            </button>
          </motion.section>
        )}
      </AnimatePresence>
    </>
  );
}
