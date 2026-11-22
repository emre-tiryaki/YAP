// Language ids are highlight.js names; the loaders live in util/highlight.ts.
const byExtension: Record<string, string> = {
    js: "javascript", mjs: "javascript", cjs: "javascript", jsx: "javascript",
    ts: "typescript", mts: "typescript", cts: "typescript", tsx: "typescript",
    json: "json", jsonc: "json", json5: "json", webmanifest: "json",
    html: "xml", htm: "xml", xml: "xml", svg: "xml", xhtml: "xml", vue: "xml", svelte: "xml",
    css: "css", scss: "scss", sass: "scss", less: "less",
    md: "markdown", markdown: "markdown",
    py: "python", pyw: "python",
    go: "go",
    rs: "rust",
    java: "java",
    kt: "kotlin", kts: "kotlin",
    swift: "swift",
    c: "c", h: "c",
    cc: "cpp", cpp: "cpp", cxx: "cpp", hpp: "cpp", hh: "cpp",
    cs: "csharp",
    rb: "ruby", rake: "ruby", gemspec: "ruby",
    php: "php",
    lua: "lua",
    pl: "perl", pm: "perl",
    r: "r",
    scala: "scala", sc: "scala",
    dart: "dart",
    sh: "bash", bash: "bash", zsh: "bash", fish: "bash",
    ps1: "powershell", psm1: "powershell",
    sql: "sql",
    yml: "yaml", yaml: "yaml",
    toml: "ini", ini: "ini", cfg: "ini", conf: "ini", properties: "ini", env: "ini",
    diff: "diff", patch: "diff",
    graphql: "graphql", gql: "graphql",
    proto: "protobuf",
    dockerfile: "dockerfile",
    mk: "makefile",
    nginx: "nginx",
};

const byFilename: Record<string, string> = {
    dockerfile: "dockerfile",
    makefile: "makefile",
    gnumakefile: "makefile",
    ".bashrc": "bash",
    ".zshrc": "bash",
    ".profile": "bash",
    ".gitignore": "ini",
    ".dockerignore": "ini",
    ".editorconfig": "ini",
    ".env": "ini",
    gemfile: "ruby",
    rakefile: "ruby",
};

// The highlight.js language of a file, or null when it is not source code
// we know (plain text stays unhighlighted).
export function codeLanguage(name: string): string | null {
    const lower = name.toLowerCase();
    const byName = byFilename[lower];
    if (byName) return byName;
    const dot = lower.lastIndexOf(".");
    if (dot < 0) return null;
    return byExtension[lower.slice(dot + 1)] ?? null;
}
