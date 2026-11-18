import { useState } from "react";
import {
    FiAlertTriangle,
    FiChevronRight,
    FiCopy,
    FiDownload,
    FiDownloadCloud,
    FiExternalLink,
    FiGitBranch,
    FiLock,
    FiRefreshCw,
    FiX,
} from "react-icons/fi";
import { useSearchParams } from "react-router";
import { repoContentUrl } from "../api/client";
import { useDateFormat } from "../hooks/useDateFormat";
import {
    useForgetRepositoryClone,
    useGitHubStatus,
    useRepoEntries,
    useRepository,
    useRepositoryClones,
    useSyncRepository,
} from "../hooks/useGitHub";
import { errorKey } from "../hooks/useResourceOperations";
import { useTranslate } from "../hooks/useTranslate";
import { useCurrentUserSettings } from "../hooks/useUserSettings";
import type { RepoEntry, Repository } from "../types/github";
import type { Resource } from "../types/resource";
import { formatSize } from "../util/resources";
import { CloneRepositoryDialog } from "./CloneRepositoryDialog";
import { CommitGraph } from "./CommitGraph";
import { LinkedTaskList } from "./LinkedTaskList";
import { RepoFilePreview } from "./RepoFilePreview";
import { ResourceActivity } from "./ResourceActivity";
import { ResourceIcon } from "./ResourceIcon";
import { ResourceMenu, type ResourceMenuItem } from "./ResourceMenu";
import { Tabs } from "./Tabs";

type Tab = "content" | "activity" | "tasks";

type RepositoryPaneProps = {
    resource: Resource;
    menu: ResourceMenuItem[];
    onLink: () => void;
};

function parentPath(path: string): string {
    const cut = path.lastIndexOf("/");
    return cut < 0 ? "" : path.slice(0, cut);
}

