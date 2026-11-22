import hljs from "highlight.js/lib/core";

// Each language is its own chunk, loaded the first time a file needs it.
const loaders: Record<string, () => Promise<{ default: Parameters<typeof hljs.registerLanguage>[1] }>> = {
    bash: () => import("highlight.js/lib/languages/bash"),
    c: () => import("highlight.js/lib/languages/c"),
    cpp: () => import("highlight.js/lib/languages/cpp"),
    csharp: () => import("highlight.js/lib/languages/csharp"),
    css: () => import("highlight.js/lib/languages/css"),
    dart: () => import("highlight.js/lib/languages/dart"),
    diff: () => import("highlight.js/lib/languages/diff"),
    dockerfile: () => import("highlight.js/lib/languages/dockerfile"),
    go: () => import("highlight.js/lib/languages/go"),
    graphql: () => import("highlight.js/lib/languages/graphql"),
    ini: () => import("highlight.js/lib/languages/ini"),
    java: () => import("highlight.js/lib/languages/java"),
    javascript: () => import("highlight.js/lib/languages/javascript"),
    json: () => import("highlight.js/lib/languages/json"),
    kotlin: () => import("highlight.js/lib/languages/kotlin"),
    less: () => import("highlight.js/lib/languages/less"),
    lua: () => import("highlight.js/lib/languages/lua"),
    makefile: () => import("highlight.js/lib/languages/makefile"),
    markdown: () => import("highlight.js/lib/languages/markdown"),
    nginx: () => import("highlight.js/lib/languages/nginx"),
    perl: () => import("highlight.js/lib/languages/perl"),
    php: () => import("highlight.js/lib/languages/php"),
    powershell: () => import("highlight.js/lib/languages/powershell"),
    protobuf: () => import("highlight.js/lib/languages/protobuf"),
    python: () => import("highlight.js/lib/languages/python"),
    r: () => import("highlight.js/lib/languages/r"),
    ruby: () => import("highlight.js/lib/languages/ruby"),
    rust: () => import("highlight.js/lib/languages/rust"),
    scala: () => import("highlight.js/lib/languages/scala"),
    scss: () => import("highlight.js/lib/languages/scss"),
    sql: () => import("highlight.js/lib/languages/sql"),
    swift: () => import("highlight.js/lib/languages/swift"),
    typescript: () => import("highlight.js/lib/languages/typescript"),
    xml: () => import("highlight.js/lib/languages/xml"),
    yaml: () => import("highlight.js/lib/languages/yaml"),
};

// Highlighting a huge file would freeze typing; past this it stays plain.
export const MAX_HIGHLIGHT_CHARS = 150_000;

const loading = new Map<string, Promise<boolean>>();

// Registers the language; resolves false when there is no such language.
export function loadLanguage(language: string): Promise<boolean> {
    if (hljs.getLanguage(language)) return Promise.resolve(true);
    const load = loaders[language];
    if (!load) return Promise.resolve(false);
    let pending = loading.get(language);
    if (!pending) {
        pending = load()
            .then((module) => {
                hljs.registerLanguage(language, module.default);
                return true;
            })
            .catch(() => {
                loading.delete(language);
                return false;
            });
        loading.set(language, pending);
    }
    return pending;
}

// The text as HTML with token spans; the language must be loaded.
export function highlightToHtml(text: string, language: string): string {
    return hljs.highlight(text, { language, ignoreIllegals: true }).value;
}
