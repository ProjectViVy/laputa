import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    host: "127.0.0.1",
    port: 5173,
    proxy: {
      "/v1": { target: "http://127.0.0.1:7373", changeOrigin: false },
      "/v2": { target: "http://127.0.0.1:7373", changeOrigin: false },
    },
  },
  build: { outDir: "dist", sourcemap: true },
});
