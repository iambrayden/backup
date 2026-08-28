"use client";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { Shell } from "@/components/Shell";
import { StatusBadge } from "@/components/StatusBadge";
import { api, type Device } from "@/lib/api";
import { getToken, isAuthenticated } from "@/lib/auth";

function fmtDate(s: string | null) {
  if (!s) return "Never";
  return new Date(s).toLocaleString();
}

export default function DashboardPage() {
  const router = useRouter();
  const [devices, setDevices] = useState<Device[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [enrollToken, setEnrollToken] = useState<string | null>(null);
  const [tokenLabel, setTokenLabel] = useState("");

  useEffect(() => {
    if (!isAuthenticated()) { router.replace("/login"); return; }
    const token = getToken()!;
    api.listDevices(token)
      .then(setDevices)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, [router]);

  async function handleCreateToken() {
    const token = getToken()!;
    try {
      const { token: t } = await api.createEnrollmentToken(token, tokenLabel || "manual");
      setEnrollToken(t);
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "Failed");
    }
  }

  return (
    <Shell>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Fleet Dashboard</h1>
        <div className="flex items-center gap-2">
          <input
            value={tokenLabel}
            onChange={(e) => setTokenLabel(e.target.value)}
            placeholder="Token label"
            className="border border-gray-300 rounded-lg px-3 py-1.5 text-sm"
          />
          <button
            onClick={handleCreateToken}
            className="bg-brand-600 hover:bg-brand-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
          >
            + Enroll Device
          </button>
        </div>
      </div>

      {enrollToken && (
        <div className="mb-6 bg-yellow-50 border border-yellow-200 rounded-xl p-4">
          <p className="text-sm font-medium text-yellow-900 mb-1">Enrollment Token (one-time use, valid 72h)</p>
          <code className="text-xs bg-yellow-100 px-3 py-1 rounded font-mono break-all">{enrollToken}</code>
          <p className="text-xs text-yellow-700 mt-2">
            Run: <code className="bg-yellow-100 px-1 rounded">backup-agent --config /etc/backup-agent/config.yaml --enroll</code> with this token in the config.
          </p>
          <button onClick={() => setEnrollToken(null)} className="text-xs text-yellow-600 underline mt-2">Dismiss</button>
        </div>
      )}

      {loading && <p className="text-gray-500">Loading devices…</p>}
      {error && <p className="text-red-600">{error}</p>}

      {!loading && !error && (
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-gray-50 border-b border-gray-200 text-left">
                <th className="px-4 py-3 font-semibold text-gray-600">Hostname</th>
                <th className="px-4 py-3 font-semibold text-gray-600">OS</th>
                <th className="px-4 py-3 font-semibold text-gray-600">Status</th>
                <th className="px-4 py-3 font-semibold text-gray-600">Last Seen</th>
                <th className="px-4 py-3 font-semibold text-gray-600">Enrolled</th>
                <th className="px-4 py-3"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {devices.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-gray-400">
                    No devices enrolled yet.
                  </td>
                </tr>
              ) : (
                devices.map((d) => (
                  <tr key={d.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-4 py-3 font-medium text-gray-900">{d.hostname}</td>
                    <td className="px-4 py-3 text-gray-600 capitalize">{d.os_type} / {d.arch}</td>
                    <td className="px-4 py-3"><StatusBadge status={d.status} /></td>
                    <td className="px-4 py-3 text-gray-500">{fmtDate(d.last_seen_at)}</td>
                    <td className="px-4 py-3 text-gray-500">{fmtDate(d.enrolled_at)}</td>
                    <td className="px-4 py-3">
                      <Link
                        href={`/devices/${d.id}`}
                        className="text-brand-600 hover:text-brand-700 font-medium"
                      >
                        View →
                      </Link>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </Shell>
  );
}
