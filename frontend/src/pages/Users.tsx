import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { get, post } from "../api/client";
import { Inbound, User } from "../api/types";

function fmtBytes(n: number) {
  if (!n) return "0";
  const u = ["B", "KB", "MB", "GB", "TB"];
  let i = 0;
  while (n >= 1024 && i < u.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(1)} ${u[i]}`;
}

const statusColor: Record<string, string> = {
  active: "bg-green-500/20 text-green-300",
  disabled: "bg-white/10 text-white/50",
  limited: "bg-amber-500/20 text-amber-300",
  expired: "bg-red-500/20 text-red-300",
};

export default function Users() {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [username, setUsername] = useState("");
  const [dataLimitGB, setDataLimitGB] = useState(0);
  const [selected, setSelected] = useState<number[]>([]);
  const [created, setCreated] = useState<User | null>(null);

  const { data: users } = useQuery({ queryKey: ["users"], queryFn: () => get<User[]>("/users") });
  const { data: inbounds } = useQuery({ queryKey: ["inbounds"], queryFn: () => get<Inbound[]>("/inbounds") });

  const create = useMutation({
    mutationFn: () =>
      post<User>("/users", {
        username,
        data_limit: dataLimitGB * 1024 * 1024 * 1024,
        inbound_ids: selected,
      }),
    onSuccess: (u) => {
      qc.invalidateQueries({ queryKey: ["users"] });
      setCreated(u);
      setOpen(false);
      setUsername("");
      setDataLimitGB(0);
      setSelected([]);
    },
  });

  const toggleEnabled = useMutation({
    mutationFn: ({ id, enable }: { id: number; enable: boolean }) =>
      post(`/users/${id}/${enable ? "enable" : "disable"}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["users"] }),
  });

  const subUrl = (u: User) => `${window.location.origin}/sub/${u.subscription_token}`;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-xl font-semibold">Users</h2>
        <button onClick={() => setOpen(true)} className="rounded-lg bg-accent px-4 py-2 text-sm font-medium hover:brightness-110">
          + New user
        </button>
      </div>

      <motion.div layout className="space-y-2">
        {users?.map((u) => (
          <motion.div
            key={u.id}
            layout
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            className="flex items-center justify-between rounded-xl bg-panel/70 p-4 ring-1 ring-white/10"
          >
            <div>
              <span className="font-medium">{u.username}</span>
              <span className={`ml-3 rounded px-2 py-0.5 text-xs ${statusColor[u.status] ?? ""}`}>{u.status}</span>
              <div className="mt-1 text-xs text-white/50">
                {fmtBytes(u.used_up + u.used_down)}
                {u.data_limit > 0 && ` / ${fmtBytes(u.data_limit)}`} · {u.inbounds?.length ?? 0} inbounds
              </div>
            </div>
            <div className="flex items-center gap-3 text-sm">
              <button
                onClick={() => navigator.clipboard.writeText(subUrl(u))}
                className="text-accent hover:underline"
              >
                Copy sub
              </button>
              <button
                onClick={() => toggleEnabled.mutate({ id: u.id, enable: u.status !== "active" })}
                className="text-white/50 hover:text-white"
              >
                {u.status === "active" ? "Disable" : "Enable"}
              </button>
            </div>
          </motion.div>
        ))}
        {users?.length === 0 && <div className="rounded-xl bg-panel/40 p-8 text-center text-white/40">No users yet.</div>}
      </motion.div>

      {created && (
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="mt-4 rounded-xl bg-green-500/10 p-4 text-sm ring-1 ring-green-500/30">
          Created <b>{created.username}</b>. Subscription link:
          <code className="ml-2 break-all text-accent">{subUrl(created)}</code>
        </motion.div>
      )}

      <AnimatePresence>
        {open && (
          <motion.div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} onClick={() => setOpen(false)}>
            <motion.div onClick={(e) => e.stopPropagation()} initial={{ scale: 0.95, y: 20 }} animate={{ scale: 1, y: 0 }} exit={{ scale: 0.95, y: 20 }} className="w-full max-w-md rounded-2xl bg-panel p-6 ring-1 ring-white/10">
              <h3 className="mb-4 text-lg font-semibold">New user</h3>
              <label className="mb-3 block">
                <span className="mb-1 block text-xs text-white/60">Username</span>
                <input className="inp" value={username} onChange={(e) => setUsername(e.target.value)} autoFocus />
              </label>
              <label className="mb-3 block">
                <span className="mb-1 block text-xs text-white/60">Data limit (GB, 0 = unlimited)</span>
                <input className="inp" type="number" value={dataLimitGB} onChange={(e) => setDataLimitGB(Number(e.target.value))} />
              </label>
              <div className="mb-4">
                <span className="mb-1 block text-xs text-white/60">Inbounds</span>
                <div className="max-h-40 space-y-1 overflow-auto rounded-lg bg-black/20 p-2">
                  {inbounds?.map((ib) => (
                    <label key={ib.id} className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 text-sm hover:bg-white/5">
                      <input
                        type="checkbox"
                        checked={selected.includes(ib.id)}
                        onChange={(e) =>
                          setSelected(e.target.checked ? [...selected, ib.id] : selected.filter((x) => x !== ib.id))
                        }
                      />
                      {ib.tag} <span className="text-white/40">({ib.protocol})</span>
                    </label>
                  ))}
                  {inbounds?.length === 0 && <div className="px-2 text-xs text-white/40">Create an inbound first.</div>}
                </div>
              </div>
              <div className="flex justify-end gap-2">
                <button onClick={() => setOpen(false)} className="rounded-lg px-4 py-2 text-sm text-white/60 hover:text-white">
                  Cancel
                </button>
                <button onClick={() => create.mutate()} className="rounded-lg bg-accent px-4 py-2 text-sm font-medium hover:brightness-110">
                  Create
                </button>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
