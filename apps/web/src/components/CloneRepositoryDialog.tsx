import { useState, type FormEvent } from "react";
import { FiArrowUp, FiCheck, FiFolder, FiHome } from "react-icons/fi";
import { ApiError } from "../api/client";
import { useCloneRepository, useLocalFolders } from "../hooks/useGitHub";
import { errorKey } from "../hooks/useResourceOperations";
import { useTranslate } from "../hooks/useTranslate";
import { Modal } from "./Modal";
import { Button } from "./Button";

type CloneRepositoryDialogProps = {
    isOpen: boolean;
    resourceId: string;
    // The folder name suggested for the clone.
    defaultName: string;
    onClose: () => void;
};

export function CloneRepositoryDialog({ isOpen, resourceId, defaultName, onClose }: CloneRepositoryDialogProps) {
    const t = useTranslate();
    return (
        <Modal isOpen={isOpen} title={t("resources.repo.clone.title")} onClose={onClose}>
            {isOpen && <CloneForm resourceId={resourceId} defaultName={defaultName} onClose={onClose} />}
        </Modal>
    );
}

// Joins a folder name to an absolute path the way the computer writes paths.
function childPath(parent: string, name: string): string {
    const sep = parent.includes("\\") && !parent.includes("/") ? "\\" : "/";
    return parent.endsWith(sep) ? parent + name : parent + sep + name;
}

function CloneForm({ resourceId, defaultName, onClose }: { resourceId: string; defaultName: string; onClose: () => void }) {
    const t = useTranslate();
    // "" asks for the home folder; the answer carries the real path.
    const [path, setPath] = useState("");
    const [name, setName] = useState(defaultName);
    const folders = useLocalFolders(path, true);
    const clone = useCloneRepository();
    const current = folders.data;
    const trimmed = name.trim();

    function submit(event: FormEvent) {
        event.preventDefault();
        if (!current || !trimmed) return;
        clone.mutate({ resourceId, parentPath: current.path, name: trimmed });
    }

    if (clone.isSuccess)
        return (
            <div className="flex flex-col gap-4">
                <p className="flex items-start gap-2 text-sm text-gray-700">
                    <FiCheck aria-hidden className="mt-0.5 shrink-0 text-emerald-600" />
                    <span>
                        {t("resources.repo.clone.done")}{" "}
                        <code className="break-all font-mono text-xs">{clone.data.path}</code>
                    </span>
                </p>
                <p className="text-xs text-gray-500">{t("resources.repo.clone.ownership")}</p>
                <div className="flex justify-end">
                    <Button variant="primary" size="lg"
                        type="button"
                        onClick={onClose}
                    >
                        {t("common.close")}
                    </Button>
                </div>
            </div>
        );

    const errorMessage = !clone.error
        ? null
        : clone.error instanceof ApiError && clone.error.message === "conflict"
          ? t("resources.repo.clone.exists")
          : clone.error instanceof ApiError && clone.error.message === "not_found"
            ? t("resources.repo.error.not_found")
            : t(errorKey(clone.error));

    return (
        <form onSubmit={submit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-2">
                <p className="text-sm font-medium text-gray-900">{t("resources.repo.clone.where")}</p>
                <div className="flex items-center gap-1">
                    <button
                        type="button"
                        disabled={!current?.parent || clone.isPending}
                        onClick={() => current?.parent && setPath(current.parent)}
                        aria-label={t("resources.repo.clone.up")}
                        title={t("resources.repo.clone.up")}
                        className="cursor-pointer rounded-md p-1.5 text-gray-600 hover:bg-gray-100 disabled:cursor-default disabled:opacity-40"
                    >
                        <FiArrowUp aria-hidden />
                    </button>
                    <button
                        type="button"
                        disabled={clone.isPending}
                        onClick={() => setPath(current?.home ?? "")}
                        aria-label={t("resources.repo.clone.home")}
                        title={t("resources.repo.clone.home")}
                        className="cursor-pointer rounded-md p-1.5 text-gray-600 hover:bg-gray-100 disabled:opacity-40"
                    >
                        <FiHome aria-hidden />
                    </button>
                    <code className="min-w-0 flex-1 truncate rounded-md bg-gray-50 px-2 py-1 font-mono text-xs text-gray-700" title={current?.path}>
                        {current?.path ?? "…"}
                    </code>
                </div>
                <div className="h-48 overflow-y-auto rounded-md border border-gray-200">
                    {folders.isLoading && <p className="p-3 text-sm text-gray-400">{t("common.loading")}</p>}
                    {folders.isError && (
                        <p role="alert" className="p-3 text-sm text-red-600">
                            {folders.error instanceof ApiError && folders.error.message === "forbidden"
                                ? t("resources.repo.clone.forbidden")
                                : t(errorKey(folders.error))}
                        </p>
                    )}
                    {current && current.folders.length === 0 && (
                        <p className="p-3 text-sm text-gray-400">{t("resources.repo.clone.noFolders")}</p>
                    )}
                    {current?.folders.map((folder) => (
                        <button
                            key={folder}
                            type="button"
                            disabled={clone.isPending}
                            onClick={() => setPath(childPath(current.path, folder))}
                            className="flex w-full cursor-pointer items-center gap-2 border-b border-gray-100 px-3 py-1.5 text-left text-sm last:border-b-0 hover:bg-gray-50"
                        >
                            <FiFolder aria-hidden className="shrink-0 text-amber-500" />
                            <span className="truncate">{folder}</span>
                        </button>
                    ))}
                </div>
            </div>
            <label className="flex flex-col gap-1 text-sm">
                <span className="font-medium text-gray-900">{t("resources.repo.clone.name")}</span>
                <input
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    disabled={clone.isPending}
                    className="rounded-md border border-gray-300 px-3 py-2 text-sm"
                />
            </label>
            {current && trimmed && (
                <p className="text-xs text-gray-500">
                    {t("resources.repo.clone.target")}{" "}
                    <code className="break-all font-mono">{childPath(current.path, trimmed)}</code>
                </p>
            )}
            {errorMessage && (
                <p role="alert" className="text-sm text-red-600">
                    {errorMessage}
                </p>
            )}
            <div className="flex items-center justify-end gap-3">
                {clone.isPending && (
                    <span role="status" className="text-sm text-gray-500">
                        {t("resources.repo.clone.cloning")}
                    </span>
                )}
                <Button variant="primary" size="lg"
                    type="submit"
                    disabled={!current || !trimmed || clone.isPending}
                >
                    {t("resources.repo.clone.submit")}
                </Button>
            </div>
        </form>
    );
}
