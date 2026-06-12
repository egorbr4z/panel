import axios from "axios";
import { useAuth } from "../store/auth";

// Single axios instance. Base path is relative so it works behind any reverse
// proxy and in the dev proxy alike.
export const api = axios.create({ baseURL: "/api" });

// Attach the access token to every request.
api.interceptors.request.use((cfg) => {
  const token = useAuth.getState().accessToken;
  if (token) cfg.headers.Authorization = `Bearer ${token}`;
  return cfg;
});

// On 401, try a one-shot refresh, then retry; otherwise log out.
let refreshing: Promise<string | null> | null = null;

api.interceptors.response.use(
  (r) => r,
  async (error) => {
    const original = error.config;
    if (error.response?.status === 401 && !original._retry) {
      original._retry = true;
      if (!refreshing) refreshing = useAuth.getState().refresh();
      const newToken = await refreshing;
      refreshing = null;
      if (newToken) {
        original.headers.Authorization = `Bearer ${newToken}`;
        return api(original);
      }
      useAuth.getState().logout();
    }
    return Promise.reject(error);
  },
);

// Unwrap the {data, error, meta} envelope.
export async function get<T>(url: string): Promise<T> {
  const res = await api.get(url);
  return res.data.data as T;
}
