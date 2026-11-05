import { loadEnv } from "vite";
import path from "node:path";
import { existsSync } from "node:fs";

const envPath = path.resolve(process.cwd(), "../../.env");

if (!existsSync(envPath)) {
    throw new Error(`Environment file not found: ${envPath}`);
}

const env = loadEnv("development", path.dirname(envPath), "");

if (!env.BACKEND_PORT) {
    throw new Error("BACKEND_PORT is required in .env");
}

export const BACKEND_PORT = env.BACKEND_PORT;
