import {
    useEffect,
    useId,
    useLayoutEffect,
    useRef,
    useState,
    type KeyboardEvent,
    type RefObject,
} from "react";
import { createPortal } from "react-dom";
import { FiCalendar, FiChevronLeft, FiChevronRight } from "react-icons/fi";
import { useDateFormat } from "../hooks/useDateFormat";
import { useTranslate } from "../hooks/useTranslate";
import {
    calendarDays,
    fromDateInputValue,
    toDateInputValue,
} from "../util/date";
import { useDismiss } from "../hooks/useDismiss";
import { Tooltip } from "./Tooltip";

type DatePickerProps = {
    label: string;
    // Local "YYYY-MM-DD", or "" for no date.
    value: string;
    onChange: (value: string) => void;
    // Shows the label above the field like TextInput; otherwise it is only
    // the accessible name.
    showLabel?: boolean;
    // A compact pill for task rows and cards: the date next to a calendar
    // icon (icon only while empty), red when `overdue`.
    pill?: { overdue: boolean };
    // Opens the calendar each time this number changes (a keyboard shortcut).
    openRequest?: number;
};

const POPOVER_WIDTH = 288;
const POPOVER_HEIGHT = 352;

// Replaces <input type="date"> so the calendar follows the user's
// first-day-of-week and date format preferences instead of the browser's.
export function DatePicker({
    label,
    value,
    onChange,
    showLabel = false,
    pill,
    openRequest = 0,
}: DatePickerProps) {
    const t = useTranslate();
    const format = useDateFormat();
    const [isOpen, setIsOpen] = useState(false);
    const [seenRequest, setSeenRequest] = useState(openRequest);
    if (openRequest !== seenRequest) {
        setSeenRequest(openRequest);
        setIsOpen(true);
    }
    const triggerRef = useRef<HTMLButtonElement>(null);
    const selected = fromDateInputValue(value);

    const pillField = pill && (
        <Tooltip label={selected ? label : t("task.dueDate.set")}>
            <button
                ref={triggerRef}
                type="button"
                aria-label={label}
                aria-haspopup="dialog"
                aria-expanded={isOpen}
                onClick={(event) => {
                    event.stopPropagation();
                    setIsOpen(!isOpen);
                }}
                className={`inline-flex h-6 cursor-pointer items-center gap-1 rounded-md border border-line bg-layer-2 px-1.5 text-caption hover:bg-layer-2-hover focus-visible:outline-2 focus-visible:outline-accent ${
                    pill.overdue ? "text-danger" : "text-fg-muted"
                }`}
            >
                <FiCalendar aria-hidden />
                {selected && format.date(selected)}
            </button>
        </Tooltip>
    );

    const field = pillField || (
        <button
            ref={triggerRef}
            type="button"
            aria-label={showLabel ? undefined : label}
            aria-haspopup="dialog"
            aria-expanded={isOpen}
            onClick={() => setIsOpen(!isOpen)}
            className="flex w-full cursor-pointer items-center gap-2 rounded-md border border-line-strong bg-surface-1 px-3 py-2 text-left text-sm hover:border-gray-400"
        >
            <FiCalendar aria-hidden className="shrink-0 text-fg-subtle" />
            <span className={selected ? "text-fg" : "text-fg-placeholder"}>
                {selected ? format.date(selected) : t("datePicker.placeholder")}
            </span>
        </button>
    );

    return (
        <>
            {showLabel ? (
                <label className="flex flex-col gap-1 text-sm text-fg-muted">
                    {label}
                    {field}
                </label>
            ) : (
                field
            )}
            {isOpen &&
                createPortal(
                    <Calendar
                        label={label}
                        anchor={triggerRef}
                        selected={selected}
                        onSelect={(date) => {
                            onChange(toDateInputValue(date));
                            setIsOpen(false);
                            triggerRef.current?.focus();
                        }}
                        onClose={() => {
                            setIsOpen(false);
                            triggerRef.current?.focus();
                        }}
                    />,
                    document.body,
                )}
        </>
    );
}

function sameDay(a: Date, b: Date) {
    return toDateInputValue(a) === toDateInputValue(b);
}

