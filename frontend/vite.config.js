import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// In dev, calls to /api are forwarded to the Go backend (no CORS headaches).
export default defineConfig({
  plugins: [react()],
  server: { port: 5173, proxy: { "/api": "http://localhost:8080" } },
});
