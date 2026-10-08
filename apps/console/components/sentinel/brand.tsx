import Image from "next/image";
import Link from "next/link";

import { cn } from "@/lib/utils";

type BrandProps = {
  variant?: "header" | "sidebar" | "hero";
  className?: string;
};

export default function SentinelBrand({ variant = "header", className }: BrandProps) {
  const hero = variant === "hero";
  const header = variant === "header";
  const size = hero ? 72 : header ? 36 : 48;

  return (
    <Link
      href="/"
      aria-label="Sentinel home"
      className={cn(
        "inline-flex min-w-0 items-center gap-3 rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-cyan-300 focus-visible:ring-offset-4 focus-visible:ring-offset-[#070a0f]",
        hero && "gap-4",
        className,
      )}
    >
      <Image
        src={header ? "/brand/sentinel-symbol.svg" : "/brand/sentinel-logo.svg"}
        width={size}
        height={size}
        alt=""
        aria-hidden="true"
        unoptimized
        loading="eager"
        className="shrink-0"
      />
      <span className="min-w-0">
        <span className={cn(
          "block font-semibold leading-none tracking-[-0.05em] text-slate-100",
          hero ? "text-3xl sm:text-4xl" : "text-xl",
        )}>
          Sentinel
        </span>
        <span className={cn(
          "mt-1.5 block font-normal leading-relaxed tracking-normal text-slate-400",
          hero ? "text-xs sm:text-sm" : "text-[0.625rem]",
          header && "hidden sm:block",
        )}>
          Threat hunting &amp; security analytics
        </span>
      </span>
    </Link>
  );
}