function Calendar({
    label,
    anchor,
    selected,
    onSelect,
    onClose,
}: {
    label: string;
    anchor: RefObject<HTMLButtonElement | null>;
    selected: Date | null;
    onSelect: (date: Date | null) => void;
    onClose: () => void;
}) {
    const t = useTranslate();
    const format = useDateFormat();
    const titleId = useId();
    const ref = useRef<HTMLDivElement>(null);
    const gridRef = useRef<HTMLDivElement>(null);
    const today = new Date();
    const [focused, setFocused] = useState<Date>(selected ?? today);
    const [position, setPosition] = useState({ top: 0, left: 0 });

    useLayoutEffect(() => {
        function place() {
            const rect = anchor.current?.getBoundingClientRect();
            if (!rect) return;
            const fitsBelow =
                rect.bottom + 4 + POPOVER_HEIGHT <= window.innerHeight;
            setPosition({
                top: fitsBelow
                    ? rect.bottom + 4
                    : Math.max(8, rect.top - 4 - POPOVER_HEIGHT),
                left: Math.min(
                    rect.left,
                    window.innerWidth - POPOVER_WIDTH - 8,
                ),
            });
        }
        place();
        window.addEventListener("resize", place);
        window.addEventListener("scroll", place, true);
        return () => {
            window.removeEventListener("resize", place);
            window.removeEventListener("scroll", place, true);
        };
    }, [anchor]);

    useDismiss(ref, true, onClose, [anchor]);

    // Moves keyboard focus along with the focused day.
    useEffect(() => {
        gridRef.current
            ?.querySelector<HTMLButtonElement>('[data-focused="true"]')
            ?.focus();
    }, [focused]);

    const days = calendarDays(focused, format.weekStart);
    const weekdays = days
        .slice(0, 7)
        .map((day) =>
            day.toLocaleDateString(format.locale, { weekday: "short" }),
        );

    function shift(days: number, months = 0) {
        setFocused(
            (d) =>
                new Date(
                    d.getFullYear(),
                    d.getMonth() + months,
                    // Clamp so Jan 31 + 1 month lands on Feb 28/29.
                    months
                        ? Math.min(
                              d.getDate(),
                              new Date(
                                  d.getFullYear(),
                                  d.getMonth() + months + 1,
                                  0,
                              ).getDate(),
                          )
                        : d.getDate() + days,
                ),
        );
    }

    function handleGridKeyDown(event: KeyboardEvent<HTMLDivElement>) {
        const moves: Record<string, [number, number]> = {
            ArrowLeft: [-1, 0],
            ArrowRight: [1, 0],
            ArrowUp: [-7, 0],
            ArrowDown: [7, 0],
            PageUp: [0, -1],
            PageDown: [0, 1],
        };
        const move = moves[event.key];
        if (!move) return;
        event.preventDefault();
        shift(...move);
    }

    return (
        <div
            ref={ref}
            role="dialog"
            aria-modal="false"
            aria-label={label}
            style={{ top: position.top, left: position.left, width: POPOVER_WIDTH }}
            className="fixed z-[60] rounded-lg border border-line bg-surface-1 p-3 shadow-xl"
        >
            <div className="mb-2 flex items-center justify-between">
                <button
                    type="button"
                    onClick={() => shift(0, -1)}
                    aria-label={t("datePicker.previousMonth")}
                    title={t("datePicker.previousMonth")}
                    className="cursor-pointer rounded-md p-1.5 text-fg-subtle hover:bg-layer-1-hover hover:text-fg"
                >
                    <FiChevronLeft aria-hidden />
                </button>
                <p
                    id={titleId}
                    aria-live="polite"
                    className="text-sm font-semibold text-fg capitalize"
                >
                    {focused.toLocaleDateString(format.locale, {
                        month: "long",
                        year: "numeric",
                    })}
                </p>
                <button
                    type="button"
                    onClick={() => shift(0, 1)}
                    aria-label={t("datePicker.nextMonth")}
                    title={t("datePicker.nextMonth")}
                    className="cursor-pointer rounded-md p-1.5 text-fg-subtle hover:bg-layer-1-hover hover:text-fg"
                >
                    <FiChevronRight aria-hidden />
                </button>
            </div>

            <div
                ref={gridRef}
                role="grid"
                aria-labelledby={titleId}
                onKeyDown={handleGridKeyDown}
                className="grid grid-cols-7 gap-0.5 text-center"
            >
                {weekdays.map((name) => (
                    <span
                        key={name}
                        role="columnheader"
                        className="py-1 text-xs font-medium text-fg-placeholder"
                    >
                        {name}
                    </span>
                ))}
                {days.map((day) => {
                    const isOtherMonth = day.getMonth() !== focused.getMonth();
                    const isSelected = selected !== null && sameDay(day, selected);
                    const isFocused = sameDay(day, focused);
                    return (
                        <button
                            key={toDateInputValue(day)}
                            type="button"
                            role="gridcell"
                            tabIndex={isFocused ? 0 : -1}
                            data-focused={isFocused}
                            aria-selected={isSelected}
                            aria-current={sameDay(day, today) ? "date" : undefined}
                            aria-label={format.date(day)}
                            onClick={() => onSelect(day)}
                            className={`size-9 cursor-pointer rounded-md text-sm hover:bg-layer-1-hover aria-[current=date]:font-bold aria-selected:bg-accent aria-selected:text-accent-fg ${
                                isOtherMonth ? "text-line-strong" : "text-fg-muted"
                            }`}
                        >
                            {day.getDate()}
                        </button>
                    );
                })}
            </div>

            <div className="mt-2 flex justify-between border-t border-line pt-2">
                <button
                    type="button"
                    onClick={() => onSelect(today)}
                    className="cursor-pointer rounded-md px-2 py-1 text-sm text-fg-muted hover:bg-layer-1-hover"
                >
                    {t("datePicker.today")}
                </button>
                <button
                    type="button"
                    onClick={() => onSelect(null)}
                    disabled={selected === null}
                    className="cursor-pointer rounded-md px-2 py-1 text-sm text-fg-subtle hover:bg-layer-1-hover disabled:cursor-default disabled:opacity-40 disabled:hover:bg-transparent"
                >
                    {t("datePicker.clear")}
                </button>
            </div>
        </div>
    );
}
