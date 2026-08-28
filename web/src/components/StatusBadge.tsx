const colors: Record<string, string> = {
  active: "bg-green-100 text-green-800",
  completed: "bg-green-100 text-green-800",
  running: "bg-blue-100 text-blue-800",
  pending: "bg-yellow-100 text-yellow-800",
  failed: "bg-red-100 text-red-800",
  inactive: "bg-gray-100 text-gray-600",
  cancelled: "bg-gray-100 text-gray-600",
  lost: "bg-orange-100 text-orange-800",
};

export function StatusBadge({ status }: { status: string }) {
  const cls = colors[status] ?? "bg-gray-100 text-gray-600";
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${cls}`}>
      {status}
    </span>
  );
}