// A repository folder: its GitHub details, the state of this device's copy
// and a read-only browser over its files. The folder and file shown inside
// the repository live in the URL (?path= and ?file=).
export function RepositoryPane({ resource, menu, onLink }: RepositoryPaneProps) {
    const t = useTranslate();
    const format = useDateFormat();
    const repo = useRepository(resource.id);
    const sync = useSyncRepository();
    const github = useGitHubStatus();
    const gitAvailable = github.data?.gitAvailable ?? false;
    const [isCloning, setIsCloning] = useState(false);
    const [activity, setActivity] = useState<"commits" | "events">("commits");
    const [tab, setTab] = useState<Tab>("content");
    const [params, setParams] = useSearchParams();
    const file = params.get("file") ?? "";
    const dir = file ? parentPath(file) : (params.get("path") ?? "");

    const data = repo.data;
    const isFetching = data?.state === "fetching" || sync.isPending;
    const tabs: { id: Tab; label: string }[] = [
        { id: "content", label: t("resources.tab.content") },
        { id: "activity", label: t("resources.tab.activity") },
        { id: "tasks", label: t("resources.tab.tasks") },
    ];

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <header className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-gray-200 px-4 pt-3">
                <div className="min-w-0 pb-2">
                    <h1 className="flex min-w-0 items-center gap-2 text-lg font-semibold text-gray-900">
                        <ResourceIcon resource={resource} className="text-xl" />
                        <span className="truncate">{resource.name}</span>
                        <ResourceMenu items={menu} isSubtle={false} />
                        <span className="shrink-0 rounded bg-gray-100 px-1.5 py-0.5 text-xs font-normal text-gray-600">
                            {t("resources.repo.readOnly")}
                        </span>
                    </h1>
                    {data && (
                        <p className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-gray-500">
                            <span className="flex items-center gap-1">
                                {data.private && <FiLock aria-label={t("resources.repo.private")} />}
                                {data.fullName}
                            </span>
                            <span className="flex items-center gap-1" title={t("resources.repo.branch")}>
                                <FiGitBranch aria-hidden />
                                {data.branch}
                            </span>
                            {data.commitSha && <code className="font-mono">{data.commitSha.slice(0, 7)}</code>}
                            {data.fetchedAt && (
                                <span title={format.dateTime(data.fetchedAt)}>
                                    {t("resources.repo.updated").replace(
                                        "{time}",
                                        format.relative(data.fetchedAt) ?? format.dateTime(data.fetchedAt),
                                    )}
                                </span>
                            )}
                        </p>
                    )}
                </div>
                <span className="flex-1" />
                {data && (
                    <div className="mb-2 flex items-center gap-2">
                        <a
                            href={data.htmlUrl}
                            target="_blank"
                            rel="noreferrer"
                            className="flex items-center gap-1.5 rounded-md border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-100"
                        >
                            <FiExternalLink aria-hidden />
                            {t("resources.repo.openOnGitHub")}
                        </a>
                        <button
                            type="button"
                            disabled={!gitAvailable}
                            onClick={() => setIsCloning(true)}
                            title={gitAvailable ? t("resources.repo.clone.hint") : t("resources.error.gitMissing")}
                            className="flex cursor-pointer items-center gap-1.5 rounded-md border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-100 disabled:cursor-default disabled:opacity-50"
                        >
                            <FiDownloadCloud aria-hidden />
                            {t("resources.repo.clone")}
                        </button>
                        <CloneRepositoryDialog
                            isOpen={isCloning}
                            resourceId={resource.id}
                            defaultName={data.fullName.split("/").pop() ?? resource.name}
                            onClose={() => setIsCloning(false)}
                        />
                        <button
                            type="button"
                            disabled={isFetching}
                            onClick={() => sync.mutate(resource.id)}
                            className="flex cursor-pointer items-center gap-1.5 rounded-md border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-100 disabled:cursor-default disabled:opacity-60"
                        >
                            <FiRefreshCw aria-hidden className={isFetching ? "animate-spin" : ""} />
                            {t(isFetching ? "resources.repo.updating" : "resources.repo.update")}
                        </button>
                    </div>
                )}
                <Tabs value={tab} onChange={setTab} tabs={tabs} />
            </header>
            {sync.isError && (
                <p role="alert" className="border-b border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700">
                    {t(errorKey(sync.error))}
                </p>
            )}
            {data && <StatusBanner repo={data} />}
            <CloneList resourceId={resource.id} />
            {tab === "content" && (
                <>
                    {repo.isLoading && <p className="p-6 text-sm text-gray-400">{t("common.loading")}</p>}
                    {repo.isError && (
                        <p role="alert" className="p-6 text-sm text-red-600">
                            {t(errorKey(repo.error))}
                        </p>
                    )}
                    {data && !data.commitSha && (
                        <p className="p-6 text-sm text-gray-500">
                            {t(data.state === "fetching" ? "resources.repo.downloading" : "resources.repo.notReady")}
                        </p>
                    )}
                    {data?.commitSha && (
                        <Browser
                            resource={resource}
                            commit={data.commitSha}
                            dir={dir}
                            file={file}
                            onOpenDir={(path) => setParams(path ? { path } : {})}
                            onOpenFile={(path) => setParams({ file: path })}
                        />
                    )}
                </>
            )}
            {tab === "activity" && (
                <div className="flex-1 overflow-y-auto p-6">
                    <div role="radiogroup" aria-label={t("resources.tab.activity")} className="mb-4 inline-flex rounded-md bg-gray-100 p-0.5">
                        {(
                            [
                                ["commits", "resources.repo.commits"],
                                ["events", "resources.repo.events"],
                            ] as const
                        ).map(([value, label]) => (
                            <button
                                key={value}
                                type="button"
                                role="radio"
                                aria-checked={activity === value}
                                onClick={() => setActivity(value)}
                                className="cursor-pointer rounded px-3 py-1 text-sm text-gray-600 hover:text-gray-900 aria-checked:bg-white aria-checked:font-medium aria-checked:text-gray-900 aria-checked:shadow-sm"
                            >
                                {t(label)}
                            </button>
                        ))}
                    </div>
                    {activity === "commits" ? (
                        <CommitGraph resourceId={resource.id} commit={data?.commitSha ?? null} />
                    ) : (
                        <ResourceActivity scope={{ kind: "resource", id: resource.id }} />
                    )}
                </div>
            )}
            {tab === "tasks" && (
                <div className="flex-1 overflow-y-auto p-6">
                    <LinkedTaskList resourceId={resource.id} onLink={onLink} />
                </div>
            )}
        </div>
    );
}

