// A task's key: the project identifier and the task number, e.g. "ABC-12".
// Empty while either is missing (a task still being created has no number).
export function taskKey(identifier: string, number: number): string {
    return identifier && number > 0 ? `${identifier}-${number}` : "";
}
