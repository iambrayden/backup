const API = process.env.NEXT_PUBLIC_API_URL ?? "";

async function request<T>(
  path: string,
  options: RequestInit = {},
  token?: string
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`${API}${path}`, { ...options, headers });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(body || `HTTP ${res.status}`);
  }
  return res.json();
}

export type User = {
  id: string;
  tenant_id: string;
  email: string;
  role: string;
  display_name: string;
};

export type Device = {
  id: string;
  tenant_id: string;
  hostname: string;
  os_type: string;
  arch: string;
  agent_version: string;
  status: string;
  last_seen_at: string | null;
  enrolled_at: string;
};

export type BackupJob = {
  id: string;
  device_id: string;
  tenant_id: string;
  status: string;
  bytes_scanned: number;
  chunks_total: number;
  chunks_uploaded: number;
  chunks_deduped: number;
  error_msg: string | null;
  started_at: string | null;
  finished_at: string | null;
  created_at: string;
};

export type Snapshot = {
  id: string;
  device_id: string;
  tenant_id: string;
  file_count: number;
  total_size: number;
  created_at: string;
};

export type FileEntry = {
  id: string;
  snapshot_id: string;
  path: string;
  parent_path: string;
  name: string;
  size: number;
  mtime: string | null;
  mode: number;
  is_dir: boolean;
};

export const api = {
  login(email: string, password: string) {
    return request<{ token: string; user: User }>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
  },

  listDevices(token: string) {
    return request<Device[]>("/api/v1/devices", {}, token);
  },

  getDevice(token: string, id: string) {
    return request<Device>(`/api/v1/devices/${id}`, {}, token);
  },

  triggerJob(token: string, deviceID: string) {
    return request<BackupJob>(`/api/v1/devices/${deviceID}/jobs`, { method: "POST", body: "{}" }, token);
  },

  listJobs(token: string, deviceID: string) {
    return request<BackupJob[]>(`/api/v1/devices/${deviceID}/jobs`, {}, token);
  },

  listSnapshots(token: string, deviceID: string) {
    return request<Snapshot[]>(`/api/v1/devices/${deviceID}/snapshots`, {}, token);
  },

  fileTree(token: string, snapshotID: string, path = "/") {
    return request<{ snapshot: Snapshot; path: string; entries: FileEntry[] }>(
      `/api/v1/snapshots/${snapshotID}/tree?path=${encodeURIComponent(path)}`,
      {},
      token
    );
  },

  downloadURL(snapshotID: string, path: string, token: string) {
    return `/api/v1/snapshots/${snapshotID}/download?path=${encodeURIComponent(path)}&token=${token}`;
  },

  createEnrollmentToken(token: string, label: string) {
    return request<{ token: string; expires_at: string }>("/api/v1/enrollment-tokens", {
      method: "POST",
      body: JSON.stringify({ label }),
    }, token);
  },
};
