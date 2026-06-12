import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { get } from "../api/client";

interface SystemStats {
  version: string;
  uptime: number;
  goroutines: number;
  heap_alloc: number;
  sys_mem: number;
  num_cpu: number;
}

const card = {
  hidden: { opacity: 0, y: 16 },
  show: { opacity: 1, y: 0 },
};

function fmtBytes(n: number) {
  if (n < 1024) return `${n} B`;
  const u = ["KB", "MB", "GB"];
  let i = -1;
  do {
    n /= 1024;
    i++;
  } while (n >= 1024 && i < u.length - 1);
  return `${n.toFixed(1)} ${u[i]}`;
}

export default function Dashboard() {
  const { data, isLoading } = useQuery({
    queryKey: ["system"],
    queryFn: () => get<SystemStats>("/system"),
    refetchInterval: 5000,
  });

  const stats = [
    { label: "Version", value: data?.version ?? "—" },
    { label: "Uptime", value: data ? `${data.uptime}s` : "—" },
    { label: "Goroutines", value: data?.goroutines ?? "—" },
    { label: "Heap", value: data ? fmtBytes(data.heap_alloc) : "—" },
    { label: "Sys mem", value: data ? fmtBytes(data.sys_mem) : "—" },
    { label: "CPUs", value: data?.num_cpu ?? "—" },
  ];

  return (
    <div>
      <h2 className="mb-6 text-xl font-semibold">Dashboard</h2>
      <motion.div
        variants={{ show: { transition: { staggerChildren: 0.05 } } }}
        initial="hidden"
        animate="show"
        className="grid grid-cols-2 gap-4 sm:grid-cols-3"
      >
        {stats.map((s) => (
          <motion.div
            key={s.label}
            variants={card}
            className="rounded-xl bg-panel/70 p-4 ring-1 ring-white/10"
          >
            <div className="text-xs uppercase tracking-wide text-white/40">{s.label}</div>
            <div className="mt-1 text-2xl font-semibold">{isLoading ? "…" : s.value}</div>
          </motion.div>
        ))}
      </motion.div>
    </div>
  );
}
