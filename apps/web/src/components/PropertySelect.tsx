import {
    useId,
    useRef,
    useState,
    type KeyboardEvent,
    type ReactNode,
} from "react";
import { FiCheck } from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";
import { Popover } from "./Popover";
import { Tooltip } from "./Tooltip";

type Option<T extends string> = {
    value: T;
    label: string;
    icon?: ReactNode;
};

type PropertySelectProps<T extends string> = {
    value: T | T[] | null;
    options: Option<T>[];
    onChange: (value: T | T[] | null) => void;
    multiple?: boolean;
    searchable?: boolean;
    placeholder: string;
    variant: "pill" | "ghost" | "cell";
    tooltip?: string;
    triggerLabel?: ReactNode;
    // Accessible name of the trigger when its text alone is not enough.
    ariaLabel?: string;
};

const triggerClasses = {
    pill: "h-6 rounded-md border border-line bg-layer-2 px-1.5 text-caption hover:bg-layer-2-hover",
    ghost: "h-8 rounded-md px-2 text-body-xs hover:bg-layer-1-hover",
    cell: "h-full w-full px-2 text-left text-body-xs hover:bg-layer-1-hover",
};

export function PropertySelect<T extends string>({
    value,
    options,
    onChange,
    multiple,
    searchable,
    placeholder,
    variant,
    tooltip,
    triggerLabel,
    ariaLabel,
}: PropertySelectProps<T>) {
    const t = useTranslate();
    const listId = useId();
    const anchorRef = useRef<HTMLButtonElement>(null);
    const [open, setOpen] = useState(false);
    const [query, setQuery] = useState("");
    const [active, setActive] = useState(0);

    const selected: T[] =
        value === null ? [] : Array.isArray(value) ? value : [value];
    const showSearch = searchable ?? options.length > 6;

    // Selected options first, then the filter.
    const visible = [
        ...options.filter((o) => selected.includes(o.value)),
        ...options.filter((o) => !selected.includes(o.value)),
    ].filter((o) => o.label.toLowerCase().includes(query.trim().toLowerCase()));

    const selectedOptions = options.filter((o) => selected.includes(o.value));
    const summary =
        triggerLabel ??
        (selectedOptions.length === 0
            ? placeholder
            : selectedOptions.map((o) => o.label).join(", "));
    const leadingIcon = selectedOptions[0]?.icon;

    function openList() {
        setQuery("");
        setActive(0);
        setOpen(true);
    }
    function close() {
        setOpen(false);
        anchorRef.current?.focus();
    }
    function pick(option: Option<T>) {
        if (multiple) {
            onChange(
                selected.includes(option.value)
                    ? selected.filter((v) => v !== option.value)
                    : [...selected, option.value],
            );
        } else {
            onChange(option.value);
            close();
        }
    }
    function handleKeyDown(event: KeyboardEvent<HTMLElement>) {
        if (event.key === "ArrowDown") {
            event.preventDefault();
            setActive((i) => Math.min(i + 1, visible.length - 1));
        } else if (event.key === "ArrowUp") {
            event.preventDefault();
            setActive((i) => Math.max(i - 1, 0));
        } else if (event.key === "Enter") {
            event.preventDefault();
            const option = visible[active];
            if (option) pick(option);
        }
    }

    const trigger = (
        <button
            ref={anchorRef}
            type="button"
            aria-label={ariaLabel}
            aria-haspopup="listbox"
            aria-expanded={open}
            onClick={(event) => {
                event.stopPropagation();
                if (open) setOpen(false);
                else openList();
            }}
            className={`inline-flex cursor-pointer items-center gap-1 whitespace-nowrap text-fg-muted focus-visible:outline-2 focus-visible:outline-accent ${triggerClasses[variant]}`}
        >
            {leadingIcon && (
                <span className="flex shrink-0 items-center">{leadingIcon}</span>
            )}
            <span
                className={`truncate ${
                    selected.length === 0 && !triggerLabel
                        ? "text-fg-placeholder"
                        : ""
                }`}
            >
                {summary}
            </span>
        </button>
    );

    return (
        <>
            {tooltip ? <Tooltip label={tooltip}>{trigger}</Tooltip> : trigger}
            <Popover open={open} onClose={close} anchorRef={anchorRef}>
                <div
                    onKeyDown={handleKeyDown}
                    onClick={(event) => event.stopPropagation()}
                >
                    {showSearch && (
                        <input
                            autoFocus
                            value={query}
                            onChange={(event) => {
                                setQuery(event.target.value);
                                setActive(0);
                            }}
                            placeholder={t("common.search")}
                            aria-label={t("common.search")}
                            aria-controls={listId}
                            aria-activedescendant={`${listId}-${active}`}
                            className="mb-1 h-7 w-full rounded-md border border-line bg-layer-1 px-2 text-body-xs text-fg outline-none focus:border-accent"
                        />
                    )}
                    <div
                        id={listId}
                        role="listbox"
                        aria-multiselectable={multiple}
                        tabIndex={showSearch ? -1 : 0}
                        ref={(element) => {
                            if (element && !showSearch) element.focus();
                        }}
                        className="max-h-64 overflow-auto outline-none"
                    >
                        {visible.length === 0 && (
                            <div className="px-2 py-1.5 text-body-xs text-fg-subtle">
                                {t("common.noOptions")}
                            </div>
                        )}
                        {visible.map((option, index) => {
                            const isSelected = selected.includes(option.value);
                            return (
                                <div
                                    key={option.value}
                                    id={`${listId}-${index}`}
                                    role="option"
                                    aria-selected={isSelected}
                                    onClick={() => pick(option)}
                                    onMouseEnter={() => setActive(index)}
                                    className={`flex h-7 cursor-pointer items-center gap-2 rounded-md px-2 text-body-xs text-fg ${
                                        index === active ? "bg-layer-1-hover" : ""
                                    }`}
                                >
                                    {multiple && (
                                        <span
                                            aria-hidden
                                            className={`grid size-3.5 shrink-0 place-items-center rounded border ${
                                                isSelected
                                                    ? "border-accent bg-accent text-accent-fg"
                                                    : "border-line-strong"
                                            }`}
                                        >
                                            {isSelected && (
                                                <FiCheck className="size-2.5" />
                                            )}
                                        </span>
                                    )}
                                    {option.icon && (
                                        <span className="flex shrink-0 items-center">
                                            {option.icon}
                                        </span>
                                    )}
                                    <span className="flex-1 truncate">
                                        {option.label}
                                    </span>
                                    {!multiple && isSelected && (
                                        <FiCheck
                                            aria-hidden
                                            className="shrink-0 text-fg-subtle"
                                        />
                                    )}
                                </div>
                            );
                        })}
                    </div>
                </div>
            </Popover>
        </>
    );
}
