// Formatting commands shared by both modes of the markdown editor.
export type MarkdownCommand =
    | "h1"
    | "h2"
    | "h3"
    | "bold"
    | "italic"
    | "strike"
    | "code"
    | "bulletList"
    | "orderedList"
    | "taskList"
    | "quote"
    | "codeBlock"
    | "divider";

export type TextEdit = {
    text: string;
    selectionStart: number;
    selectionEnd: number;
};

const headingPrefix = /^#{1,6} /;
const listPrefix = /^(\s*)([-*+] \[[ xX]\] |[-*+] |\d+\. )/;
const quotePrefix = /^> ?/;

const wrapMarkers: Partial<Record<MarkdownCommand, string>> = {
    bold: "**",
    italic: "*",
    strike: "~~",
    code: "`",
};

// Applies a formatting command to raw markdown the way the rich editor
// would: line commands toggle a prefix on every selected line, inline
// commands wrap or unwrap the selection.
export function applyMarkdownCommand(
    text: string,
    start: number,
    end: number,
    command: MarkdownCommand,
): TextEdit {
    const marker = wrapMarkers[command];
    if (marker) return toggleWrap(text, start, end, marker);

    switch (command) {
        case "h1":
        case "h2":
        case "h3": {
            const prefix = `${"#".repeat(Number(command[1]))} `;
            return editLines(text, start, end, (lines) => {
                const all = lines.every((line) => line.startsWith(prefix));
                return lines.map((line) => {
                    const bare = line.replace(headingPrefix, "");
                    return all ? bare : prefix + bare;
                });
            });
        }
        case "bulletList":
        case "orderedList":
        case "taskList": {
            const prefixFor = (i: number) =>
                command === "bulletList"
                    ? "- "
                    : command === "taskList"
                      ? "- [ ] "
                      : `${i + 1}. `;
            const matches = (line: string) => {
                const found = listPrefix.exec(line)?.[2];
                if (!found) return false;
                if (command === "orderedList") return /^\d+\. $/.test(found);
                if (command === "taskList") return found.includes("[");
                return !found.includes("[") && !/^\d/.test(found);
            };
            return editLines(text, start, end, (lines) => {
                const all = lines.every(matches);
                return lines.map((line, i) => {
                    const bare = line.replace(listPrefix, "$1");
                    if (all) return bare;
                    const indent = /^\s*/.exec(bare)?.[0] ?? "";
                    return indent + prefixFor(i) + bare.slice(indent.length);
                });
            });
        }
        case "quote":
            return editLines(text, start, end, (lines) => {
                const all = lines.every((line) => quotePrefix.test(line));
                return lines.map((line) =>
                    all ? line.replace(quotePrefix, "") : `> ${line}`,
                );
            });
        case "codeBlock": {
            const before = start > 0 && text[start - 1] !== "\n" ? "\n" : "";
            const open = `${before}\`\`\`\n`;
            const inserted = `${open}${text.slice(start, end)}\n\`\`\`\n`;
            const caret = start + open.length + (end - start);
            return {
                text: text.slice(0, start) + inserted + text.slice(end),
                selectionStart: start + open.length,
                selectionEnd: caret,
            };
        }
        case "divider": {
            const before = start > 0 && text[start - 1] !== "\n" ? "\n" : "";
            const inserted = `${before}\n---\n\n`;
            const caret = start + inserted.length;
            return {
                text: text.slice(0, start) + inserted + text.slice(end),
                selectionStart: caret,
                selectionEnd: caret,
            };
        }
        default:
            return { text, selectionStart: start, selectionEnd: end };
    }
}

function editLines(
    text: string,
    start: number,
    end: number,
    transform: (lines: string[]) => string[],
): TextEdit {
    const lineStart = text.lastIndexOf("\n", start - 1) + 1;
    // A selection ending right after a newline does not include the next line.
    const last = end > start && text[end - 1] === "\n" ? end - 1 : end;
    const found = text.indexOf("\n", last);
    const lineEnd = found === -1 ? text.length : found;

    const oldBlock = text.slice(lineStart, lineEnd);
    const lines = oldBlock.split("\n");
    const newLines = transform(lines);
    const newBlock = newLines.join("\n");
    const result = text.slice(0, lineStart) + newBlock + text.slice(lineEnd);

    if (lines.length === 1 && start === end) {
        const caret = Math.max(
            lineStart,
            start + newBlock.length - oldBlock.length,
        );
        return { text: result, selectionStart: caret, selectionEnd: caret };
    }
    return {
        text: result,
        selectionStart: lineStart,
        selectionEnd: lineStart + newBlock.length,
    };
}

function toggleWrap(
    text: string,
    start: number,
    end: number,
    marker: string,
): TextEdit {
    const m = marker.length;
    const selected = text.slice(start, end);

    // Markers just outside the selection: **|text|**
    if (
        text.slice(start - m, start) === marker &&
        text.slice(end, end + m) === marker
    ) {
        return {
            text: text.slice(0, start - m) + selected + text.slice(end + m),
            selectionStart: start - m,
            selectionEnd: end - m,
        };
    }
    // Markers inside the selection: |**text**|
    if (
        selected.length >= 2 * m &&
        selected.startsWith(marker) &&
        selected.endsWith(marker)
    ) {
        const inner = selected.slice(m, -m);
        return {
            text: text.slice(0, start) + inner + text.slice(end),
            selectionStart: start,
            selectionEnd: start + inner.length,
        };
    }
    return {
        text: text.slice(0, start) + marker + selected + marker + text.slice(end),
        selectionStart: start + m,
        selectionEnd: end + m,
    };
}

// Rough plain-text rendering of markdown for short previews (task cards).
export function markdownToPlainText(markdown: string): string {
    return markdown
        .replace(/^```.*$/gm, "")
        .replace(
            /^\s{0,3}(#{1,6}\s+|>\s?|[-*+]\s+\[[ xX]\]\s+|[-*+]\s+|\d+\.\s+)/gm,
            "",
        )
        .replace(/^\s*([-*_]\s*){3,}$/gm, "")
        .replace(/!?\[([^\]]*)\]\([^)]*\)/g, "$1")
        .replace(/\*\*|__|~~|`|\*/g, "")
        .replace(/\n{2,}/g, "\n")
        .trim();
}
