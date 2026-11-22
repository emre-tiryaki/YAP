import { lazy, Suspense, useEffect, useRef, useState, type KeyboardEvent } from "react";
import { ApiError, getResource, saveResourceText } from "../api/client";
import { useResourceText, useSaveResourceText } from "../hooks/useResources";
import { useTranslate } from "../hooks/useTranslate";
import { defaultUserSettings, useUserSettings } from "../hooks/useUserSettings";
import type { Resource } from "../types/resource";
import { codeLanguage } from "../util/codeLanguage";
import { Button } from "./Button";
import { CodeEditor } from "./CodeEditor";

// TipTap is only needed for the markdown preview.
const MarkdownView = lazy(() =>
    import("./MarkdownView").then((module) => ({ default: module.MarkdownView })),
);

const MD_MODE_KEY = "yap.resources.markdownMode";

type TextFileEditorProps = {
    resource: Resource;
    isMarkdown: boolean;
    readOnly: boolean;
};

// Loads a text file and hands it to the editor. Reloading after a conflict
// loads the newest revision into a fresh editor.
export function TextFileEditor({ resource, isMarkdown, readOnly }: TextFileEditorProps) {
    const t = useTranslate();
    // The revision the editor started from; later saves bump it inside the
    // editor without reloading the text.
    const [base, setBase] = useState(resource);
    const text = useResourceText(base);

    if (text.isLoading) return <p className="p-6 text-sm text-gray-400">{t("common.loading")}</p>;
    if (text.isError || text.data === undefined)
        return (
            <p role="alert" className="p-6 text-sm text-red-600">
                {text.error instanceof ApiError && text.error.message === "file_missing"
                    ? t("resources.error.missing")
                    : t("common.error")}
            </p>
        );
    return (
        <EditorBody
            key={`${base.id}-${base.revision}`}
            resource={base}
            initialText={text.data}
            isMarkdown={isMarkdown}
            readOnly={readOnly}
            onReload={async () => setBase(await getResource(base.id))}
        />
    );
}

type Status = "saved" | "dirty" | "saving" | "conflict" | "error";

type EditorBodyProps = {
    resource: Resource;
    initialText: string;
    isMarkdown: boolean;
    readOnly: boolean;
    onReload: () => Promise<void>;
};

