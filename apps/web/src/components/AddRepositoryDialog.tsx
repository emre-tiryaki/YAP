import { useState, type FormEvent } from "react";
import { FiGithub, FiLock } from "react-icons/fi";
import { ApiError } from "../api/client";
import { useAddRepository, useGitHubRepos, useGitHubStatus } from "../hooks/useGitHub";
import { errorKey } from "../hooks/useResourceOperations";
import { useTranslate } from "../hooks/useTranslate";
import type { Resource } from "../types/resource";
import { Modal } from "./Modal";

type AddRepositoryDialogProps = {
    isOpen: boolean;
    // The folder the repository goes into; null is the top.
    parentId: string | null;
    onClose: () => void;
    onAdded: (resource: Resource) => void;
};

export function AddRepositoryDialog({ isOpen, parentId, onClose, onAdded }: AddRepositoryDialogProps) {
    const t = useTranslate();
    return (
        <Modal isOpen={isOpen} title={t("resources.repo.dialogTitle")} onClose={onClose}>
            {isOpen && <AddRepositoryForm parentId={parentId} onAdded={onAdded} />}
        </Modal>
    );
}

// "owner/name" as GitHub allows it.
const FULL_NAME = /^[\w.-]+\/[\w.-]+$/;

function AddRepositoryForm({ parentId, onAdded }: { parentId: string | null; onAdded: (resource: Resource) => void }) {
    const t = useTranslate();
    const status = useGitHubStatus();
    const connected = status.data?.account != null;
    const repos = useGitHubRepos(connected);
    const add = useAddRepository();
    const [query, setQuery] = useState("");
    const [selected, setSelected] = useState("");

    if (status.isLoading) return <p className="text-sm text-gray-400">{t("common.loading")}</p>;
    if (status.isError) return <p role="alert" className="text-sm text-red-600">{t(errorKey(status.error))}</p>;
    if (!status.data?.configured) return <p className="text-sm text-gray-600">{t("settings.github.notConfigured")}</p>;
    if (!connected) return <p className="text-sm text-gray-600">{t("resources.repo.notConnected")}</p>;

    const needle = query.trim().toLowerCase();
    const matches = (repos.data ?? []).filter((r) => r.fullName.toLowerCase().includes(needle)).slice(0, 50);
    // A typed "owner/name" can be added even when it is not in the list
    // (a public repository of someone else).
    const fullName = selected || (FULL_NAME.test(query.trim()) ? query.trim() : "");

    function submit(event: FormEvent) {
        event.preventDefault();
        if (!fullName) return;
        add.mutate({ parentId, fullName }, { onSuccess: onAdded });
    }

    return (
        <form onSubmit={submit} className="flex flex-col gap-4">
            <input
                type="search"
                autoFocus
                value={query}
                onChange={(event) => {
                    setQuery(event.target.value);
                    setSelected("");
                }}
                placeholder={t("resources.repo.search")}
                aria-label={t("resources.repo.search")}
                className="rounded-md border border-gray-300 px-3 py-2 text-sm"
            />
            <div role="listbox" aria-label={t("resources.repo.dialogTitle")} className="h-56 overflow-y-auto rounded-md border border-gray-200">
                {repos.isLoading && <p className="p-3 text-sm text-gray-400">{t("resources.repo.loading")}</p>}
                {repos.isError && <p role="alert" className="p-3 text-sm text-red-600">{t(errorKey(repos.error))}</p>}
                {repos.isSuccess && matches.length === 0 && (
                    <p className="p-3 text-sm text-gray-400">{t("resources.repo.noMatch")}</p>
                )}
                {matches.map((repo) => (
                    <button
                        key={repo.id}
                        type="button"
                        role="option"
                        aria-selected={selected === repo.fullName}
                        onClick={() => setSelected(repo.fullName)}
                        className="flex w-full cursor-pointer flex-col items-start gap-0.5 border-b border-gray-100 px-3 py-2 text-left last:border-b-0 hover:bg-gray-50 aria-selected:bg-accent/10"
                    >
                        <span className="flex w-full items-center gap-2 text-sm font-medium text-gray-900">
                            <FiGithub aria-hidden className="shrink-0 text-gray-500" />
                            <span className="truncate">{repo.fullName}</span>
                            {repo.private && (
                                <span className="ml-auto flex shrink-0 items-center gap-1 rounded bg-gray-100 px-1.5 text-xs text-gray-600">
                                    <FiLock aria-hidden />
                                    {t("resources.repo.private")}
                                </span>
                            )}
                        </span>
                        {repo.description && <span className="line-clamp-1 text-xs text-gray-500">{repo.description}</span>}
                    </button>
                ))}
            </div>
            {add.isError && (
                <p role="alert" className="text-sm text-red-600">
                    {add.error instanceof ApiError && add.error.message === "not_found"
                        ? t("resources.repo.error.not_found")
                        : t(errorKey(add.error))}
                </p>
            )}
            <div className="flex justify-end">
                <button
                    type="submit"
                    disabled={!fullName || add.isPending}
                    className="cursor-pointer rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg hover:bg-accent/85 disabled:cursor-default disabled:opacity-50"
                >
                    {t("resources.repo.add")}
                </button>
            </div>
        </form>
    );
}
