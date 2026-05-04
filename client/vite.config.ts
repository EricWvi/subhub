import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { default as _monacoEditorPlugin } from "vite-plugin-monaco-editor";

const monacoEditorPlugin = (_monacoEditorPlugin as any).default || _monacoEditorPlugin;

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    monacoEditorPlugin({
      languageWorkers: ["editorWorkerService", "typescript"],
    }),
  ],
  server: {
    host: "0.0.0.0",
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
