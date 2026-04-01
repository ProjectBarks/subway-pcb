import { resolve } from "path";
import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

const isWatch = process.argv.includes("--watch");

export default defineConfig({
  plugins: [
    tailwindcss(),
  ],
  esbuild: {
    jsx: "automatic",
    jsxImportSource: "preact",
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8080",
      "/boards": "http://localhost:8080",
      "/community": "http://localhost:8080",
      "/editor": "http://localhost:8080",
      "/partials": "http://localhost:8080",
      "/health": "http://localhost:8080",
      "/landing": "http://localhost:8080",
    },
  },
  build: {
    outDir: resolve(__dirname, "static/dist"),
    emptyOutDir: true,
    manifest: true,
    // In watch mode, ignore Go-generated files to prevent rebuild loops with air
    watch: isWatch ? {
      chokidar: {
        ignored: ["**/*.go", "**/static/dist/**"],
      },
    } : undefined,
    rollupOptions: {
      input: {
        "app-shell": resolve(__dirname, "ui/components/layout/app-shell.ts"),
        "board-page": resolve(__dirname, "ui/board/board-page.ts"),
        "landing": resolve(__dirname, "ui/landing/landing.ts"),
        "community": resolve(__dirname, "ui/community/community.ts"),
        "dashboard": resolve(__dirname, "ui/dashboard/dashboard.ts"),
        "editor": resolve(__dirname, "ui/editor/editor.tsx"),
        main: resolve(__dirname, "ui/components/layout/global.css"),
      },
      output: {
        entryFileNames: isWatch ? "[name].js" : "[name]-[hash].js",
        chunkFileNames: isWatch ? "[name].js" : "[name]-[hash].js",
        assetFileNames: isWatch ? "[name].[ext]" : "[name]-[hash].[ext]",
        manualChunks: {
          htmx: ["htmx.org"],
          alpine: ["alpinejs", "@alpinejs/intersect"],
          three: ["three"],
          wasmoon: ["wasmoon"],
        },
      },
    },
  },
});
