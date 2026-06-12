import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Build straight into the Go embed directory so `make build` ships one binary.
export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "../internal/web/dist",
    emptyOutDir: true,
  },
  server: {
    // Dev proxy: forward API + subscription calls to the Go backend.
    proxy: {
      "/api": "http://localhost:8080",
      "/sub": "http://localhost:8080",
    },
  },
});
