import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { BACKEND_PORT } from "./src/util/env.ts";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(() => {
    return {
        plugins: [react(), tailwindcss()],
        server: {
            proxy: {
                "/api": {
                    target: `http://localhost:${BACKEND_PORT}`,
                    changeOrigin: true,
                },
            },
        },
    };
});
