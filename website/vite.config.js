import { defineConfig } from "vite";
import { fileURLToPath } from "node:url";

const repositoryRoot = fileURLToPath(new URL("../", import.meta.url));

export default defineConfig({
  // './' makes every generated URL work from a GitHub Pages project subfolder.
  base: "./",
  server: {
    // Shared frontend modules live at the repository root so desktop and web
    // consume the exact same source files during development and production builds.
    fs: { allow: [repositoryRoot] },
  },
  build: {
    target: "es2022",
    sourcemap: true,
  },
});
