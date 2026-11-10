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
