import { useLayoutEffect, useRef, useState } from "react";
import { useTranslate } from "../hooks/useTranslate";

type ExpandableTextProps = {
    text: string;
    className?: string;
};

// Clamps long text to a few lines and offers a toggle only when the text
// actually overflows the clamp.
export function ExpandableText({ text, className = "" }: ExpandableTextProps) {
    const t = useTranslate();
    const ref = useRef<HTMLParagraphElement>(null);
    const [expanded, setExpanded] = useState(false);
    const [overflowing, setOverflowing] = useState(false);

    useLayoutEffect(() => {
        const element = ref.current;
        if (!element || expanded) return;
        const measure = () =>
            setOverflowing(element.scrollHeight > element.clientHeight + 1);
        measure();
        const observer = new ResizeObserver(measure);
        observer.observe(element);
        return () => observer.disconnect();
    }, [text, expanded]);

    return (
        <div>
            <p
                ref={ref}
                className={`break-words whitespace-pre-wrap ${
                    expanded ? "" : "line-clamp-4"
                } ${className}`}
            >
                {text}
            </p>
            {(overflowing || expanded) && (
                <button
                    type="button"
                    onClick={() => setExpanded((value) => !value)}
                    className="mt-1 text-xs font-medium text-gray-900 hover:underline"
                >
                    {expanded ? t("common.showLess") : t("common.showAll")}
                </button>
            )}
        </div>
    );
}
