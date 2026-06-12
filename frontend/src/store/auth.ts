import { create } from "zustand";
import { persist } from "zustand/middleware";
import axios from "axios";

interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  login: (username: string, password: string) => Promise<void>;
  refresh: () => Promise<string | null>;
  logout: () => void;
}

// Auth/session state persisted to localStorage so reloads stay logged in.
export const useAuth = create<AuthState>()(
  persist(
    (set, get) => ({
      accessToken: null,
      refreshToken: null,

      login: async (username, password) => {
        const res = await axios.post("/api/auth/login", { username, password });
        const d = res.data.data;
        set({ accessToken: d.access_token, refreshToken: d.refresh_token });
      },

      refresh: async () => {
        const rt = get().refreshToken;
        if (!rt) return null;
        try {
          const res = await axios.post("/api/auth/refresh", { refresh_token: rt });
          const d = res.data.data;
          set({ accessToken: d.access_token, refreshToken: d.refresh_token });
          return d.access_token as string;
        } catch {
          set({ accessToken: null, refreshToken: null });
          return null;
        }
      },

      logout: () => set({ accessToken: null, refreshToken: null }),
    }),
    { name: "vpanel-auth" },
  ),
);
