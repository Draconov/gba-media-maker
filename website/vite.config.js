import { defineConfig } from "vite";

export default defineConfig({
  // './' makes every generated URL work from a GitHub Pages project subfolder.
  base: "./",
  build: {
    target: "es2022",
    sourcemap: true,
  },
});
