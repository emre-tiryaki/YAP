import { useEffect, useRef, useState } from "react";
import { ApiError } from "../api/client";
import { useTranslate } from "../hooks/useTranslate";
import { accentFor, initials } from "../util/accent";
import { FormError } from "./FormError";
import { IconPicker } from "./IconPicker";

type IconButtonProps = {
    // Used for the accent colour of the initials tile.
    id: string;
    name: string;
    // A single emoji, or "" for none.
    icon: string;
    error: Error | null;
    // Saves the icon and calls onSaved when it is stored.
    onSelect: (icon: string, onSaved: () => void) => void;
    // Clears the save error.
    reset: () => void;
};

// An icon tile (emoji, or initials when there is none). Clicking it opens a
// picker that saves the chosen icon right away.
export function IconButton({
    id,
    name,
    icon,
    error,
    onSelect,
    reset,
}: IconButtonProps) {
    const t = useTranslate();
    const [isOpen, setIsOpen] = useState(false);
    const ref = useRef<HTMLDivElement>(null);
    const accent = accentFor(id);

    useEffect(() => {
        if (!isOpen) return;

        const close = () => {
            setIsOpen(false);
            reset();
        };
        function handlePointerDown(event: PointerEvent) {
            if (!ref.current?.contains(event.target as Node)) close();
        }
        function handleKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") close();
        }

        document.addEventListener("pointerdown", handlePointerDown, true);
        document.addEventListener("keydown", handleKeyDown, true);
        return () => {
            document.removeEventListener(
                "pointerdown",
                handlePointerDown,
                true,
            );
            document.removeEventListener("keydown", handleKeyDown, true);
        };
    }, [isOpen, reset]);

    function select(next: string) {
        onSelect(next, () => {
            setIsOpen(false);
            reset();
        });
    }

    const errorMessage = !error
        ? null
        : error instanceof ApiError && error.message === "invalid_input"
          ? t("icon.invalid")
          : t("common.error");

    return (
        <div ref={ref} className="relative shrink-0">
            <button
                type="button"
                onClick={() => {
                    setIsOpen(!isOpen);
                    reset();
                }}
                aria-label={t("icon.change")}
                title={t("icon.change")}
                aria-haspopup="dialog"
                aria-expanded={isOpen}
                className={`grid size-12 cursor-pointer place-items-center rounded-lg font-semibold transition hover:ring-2 hover:ring-gray-300 focus-visible:outline-2 focus-visible:outline-gray-900 ${
                    icon ? "text-3xl" : `text-base ${accent.tile}`
                }`}
            >
                {icon || initials(name)}
            </button>
            {isOpen && (
                <div
                    role="dialog"
                    aria-label={t("icon.change")}
                    className="absolute top-full left-0 z-40 mt-2 w-80 rounded-lg border border-gray-200 bg-white p-3 shadow-lg"
                >
                    <IconPicker value={icon} onChange={select} />
                    {errorMessage && (
                        <div className="mt-2">
                            <FormError message={errorMessage} />
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
