import { FiCopy, FiRefreshCw } from "react-icons/fi";
import { aiErrorCodeKey } from "../hooks/useAi";
import { pushInfoToast } from "../hooks/useToasts";
import { useTranslate } from "../hooks/useTranslate";
import type { AiAttachment, AiMessage as AiMessageData } from "../types/ai";
import { AiAttachmentChip } from "./AiAttachmentChip";
import { Button } from "./Button";
import { Tooltip } from "./Tooltip";
import { MarkdownView } from "./MarkdownView";

function attachmentHref(a: AiAttachment): string | null {
    switch (a.kind) {
        case "task":
            return a.projectId && a.planId
                ? `/projects/${a.projectId}/plans/${a.planId}/tasks/${a.targetId}`
                : null;
        case "plan":
            return a.projectId ? `/projects/${a.projectId}/plans/${a.targetId}` : null;
        case "project":
            return `/projects/${a.targetId}`;
        case "resource":
        case "repo_file":
            return `/resources/${a.targetId}`;
    }
}

type AiMessageProps = {
    message: AiMessageData;
    // The text received so far while the answer is being written.
    liveContent: string | null;
    // Offered on the last answer when nothing is being generated.
    onRegenerate: (() => void) | null;
};

export function AiMessage({ message, liveContent, onRegenerate }: AiMessageProps) {
    const t = useTranslate();

    if (message.role === "user") {
        return (
            <div className="flex flex-col items-end gap-1.5">
                {message.attachments.length > 0 && (
                    <div className="flex max-w-[80%] flex-wrap justify-end gap-1.5">
                        {message.attachments.map((a) => (
                            <AiAttachmentChip
                                key={a.id}
                                kind={a.kind}
                                label={a.label}
                                href={attachmentHref(a)}
                                missing={a.missing}
                                status={a.lastStatus}
                            />
                        ))}
                    </div>
                )}
                <p className="max-w-[80%] rounded-lg bg-layer-3 px-3 py-2 text-sm whitespace-pre-wrap text-fg">
                    {message.content}
                </p>
            </div>
        );
    }

    const streaming = message.status === "streaming";
    const content = streaming ? (liveContent ?? message.content) : message.content;

    async function copy() {
        try {
            await navigator.clipboard.writeText(message.content);
            pushInfoToast(t("assistant.message.copy"), t("assistant.message.copied"));
        } catch {
            // Clipboard refused: nothing to confirm.
        }
    }

    return (
        <div className="flex flex-col gap-1">
            {/* Plain text while streaming: markdown is parsed once it is whole. */}
            {streaming ? (
                <p
                    aria-live="polite"
                    className="px-1 text-sm whitespace-pre-wrap text-fg"
                >
                    {content || t("assistant.chat.generating")}
                </p>
            ) : (
                content !== "" && <MarkdownView markdown={content} />
            )}
            {message.status === "error" && (
                <p role="alert" className="px-1 text-body-xs text-danger">
                    {t(aiErrorCodeKey(message.errorCode))}
                </p>
            )}
            {(message.status === "stopped" || message.status === "interrupted") && (
                <p className="px-1 text-caption text-fg-muted">
                    {t(
                        message.status === "stopped"
                            ? "assistant.message.stopped"
                            : "assistant.message.interrupted",
                    )}
                </p>
            )}
            {!streaming && (
                <div className="flex items-center gap-1 px-1">
                    {message.content !== "" && (
                        <Tooltip label={t("assistant.message.copy")} side="top">
                            <Button
                                variant="ghost"
                                size="md"
                                aria-label={t("assistant.message.copy")}
                                onClick={() => void copy()}
                            >
                                <FiCopy aria-hidden />
                            </Button>
                        </Tooltip>
                    )}
                    {onRegenerate && (
                        <Tooltip label={t("assistant.message.regenerate")} side="top">
                            <Button
                                variant="ghost"
                                size="md"
                                aria-label={t("assistant.message.regenerate")}
                                onClick={onRegenerate}
                            >
                                <FiRefreshCw aria-hidden />
                            </Button>
                        </Tooltip>
                    )}
                    {message.model && (
                        <span className="ml-1 text-caption text-fg-placeholder">
                            {message.model}
                        </span>
                    )}
                </div>
            )}
        </div>
    );
}
