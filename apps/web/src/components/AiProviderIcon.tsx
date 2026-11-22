import type { ComponentType } from "react";
import { FiServer } from "react-icons/fi";
import {
    SiAnthropic,
    SiGooglegemini,
    SiLmstudio,
    SiMistralai,
    SiOllama,
    SiOpenrouter,
} from "react-icons/si";
import { TbBrandOpenai } from "react-icons/tb";
import type { AiPreset } from "../types/ai";

type IconType = ComponentType<{ className?: string; "aria-hidden"?: boolean }>;

function Monogram({ letter }: { letter: string }) {
    return (
        <span aria-hidden className="text-[0.6875rem] leading-none font-bold">
            {letter}
        </span>
    );
}

const icons: Partial<Record<AiPreset, IconType>> = {
    ollama: SiOllama,
    lmstudio: SiLmstudio,
    openai: TbBrandOpenai,
    anthropic: SiAnthropic,
    gemini: SiGooglegemini,
    openrouter: SiOpenrouter,
    mistral: SiMistralai,
    custom: FiServer,
};

const monograms: Partial<Record<AiPreset, string>> = {
    llamacpp: "ll",
    groq: "G",
};

const sizes = { sm: "size-7 text-sm", md: "size-9 text-base" };

// A neutral tile with the provider mark: keeps the list scannable without
// brand colors fighting the app's own palette.
export function AiProviderIcon({
    preset,
    size = "md",
}: {
    preset: AiPreset;
    size?: keyof typeof sizes;
}) {
    const Icon = icons[preset];
    return (
        <span
            aria-hidden
            className={`flex shrink-0 items-center justify-center rounded-md border border-line bg-layer-1 text-fg ${sizes[size]}`}
        >
            {Icon ? <Icon aria-hidden /> : <Monogram letter={monograms[preset] ?? "?"} />}
        </span>
    );
}
