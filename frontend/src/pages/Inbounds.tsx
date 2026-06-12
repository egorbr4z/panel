import { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { get, post, del } from "../api/client";
import { Inbound, PROTOCOLS } from "../api/types";

const blank = {
  tag: "",
  core: "xray" as "xray" | "singbox",
  protocol: "vless",
  port: 443,
  listen: "0.0.0.0",
  settings: "{\n  \"flow\": \"xtls-rprx-vision\",\n  \"decryption\": \"none\"\n}",
  stream:
    '{\n  "network": "tcp",\n  "security": "reality",\n  "realitySettings": {\n    "dest": "www.google.com:443",\n    "serverNames": ["www.google.com"],\n    "privateKey": "",\n    "shortIds": [""]\n  }\n}',
};

export default function Inbounds() {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState(blank);
  const [error, setError] = useState("");

  const { data: inbounds } = useQuery({
    queryKey: ["inbounds"],
    queryFn: () => get<Inbound[]>("/inbounds"),
  });

  const create = useMutation({
    mutationFn: async () => {
      let settings: unknown, stream: unknown;
      try {
        settings = form.settings ? JSON.parse(form.settings) : {};
        stream = form.stream ? JSON.parse(form.stream) : {};
      } catch {
        throw new Error("settings/stream must be valid JSON");
      }
      return post("/inbounds", {
        tag: form.tag,
        core: form.core,
        protocol: form.protocol,
        port: Number(form.port),
        listen: form.listen,
        settings,
        stream,
      });
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["inbounds"] });
      setOpen(false);
      setForm(blank);
      setError("");
    },
    onError: (e: Error) => setError(e.message),
  });

  const remove = useMutation({
    mutationFn: (id: number) => del(`/inbounds/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["inbounds"] }),
  });

  const genReality = async () => {
    const kp = await post<{ private_key: string; public_key: string; short_id: string }>(
      "/inbounds/reality-keys",
    );
    try {
      const s = JSON.parse(form.stream);
      s.realitySettings = {
        ...(s.realitySettings || {}),
        privateKey: kp.private_key,
        shortIds: [kp.short_id],
      };
      setForm({ ...form, stream: JSON.stringify(s, null, 2) });
      alert(`Public key (give to clients):\n${kp.public_key}`);
    } catch {
      setError("fix stream JSON before generating keys");
    }
  };

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-xl font-semibold">Inbounds</h2>
        <button
          onClick={() => setOpen(true)}
          className="rounded-lg bg-accent px-4 py-2 text-sm font-medium hover:brightness-110"
        >
          + New inbound
        </button>
      </div>

      <motion.div layout className="space-y-2">
        {inbounds?.map((ib) => (
          <motion.div
            key={ib.id}
            layout
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            className="flex items-center justify-between rounded-xl bg-panel/70 p-4 ring-1 ring-white/10"
          >
            <div>
              <span className="font-medium">{ib.tag}</span>
              <span className="ml-3 rounded bg-white/10 px-2 py-0.5 text-xs uppercase">{ib.core}</span>
              <span className="ml-2 text-sm text-white/60">
                {ib.protocol} · :{ib.port}
              </span>
            </div>
            <button
              onClick={() => remove.mutate(ib.id)}
              className="text-sm text-white/40 hover:text-red-400"
            >
              Delete
            </button>
          </motion.div>
        ))}
        {inbounds?.length === 0 && (
          <div className="rounded-xl bg-panel/40 p-8 text-center text-white/40">No inbounds yet.</div>
        )}
      </motion.div>

      <AnimatePresence>
        {open && (
          <motion.div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={() => setOpen(false)}
          >
            <motion.div
              onClick={(e) => e.stopPropagation()}
              initial={{ scale: 0.95, y: 20 }}
              animate={{ scale: 1, y: 0 }}
              exit={{ scale: 0.95, y: 20 }}
              className="max-h-[90vh] w-full max-w-lg overflow-auto rounded-2xl bg-panel p-6 ring-1 ring-white/10"
            >
              <h3 className="mb-4 text-lg font-semibold">New inbound</h3>
              <div className="grid grid-cols-2 gap-3">
                <Field label="Tag">
                  <input className="inp" value={form.tag} onChange={(e) => setForm({ ...form, tag: e.target.value })} />
                </Field>
                <Field label="Port">
                  <input className="inp" type="number" value={form.port} onChange={(e) => setForm({ ...form, port: Number(e.target.value) })} />
                </Field>
                <Field label="Core">
                  <select
                    className="inp"
                    value={form.core}
                    onChange={(e) => {
                      const core = e.target.value as "xray" | "singbox";
                      setForm({ ...form, core, protocol: PROTOCOLS[core][0] });
                    }}
                  >
                    <option value="xray">xray</option>
                    <option value="singbox">sing-box</option>
                  </select>
                </Field>
                <Field label="Protocol">
                  <select className="inp" value={form.protocol} onChange={(e) => setForm({ ...form, protocol: e.target.value })}>
                    {PROTOCOLS[form.core].map((p) => (
                      <option key={p} value={p}>
                        {p}
                      </option>
                    ))}
                  </select>
                </Field>
              </div>

              <Field label="Settings (JSON)">
                <textarea className="inp h-28 font-mono text-xs" value={form.settings} onChange={(e) => setForm({ ...form, settings: e.target.value })} />
              </Field>
              <Field label="Stream / transport (JSON)">
                <textarea className="inp h-40 font-mono text-xs" value={form.stream} onChange={(e) => setForm({ ...form, stream: e.target.value })} />
              </Field>

              {error && <p className="mt-2 text-sm text-red-400">{error}</p>}

              <div className="mt-4 flex items-center justify-between">
                <button onClick={genReality} className="text-sm text-accent hover:underline">
                  Generate Reality keys
                </button>
                <div className="flex gap-2">
                  <button onClick={() => setOpen(false)} className="rounded-lg px-4 py-2 text-sm text-white/60 hover:text-white">
                    Cancel
                  </button>
                  <button onClick={() => create.mutate()} className="rounded-lg bg-accent px-4 py-2 text-sm font-medium hover:brightness-110">
                    Create
                  </button>
                </div>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="mb-3 block">
      <span className="mb-1 block text-xs text-white/60">{label}</span>
      {children}
    </label>
  );
}
