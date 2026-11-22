import { useState } from "react";

const storageKey = "yap.ai.model";

export type AiModelChoice = { connectionId: string; model: string };

function stored(): AiModelChoice | null {
    try {
        const parsed: unknown = JSON.parse(localStorage.getItem(storageKey) ?? "null");
        if (
            typeof parsed === "object" &&
            parsed !== null &&
            "connectionId" in parsed &&
            "model" in parsed &&
            typeof parsed.connectionId === "string" &&
            typeof parsed.model === "string"
        ) {
            return { connectionId: parsed.connectionId, model: parsed.model };
        }
    } catch {
        // Unreadable: fall through to no choice.
    }
    return null;
}

// The model picked in the composer. It starts from `initial` (the chat's own
// model) or the one used last, and a choice is remembered for next time.
export function useAiModelChoice(initial: AiModelChoice | null) {
    const [choice, setChoice] = useState<AiModelChoice | null>(
        () => initial ?? stored(),
    );
    function choose(next: AiModelChoice) {
        setChoice(next);
        localStorage.setItem(storageKey, JSON.stringify(next));
    }
    return [choice, choose] as const;
}
