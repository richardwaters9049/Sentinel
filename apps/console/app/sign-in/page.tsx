import SentinelBrand from "@/components/sentinel/brand";
import SignInForm from "@/components/sentinel/sign-in-form";

export default function SignInPage() {
  return (
    <main className="flex min-h-dvh items-center justify-center px-6 py-12">
      <section
        aria-labelledby="sign-in-title"
        className="w-full max-w-md rounded-2xl border border-slate-800 bg-[#0c111b] p-7 sm:p-9"
      >
        <SentinelBrand variant="hero" className="mb-8" />
        <h1 id="sign-in-title" className="text-2xl font-semibold tracking-tight">
          Analyst console sign-in
        </h1>
        <p className="mt-3 text-sm leading-6 text-slate-400">
          Access evidence, hunts and investigations using your verified lab identity.
        </p>
        <SignInForm />
      </section>
    </main>
  );
}
