import { useState } from "react";
import { motion } from "framer-motion";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { get, post } from "../api/client";
import { CoreStatus } from "../api/types";

const stateColor: Record<string, string> = {
  running: "bg-green-500/20 text-green-300",
  starting: "bg-amber-500/20 text-amber-300",
  stopped: "bg-white/10 text-white/50",
  crashed: "bg-red-500/20 text-red-300",
};

export default function Cores() {
  const qc = useQueryClient();
  const [logsFor, setLogsFor] = useState<string | null>(null);

  const { data: cores } = useQuery({
    queryKey: ["cores"],
    queryFn: () => get<CoreStatus[]>("/cores"),
    refetchInterval: 4000,
  });

  const { data: logs } = useQuery({
    queryKey: ["core-logs", logsFor],
    queryFn: () => get<{ lines: string[] }>(`/cores/${logsFor}/logs`),
    enabled: !!logsFor,
    refetchInterval: 3000,
  });

  const restart = useMutation({
    mutationFn: (name: string) => post(`/cores/${name}/restart`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cores"] }),
  });

  return (
    <div>
      <h2 className="mb-6 text-xl font-semibold">Cores</h2>
      <div className="grid gap-4 sm:grid-cols-2">
        {cores?.map((core) => (
          <motion.div key={core.name} layout initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} className="rounded-xl bg-panel/70 p-5 ring-1 ring-white/10">
            <div className="flex items-center justify-between">
              <span className="text-lg font-medium">{core.name}</span>
              <span className={`rounded px-2 py-0.5 text-xs ${stateColor[core.state] ?? ""}`}>{core.state}</span>
            </div>
            <div className="mt-2 text-sm text-white/50">
              {core.inbounds} inbounds · {core.restarts} restarts
            </div>
            <div className="mt-4 flex gap-2">
              <button onClick={() => restart.mutate(core.name)} className="rounded-lg bg-white/10 px-3 py-1.5 text-sm hover:bg-white/20">
                Restart
              </button>
              <button onClick={() => setLogsFor(core.name)} className="rounded-lg bg-white/10 px-3 py-1.5 text-sm hover:bg-white/20">
                Logs
              </button>
            </div>
          </motion.div>
        ))}
      </div>

      {logsFor && (
        <motion.div initial={{ opacity: 0, y: 12 }} animate={{ opacity: 1, y: 0 }} className="mt-6 rounded-xl bg-black/50 p-4 ring-1 ring-white/10">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-sm font-medium">{logsFor} logs</span>
            <button onClick={() => setLogsFor(null)} className="text-sm text-white/40 hover:text-white">
              Close
            </button>
          </div>
          <pre className="max-h-80 overflow-auto whitespace-pre-wrap font-mono text-xs text-white/70">
            {logs?.lines?.join("\n") || "no output"}
          </pre>
        </motion.div>
      )}
    </div>
  );
}
