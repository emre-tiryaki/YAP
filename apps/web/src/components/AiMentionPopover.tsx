import type { RefObject } from "react";
import { useTranslate } from "../hooks/useTranslate";
import type { AiMentionHit } from "../hooks/useAiMentions";
import { Popover } from "./Popover";

type AiMentionPopoverProps = {
    anchorRef: RefObject<HTMLElement | null>;
    hits: AiMentionHit[];
    isPending: boolean;
    activeIndex: number;
    onPick: (hit: AiMentionHit) => void;
    onClose: () => void;
};

// The list under "@query" in the composer. The textarea keeps focus, so the
// composer handles the arrow keys and Enter; this only draws the list.
export function AiMentionPopover({
    anchorRef,
    hits,
    isPending,
    activeIndex,
    onPick,
    onClose,
}: AiMentionPopoverProps) {
    const t = useTranslate();
    return (
        <Popover
            open
            onClose={onClose}
            anchorRef={anchorRef}
            className="w-80 rounded-lg border border-line bg-surface-1 p-1 shadow-lg"
        >
            <div role="listbox" aria-label={t("assistant.mention.label")}>
                {hits.length === 0 ? (
                    <p className="px-2 py-1.5 text-body-xs text-fg-subtle">
                        {isPending
                            ? t("common.loading")
                            : t("assistant.mention.none")}
                    </p>
                ) : (
                    hits.map((hit, index) => (
                        <button
                            key={hit.key}
                            type="button"
                            role="option"
                            aria-selected={index === activeIndex}
                            // Keep focus in the textarea.
                            onMouseDown={(event) => event.preventDefault()}
                            onClick={() => onPick(hit)}
                            className={`flex w-full cursor-pointer items-baseline gap-2 rounded-md px-2 py-1.5 text-left text-body-xs ${
                                index === activeIndex
                                    ? "bg-layer-1-hover"
                                    : "hover:bg-layer-1-hover"
                            }`}
                        >
                            <span className="truncate font-medium">{hit.label}</span>
                            {hit.detail && (
                                <span className="truncate text-caption text-fg-muted">
                                    {hit.detail}
                                </span>
                            )}
                        </button>
                    ))
                )}
            </div>
        </Popover>
    );
}
