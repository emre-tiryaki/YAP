import {
    formatDate,
    formatRelativeTime,
    formatTime,
} from "../util/date";
import { usePreferences } from "./usePreferences";

// Date and time formatters that follow the user's preferences.
export function useDateFormat() {
    const { language, dateFormat, timeFormat, weekStart } = usePreferences();
    const locale = language;
    return {
        locale,
        weekStart,
        date: (date: Date) => formatDate(date, dateFormat),
        time: (date: Date) => formatTime(date, timeFormat, locale),
        dateTime: (date: Date) =>
            `${formatDate(date, dateFormat)} ${formatTime(date, timeFormat, locale)}`,
        relative: (date: Date) => formatRelativeTime(date, locale),
    };
}
