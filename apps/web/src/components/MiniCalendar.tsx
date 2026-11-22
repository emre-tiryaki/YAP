import { useState } from "react";
import { FiChevronLeft, FiChevronRight } from "react-icons/fi";
import { useDateFormat } from "../hooks/useDateFormat";
import { useTranslate } from "../hooks/useTranslate";
import type { Task } from "../types/task";
import { addMonths, calendarDays, isSameDay, weekDays } from "../util/calendar";
import { toDateInputValue } from "../util/date";

type MiniCalendarProps = {
    // Days with at least one of these tasks get a dot.
    tasks: Task[];
};

// A small month overview for the home page: today highlighted, dots on days
// that have tasks due.
export function MiniCalendar({ tasks }: MiniCalendarProps) {
    const t = useTranslate();
    const format = useDateFormat();
    const [cursor, setCursor] = useState(() => new Date());
    const today = new Date();
    const days = calendarDays(cursor, format.weekStart);
    const counts = new Map<string, number>();
    for (const task of tasks) {
        if (!task.dueDate) continue;
        const key = toDateInputValue(task.dueDate);
        counts.set(key, (counts.get(key) ?? 0) + 1);
    }
    const weekdayNames = weekDays(today, format.weekStart).map((day) =>
        day.toLocaleDateString(format.locale, { weekday: "narrow" }),
    );

    return (
        <section
            aria-label={t("home.calendar")}
            className="rounded-xl border border-line bg-layer-1 p-3 shadow-sm"
        >
            <div className="mb-2 flex items-center justify-between">
                <h2 className="text-sm font-semibold capitalize">
                    {cursor.toLocaleDateString(format.locale, {
                        month: "long",
                        year: "numeric",
                    })}
                </h2>
                <div className="flex items-center gap-0.5">
                    <button
                        type="button"
                        aria-label={t("calendar.prev")}
                        onClick={() => setCursor(addMonths(cursor, -1))}
                        className="grid size-6 cursor-pointer place-items-center rounded text-fg-muted hover:bg-layer-1-hover"
                    >
                        <FiChevronLeft aria-hidden />
                    </button>
                    <button
                        type="button"
                        onClick={() => setCursor(new Date())}
                        className="cursor-pointer rounded px-1.5 py-0.5 text-xs text-fg-muted hover:bg-layer-1-hover"
                    >
                        {t("calendar.today")}
                    </button>
                    <button
                        type="button"
                        aria-label={t("calendar.next")}
                        onClick={() => setCursor(addMonths(cursor, 1))}
                        className="grid size-6 cursor-pointer place-items-center rounded text-fg-muted hover:bg-layer-1-hover"
                    >
                        <FiChevronRight aria-hidden />
                    </button>
                </div>
            </div>
            <div className="grid grid-cols-7 text-center text-[11px]">
                {weekdayNames.map((name, index) => (
                    <span key={index} className="py-1 text-fg-placeholder">
                        {name}
                    </span>
                ))}
                {days.map((day) => {
                    const count = counts.get(toDateInputValue(day)) ?? 0;
                    const isToday = isSameDay(day, today);
                    const inMonth = day.getMonth() === cursor.getMonth();
                    return (
                        <span
                            key={day.getTime()}
                            title={
                                count > 0
                                    ? `${count} ${t("workspace.projects.taskCount")}`
                                    : undefined
                            }
                            className={`relative mx-auto my-0.5 grid size-7 place-items-center rounded-full text-xs tabular-nums ${
                                isToday
                                    ? "bg-accent font-semibold text-accent-fg"
                                    : inMonth
                                      ? "text-fg"
                                      : "text-fg-placeholder"
                            }`}
                        >
                            {day.getDate()}
                            {count > 0 && (
                                <span
                                    aria-hidden
                                    className={`absolute bottom-0.5 size-1 rounded-full ${
                                        isToday ? "bg-accent-fg" : "bg-accent"
                                    }`}
                                />
                            )}
                        </span>
                    );
                })}
            </div>
        </section>
    );
}
