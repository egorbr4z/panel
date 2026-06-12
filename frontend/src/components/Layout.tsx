import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { motion, AnimatePresence } from "framer-motion";
import { useAuth } from "../store/auth";

const nav = [
  { to: "/", label: "Dashboard" },
  { to: "/users", label: "Users" },
  { to: "/inbounds", label: "Inbounds" },
  { to: "/cores", label: "Cores" },
  { to: "/settings", label: "Settings" },
];

export default function Layout() {
  const location = useLocation();
  const navigate = useNavigate();
  const logout = useAuth((s) => s.logout);

  return (
    <div className="flex min-h-screen">
      <aside className="w-56 shrink-0 border-r border-white/10 bg-panel/40 p-4">
        <div className="mb-8 px-2 text-lg font-semibold tracking-tight">vpanel</div>
        <nav className="space-y-1">
          {nav.map((n) => {
            const active = location.pathname === n.to;
            return (
              <Link
                key={n.to}
                to={n.to}
                className={`relative block rounded-lg px-3 py-2 text-sm transition ${
                  active ? "text-white" : "text-white/55 hover:text-white"
                }`}
              >
                {active && (
                  <motion.span
                    layoutId="nav-active"
                    className="absolute inset-0 -z-10 rounded-lg bg-accent/20 ring-1 ring-accent/40"
                    transition={{ type: "spring", stiffness: 350, damping: 30 }}
                  />
                )}
                {n.label}
              </Link>
            );
          })}
        </nav>
        <button
          onClick={() => {
            logout();
            navigate("/login");
          }}
          className="mt-8 w-full rounded-lg px-3 py-2 text-left text-sm text-white/55 hover:text-red-400"
        >
          Sign out
        </button>
      </aside>

      <main className="flex-1 p-8">
        <AnimatePresence mode="wait">
          <motion.div
            key={location.pathname}
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            transition={{ duration: 0.2 }}
          >
            <Outlet />
          </motion.div>
        </AnimatePresence>
      </main>
    </div>
  );
}
