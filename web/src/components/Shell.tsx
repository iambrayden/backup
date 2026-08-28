"use client";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { clearAuth, getUser } from "@/lib/auth";

export function Shell({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const user = getUser();

  function handleLogout() {
    clearAuth();
    router.replace("/login");
  }

  return (
    <div className="min-h-screen flex flex-col">
      <header className="bg-brand-900 text-white px-6 py-3 flex items-center justify-between shadow-lg">
        <Link href="/dashboard" className="flex items-center gap-2 font-bold text-lg hover:opacity-90">
          <span>🛡️</span>
          <span>Waylon Backup</span>
        </Link>
        <div className="flex items-center gap-4 text-sm">
          {user && (
            <span className="text-brand-100">
              {user.display_name || user.email}
              <span className="ml-2 text-xs bg-brand-700 px-2 py-0.5 rounded-full">{user.role}</span>
            </span>
          )}
          <button
            onClick={handleLogout}
            className="text-brand-200 hover:text-white transition-colors"
          >
            Sign out
          </button>
        </div>
      </header>
      <main className="flex-1 p-6 max-w-7xl mx-auto w-full">{children}</main>
    </div>
  );
}
