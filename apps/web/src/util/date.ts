import type { DateFormat, TimeFormat, WeekStart } from "../types/settings";

// Converts between Date and the local "YYYY-MM-DD" value of <input type="date">.

export function toDateInputValue(date: Date | null): string {
    if (!date) return "";
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    return `${date.getFullYear()}-${month}-${day}`;
}

export function fromDateInputValue(value: string): Date | null {
    return value ? new Date(`${value}T00:00:00`) : null;
}

// Local calendar day of `date` relative to today: 0 today, 1 tomorrow, -1 yesterday.
export function daysFromToday(date: Date, now: Date = new Date()): number {
    const day = new Date(date.getFullYear(), date.getMonth(), date.getDate());
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    return Math.round((day.getTime() - today.getTime()) / 86_400_000);
}

// "5 minutes ago", "yesterday"... for the last week. Returns null under a
// minute so the caller can say "just now", and for older dates so the caller
// can show the calendar date instead.
export function formatRelativeTime(
    date: Date,
    locale: string,
    now: Date = new Date(),
): string | null {
    const seconds = Math.round((date.getTime() - now.getTime()) / 1000);
    const elapsed = Math.abs(seconds);
    if (elapsed < 60 || elapsed >= 7 * 86_400) return null;
    const format = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
    if (elapsed < 3600) return format.format(Math.round(seconds / 60), "minute");
    if (elapsed < 86_400) return format.format(Math.round(seconds / 3600), "hour");
    return format.format(daysFromToday(date, now), "day");
}

export function isWithinLastMinute(date: Date, now: Date = new Date()): boolean {
    return Math.abs(now.getTime() - date.getTime()) < 60_000;
}

// Numeric calendar date in the chosen format, from the local day of `date`.
export function formatDate(date: Date, format: DateFormat): string {
    const [year, month, day] = toDateInputValue(date).split("-");
    if (format === "mdy") return `${month}/${day}/${year}`;
    if (format === "iso") return `${year}-${month}-${day}`;
    return `${day}.${month}.${year}`;
}

export function formatTime(
    date: Date,
    format: TimeFormat,
    locale: string,
): string {
    return date.toLocaleTimeString(locale, {
        hour: format === "12h" ? "numeric" : "2-digit",
        minute: "2-digit",
        hour12: format === "12h",
    });
}

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