// The clones made of this repository on this computer. YAP only remembers
// where they are; it never changes them.
function CloneList({ resourceId }: { resourceId: string }) {
    const t = useTranslate();
    const clones = useRepositoryClones(resourceId);
    const forget = useForgetRepositoryClone(resourceId);
    if (!clones.data || clones.data.length === 0) return null;
    return (
        <div className="flex flex-wrap items-center gap-2 border-b border-gray-200 px-4 py-2 text-xs text-gray-600">
            <span className="font-medium">{t("resources.repo.clones")}</span>
            {clones.data.map((c) => (
                <span key={c.id} className="flex min-w-0 items-center gap-1 rounded bg-gray-100 py-0.5 pr-1 pl-2">
                    <code className={`truncate font-mono ${c.exists ? "" : "text-gray-400 line-through"}`} title={c.path}>
                        {c.path}
                    </code>
                    {!c.exists && <span className="text-gray-400">({t("resources.repo.clones.missing")})</span>}
                    <button
                        type="button"
                        onClick={() => void navigator.clipboard.writeText(c.path)}
                        aria-label={t("resources.repo.clones.copy")}
                        title={t("resources.repo.clones.copy")}
                        className="cursor-pointer rounded p-0.5 hover:bg-gray-200"
                    >
                        <FiCopy aria-hidden />
                    </button>
                    <button
                        type="button"
                        disabled={forget.isPending}
                        onClick={() => forget.mutate(c.id)}
                        aria-label={t("resources.repo.clones.forget")}
                        title={t("resources.repo.clones.forget")}
                        className="cursor-pointer rounded p-0.5 hover:bg-gray-200"
                    >
                        <FiX aria-hidden />
                    </button>
                </span>
            ))}
        </div>
    );
}

// Says when the copy may be out of date or why it could not be fetched.
function StatusBanner({ repo }: { repo: Repository }) {
    const t = useTranslate();
    const format = useDateFormat();
    const settings = useCurrentUserSettings();

    if (repo.state === "fetching")
        return (
            <p role="status" className="flex items-center gap-2 border-b border-gray-200 bg-gray-50 px-4 py-2 text-sm text-gray-600">
                <FiRefreshCw aria-hidden className="animate-spin" />
                {t(repo.commitSha ? "resources.repo.updating" : "resources.repo.downloading")}
            </p>
        );
    if (repo.error === "offline") {
        const message = !repo.commitSha
            ? t("resources.repo.waitingOffline")
            : repo.checkedAt
              ? t("resources.repo.outdated").replace("{time}", format.dateTime(repo.checkedAt))
              : t("resources.repo.outdatedNever");
        return (
            <p role="status" className="flex items-center gap-2 border-b border-amber-200 bg-amber-50 px-4 py-2 text-sm text-amber-800">
                <FiAlertTriangle aria-hidden className="shrink-0" />
                {message}
            </p>
        );
    }
    if (repo.error)
        return (
            <p role="alert" className="flex items-center gap-2 border-b border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700">
                <FiAlertTriangle aria-hidden className="shrink-0" />
                {t(`resources.repo.error.${repo.error}`)}
            </p>
        );
    if (!settings.repositoriesAutoUpdate)
        return (
            <p className="border-b border-gray-200 bg-gray-50 px-4 py-2 text-xs text-gray-500">
                {t("resources.repo.autoUpdateOff")}
            </p>
        );
    return null;
}

type BrowserProps = {
    resource: Resource;
    commit: string;
    dir: string;
    file: string;
    onOpenDir: (path: string) => void;
    onOpenFile: (path: string) => void;
};

