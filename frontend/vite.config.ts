import { defineConfig } from "vite";
import { fileURLToPath } from "node:url";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import wails from "@wailsio/runtime/plugins/vite";

const mock = (file: string) => fileURLToPath(new URL(`./e2e/mock/${file}`, import.meta.url));

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  // --mode e2e: the frontend runs in a normal browser, with a fake backend instead of Wails
  const e2e = mode === "e2e";
  return {
    server: {
      host: "127.0.0.1",
      port: e2e ? 5179 : Number(process.env.WAILS_VITE_PORT) || 9245,
      strictPort: true,
    },
    resolve: e2e
      ? {
          alias: [
            { find: /^.*\/bindings\/github\.com\/jacopofilonzi\/project-library\/internal\/core\/library\.js$/, replacement: mock("library.ts") },
            { find: /^@wailsio\/runtime$/, replacement: mock("runtime.ts") },
          ],
        }
      : undefined,
    plugins: e2e ? [svelte()] : [svelte(), wails("./bindings")],
  };
});