function EditorBody({ resource, initialText, isMarkdown, readOnly, onReload }: EditorBodyProps) {
    const t = useTranslate();
    const settings = useUserSettings().data ?? defaultUserSettings;
    const save = useSaveResourceText();
    const [text, setText] = useState(initialText);
    const [savedText, setSavedText] = useState(initialText);
    const [revision, setRevision] = useState(resource.revision);
    const [status, setStatus] = useState<Status>("saved");
    const [mode, setMode] = useState<"source" | "preview">(() =>
        isMarkdown && localStorage.getItem(MD_MODE_KEY) !== "source" ? "preview" : "source",
    );
    const isDirty = text !== savedText;

    // The latest state for the unmount flush and the unload warning.
    const latest = useRef({ text, isDirty, revision, status });
    useEffect(() => {
        latest.current = { text, isDirty, revision, status };
    });

    async function saveNow(force = false) {
        if (readOnly || save.isPending) return;
        const sent = text;
        if (!force && sent === savedText) return;
        setStatus("saving");
        try {
            const rev = force ? (await getResource(resource.id)).revision : revision;
            const saved = await save.mutateAsync({ id: resource.id, revision: rev, text: sent });
            setRevision(saved.revision);
            setSavedText(sent);
            setStatus(latest.current.text === sent ? "saved" : "dirty");
        } catch (error) {
            setStatus(error instanceof ApiError && error.message === "stale" ? "conflict" : "error");
        }
    }

    // Autosave after a pause in typing. A conflict stops it until resolved.
    useEffect(() => {
        if (!isDirty || readOnly || !settings.autosaveEnabled) return;
        if (status === "conflict" || status === "saving") return;
        const timer = window.setTimeout(() => void saveNow(), settings.autosaveDelayMs);
        return () => window.clearTimeout(timer);
    });

    useEffect(() => {
        function handleBeforeUnload(event: BeforeUnloadEvent) {
            if (latest.current.isDirty) event.preventDefault();
        }
        window.addEventListener("beforeunload", handleBeforeUnload);
        return () => {
            window.removeEventListener("beforeunload", handleBeforeUnload);
            // Leaving the file (another file opened, page left) keeps the
            // unsaved text instead of dropping it, unless it would overwrite
            // someone else's change.
            const { isDirty, text, revision, status } = latest.current;
            if (isDirty && status !== "conflict" && !readOnly)
                void saveResourceText(resource.id, revision, text).catch(() => undefined);
        };
    }, [resource.id, readOnly]);

    function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
        if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "s") {
            event.preventDefault();
            void saveNow();
        }
    }

    function changeMode(next: "source" | "preview") {
        setMode(next);
        localStorage.setItem(MD_MODE_KEY, next);
    }

    const statusText: Record<Status, string> = {
        saved: t("resources.editor.saved"),
        dirty: settings.autosaveEnabled ? t("resources.editor.unsaved") : t("resources.editor.unsavedManual"),
        saving: t("resources.editor.saving"),
        conflict: t("resources.editor.conflict"),
        error: t("resources.editor.error"),
    };
    const shownStatus: Status = status === "saved" && isDirty ? "dirty" : status;

    return (
        <div onKeyDown={handleKeyDown} className="flex min-h-0 flex-1 flex-col">
            <div className="flex items-center gap-3 border-b border-gray-200 px-4 py-2 text-sm">
                {isMarkdown && (
                    <div role="radiogroup" aria-label={t("resources.editor.mode")} className="flex rounded-md bg-gray-100 p-0.5">
                        {(["source", "preview"] as const).map((m) => (
                            <button
                                key={m}
                                type="button"
                                role="radio"
                                aria-checked={mode === m}
                                onClick={() => changeMode(m)}
                                className="cursor-pointer rounded px-3 py-0.5 text-gray-600 aria-checked:bg-white aria-checked:font-medium aria-checked:text-gray-900 aria-checked:shadow-sm"
                            >
                                {t(m === "source" ? "resources.editor.source" : "resources.editor.preview")}
                            </button>
                        ))}
                    </div>
                )}
                {readOnly ? (
                    <span className="text-gray-500">{t("resources.editor.readOnly")}</span>
                ) : (
                    <span
                        role="status"
                        className={shownStatus === "conflict" || shownStatus === "error" ? "text-red-600" : "text-gray-500"}
                    >
                        {statusText[shownStatus]}
                    </span>
                )}
                <span className="flex-1" />
                {!readOnly && (
                    <Button variant="primary" size="md"
                        type="button"
                        onClick={() => void saveNow()}
                        disabled={!isDirty || status === "saving" || status === "conflict"}
                        title="Ctrl+S"
                    >
                        {t("common.save")}
                    </Button>
                )}
            </div>
            {status === "conflict" && (
                <div role="alert" className="flex flex-wrap items-center gap-3 border-b border-amber-200 bg-amber-50 px-4 py-2 text-sm text-amber-800">
                    <span className="flex-1">{t("resources.editor.conflictHint")}</span>
                    <button
                        type="button"
                        onClick={() => void onReload()}
                        className="cursor-pointer rounded-md border border-amber-300 bg-white px-3 py-1 hover:bg-amber-100"
                    >
                        {t("resources.editor.reload")}
                    </button>
                    <button
                        type="button"
                        onClick={() => void saveNow(true)}
                        className="cursor-pointer rounded-md bg-amber-600 px-3 py-1 text-white hover:bg-amber-700"
                    >
                        {t("resources.editor.overwrite")}
                    </button>
                </div>
            )}
            {mode === "preview" ? (
                <Suspense fallback={<p className="p-6 text-sm text-gray-400">{t("common.loading")}</p>}>
                    <MarkdownView markdown={text} />
                </Suspense>
            ) : (
                <CodeEditor
                    label={resource.name}
                    value={text}
                    readOnly={readOnly}
                    language={codeLanguage(resource.name)}
                    onChange={(value) => {
                        setText(value);
                        if (status === "error") setStatus("dirty");
                    }}
                />
            )}
        </div>
    );
}
