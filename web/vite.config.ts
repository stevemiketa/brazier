import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react({ jsxRuntime: "classic" })],
  server: {
    proxy: {
      // Proxy gRPC-web requests to the master service during development.
      "/brazier.BrazierAPI": {
        target: "http://localhost:9000",
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: "./src/test-setup.ts",
    includeSource: ["src/**/*.{ts,tsx}"],
  },
});
