import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { FiCheckSquare, FiFileText, FiMessageSquare, FiSearch } from "react-icons/fi";
import { TbSparkles } from "react-icons/tb";
import { ApiError } from "../api/client";
import {
    useAiChat,
    useAiStatus,
    useAiStream,
    useCreateAiChat,
    useRegenerateAiMessage,
    useSendAiMessage,
    useStopAiChat,
} from "../hooks/useAi";
import { useAiModelChoice } from "../hooks/useAiModelChoice";
import { useTranslate } from "../hooks/useTranslate";
import type { AiAttachmentInput, AiChatDetail } from "../types/ai";
import { AiComposer } from "./AiComposer";
import { AiMessage } from "./AiMessage";
import { EmptyState } from "./EmptyState";
import { InlineError } from "./InlineError";
import { Skeleton } from "./Skeleton";

// One chat, or (chatId null) the empty page for a new one.
export function AiChatView({ chatId }: { chatId: string | null }) {
    const t = useTranslate();
    const chat = useAiChat(chatId ?? "");

    if (chatId === null) return <AiChatBody key="new" chatId={null} detail={null} />;
    if (chat.isPending) {
        return (
            <div className="flex flex-1 flex-col gap-4 p-6">
                <Skeleton className="h-10 w-1/2 self-end" />
                <Skeleton className="h-24 w-2/3" />
            </div>
        );
    }
    if (chat.isError) {
        const gone = chat.error instanceof ApiError && chat.error.status === 404;
        return (
            <div className="p-6">
                {gone ? (
                    <EmptyState
                        icon={FiMessageSquare}
                        title={t("assistant.chat.notFound.title")}
                        description={t("assistant.chat.notFound.message")}
                    />
                ) : (
                    <InlineError
                        message={t("assistant.chat.loadError")}
                        onRetry={() => void chat.refetch()}
                    />
                )}
            </div>
        );
    }
    return <AiChatBody key={chatId} chatId={chatId} detail={chat.data} />;
}

type AiChatBodyProps = { chatId: string | null; detail: AiChatDetail | null };

function AiChatBody({ chatId, detail }: AiChatBodyProps) {
    const navigate = useNavigate();
    const id = chatId ?? "";
    const [choice, setChoice] = useAiModelChoiceFor(detail);
    const status = useAiStatus();
    const create = useCreateAiChat();
    const send = useSendAiMessage(id);
    const regenerate = useRegenerateAiMessage(id);
    const stop = useStopAiChat();
    const [draft, setDraft] = useState<{ text: string; n: number } | null>(null);

    const messages = detail?.messages ?? [];
    const generating =
        detail?.chat.generating === true ||
        messages.some((m) => m.status === "streaming");
    const stream = useAiStream(id, generating);
    const last = messages[messages.length - 1];
    const lastLength = stream?.content.length ?? last?.content.length ?? 0;

    const endRef = useRef<HTMLDivElement>(null);
    useEffect(() => {
        endRef.current?.scrollIntoView({ block: "end" });
    }, [messages.length, lastLength]);

    const busyChatId =
        status.activeChatId !== null && status.activeChatId !== chatId
            ? status.activeChatId
            : null;
    const error = create.error ?? send.error ?? regenerate.error ?? stop.error;

    async function handleSend(content: string, attachments: AiAttachmentInput[]) {
        if (!choice) return;
        const input = {
            connectionId: choice.connectionId,
            model: choice.model,
            content,
            attachments,
        };
        if (chatId === null) {
            const result = await create.mutateAsync(input);
            void navigate(`/assistant/${result.chat.id}`);
        } else {
            await send.mutateAsync(input);
        }
    }

    const lastAssistant =
        last && last.role === "assistant" && !generating && choice ? last : null;

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <div className="min-h-0 flex-1 overflow-y-auto">
                <div className="mx-auto flex max-w-3xl flex-col gap-5 px-4 py-6">
                    {messages.length === 0 ? (
                        <Welcome
                            onPick={(text) =>
                                setDraft({ text, n: (draft?.n ?? 0) + 1 })
                            }
                        />
                    ) : (
                        messages.map((message) => (
                            <AiMessage
                                key={message.id}
                                message={message}
                                liveContent={
                                    stream && stream.messageId === message.id
                                        ? stream.content
                                        : null
                                }
                                onRegenerate={
                                    lastAssistant === message && choice
                                        ? () =>
                                              regenerate.mutate({
                                                  connectionId: choice.connectionId,
                                                  model: choice.model,
                                              })
                                        : null
                                }
                            />
                        ))
                    )}
                    <div ref={endRef} />
                </div>
            </div>
            <AiComposer
                choice={choice}
                onChoiceChange={setChoice}
                onSend={handleSend}
                onStop={() => id && stop.mutate(id)}
                generating={generating}
                busyChatId={busyChatId}
                error={error}
                draft={draft}
            />
        </div>
    );
}

const suggestions = [
    { icon: FiCheckSquare, key: "assistant.suggest.overdue" },
    { icon: FiFileText, key: "assistant.suggest.summary" },
    { icon: FiSearch, key: "assistant.suggest.next" },
] as const;

function Welcome({ onPick }: { onPick: (text: string) => void }) {
    const t = useTranslate();
    return (
        <div className="flex flex-col items-center gap-6 pt-[12vh] text-center">
            <span
                aria-hidden
                className="flex size-12 items-center justify-center rounded-xl border border-line bg-layer-2 text-2xl text-fg shadow-raised"
            >
                <TbSparkles />
            </span>
            <div className="flex flex-col gap-1">
                <h2 className="text-lg font-semibold text-fg">
                    {t("assistant.empty.title")}
                </h2>
                <p className="mx-auto max-w-md text-body-xs text-fg-subtle">
                    {t("assistant.empty.description")}
                </p>
            </div>
            <ul className="flex w-full max-w-md flex-col gap-2">
                {suggestions.map(({ icon: Icon, key }) => (
                    <li key={key}>
                        <button
                            type="button"
                            onClick={() => onPick(t(key))}
                            className="flex w-full cursor-pointer items-center gap-2.5 rounded-lg border border-line bg-layer-2 px-3 py-2 text-left text-body-xs text-fg-muted hover:border-line-strong hover:bg-layer-2-hover hover:text-fg focus-visible:outline-2 focus-visible:outline-accent"
                        >
                            <Icon aria-hidden className="shrink-0 text-fg-placeholder" />
                            {t(key)}
                        </button>
                    </li>
                ))}
            </ul>
        </div>
    );
}

// The chat's own model when it has one, else the last one used.
function useAiModelChoiceFor(detail: AiChatDetail | null) {
    const chat = detail?.chat;
    return useAiModelChoice(
        chat && chat.connectionId && chat.model
            ? { connectionId: chat.connectionId, model: chat.model }
            : null,
    );
}
