import { useEffect, useMemo, useState } from "react";
import { MAX_HIGHLIGHT_CHARS, highlightToHtml, loadLanguage } from "../util/highlight";

// The text highlighted as HTML, or null while the language loads, when the
// language is unknown or when the text is too big. Callers show plain text
// for null.
export function useHighlightedHtml(text: string, language: string | null): string | null {
    const [loaded, setLoaded] = useState<string | null>(null);

    useEffect(() => {
        if (language === null) return;
        let cancelled = false;
        void loadLanguage(language).then((ok) => {
            if (!cancelled) setLoaded(ok ? language : null);
        });
        return () => {
            cancelled = true;
        };
    }, [language]);

    return useMemo(() => {
        if (language === null || loaded !== language) return null;
        if (text.length > MAX_HIGHLIGHT_CHARS) return null;
        return highlightToHtml(text, language);
    }, [text, language, loaded]);
}
