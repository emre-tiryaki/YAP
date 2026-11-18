import { lazy, Suspense } from "react";
import { FiDownload } from "react-icons/fi";
import { repoContentUrl } from "../api/client";
import { useRepoText } from "../hooks/useGitHub";
import { errorKey } from "../hooks/useResourceOperations";
import { useTranslate } from "../hooks/useTranslate";
import type { RepoEntry } from "../types/github";
import { fileCategory } from "../util/resources";
import { ResourceIcon } from "./ResourceIcon";

// TipTap is only needed for markdown.
const MarkdownView = lazy(() =>
    import("./MarkdownView").then((module) => ({ default: module.MarkdownView })),
);

type RepoFilePreviewProps = {
    resourceId: string;
    entry: RepoEntry;
    // The commit of the copy, so an update loads the new bytes.
    commit: string;
};

// Shows a file of a repository. Nothing here can be edited: the files come
// from GitHub.
export function RepoFilePreview({ resourceId, entry, commit }: RepoFilePreviewProps) {
    const t = useTranslate();
    const look = { kind: entry.kind, name: entry.name, mimeType: entry.mimeType, isText: entry.isText };
    const category = fileCategory(look);
    const url = repoContentUrl(resourceId, entry.path, commit);
    const downloadUrl = repoContentUrl(resourceId, entry.path, commit, true);

    switch (category) {
        case "markdown":
        case "text":
            return <RepoText resourceId={resourceId} path={entry.path} commit={commit} isMarkdown={category === "markdown"} />;
        case "image":
            return (
                <div className="flex min-h-0 flex-1 overflow-auto bg-[repeating-conic-gradient(#f3f4f6_0_25%,#fff_0_50%)] bg-[length:20px_20px] p-4">
                    <img src={url} alt={entry.name} className="m-auto max-h-full max-w-full object-contain" />
                </div>
            );
        case "pdf":
            return <iframe src={url} title={entry.name} className="min-h-0 w-full flex-1 border-0" />;
        case "audio":
            return (
                <div className="flex flex-1 items-center justify-center p-8">
                    <audio controls src={url} className="w-full max-w-lg" />
                </div>
            );
        case "video":
            return (
                <div className="flex min-h-0 flex-1 items-center justify-center bg-black/90 p-4">
                    <video controls src={url} className="max-h-full max-w-full" />
                </div>
            );
        default:
            return (
                <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
                    <ResourceIcon resource={look} className="text-7xl" />
                    <p className="text-sm text-gray-500">{t("resources.preview.unavailable")}</p>
                    <a
                        href={downloadUrl}
                        className="flex items-center gap-2 rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:bg-accent/85"
                    >
                        <FiDownload aria-hidden />
                        {t("resources.download")}
                    </a>
                </div>
            );
    }
}

function RepoText({ resourceId, path, commit, isMarkdown }: { resourceId: string; path: string; commit: string; isMarkdown: boolean }) {
    const t = useTranslate();
    const text = useRepoText(resourceId, path, commit);
    if (text.isLoading) return <p className="p-6 text-sm text-gray-400">{t("common.loading")}</p>;
    if (text.isError || text.data === undefined)
        return (
            <p role="alert" className="p-6 text-sm text-red-600">
                {t(errorKey(text.error))}
            </p>
        );
    if (isMarkdown)
        return (
            <div className="min-h-0 flex-1">
                <Suspense fallback={<p className="p-6 text-sm text-gray-400">{t("common.loading")}</p>}>
                    <MarkdownView markdown={text.data} />
                </Suspense>
            </div>
        );
    return (
        <pre className="min-h-0 flex-1 overflow-auto bg-gray-50 px-6 py-4 font-mono text-xs leading-5 text-gray-800">
            {text.data}
        </pre>
    );
}
