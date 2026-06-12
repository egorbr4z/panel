// Placeholder pages for sections delivered in later phases (inbounds, users,
// cores, settings). Keeps routing/animations wired so each phase just swaps in
// real content.
export default function Placeholder({ title }: { title: string }) {
  return (
    <div>
      <h2 className="mb-4 text-xl font-semibold">{title}</h2>
      <div className="rounded-xl bg-panel/50 p-8 text-center text-white/40 ring-1 ring-white/10">
        Coming in a later phase.
      </div>
    </div>
  );
}
