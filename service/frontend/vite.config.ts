import { resolve } from "path";
import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [
    tailwindcss(),
  ],
  esbuild: {
    jsx: "automatic",
    jsxImportSource: "preact",
  },
  // Dev server proxies all non-static requests to the Go backend
  server: {
    port: 5173,
    proxy: {
      // Proxy everything except /static/dist (which Vite serves) to Go backend
      "/api": "http://localhost:8080",
      "/boards": "http://localhost:8080",
      "/community": "http://localhost:8080",
      "/editor": "http://localhost:8080",
      "/partials": "http://localhost:8080",
      "/health": "http://localhost:8080",
      "/landing": "http://localhost:8080",
      "/static/leds.json": "http://localhost:8080",
      "/static/led_map.json": "http://localhost:8080",
    },
  },
  build: {
    outDir: resolve(__dirname, "../static/dist"),
    emptyOutDir: true,
    rollupOptions: {
      input: {
        "app-shell": resolve(__dirname, "src/entries/app-shell.ts"),
        "board-page": resolve(__dirname, "src/entries/board-page.ts"),
        "landing": resolve(__dirname, "src/entries/landing.ts"),
        "editor": resolve(__dirname, "src/entries/editor.tsx"),
        main: resolve(__dirname, "styles/main.css"),
      },
      output: {
        entryFileNames: "[name].js",
        chunkFileNames: "[name].js",
        assetFileNames: "[name].[ext]",
        manualChunks: {
          three: ["three"],
        },
      },
    },
  },
});
