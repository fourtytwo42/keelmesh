import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.ts", "src/**/*.test.tsx"],
  },
  build: {
    // Codex's embedded browser can lag desktop Chrome. Keep the appliance
    // bundle within a conservative WebView baseline instead of Vite's
    // rolling "widely available" target.
    target: "es2020",
    cssTarget: "chrome90",
    outDir: "dist",
    emptyOutDir: true,
  },
  esbuild: {
    target: "es2020",
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8080",
      "/healthz": "http://localhost:8080",
      "/readyz": "http://localhost:8080",
    },
  },
});
