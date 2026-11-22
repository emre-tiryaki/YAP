import { usePreferences } from "../hooks/usePreferences";
import { useHighlightedHtml } from "../hooks/useHighlightedHtml";

type CodeViewProps = {
    text: string;
    // highlight.js language; null shows the text plain.
    language: string | null;
};

// Read-only, syntax-highlighted source text.
export function CodeView({ text, language }: CodeViewProps) {
    const { codeFontSize, codeWrap } = usePreferences();
    const html = useHighlightedHtml(text, language);
    return (
        <pre
            style={{ fontSize: codeFontSize }}
            className={`m-0 min-h-0 flex-1 overflow-auto bg-gray-50 px-6 py-4 font-mono leading-[1.6] text-gray-900 [tab-size:4] ${
                codeWrap ? "whitespace-pre-wrap break-words" : "whitespace-pre"
            }`}
        >
            {html === null ? (
                text
            ) : (
                <code dangerouslySetInnerHTML={{ __html: html }} />
            )}
        </pre>
    );
}
