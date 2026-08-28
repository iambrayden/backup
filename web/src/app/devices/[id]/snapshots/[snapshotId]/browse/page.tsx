"use client";
import { useEffect, useState } from "react";
import { useRouter, useParams, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Shell } from "@/components/Shell";
import { api, type FileEntry, type Snapshot } from "@/lib/api";
import { getToken, isAuthenticated } from "@/lib/auth";

function fmtSize(bytes: number) {
  if (bytes === 0) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
  return `${(bytes / 1024 ** 3).toFixed(2)} GB`;
}

function parentOf(path: string) {
  if (path === "/") return null;
  const trimmed = path.replace(/\/$/, "");
  const idx = trimmed.lastIndexOf("/");
  if (idx <= 0) return "/";
  return trimmed.slice(0, idx + 1);
}

export default function BrowsePage() {
  const router = useRouter();
  const params = useParams<{ id: string; snapshotId: string }>();
  const searchParams = useSearchParams();
  const currentPath = searchParams.get("path") ?? "/";

  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [entries, setEntries] = useState<FileEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  function token() { return getToken()!; }

  async function load(path: string) {
    setLoading(true);
    setError("");
    try {
      const res = await api.fileTree(token(), params.snapshotId, path);
      setSnapshot(res.snapshot);
      setEntries(res.entries ?? []);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : "Failed");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (!isAuthenticated()) { router.replace("/login"); return; }
    load(currentPath);
  }, [params.snapshotId, currentPath, router]);

  function navigate(path: string) {
    router.push(`/devices/${params.id}/snapshots/${params.snapshotId}/browse?path=${encodeURIComponent(path)}`);
  }

  function downloadURL(path: string) {
    return `/api/v1/snapshots/${params.snapshotId}/download?path=${encodeURIComponent(path)}`;
  }

  const up = parentOf(currentPath);

  return (
    <Shell>
      <div className="mb-4">
        <Link href={`/devices/${params.id}`} className="text-brand-600 hover:underline text-sm">
          ← Device
        </Link>
        {snapshot && (
          <p className="text-gray-500 text-xs mt-1">
            Snapshot {new Date(snapshot.created_at).toLocaleString()} · {snapshot.file_count.toLocaleString()} files
          </p>
        )}
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <div className="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center gap-2">
          <span className="text-gray-400">📁</span>
          <span className="font-mono text-sm text-gray-700 flex-1">{currentPath}</span>
        </div>

        {loading && <p className="px-4 py-6 text-gray-400 text-sm text-center">Loading…</p>}
        {error && <p className="px-4 py-4 text-red-600 text-sm">{error}</p>}

        {!loading && !error && (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-100 text-left">
                <th className="px-4 py-2 font-semibold text-gray-500">Name</th>
                <th className="px-4 py-2 font-semibold text-gray-500">Size</th>
                <th className="px-4 py-2 font-semibold text-gray-500">Modified</th>
                <th className="px-4 py-2"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {up !== null && (
                <tr
                  className="hover:bg-gray-50 cursor-pointer"
                  onClick={() => navigate(up)}
                >
                  <td className="px-4 py-2.5 text-gray-600" colSpan={4}>
                    <span className="mr-2">📁</span>..
                  </td>
                </tr>
              )}
              {entries.length === 0 && (
                <tr>
                  <td colSpan={4} className="px-4 py-6 text-gray-400 text-center">Empty directory</td>
                </tr>
              )}
              {entries.map((e) => (
                <tr key={e.id} className="hover:bg-gray-50">
                  <td className="px-4 py-2.5">
                    {e.is_dir ? (
                      <button
                        onClick={() => navigate(e.path)}
                        className="flex items-center gap-2 text-gray-800 hover:text-brand-600 transition-colors"
                      >
                        <span>📁</span>
                        <span className="font-medium">{e.name}</span>
                      </button>
                    ) : (
                      <span className="flex items-center gap-2 text-gray-700">
                        <span>📄</span>
                        {e.name}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 text-gray-500">{fmtSize(e.size)}</td>
                  <td className="px-4 py-2.5 text-gray-500">
                    {e.mtime ? new Date(e.mtime).toLocaleDateString() : "—"}
                  </td>
                  <td className="px-4 py-2.5 text-right">
                    {!e.is_dir && (
                      <a
                        href={downloadURL(e.path)}
                        download={e.name}
                        className="text-brand-600 hover:text-brand-700 text-xs font-medium"
                        onClick={(ev) => {
                          ev.preventDefault();
                          const t = token();
                          const url = `/api/v1/snapshots/${params.snapshotId}/download?path=${encodeURIComponent(e.path)}`;
                          fetch(url, { headers: { Authorization: `Bearer ${t}` } })
                            .then((r) => r.blob())
                            .then((blob) => {
                              const a = document.createElement("a");
                              a.href = URL.createObjectURL(blob);
                              a.download = e.name;
                              a.click();
                            });
                        }}
                      >
                        ↓ Download
                      </a>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </Shell>
  );
}
