import { useRef, useState } from "react";
import { ApiError, accountExportUrl } from "../api/client";
import { useImportAccount, useResetAccount } from "../hooks/useAccount";
import { type TranslationKey, useTranslate } from "../hooks/useTranslate";
import { type ImportResult } from "../types/account";
import { ConfirmDialog } from "./ConfirmDialog";
import { SettingsGroup } from "./SettingsGroup";
import { SettingRow } from "./SettingRow";

export function SettingsData() {
    const t = useTranslate();
    const reset = useResetAccount();
    const [isConfirmOpen, setIsConfirmOpen] = useState(false);

    return (
        <SettingsGroup title={t("settings.account.data")}>
            <SettingRow
                label={t("settings.account.export")}
                hint={t("settings.account.export.hint")}
            >
                <a
                    href={accountExportUrl}
                    download
                    className="inline-block cursor-pointer rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
                >
                    {t("settings.account.export.action")}
                </a>
            </SettingRow>
            <ImportSetting />
            <SettingRow
                label={t("settings.account.reset")}
                hint={t("settings.account.reset.hint")}
            >
                <button
                    type="button"
                    onClick={() => setIsConfirmOpen(true)}
                    className="cursor-pointer rounded-md border border-red-200 bg-white px-3 py-1.5 text-sm font-medium text-red-600 hover:bg-red-50"
                >
                    {t("settings.account.reset.action")}
                </button>
            </SettingRow>
            <ConfirmDialog
                isOpen={isConfirmOpen}
                title={t("settings.account.reset.confirmTitle")}
                message={t("settings.account.reset.confirmMessage")}
                confirmLabel={t("settings.account.reset.action")}
                isPending={reset.isPending}
                isError={reset.isError}
                onConfirm={() => reset.mutate()}
                onClose={() => {
                    setIsConfirmOpen(false);
                    reset.reset();
                }}
            />
        </SettingsGroup>
    );
}

function importSummary(
    result: ImportResult,
    t: (key: TranslationKey) => string,
): string {
    const { added } = result;
    const parts: string[] = [];
    if (added.projects + added.tasks + added.resources + added.comments === 0) {
        parts.push(t("settings.account.import.nothingNew"));
    } else {
        parts.push(
            t(
                result.restored
                    ? "settings.account.import.restored"
                    : "settings.account.import.merged",
            )
                .replace("{projects}", String(added.projects))
                .replace("{tasks}", String(added.tasks))
                .replace("{files}", String(added.resources)),
        );
    }
    if (result.renamed > 0) {
        parts.push(
            t("settings.account.import.renamed").replace(
                "{count}",
                String(result.renamed),
            ),
        );
    }
    if (result.missingFiles.length > 0) {
        parts.push(
            t("settings.account.import.missing").replace(
                "{count}",
                String(result.missingFiles.length),
            ),
        );
    }
    return parts.join(" ");
}

function ImportSetting() {
    const t = useTranslate();
    const importData = useImportAccount();
    const inputRef = useRef<HTMLInputElement>(null);

    function handleFile(file: File | undefined) {
        if (file) importData.mutate(file);
    }

    let hint = t("settings.account.import.hint");
    if (importData.isPending) hint = t("settings.account.import.pending");
    if (importData.data) hint = importSummary(importData.data, t);
    if (importData.error) {
        const error = importData.error;
        hint =
            error instanceof ApiError && error.status === 400
                ? t("settings.account.import.invalid")
                : error instanceof ApiError && error.status === 413
                    ? t("settings.account.import.tooLarge")
                    : t("common.error");
    }

    return (
        <SettingRow
            label={t("settings.account.import")}
            hint={hint}
            hintTone={importData.isError ? "error" : "default"}
        >
            <input
                ref={inputRef}
                type="file"
                accept=".zip,application/zip"
                className="hidden"
                aria-hidden
                tabIndex={-1}
                onChange={(event) => {
                    handleFile(event.target.files?.[0]);
                    // Lets the same file be picked again.
                    event.target.value = "";
                }}
            />
            <button
                type="button"
                onClick={() => inputRef.current?.click()}
                disabled={importData.isPending}
                className="cursor-pointer rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
            >
                {t("settings.account.import.action")}
            </button>
        </SettingRow>
    );
}
