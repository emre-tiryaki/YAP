import { useState } from "react";
import { FiChevronRight, FiFile, FiFolder, FiGithub } from "react-icons/fi";
import { useRepoEntries, useRepository } from "../hooks/useGitHub";
import { useResources } from "../hooks/useResources";
import { useTranslate } from "../hooks/useTranslate";
import { Button } from "./Button";
import { Dialog } from "./Dialog";
import { InlineError } from "./InlineError";

type RepoFilePickerDialogProps = {
    // The repository's resource id, the file's path and a display label
    // ("owner/repo/path").
    onPick: (resourceId: string, path: string, label: string) => void;
    onClose: () => void;
};

const rowClass =
    "flex h-8 w-full cursor-pointer items-center gap-2 rounded-md px-2 text-left text-body-xs hover:bg-layer-1-hover disabled:cursor-default disabled:text-fg-placeholder disabled:hover:bg-transparent";

// Step 1 picks a repository, step 2 browses it for a text file.
export function RepoFilePickerDialog({
    onPick,
    onClose,
}: RepoFilePickerDialogProps) {
    const t = useTranslate();
    const resources = useResources();
    const [resourceId, setResourceId] = useState("");
    const repositories = (resources.data ?? []).filter(
        (r) => r.isRepository && r.trashedAt === null,
    );

    return (
        <Dialog
            open
            onClose={onClose}
            title={t("assistant.picker.repo")}
            footer={
                <Button size="lg" onClick={onClose}>
                    {t("common.cancel")}
                </Button>
            }
        >
            {resourceId === "" ? (
                <div className="flex flex-col gap-1">
                    <p className="mb-1 text-caption text-fg-muted">
                        {t("assistant.picker.chooseRepo")}
                    </p>
                    {resources.isError ? (
                        <InlineError
                            message={t("common.error")}
                            onRetry={() => void resources.refetch()}
                        />
                    ) : repositories.length === 0 ? (
                        <p className="text-body-xs text-fg-subtle">
                            {resources.isPending
                                ? t("common.loading")
                                : t("assistant.picker.noRepos")}
                        </p>
                    ) : (
                        repositories.map((r) => (
                            <button
                                key={r.id}
                                type="button"
                                className={rowClass}
                                onClick={() => setResourceId(r.id)}
                            >
                                <FiGithub aria-hidden className="text-fg-subtle" />
                                <span className="truncate">{r.name}</span>
                            </button>
                        ))
                    )}
                </div>
            ) : (
                <RepoBrowser
                    resourceId={resourceId}
                    onBack={() => setResourceId("")}
                    onPick={onPick}
                />
            )}
        </Dialog>
    );
}

type RepoBrowserProps = {
    resourceId: string;
    onBack: () => void;
    onPick: (resourceId: string, path: string, label: string) => void;
};

function RepoBrowser({ resourceId, onBack, onPick }: RepoBrowserProps) {
    const t = useTranslate();
    const repository = useRepository(resourceId);
    const [path, setPath] = useState("");
    const commit = repository.data?.commitSha ?? null;
    const entries = useRepoEntries(resourceId, path, commit);
    const fullName = repository.data?.fullName ?? "";
    const segments = path === "" ? [] : path.split("/");

    return (
        <div className="flex flex-col gap-2">
            <nav
                aria-label={fullName}
                className="flex flex-wrap items-center gap-1 text-caption text-fg-muted"
            >
                <button
                    type="button"
                    onClick={onBack}
                    className="cursor-pointer underline hover:text-fg"
                >
                    {t("assistant.picker.chooseRepo")}
                </button>
                <FiChevronRight aria-hidden />
                <button
                    type="button"
                    onClick={() => setPath("")}
                    className="cursor-pointer hover:text-fg"
                >
                    {fullName}
                </button>
                {segments.map((segment, index) => (
                    <span key={segment + index} className="flex items-center gap-1">
                        <FiChevronRight aria-hidden />
                        <button
                            type="button"
                            onClick={() =>
                                setPath(segments.slice(0, index + 1).join("/"))
                            }
                            className="cursor-pointer hover:text-fg"
                        >
                            {segment}
                        </button>
                    </span>
                ))}
            </nav>
            <div className="max-h-80 overflow-y-auto">
                {commit === null ? (
                    <p className="text-body-xs text-fg-subtle">
                        {repository.isPending
                            ? t("common.loading")
                            : t("resources.repo.notReady")}
                    </p>
                ) : entries.isError ? (
                    <InlineError
                        message={t("common.error")}
                        onRetry={() => void entries.refetch()}
                    />
                ) : entries.isPending ? (
                    <p className="text-body-xs text-fg-subtle">
                        {t("common.loading")}
                    </p>
                ) : entries.data.length === 0 ? (
                    <p className="text-body-xs text-fg-subtle">
                        {t("assistant.picker.noFiles")}
                    </p>
                ) : (
                    entries.data.map((entry) =>
                        entry.kind === "folder" ? (
                            <button
                                key={entry.path}
                                type="button"
                                className={rowClass}
                                onClick={() => setPath(entry.path)}
                            >
                                <FiFolder aria-hidden className="text-fg-subtle" />
                                <span className="truncate">{entry.name}</span>
                            </button>
                        ) : (
                            <button
                                key={entry.path}
                                type="button"
                                className={rowClass}
                                disabled={!entry.isText}
                                onClick={() =>
                                    onPick(
                                        resourceId,
                                        entry.path,
                                        `${fullName}/${entry.path}`,
                                    )
                                }
                            >
                                <FiFile aria-hidden className="text-fg-subtle" />
                                <span className="truncate">{entry.name}</span>
                            </button>
                        ),
                    )
                )}
            </div>
        </div>
    );
}