function Browser({ resource, commit, dir, file, onOpenDir, onOpenFile }: BrowserProps) {
    const t = useTranslate();
    const entries = useRepoEntries(resource.id, dir, commit);
    const segments = dir ? dir.split("/") : [];
    const opened = file ? entries.data?.find((e) => e.path === file && e.kind === "file") : undefined;

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <nav aria-label={resource.name} className="flex flex-wrap items-center gap-1 border-b border-gray-100 px-4 py-2 text-sm">
                <button type="button" onClick={() => onOpenDir("")} className="cursor-pointer rounded px-1 text-gray-700 hover:bg-gray-100">
                    {resource.name}
                </button>
                {segments.map((seg, i) => (
                    <span key={segments.slice(0, i + 1).join("/")} className="flex items-center gap-1">
                        <FiChevronRight aria-hidden className="text-gray-400" />
                        <button
                            type="button"
                            onClick={() => onOpenDir(segments.slice(0, i + 1).join("/"))}
                            className="cursor-pointer rounded px-1 text-gray-700 hover:bg-gray-100"
                        >
                            {seg}
                        </button>
                    </span>
                ))}
                {opened && (
                    <span className="flex items-center gap-1">
                        <FiChevronRight aria-hidden className="text-gray-400" />
                        <span className="px-1 font-medium text-gray-900">{opened.name}</span>
                        <span className="text-xs text-gray-400">{formatSize(opened.size)}</span>
                        <a
                            href={repoContentUrl(resource.id, opened.path, commit, true)}
                            aria-label={t("resources.download")}
                            title={t("resources.download")}
                            className="ml-1 rounded p-1 text-gray-500 hover:bg-gray-100"
                        >
                            <FiDownload aria-hidden />
                        </a>
                    </span>
                )}
            </nav>
            {entries.isLoading && <p className="p-6 text-sm text-gray-400">{t("common.loading")}</p>}
            {entries.isError && (
                <p role="alert" className="p-6 text-sm text-red-600">
                    {t(errorKey(entries.error))}
                </p>
            )}
            {entries.isSuccess && file && !opened && <p className="p-6 text-sm text-gray-500">{t("resources.error.notFound")}</p>}
            {opened && <RepoFilePreview key={opened.path} resourceId={resource.id} entry={opened} commit={commit} />}
            {entries.isSuccess && !file && <EntryTable entries={entries.data} onOpenDir={onOpenDir} onOpenFile={onOpenFile} />}
        </div>
    );
}

function EntryTable({
    entries,
    onOpenDir,
    onOpenFile,
}: {
    entries: RepoEntry[];
    onOpenDir: (path: string) => void;
    onOpenFile: (path: string) => void;
}) {
    const t = useTranslate();
    const format = useDateFormat();
    if (entries.length === 0) return <p className="p-6 text-sm text-gray-400">{t("resources.repo.empty")}</p>;
    return (
        <div className="min-h-0 flex-1 overflow-y-auto p-2">
            <table className="w-full table-fixed text-left text-sm">
                <thead className="border-b border-gray-200 text-xs text-gray-500">
                    <tr>
                        <th scope="col" className="px-3 py-2 font-medium">{t("resources.column.name")}</th>
                        <th scope="col" className="w-28 px-3 py-2 font-medium">{t("resources.column.size")}</th>
                        <th scope="col" className="w-44 px-3 py-2 font-medium">{t("resources.column.modified")}</th>
                    </tr>
                </thead>
                <tbody>
                    {entries.map((e) => (
                        <tr key={e.path} className="hover:bg-gray-50">
                            <td className="px-3 py-1.5">
                                <button
                                    type="button"
                                    onClick={() => (e.kind === "folder" ? onOpenDir(e.path) : onOpenFile(e.path))}
                                    className="flex w-full min-w-0 cursor-pointer items-center gap-2 text-left"
                                >
                                    <ResourceIcon
                                        resource={{ kind: e.kind, name: e.name, mimeType: e.mimeType, isText: e.isText }}
                                        className="text-base"
                                    />
                                    <span className="truncate" title={e.name}>
                                        {e.name}
                                    </span>
                                </button>
                            </td>
                            <td className="px-3 py-1.5 text-gray-500">{e.kind === "file" ? formatSize(e.size) : ""}</td>
                            <td className="px-3 py-1.5 text-gray-500">{format.dateTime(e.modifiedAt)}</td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}
