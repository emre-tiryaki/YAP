import type { TranslationKey } from "../hooks/useTranslate";
import type { AiPreset, AiProvider } from "../types/ai";

type PresetInfo = {
    provider: AiProvider;
    baseUrl: string;
    keyRequired: boolean;
    labelKey: TranslationKey;
    // Where to create a key; absent for local servers.
    keyUrl?: string;
    local: boolean;
};

// Mirrors the presets of the backend (service/ai_connection.go).
export const aiPresets: Record<AiPreset, PresetInfo> = {
    ollama: {
        provider: "openai_compatible",
        baseUrl: "http://localhost:11434/v1",
        keyRequired: false,
        labelKey: "ai.preset.ollama",
        local: true,
    },
    lmstudio: {
        provider: "openai_compatible",
        baseUrl: "http://localhost:1234/v1",
        keyRequired: false,
        labelKey: "ai.preset.lmstudio",
        local: true,
    },
    llamacpp: {
        provider: "openai_compatible",
        baseUrl: "http://localhost:8081/v1",
        keyRequired: false,
        labelKey: "ai.preset.llamacpp",
        local: true,
    },
    custom: {
        provider: "openai_compatible",
        baseUrl: "",
        keyRequired: false,
        labelKey: "ai.preset.custom",
        local: false,
    },
    openai: {
        provider: "openai_compatible",
        baseUrl: "https://api.openai.com/v1",
        keyRequired: true,
        labelKey: "ai.preset.openai",
        keyUrl: "https://platform.openai.com/api-keys",
        local: false,
    },
    openrouter: {
        provider: "openai_compatible",
        baseUrl: "https://openrouter.ai/api/v1",
        keyRequired: true,
        labelKey: "ai.preset.openrouter",
        keyUrl: "https://openrouter.ai/keys",
        local: false,
    },
    gemini: {
        provider: "openai_compatible",
        baseUrl: "https://generativelanguage.googleapis.com/v1beta/openai/",
        keyRequired: true,
        labelKey: "ai.preset.gemini",
        keyUrl: "https://aistudio.google.com/apikey",
        local: false,
    },
    groq: {
        provider: "openai_compatible",
        baseUrl: "https://api.groq.com/openai/v1",
        keyRequired: true,
        labelKey: "ai.preset.groq",
        keyUrl: "https://console.groq.com/keys",
        local: false,
    },
    mistral: {
        provider: "openai_compatible",
        baseUrl: "https://api.mistral.ai/v1",
        keyRequired: true,
        labelKey: "ai.preset.mistral",
        keyUrl: "https://console.mistral.ai/api-keys",
        local: false,
    },
    anthropic: {
        provider: "anthropic",
        baseUrl: "https://api.anthropic.com",
        keyRequired: true,
        labelKey: "ai.preset.anthropic",
        keyUrl: "https://console.anthropic.com/settings/keys",
        local: false,
    },
};

export const aiPresetOrder: AiPreset[] = [
    "ollama",
    "lmstudio",
    "llamacpp",
    "openai",
    "anthropic",
    "gemini",
    "openrouter",
    "groq",
    "mistral",
    "custom",
];
