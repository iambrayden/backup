"use client";
import { useEffect, useState } from "react";
import { useRouter, useParams } from "next/navigation";
import Link from "next/link";
import { Shell } from "@/components/Shell";
import { StatusBadge } from "@/components/StatusBadge";
import { api, type Device, type BackupJob, type Snapshot } from "@/lib/api";
import { getToken, isAuthenticated } from "@/lib/auth";

function fmtSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
  return `${(bytes / 1024 ** 3).toFixed(2)} GB`;
}

function fmtDate(s: string | null) {
  if (!s) return "—";
  return new Date(s).toLocaleString();
}

export default function DevicePage() {
  const router = useRouter();
  const { id } = useParams<{ id: string }>();
  const [device, setDevice] = useState<Device | null>(null);
  const [jobs, setJobs] = useState<BackupJob[]>([]);
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [triggering, setTriggering] = useState(false);
  const [loading, setLoading] = useState(true);

  async function load() {
    const token = getToken()!;
    const [dev, jbs, snaps] = await Promise.all([
      api.getDevice(token, id),
      api.listJobs(token, id),
      api.listSnapshots(token, id),
    ]);
    setDevice(dev);
    setJobs(jbs ?? []);
    setSnapshots(snaps ?? []);
  }

  useEffect(() => {
    if (!isAuthenticated()) { router.replace("/login"); return; }
    load().catch(console.error).finally(() => setLoading(false));
  }, [id, router]);

  async function handleTrigger() {
    const token = getToken()!;
    setTriggering(true);
    try {
      await api.triggerJob(token, id);
      await load();
    } catch (e: unknown) {
      alert(e instanceof Error ? e.message : "Failed");
    } finally {
      setTriggering(false);
    }
  }

  if (loading) return <Shell><p className="text-gray-500">Loading…</p></Shell>;
  if (!device) return <Shell><p className="text-red-600">Device not found.</p></Shell>;

  return (
    <Shell>
      <div className="mb-6">
        <Link href="/dashboard" className="text-brand-600 hover:underline text-sm">← Fleet</Link>
        <div className="mt-2 flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">{device.hostname}</h1>
            <p className="text-gray-500 text-sm mt-1 capitalize">{device.os_type} · {device.arch} · Agent {device.agent_version}</p>
          </div>
          <div className="flex items-center gap-3">
            <StatusBadge status={device.status} />
            <button
              onClick={handleTrigger}
              disabled={triggering}
              className="bg-brand-600 hover:bg-brand-700 disabled:opacity-50 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
            >
              {triggering ? "Triggering…" : "▶ Run Backup"}
            </button>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <section>
          <h2 className="text-lg font-semibold text-gray-800 mb-3">Recent Jobs</h2>
          <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
            {jobs.length === 0 ? (
              <p className="px-4 py-6 text-gray-400 text-sm text-center">No jobs yet.</p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="bg-gray-50 border-b border-gray-200 text-left">
                    <th className="px-4 py-2 font-semibold text-gray-600">Status</th>
                    <th className="px-4 py-2 font-semibold text-gray-600">Started</th>
                    <th className="px-4 py-2 font-semibold text-gray-600">Size</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {jobs.map((j) => (
                    <tr key={j.id} className="hover:bg-gray-50">
                      <td className="px-4 py-2"><StatusBadge status={j.status} /></td>
                      <td className="px-4 py-2 text-gray-500">{fmtDate(j.started_at)}</td>
                      <td className="px-4 py-2 text-gray-600">{fmtSize(j.bytes_scanned)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </section>

        <section>
          <h2 className="text-lg font-semibold text-gray-800 mb-3">Snapshots</h2>
          <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
            {snapshots.length === 0 ? (
              <p className="px-4 py-6 text-gray-400 text-sm text-center">No snapshots yet.</p>
            ) : (
              <table className="w-full text-sm">
                <thead>
                  <tr className="bg-gray-50 border-b border-gray-200 text-left">
                    <th className="px-4 py-2 font-semibold text-gray-600">Date</th>
                    <th className="px-4 py-2 font-semibold text-gray-600">Files</th>
                    <th className="px-4 py-2 font-semibold text-gray-600">Size</th>
                    <th className="px-4 py-2"></th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {snapshots.map((s) => (
                    <tr key={s.id} className="hover:bg-gray-50">
                      <td className="px-4 py-2 text-gray-700">{fmtDate(s.created_at)}</td>
                      <td className="px-4 py-2 text-gray-600">{s.file_count.toLocaleString()}</td>
                      <td className="px-4 py-2 text-gray-600">{fmtSize(s.total_size)}</td>
                      <td className="px-4 py-2">
                        <Link
                          href={`/devices/${id}/snapshots/${s.id}/browse`}
                          className="text-brand-600 hover:text-brand-700 font-medium"
                        >
                          Browse →
                        </Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </section>
      </div>
    </Shell>
  );
}
