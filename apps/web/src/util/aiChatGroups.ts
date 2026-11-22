import type { TranslationKey } from "../hooks/useTranslate";
import type { AiChat } from "../types/ai";

export type AiChatGroup = { titleKey: TranslationKey; chats: AiChat[] };

const DAY = 24 * 60 * 60 * 1000;

// Chats grouped by when they were last used, newest first; empty groups are
// left out.
export function groupAiChats(chats: AiChat[], now = new Date()): AiChatGroup[] {
    const startOfToday = new Date(now);
    startOfToday.setHours(0, 0, 0, 0);
    const today = startOfToday.getTime();
    const groups: AiChatGroup[] = [
        { titleKey: "assistant.group.today", chats: [] },
        { titleKey: "assistant.group.yesterday", chats: [] },
        { titleKey: "assistant.group.week", chats: [] },
        { titleKey: "assistant.group.older", chats: [] },
    ];
    for (const chat of chats) {
        const time = chat.updatedAt.getTime();
        const index =
            time >= today ? 0 : time >= today - DAY ? 1 : time >= today - 7 * DAY ? 2 : 3;
        groups[index].chats.push(chat);
    }
    return groups.filter((group) => group.chats.length > 0);
}
