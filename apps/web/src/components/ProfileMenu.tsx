import { useCallback, useState } from "react";
import {
    FiBell,
    FiCheck,
    FiChevronUp,
    FiCommand,
    FiMonitor,
    FiMoon,
    FiSettings,
    FiSun,
    FiUser,
} from "react-icons/fi";
import { updatePreferences, usePreferences } from "../hooks/usePreferences";
import { useShortcut } from "../hooks/useShortcut";
import { useTranslate } from "../hooks/useTranslate";
import { useWorkspace } from "../hooks/useWorkspace";
import type { SettingsSection, Theme } from "../types/settings";
import { Menu } from "./Menu";
import { MenuItem } from "./MenuItem";
import { MenuSeparator } from "./MenuSeparator";
import { SettingsModal } from "./SettingsModal";
import { Tooltip } from "./Tooltip";

const themeItems = [
    { value: "light", label: "settings.appearance.theme.light", Icon: FiSun },
    { value: "dark", label: "settings.appearance.theme.dark", Icon: FiMoon },
    { value: "system", label: "settings.appearance.theme.system", Icon: FiMonitor },
] as const satisfies readonly { value: Theme; label: string; Icon: unknown }[];

export function ProfileMenu({ collapsed }: { collapsed: boolean }) {
    const t = useTranslate();
    const workspace = useWorkspace();
    const { theme } = usePreferences();
    const [section, setSection] = useState<SettingsSection | null>(null);
    const closeSettings = useCallback(() => setSection(null), []);

    useShortcut({ key: ",", mod: true }, () => setSection("general"));

    const username = workspace.data?.username ?? "";
    const label = username || t("sidebar.profile.label");

    const trigger = (props: Parameters<Parameters<typeof Menu>[0]["trigger"]>[0]) => {
        const { ref, ...rest } = props;
        const button = (
            <button
                ref={ref}
                type="button"
                aria-label={t("sidebar.profile.label")}
                {...rest}
                className={`flex w-full cursor-pointer items-center gap-2 rounded-md p-1.5 text-left hover:bg-layer-1-hover ${
                    collapsed ? "justify-center" : ""
                } ${rest["aria-expanded"] ? "bg-layer-1-hover" : ""}`}
            >
                <span
                    aria-hidden
                    className="flex size-7 shrink-0 items-center justify-center rounded-full bg-accent text-caption font-semibold text-accent-fg"
                >
                    {username ? username[0].toUpperCase() : <FiUser />}
                </span>
                {!collapsed && (
                    <>
                        <span className="min-w-0 flex-1">
                            <span className="block truncate text-body-xs font-medium text-fg">
                                {username}
                            </span>
                            <span className="block truncate text-caption text-fg-subtle">
                                {workspace.data?.name}
                            </span>
                        </span>
                        <FiChevronUp
                            aria-hidden
                            className="shrink-0 text-fg-placeholder"
                        />
                    </>
                )}
            </button>
        );
        return collapsed ? (
            <Tooltip label={label} side="right">
                {button}
            </Tooltip>
        ) : (
            button
        );
    };

    return (
        <div className="shrink-0 border-t border-line p-2">
            <Menu placement={collapsed ? "right-start" : "bottom-start"} trigger={trigger}>
                <MenuItem
                    icon={<FiUser aria-hidden />}
                    label={t("sidebar.profile.profile")}
                    onSelect={() => setSection("account")}
                />
                <MenuItem
                    icon={<FiSettings aria-hidden />}
                    label={t("sidebar.profile.settings")}
                    shortcut={["mod", ","]}
                    onSelect={() => setSection("general")}
                />
                <MenuItem
                    icon={<FiBell aria-hidden />}
                    label={t("sidebar.profile.notifications")}
                    onSelect={() => setSection("notifications")}
                />
                <MenuItem
                    icon={<FiCommand aria-hidden />}
                    label={t("sidebar.profile.shortcuts")}
                    onSelect={() => setSection("shortcuts")}
                />
                <MenuSeparator />
                <p className="px-2 pt-1 pb-0.5 text-caption text-fg-subtle">
                    {t("sidebar.profile.theme")}
                </p>
                {themeItems.map(({ value, label: itemLabel, Icon }) => (
                    <MenuItem
                        key={value}
                        icon={
                            theme === value ? (
                                <FiCheck aria-hidden />
                            ) : (
                                <Icon aria-hidden />
                            )
                        }
                        label={t(itemLabel)}
                        onSelect={() => updatePreferences({ theme: value })}
                    />
                ))}
            </Menu>
            <SettingsModal
                isOpen={section !== null}
                initialSection={section ?? undefined}
                onClose={closeSettings}
            />
        </div>
    );
}
