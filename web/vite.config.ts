import { defineConfig } from "vite";
import path from "node:path";

export default defineConfig({
  base: "/assets/",
  build: {
    outDir: path.resolve(__dirname, "dist"),
    emptyOutDir: true,
    manifest: "manifest.json",
    assetsDir: ".",
    rollupOptions: {
      input: path.resolve(__dirname, "index.html"),
    },
  },
});
