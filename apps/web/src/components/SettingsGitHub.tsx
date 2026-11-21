import { useEffect, useState } from "react";
import { FiGithub } from "react-icons/fi";
import { ApiError } from "../api/client";
import { useDisconnectGitHub, useGitHubStatus, usePollGitHubConnect, useStartGitHubConnect } from "../hooks/useGitHub";
import { errorKey } from "../hooks/useResourceOperations";
import { useTranslate } from "../hooks/useTranslate";
import { defaultUserSettings, useUpdateUserSettings, useUserSettings } from "../hooks/useUserSettings";
import { Button } from "./Button";
import { SettingsGroup } from "./SettingsGroup";
import { SettingRow } from "./SettingRow";
import { Toggle } from "./Toggle";

export function SettingsGitHub() {
    const t = useTranslate();
    const status = useGitHubStatus();
    const disconnect = useDisconnectGitHub();
    const settings = useUserSettings();
    const update = useUpdateUserSettings();
    const autoUpdate = (settings.data ?? defaultUserSettings).repositoriesAutoUpdate;

    if (status.isLoading)
        return <p className="text-sm text-gray-400">{t("common.loading")}</p>;
    if (status.isError || !status.data)
        return (
            <p role="alert" className="text-sm text-red-600">
                {t(errorKey(status.error))}
            </p>
        );
    const { configured, gitAvailable, account } = status.data;

    return (
        <div className="flex flex-col gap-6">
            <SettingsGroup title={t("settings.github.account")}>
                {!configured ? (
                    <p className="px-5 py-4 text-sm text-gray-600">
                        {t("settings.github.notConfigured")}
                    </p>
                ) : account ? (
                    <SettingRow
                        label={account.name || account.login}
                        hint={`@${account.login} · ${t("settings.github.disconnect.hint")}`}
                        hintTone={disconnect.isError ? "error" : "default"}
                    >
                        <div className="flex items-center gap-3">
                            {account.avatarUrl && (
                                <img
                                    src={account.avatarUrl}
                                    alt=""
                                    className="size-8 rounded-full"
                                />
                            )}
                            <button
                                type="button"
                                disabled={disconnect.isPending}
                                onClick={() => disconnect.mutate()}
                                className="cursor-pointer rounded-md border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-100 disabled:opacity-50"
                            >
                                {t("settings.github.disconnect")}
                            </button>
                        </div>
                    </SettingRow>
                ) : (
                    <GitHubConnect />
                )}
            </SettingsGroup>

            <SettingsGroup title={t("settings.github.repositories")}>
                <SettingRow
                    label={t("settings.github.autoUpdate")}
                    hint={
                        update.isError
                            ? t("common.error")
                            : t("settings.github.autoUpdate.hint")
                    }
                    hintTone={update.isError ? "error" : "default"}
                >
                    <Toggle
                        label={t("settings.github.autoUpdate")}
                        checked={autoUpdate}
                        onChange={(repositoriesAutoUpdate) =>
                            update.mutate({ repositoriesAutoUpdate })
                        }
                    />
                </SettingRow>
                <SettingRow
                    label={t("settings.github.git")}
                    hint={t(
                        gitAvailable
                            ? "settings.github.git.available"
                            : "settings.github.git.missing",
                    )}
                >
                    <span
                        aria-hidden
                        className={`block size-2.5 rounded-full ${gitAvailable ? "bg-emerald-500" : "bg-gray-300"}`}
                    />
                </SettingRow>
            </SettingsGroup>
        </div>
    );
}

// Connects an account with GitHub's device flow: the user enters a code on
// github.com while this polls until they approve.
function GitHubConnect() {
    const t = useTranslate();
    const start = useStartGitHubConnect();
    const poll = usePollGitHubConnect();
    const [copied, setCopied] = useState(false);
    const flow = start.data;
    const outcome = poll.data?.state;
    // The backend forgets the flow when it restarts; polling it is pointless
    // then.
    const isLost = poll.error instanceof ApiError && poll.error.status === 404;
    const isWaiting =
        flow !== undefined &&
        !isLost &&
        (outcome === undefined || outcome === "pending");
    const { mutate: pollNow } = poll;

    // Polling GitHub (through the backend) is an external system to keep in
    // step with while the code is shown.
    useEffect(() => {
        if (!isWaiting || !flow) return;
        const timer = window.setInterval(
            () => pollNow(),
            Math.max(flow.intervalSeconds, 1) * 1000,
        );
        return () => window.clearInterval(timer);
    }, [isWaiting, flow, pollNow]);

    function begin() {
        poll.reset();
        setCopied(false);
        start.mutate();
    }

    const error = start.error ?? poll.error;
    let hint = t("settings.github.scope.hint");
    if (outcome === "expired") hint = t("settings.github.expired");
    if (outcome === "denied") hint = t("settings.github.denied");
    if (error) hint = t(errorKey(error));

    if (!isWaiting || !flow)
        return (
            <SettingRow
                label={t("settings.github.notConnected")}
                hint={hint}
                hintTone={error || outcome === "denied" ? "error" : "default"}
            >
                <Button variant="primary" size="lg"
                    type="button"
                    disabled={start.isPending}
                    onClick={begin}
                >
                    <FiGithub aria-hidden />
                    {t("settings.github.connect")}
                </Button>
            </SettingRow>
        );

    return (
        <div className="flex flex-col gap-3 px-5 py-4">
            <p className="text-sm text-gray-700">{t("settings.github.code")}</p>
            <div className="flex flex-wrap items-center gap-3">
                <code className="rounded-md border border-gray-300 bg-gray-50 px-4 py-2 font-mono text-xl tracking-widest text-gray-900">
                    {flow.userCode}
                </code>
                <button
                    type="button"
                    onClick={() => {
                        void navigator.clipboard
                            .writeText(flow.userCode)
                            .then(() => setCopied(true));
                    }}
                    className="cursor-pointer rounded-md border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-100"
                >
                    {t(copied ? "settings.github.copied" : "settings.github.copy")}
                </button>
                <a
                    href={flow.verificationUri}
                    target="_blank"
                    rel="noreferrer"
                    className="rounded-md bg-accent px-3 py-1.5 text-sm font-medium text-accent-fg hover:bg-accent/85"
                >
                    {t("settings.github.open")}
                </a>
                <button
                    type="button"
                    onClick={() => start.reset()}
                    className="cursor-pointer rounded-md px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100"
                >
                    {t("common.cancel")}
                </button>
            </div>
            <p role="status" className="text-xs text-gray-500">
                {t("settings.github.waiting")}
            </p>
        </div>
    );
}
