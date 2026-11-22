import type { TranslationKey } from "../hooks/useTranslate";
import type { AiPreset, AiProvider } from "../types/ai";

type PresetInfo = {
    provider: AiProvider;
    baseUrl: string;
    keyRequired: boolean;
    labelKey: TranslationKey;
};

// Mirrors the presets of the backend (service/ai_connection.go).
export const aiPresets: Record<AiPreset, PresetInfo> = {
    ollama: {
        provider: "openai_compatible",
        baseUrl: "http://localhost:11434/v1",
        keyRequired: false,
        labelKey: "ai.preset.ollama",
    },
    lmstudio: {
        provider: "openai_compatible",
        baseUrl: "http://localhost:1234/v1",
        keyRequired: false,
        labelKey: "ai.preset.lmstudio",
    },
    llamacpp: {
        provider: "openai_compatible",
        baseUrl: "http://localhost:8081/v1",
        keyRequired: false,
        labelKey: "ai.preset.llamacpp",
    },
    custom: {
        provider: "openai_compatible",
        baseUrl: "",
        keyRequired: false,
        labelKey: "ai.preset.custom",
    },
    openai: {
        provider: "openai_compatible",
        baseUrl: "https://api.openai.com/v1",
        keyRequired: true,
        labelKey: "ai.preset.openai",
    },
    openrouter: {
        provider: "openai_compatible",
        baseUrl: "https://openrouter.ai/api/v1",
        keyRequired: true,
        labelKey: "ai.preset.openrouter",
    },
    gemini: {
        provider: "openai_compatible",
        baseUrl: "https://generativelanguage.googleapis.com/v1beta/openai/",
        keyRequired: true,
        labelKey: "ai.preset.gemini",
    },
    groq: {
        provider: "openai_compatible",
        baseUrl: "https://api.groq.com/openai/v1",
        keyRequired: true,
        labelKey: "ai.preset.groq",
    },
    mistral: {
        provider: "openai_compatible",
        baseUrl: "https://api.mistral.ai/v1",
        keyRequired: true,
        labelKey: "ai.preset.mistral",
    },
    anthropic: {
        provider: "anthropic",
        baseUrl: "https://api.anthropic.com",
        keyRequired: true,
        labelKey: "ai.preset.anthropic",
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
