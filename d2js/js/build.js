import { build } from "bun";
import { copyFile, mkdir } from "node:fs/promises";
import { join } from "node:path";

await mkdir("./dist/esm", { recursive: true });
await mkdir("./dist/cjs", { recursive: true });
await mkdir("./dist/browser", { recursive: true });

const commonConfig = {
  splitting: false,
  sourcemap: "external",
  minify: true,
  naming: {
    entry: "[dir]/[name].js",
    chunk: "[name]-[hash].js",
    asset: "[name]-[hash][ext]",
  },
};

await build({
  ...commonConfig,
  target: "node",
  entrypoints: ["./src/index.js", "./src/worker.js", "./src/platform.js"],
  outdir: "./dist/esm",
  format: "esm",
});

await build({
  ...commonConfig,
  target: "node",
  entrypoints: ["./src/index.js", "./src/worker.js", "./src/platform.js"],
  outdir: "./dist/cjs",
  format: "cjs",
});

await build({
  ...commonConfig,
  target: "browser",
  entrypoints: ["./src/index.js", "./src/worker.js", "./src/platform.js"],
  outdir: "./dist/browser",
  format: "esm",
});

const dirs = ['esm', 'cjs', 'browser'];
for (const dir of dirs) {
  await copyFile("./wasm/d2.wasm", join(`./dist/${dir}`, "d2.wasm"));
  await copyFile("./wasm/wasm_exec.js", join(`./dist/${dir}`, "wasm_exec.js"));
}
