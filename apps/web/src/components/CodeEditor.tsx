import { useRef } from "react";
import { usePreferences } from "../hooks/usePreferences";
import { useHighlightedHtml } from "../hooks/useHighlightedHtml";

type CodeEditorProps = {
    value: string;
    onChange: (value: string) => void;
    // highlight.js language; null edits the text plain.
    language: string | null;
    readOnly: boolean;
    label: string;
};

// The text sits in a transparent textarea laid over an identically laid out
// highlighted copy, so the editing stays native (selection, undo, IME).
// Both layers must share every metric that affects layout.
const layer =
    "m-0 p-4 font-mono leading-[1.6] [tab-size:4] [scrollbar-gutter:stable]";

export function CodeEditor({ value, onChange, language, readOnly, label }: CodeEditorProps) {
    const { codeFontSize, codeWrap } = usePreferences();
    const html = useHighlightedHtml(value, language);
    const highlightRef = useRef<HTMLPreElement>(null);
    const wrap = codeWrap ? "whitespace-pre-wrap break-words" : "whitespace-pre";

    return (
        <div
            style={{ fontSize: codeFontSize }}
            className="relative min-h-0 flex-1 overflow-hidden bg-white"
        >
            {html !== null && (
                <pre
                    ref={highlightRef}
                    aria-hidden
                    className={`${layer} ${wrap} pointer-events-none absolute inset-0 overflow-auto text-gray-900 [scrollbar-color:transparent_transparent]`}
                >
                    {/* The newline keeps a final empty line as tall as the textarea's. */}
                    <code dangerouslySetInnerHTML={{ __html: `${html}\n` }} />
                </pre>
            )}
            <textarea
                aria-label={label}
                value={value}
                readOnly={readOnly}
                spellCheck={false}
                wrap={codeWrap ? "soft" : "off"}
                onChange={(event) => onChange(event.target.value)}
                onScroll={(event) => {
                    const pre = highlightRef.current;
                    if (!pre) return;
                    pre.scrollTop = event.currentTarget.scrollTop;
                    pre.scrollLeft = event.currentTarget.scrollLeft;
                }}
                onKeyDown={(event) => {
                    if (event.key !== "Tab" || event.shiftKey || readOnly) return;
                    event.preventDefault();
                    const el = event.currentTarget;
                    el.setRangeText("\t", el.selectionStart, el.selectionEnd, "end");
                    onChange(el.value);
                }}
                className={`${layer} ${wrap} absolute inset-0 size-full resize-none overflow-auto bg-transparent focus:outline-none ${
                    html === null ? "text-gray-900" : "text-transparent caret-gray-900"
                }`}
            />
        </div>
    );
}
