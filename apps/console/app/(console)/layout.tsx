import { redirect } from "next/navigation";
import SessionProvider from "@/components/sentinel/session-provider";
import { consoleAuthMode, readServerSession } from "@/lib/sentinel/session-server";

export default async function ConsoleLayout({ children }: { children: React.ReactNode }) {
  if (consoleAuthMode === "development") return <SessionProvider session={null} development>{children}</SessionProvider>;
  const session = await readServerSession();
  if (!session) redirect("/sign-in");
  return <SessionProvider session={session}>{children}</SessionProvider>;
}
