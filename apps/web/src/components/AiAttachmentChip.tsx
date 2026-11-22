import { Link } from "react-router";
import {
    FiAlertTriangle,
    FiX,
} from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";
import type { AiAttachmentKind, AiAttachmentStatus } from "../types/ai";
import { attachmentIcons } from "../util/aiAttachmentIcons";
import { Tooltip } from "./Tooltip";

type AiAttachmentChipProps = {
    kind: AiAttachmentKind;
    label: string;
    // Set in the composer: the chip can be taken away again.
    onRemove?: () => void;
    // Set in the history: where the target lives; missing ones have none.
    href?: string | null;
    // The target is gone; the chip is struck through and does not link.
    missing?: boolean;
    status?: AiAttachmentStatus;
};

// What was attached to a message: a task, plan, project or file.
export function AiAttachmentChip({
    kind,
    label,
    onRemove,
    href,
    missing = false,
    status,
}: AiAttachmentChipProps) {
    const t = useTranslate();
    const Icon = attachmentIcons[kind];
    const warning =
        missing || status === "missing"
            ? t("assistant.attachment.missing")
            : status === "truncated"
              ? t("assistant.attachment.truncated")
              : status === "omitted"
                ? t("assistant.attachment.omitted")
                : status === "unreadable"
                  ? t("assistant.attachment.unreadable")
                  : null;
    const body = (
        <>
            <Icon aria-hidden className="shrink-0 text-fg-subtle" />
            <span
                className={`max-w-48 truncate ${missing ? "text-fg-subtle line-through" : ""}`}
            >
                {label}
            </span>
        </>
    );
    const className =
        "inline-flex h-6 items-center gap-1 rounded-md border border-line bg-layer-2 px-1.5 text-caption text-fg-muted";

    return (
        <span className="inline-flex items-center gap-1">
            {href && !missing ? (
                <Link to={href} className={`${className} hover:text-fg`}>
                    {body}
                </Link>
            ) : (
                <span className={className}>{body}</span>
            )}
            {warning && !missing && (
                <Tooltip label={warning} side="top">
                    <span tabIndex={0} className="text-warning" aria-label={warning}>
                        <FiAlertTriangle aria-hidden />
                    </span>
                </Tooltip>
            )}
            {onRemove && (
                <button
                    type="button"
                    onClick={onRemove}
                    aria-label={`${t("assistant.attach.remove")}: ${label}`}
                    className="-ml-0.5 cursor-pointer rounded p-0.5 text-fg-placeholder hover:bg-layer-3-hover hover:text-fg"
                >
                    <FiX aria-hidden />
                </button>
            )}
        </span>
    );
}
