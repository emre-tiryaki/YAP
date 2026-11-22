import type { KeyboardEvent } from "react";

// Ctrl/Cmd + Enter sends the textarea's form; plain Enter adds a new line.
export function submitOnModEnter(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) {
        event.preventDefault();
        event.currentTarget.form?.requestSubmit();
    }
}
