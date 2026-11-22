import { useEffect, useRef } from "react";
import { useNavigate } from "react-router";
import { FiMessageSquare } from "react-icons/fi";
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
    const t = useTranslate();
    const navigate = useNavigate();
    const id = chatId ?? "";
    const [choice, setChoice] = useAiModelChoiceFor(detail);
    const status = useAiStatus();
    const create = useCreateAiChat();
    const send = useSendAiMessage(id);
    const regenerate = useRegenerateAiMessage(id);
    const stop = useStopAiChat();

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
                        <EmptyState
                            icon={FiMessageSquare}
                            title={t("assistant.empty.title")}
                            description={t("assistant.empty.description")}
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
            />
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
