import { useState, type ComponentType } from "react";
import {
    FiBell,
    FiCommand,
    FiGithub,
    FiInfo,
    FiMoon,
    FiSettings,
    FiTag,
    FiUser,
} from "react-icons/fi";
import { useTranslate } from "../hooks/useTranslate";
import type { SettingsSection } from "../types/settings";
import { Dialog } from "./Dialog";
import { SettingsAccount } from "./SettingsAccount";
import { SettingsAppearance } from "./SettingsAppearance";
import { SettingsCategories } from "./SettingsCategories";
import { SettingsGeneral } from "./SettingsGeneral";
import { SettingsGitHub } from "./SettingsGitHub";
import { SettingsNotifications } from "./SettingsNotifications";
import { SettingsShortcuts } from "./SettingsShortcuts";

type IconType = ComponentType<{ className?: string; "aria-hidden"?: boolean }>;

const sections: { id: SettingsSection; Icon: IconType }[] = [
    { id: "general", Icon: FiSettings },
    { id: "account", Icon: FiUser },
    { id: "appearance", Icon: FiMoon },
    { id: "categories", Icon: FiTag },
    { id: "notifications", Icon: FiBell },
    { id: "github", Icon: FiGithub },
    { id: "shortcuts", Icon: FiCommand },
];

type SettingsModalProps = {
    isOpen: boolean;
    initialSection?: SettingsSection;
    onClose: () => void;
};

export function SettingsModal({
    isOpen,
    initialSection = "general",
    onClose,
}: SettingsModalProps) {
    const t = useTranslate();
    if (!isOpen) return null;
    // Keyed so reopening at another section starts from that section.
    return (
        <Dialog
            key={initialSection}
            open
            onClose={onClose}
            title={t("settings.title")}
            size="xl"
            bare
        >
            <SettingsBody initialSection={initialSection} />
        </Dialog>
    );
}

function SettingsBody({ initialSection }: { initialSection: SettingsSection }) {
    const t = useTranslate();
    const [active, setActive] = useState<SettingsSection>(initialSection);

    return (
        <div className="flex h-[min(680px,80vh)]">
            <nav
                aria-label={t("settings.nav.label")}
                className="flex w-56 shrink-0 flex-col border-r border-line bg-surface-2 p-3"
            >
                <ul className="flex flex-col gap-1">
                    {sections.map(({ id, Icon }) => (
                        <li key={id}>
                            <button
                                type="button"
                                onClick={() => setActive(id)}
                                aria-current={active === id ? "page" : undefined}
                                className="flex h-8 w-full cursor-pointer items-center gap-3 rounded-md px-3 text-left text-body-xs text-fg-muted hover:bg-layer-1-hover aria-[current=page]:bg-layer-1-hover aria-[current=page]:font-medium aria-[current=page]:text-fg"
                            >
                                <Icon
                                    aria-hidden
                                    className={`shrink-0 text-base ${active === id ? "text-accent" : "text-fg-subtle"}`}
                                />
                                {t(`settings.${id}`)}
                            </button>
                        </li>
                    ))}
                </ul>
                <p className="mt-auto flex items-start gap-2 rounded-md px-3 py-2 text-caption text-fg-placeholder">
                    <FiInfo aria-hidden className="mt-0.5 shrink-0" />
                    {t("settings.preview")}
                </p>
            </nav>

            <div className="flex min-w-0 flex-1 flex-col">
                <header className="shrink-0 border-b border-line px-8 py-4">
                    <h3 className="text-lg font-semibold text-fg">
                        {t(`settings.${active}`)}
                    </h3>
                    <p className="mt-0.5 text-body-xs text-fg-subtle">
                        {t(`settings.${active}.description`)}
                    </p>
                </header>
                <div className="min-h-0 flex-1 overflow-y-auto px-8 py-6">
                    {active === "general" && <SettingsGeneral />}
                    {active === "account" && <SettingsAccount />}
                    {active === "appearance" && <SettingsAppearance />}
                    {active === "categories" && <SettingsCategories />}
                    {active === "notifications" && <SettingsNotifications />}
                    {active === "github" && <SettingsGitHub />}
                    {active === "shortcuts" && <SettingsShortcuts />}
                </div>
            </div>
        </div>
    );
}
