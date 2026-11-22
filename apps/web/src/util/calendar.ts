import type { WeekStart } from "../types/settings";
import { toDateInputValue } from "./date";

// 0 = Sunday ... 6 = Saturday, as returned by Date.getDay().
export function firstDayIndex(weekStart: WeekStart): number {
    return weekStart === "sunday" ? 0 : 1;
}

// The 42 days (6 weeks) shown for the month of `month`, starting on the first
// day of the week that contains the 1st.
export function calendarDays(month: Date, weekStart: WeekStart): Date[] {
    const first = new Date(month.getFullYear(), month.getMonth(), 1);
    const offset = (first.getDay() - firstDayIndex(weekStart) + 7) % 7;
    return Array.from(
        { length: 42 },
        (_, i) =>
            new Date(first.getFullYear(), first.getMonth(), 1 - offset + i),
    );
}

export function startOfDay(date: Date): Date {
    return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

export function addDays(date: Date, days: number): Date {
    return new Date(date.getFullYear(), date.getMonth(), date.getDate() + days);
}

// The first day of the month `months` away from the month of `date`.
export function addMonths(date: Date, months: number): Date {
    return new Date(date.getFullYear(), date.getMonth() + months, 1);
}

export function isSameDay(a: Date, b: Date): boolean {
    return toDateInputValue(a) === toDateInputValue(b);
}

// The 7 days of the week that contains `date`.
export function weekDays(date: Date, weekStart: WeekStart): Date[] {
    const offset = (date.getDay() - firstDayIndex(weekStart) + 7) % 7;
    const first = addDays(date, -offset);
    return Array.from({ length: 7 }, (_, i) => addDays(first, i));
}
