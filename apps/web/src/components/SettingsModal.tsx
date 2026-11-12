import {
  useEffect,
  useRef,
  useState,
  type ComponentType,
  type KeyboardEvent as ReactKeyboardEvent,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import {
  FiBell,
  FiCommand,
  FiInfo,
  FiMoon,
  FiSettings,
  FiTag,
  FiUser,
  FiX,
} from "react-icons/fi";
import { useDateFormat } from "../hooks/useDateFormat";
import {
  accents,
  updatePreferences,
  usePreferences,
} from "../hooks/usePreferences";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import { useUpdateWorkspace, useWorkspace } from "../hooks/useWorkspace";
import type {
  Accent,
  DateFormat,
  Density,
  Language,
  SettingsSection,
  Theme,
  TimeFormat,
  WeekStart,
} from "../types/settings";
import type { UpdateWorkspaceInput } from "../types/workspace";
import { formatDate } from "../util/date";
import type { StatusCategory } from "../types/task";

// General, account, shortcut and appearance settings (except the collapsed
// sidebar default) are applied and saved. Categories and notifications are
// still design only: their controls keep local state and nothing is saved.

type IconType = ComponentType<{ className?: string; "aria-hidden"?: boolean }>;

const sections: { id: SettingsSection; Icon: IconType }[] = [
  { id: "general", Icon: FiSettings },
  { id: "account", Icon: FiUser },
  { id: "appearance", Icon: FiMoon },
  { id: "categories", Icon: FiTag },
  { id: "notifications", Icon: FiBell },
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
  if (!isOpen) return null;
  // Keyed so reopening at another section starts from that section.
  return createPortal(
    <SettingsDialog
      key={initialSection}
      initialSection={initialSection}
      onClose={onClose}
    />,
    document.body,
  );
}

function SettingsDialog({
  initialSection,
  onClose,
}: {
  initialSection: SettingsSection;
  onClose: () => void;
}) {
  const t = useTranslate();
  const [active, setActive] = useState<SettingsSection>(initialSection);
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    dialogRef.current?.focus();
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onClose]);

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={onClose}
    >
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-label={t("settings.title")}
        tabIndex={-1}
        className="flex h-[min(680px,90vh)] w-full max-w-4xl overflow-hidden rounded-xl bg-white shadow-2xl outline-none"
        onClick={(event) => event.stopPropagation()}
      >
        <nav
          aria-label={t("settings.nav.label")}
          className="flex w-56 shrink-0 flex-col border-r border-gray-200 bg-gray-50 p-3"
        >
          <h2 className="px-3 pt-2 pb-3 text-xs font-semibold tracking-wide text-gray-400 uppercase">
            {t("settings.title")}
          </h2>
          <ul className="flex flex-col gap-1">
            {sections.map(({ id, Icon }) => (
              <li key={id}>
                <button
                  type="button"
                  onClick={() => setActive(id)}
                  aria-current={active === id ? "page" : undefined}
                  className="flex w-full cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-200 aria-[current=page]:bg-gray-200 aria-[current=page]:font-semibold aria-[current=page]:text-gray-900"
                >
                  <Icon
                    aria-hidden
                    className={`shrink-0 text-base ${active === id ? "text-accent" : "text-gray-500"}`}
                  />
                  {t(`settings.${id}`)}
                </button>
              </li>
            ))}
          </ul>
          <p className="mt-auto flex items-start gap-2 rounded-md px-3 py-2 text-xs text-gray-400">
            <FiInfo aria-hidden className="mt-0.5 shrink-0" />
            {t("settings.preview")}
          </p>
        </nav>

        <div className="flex min-w-0 flex-1 flex-col">
          <header className="flex shrink-0 items-start justify-between gap-4 border-b border-gray-200 px-8 py-5">
            <div className="min-w-0">
              <h3 className="text-lg font-semibold text-gray-900">
                {t(`settings.${active}`)}
              </h3>
              <p className="mt-0.5 text-sm text-gray-500">
                {t(`settings.${active}.description`)}
              </p>
            </div>
            <button
              type="button"
              onClick={onClose}
              aria-label={t("common.close")}
              title={t("common.close")}
              className="cursor-pointer rounded-md p-1.5 text-lg text-gray-500 hover:bg-gray-100 hover:text-gray-900"
            >
              <FiX aria-hidden />
            </button>
          </header>
          <div className="min-h-0 flex-1 overflow-y-auto px-8 py-6">
            {active === "general" && <GeneralSettings />}
            {active === "account" && <AccountSettings />}
            {active === "appearance" && <AppearanceSettings />}
            {active === "categories" && <CategorySettings />}
            {active === "notifications" && <NotificationSettings />}
            {active === "shortcuts" && <ShortcutSettings />}
          </div>
        </div>
      </div>
    </div>
  );
}

function GeneralSettings() {
  const t = useTranslate();
  const preferences = usePreferences();
  const format = useDateFormat();
  const now = new Date();
  const dateFormats: DateFormat[] = ["dmy", "mdy", "iso"];

  return (
    <div className="flex flex-col gap-6">
      <SettingsGroup>
        <SettingRow
          label={t("settings.general.language")}
          hint={t("settings.general.language.hint")}
        >
          <Select
            label={t("settings.general.language")}
            value={preferences.language}
            onChange={(language) =>
              updatePreferences({ language: language as Language })
            }
            options={[
              { value: "en", label: "English" },
              { value: "tr", label: "Türkçe" },
            ]}
          />
        </SettingRow>
      </SettingsGroup>

      <SettingsGroup title={t("settings.general.dateTime")}>
        <SettingRow
          label={t("settings.general.weekStart")}
          hint={t("settings.general.weekStart.hint")}
        >
          <Segmented
            label={t("settings.general.weekStart")}
            value={preferences.weekStart}
            onChange={(weekStart) =>
              updatePreferences({ weekStart: weekStart as WeekStart })
            }
            options={[
              { value: "monday", label: t("settings.general.weekStart.monday") },
              { value: "sunday", label: t("settings.general.weekStart.sunday") },
            ]}
          />
        </SettingRow>
        <SettingRow label={t("settings.general.dateFormat")}>
          <Select
            label={t("settings.general.dateFormat")}
            value={preferences.dateFormat}
            onChange={(dateFormat) =>
              updatePreferences({ dateFormat: dateFormat as DateFormat })
            }
            options={dateFormats.map((value) => ({
              value,
              label: formatDate(now, value),
            }))}
          />
        </SettingRow>
        <SettingRow
          label={t("settings.general.timeFormat")}
          hint={`${t("settings.general.example")}: ${format.dateTime(now)}`}
        >
          <Segmented
            label={t("settings.general.timeFormat")}
            value={preferences.timeFormat}
            onChange={(timeFormat) =>
              updatePreferences({ timeFormat: timeFormat as TimeFormat })
            }
            options={[
              { value: "24h", label: t("settings.general.timeFormat.24h") },
              { value: "12h", label: t("settings.general.timeFormat.12h") },
            ]}
          />
        </SettingRow>
      </SettingsGroup>
    </div>
  );
}

function AccountSettings() {
  const t = useTranslate();
  const workspace = useWorkspace();
  const username = workspace.data?.username ?? "";

  return (
    <div className="flex flex-col gap-6">
      <SettingsGroup>
        <div className="flex items-center gap-4 px-5 py-4">
          <span
            aria-hidden
            className="flex size-14 shrink-0 items-center justify-center rounded-full bg-gray-900 text-xl font-semibold text-white"
          >
            {username ? username[0].toUpperCase() : <FiUser />}
          </span>
          <div className="min-w-0">
            <p className="truncate font-medium text-gray-900">{username}</p>
            <p className="truncate text-sm text-gray-500">
              {workspace.data?.name}
            </p>
          </div>
        </div>
        {workspace.data && (
          <>
            <NameSetting
              label={t("settings.account.username")}
              value={workspace.data.username}
              toInput={(username) => ({ username })}
            />
            <NameSetting
              label={t("settings.account.workspace")}
              value={workspace.data.name}
              toInput={(name) => ({ name })}
            />
          </>
        )}
      </SettingsGroup>

      <SettingsGroup title={t("settings.account.data")}>
        <SettingRow
          label={t("settings.account.export")}
          hint={t("settings.account.export.hint")}
        >
          <button
            type="button"
            className="cursor-pointer rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            {t("settings.account.export.action")}
          </button>
        </SettingRow>
        <SettingRow
          label={t("settings.account.reset")}
          hint={t("settings.account.reset.hint")}
        >
          <button
            type="button"
            className="cursor-pointer rounded-md border border-red-200 bg-white px-3 py-1.5 text-sm font-medium text-red-600 hover:bg-red-50"
          >
            {t("settings.account.reset.action")}
          </button>
        </SettingRow>
      </SettingsGroup>
    </div>
  );
}

// A name saved when the field loses focus or Enter is pressed.
function NameSetting({
  label,
  value,
  toInput,
}: {
  label: string;
  value: string;
  toInput: (name: string) => UpdateWorkspaceInput;
}) {
  const t = useTranslate();
  const update = useUpdateWorkspace();
  const [draft, setDraft] = useState(value);
  const [error, setError] = useState<string | null>(null);
  const [isSaved, setIsSaved] = useState(false);

  function save() {
    const name = draft.trim();
    setIsSaved(false);
    if (name === "") {
      setDraft(value);
      setError(t("settings.account.nameRequired"));
      return;
    }
    setError(null);
    if (name === value) {
      setDraft(value);
      return;
    }
    update.mutate(toInput(name), {
      onSuccess: () => {
        setDraft(name);
        setIsSaved(true);
      },
      onError: () => {
        setDraft(value);
        setError(t("settings.account.saveError"));
      },
    });
  }

  function handleKeyDown(event: ReactKeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter") event.currentTarget.blur();
  }

  return (
    <SettingRow
      label={label}
      hint={error ?? (isSaved ? t("settings.account.saved") : undefined)}
      hintTone={error ? "error" : "default"}
    >
      <input
        value={draft}
        onChange={(event) => {
          setDraft(event.target.value);
          setIsSaved(false);
        }}
        onBlur={save}
        onKeyDown={handleKeyDown}
        disabled={update.isPending}
        maxLength={100}
        aria-label={label}
        aria-invalid={error !== null}
        className="w-56 rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm aria-invalid:border-red-400 disabled:opacity-60"
      />
    </SettingRow>
  );
}

// Previews use literal colors so they look the same in either theme (the
// theme variables like `bg-white` are redefined in dark mode).
const themes: { value: Theme; label: TranslationKey; preview: string }[] = [
  { value: "light", label: "settings.appearance.theme.light", preview: "bg-[#fff]" },
  { value: "dark", label: "settings.appearance.theme.dark", preview: "bg-[#1b1d22]" },
  {
    value: "system",
    label: "settings.appearance.theme.system",
    preview: "bg-linear-to-r from-[#fff] from-50% to-[#1b1d22] to-50%",
  },
];

// Must match the `data-accent` palettes in index.css.
const accentSwatches: Record<Accent, string> = {
  neutral: "var(--color-gray-900)",
  blue: "var(--color-blue-600)",
  indigo: "var(--color-indigo-600)",
  violet: "var(--color-violet-600)",
  fuchsia: "var(--color-fuchsia-600)",
  rose: "var(--color-rose-600)",
  teal: "var(--color-teal-700)",
  emerald: "var(--color-emerald-700)",
};

function AppearanceSettings() {
  const t = useTranslate();
  const { theme, accent, density } = usePreferences();

  return (
    <div className="flex flex-col gap-6">
      <SettingsGroup title={t("settings.appearance.theme")}>
        <div
          role="radiogroup"
          aria-label={t("settings.appearance.theme")}
          className="grid grid-cols-3 gap-3 px-5 py-4"
        >
          {themes.map((option) => (
            <button
              key={option.value}
              type="button"
              role="radio"
              aria-checked={theme === option.value}
              onClick={() => updatePreferences({ theme: option.value })}
              className="cursor-pointer rounded-lg border-2 border-gray-200 p-2 text-left hover:border-gray-300 aria-checked:border-accent"
            >
              <span
                aria-hidden
                className={`block h-16 rounded-md border border-gray-200 ${option.preview}`}
              />
              <span className="mt-2 block text-sm font-medium text-gray-700">
                {t(option.label)}
              </span>
            </button>
          ))}
        </div>
      </SettingsGroup>

      <SettingsGroup>
        <SettingRow
          label={t("settings.appearance.accent")}
          hint={t("settings.appearance.accent.hint")}
        >
          <div
            role="radiogroup"
            aria-label={t("settings.appearance.accent")}
            className="flex gap-1.5"
          >
            {accents.map((option) => (
              <button
                key={option}
                type="button"
                role="radio"
                aria-checked={accent === option}
                aria-label={t(`settings.appearance.accent.${option}`)}
                title={t(`settings.appearance.accent.${option}`)}
                onClick={() => updatePreferences({ accent: option })}
                style={{ backgroundColor: accentSwatches[option] }}
                className="size-6 cursor-pointer rounded-full ring-offset-2 ring-offset-white aria-checked:ring-2 aria-checked:ring-gray-900"
              />
            ))}
          </div>
        </SettingRow>
        <SettingRow label={t("settings.appearance.density")}>
          <Segmented
            label={t("settings.appearance.density")}
            value={density}
            onChange={(density) =>
              updatePreferences({ density: density as Density })
            }
            options={[
              {
                value: "comfortable",
                label: t("settings.appearance.density.comfortable"),
              },
              {
                value: "compact",
                label: t("settings.appearance.density.compact"),
              },
            ]}
          />
        </SettingRow>
        <SettingRow
          label={t("settings.appearance.sidebarCollapsed")}
          hint={t("settings.appearance.sidebarCollapsed.hint")}
        >
          <Toggle label={t("settings.appearance.sidebarCollapsed")} />
        </SettingRow>
      </SettingsGroup>
    </div>
  );
}

const categoryDefaults: { category: StatusCategory; color: string }[] = [
  { category: "todo", color: "#9ca3af" },
  { category: "in_progress", color: "#3b82f6" },
  { category: "done", color: "#22c55e" },
];

function CategorySettings() {
  const t = useTranslate();

  return (
    <div className="flex flex-col gap-6">
      <SettingsGroup title={t("settings.categories.defaults")}>
        {categoryDefaults.map(({ category, color }) => (
          <div key={category} className="flex items-center gap-4 px-5 py-4">
            <span
              aria-hidden
              style={{ backgroundColor: color }}
              className="size-3 shrink-0 rounded-full"
            />
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-gray-900">
                {t(`status.category.${category}`)}
              </p>
              <p className="text-xs text-gray-500">
                {t(`settings.categories.${category}.hint`)}
              </p>
            </div>
            <input
              defaultValue={t(`status.category.${category}`)}
              aria-label={`${t("settings.categories.defaultName")}: ${t(`status.category.${category}`)}`}
              placeholder={t("settings.categories.defaultName")}
              className="w-44 rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm"
            />
          </div>
        ))}
      </SettingsGroup>

      <SettingsGroup>
        <SettingRow
          label={t("settings.categories.doneToBottom")}
          hint={t("settings.categories.doneToBottom.hint")}
        >
          <Toggle label={t("settings.categories.doneToBottom")} />
        </SettingRow>
        <SettingRow
          label={t("settings.categories.hideDone")}
          hint={t("settings.categories.hideDone.hint")}
        >
          <Toggle label={t("settings.categories.hideDone")} />
        </SettingRow>
      </SettingsGroup>
    </div>
  );
}

function NotificationSettings() {
  const t = useTranslate();

  return (
    <div className="flex flex-col gap-6">
      <SettingsGroup title={t("settings.notifications.tasks")}>
        <SettingRow
          label={t("settings.notifications.dueSoon")}
          hint={t("settings.notifications.dueSoon.hint")}
        >
          <Toggle label={t("settings.notifications.dueSoon")} defaultOn />
        </SettingRow>
        <SettingRow label={t("settings.notifications.reminder")}>
          <Select
            label={t("settings.notifications.reminder")}
            options={[
              { value: "1h", label: t("settings.notifications.reminder.1h") },
              { value: "1d", label: t("settings.notifications.reminder.1d") },
              { value: "1w", label: t("settings.notifications.reminder.1w") },
            ]}
          />
        </SettingRow>
        <SettingRow
          label={t("settings.notifications.overdue")}
          hint={t("settings.notifications.overdue.hint")}
        >
          <Toggle label={t("settings.notifications.overdue")} defaultOn />
        </SettingRow>
      </SettingsGroup>

      <SettingsGroup title={t("settings.notifications.activity")}>
        <SettingRow
          label={t("settings.notifications.comments")}
          hint={t("settings.notifications.comments.hint")}
        >
          <Toggle label={t("settings.notifications.comments")} />
        </SettingRow>
        <SettingRow label={t("settings.notifications.sound")}>
          <Toggle label={t("settings.notifications.sound")} />
        </SettingRow>
      </SettingsGroup>
    </div>
  );
}

// The global shortcuts are registered with useShortcut; sending a comment
// and closing a dialog are handled by those components and always work.
const isMac = /mac/i.test(navigator.userAgent);
const mod = isMac ? "⌘" : "Ctrl";
const shortcuts: { label: TranslationKey; keys: string[]; global: boolean }[] = [
  { label: "settings.shortcuts.search", keys: [mod, "K"], global: true },
  { label: "settings.shortcuts.newTask", keys: ["N"], global: true },
  { label: "settings.shortcuts.toggleSidebar", keys: [mod, "B"], global: true },
  { label: "settings.shortcuts.openSettings", keys: [mod, ","], global: true },
  { label: "settings.shortcuts.sendComment", keys: [mod, "Enter"], global: false },
  { label: "settings.shortcuts.closeDialog", keys: ["Esc"], global: false },
];

function ShortcutSettings() {
  const t = useTranslate();
  const { shortcutsEnabled } = usePreferences();

  return (
    <div className="flex flex-col gap-6">
      <SettingsGroup>
        <SettingRow
          label={t("settings.shortcuts.enabled")}
          hint={t("settings.shortcuts.enabled.hint")}
        >
          <Toggle
            label={t("settings.shortcuts.enabled")}
            checked={shortcutsEnabled}
            onChange={(enabled) =>
              updatePreferences({ shortcutsEnabled: enabled })
            }
          />
        </SettingRow>
      </SettingsGroup>
      <SettingsGroup title={t("settings.shortcuts.list")}>
        {shortcuts.map(({ label, keys, global }) => (
          <SettingRow
            key={label}
            label={t(label)}
            hint={
              label === "settings.shortcuts.newTask"
                ? t("settings.shortcuts.newTask.hint")
                : undefined
            }
          >
            <span
              className={`flex items-center gap-1 ${
                global && !shortcutsEnabled ? "opacity-40" : ""
              }`}
            >
              {keys.map((key) => (
                <kbd
                  key={key}
                  className="min-w-7 rounded border border-gray-300 bg-gray-50 px-1.5 py-0.5 text-center font-sans text-xs font-medium text-gray-600 shadow-[0_1px_0] shadow-gray-300"
                >
                  {key}
                </kbd>
              ))}
            </span>
          </SettingRow>
        ))}
      </SettingsGroup>
    </div>
  );
}

function SettingsGroup({
  title,
  children,
}: {
  title?: string;
  children: ReactNode;
}) {
  return (
    <section aria-label={title}>
      {title && (
        <h4 className="mb-2 text-xs font-semibold tracking-wide text-gray-400 uppercase">
          {title}
        </h4>
      )}
      <div className="divide-y divide-gray-100 rounded-lg border border-gray-200">
        {children}
      </div>
    </section>
  );
}

function SettingRow({
  label,
  hint,
  hintTone = "default",
  children,
}: {
  label: string;
  hint?: string;
  hintTone?: "default" | "error";
  children: ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-6 px-5 py-4">
      <div className="min-w-0">
        <p className="text-sm font-medium text-gray-900">{label}</p>
        {hint && (
          <p
            role={hintTone === "error" ? "alert" : undefined}
            className={`mt-0.5 text-xs ${
              hintTone === "error" ? "text-red-600" : "text-gray-500"
            }`}
          >
            {hint}
          </p>
        )}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

type Option = { value: string; label: string };

// Controls are controlled when given value and onChange; the design-only
// sections leave them out and the control keeps its own state.
function Select({
  label,
  options,
  value,
  onChange,
}: {
  label: string;
  options: Option[];
  value?: string;
  onChange?: (value: string) => void;
}) {
  return (
    <select
      aria-label={label}
      value={value}
      onChange={onChange && ((event) => onChange(event.target.value))}
      className="w-56 cursor-pointer rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm"
    >
      {options.map((option) => (
        <option key={option.value} value={option.value}>
          {option.label}
        </option>
      ))}
    </select>
  );
}

function Segmented({
  label,
  options,
  value,
  onChange,
}: {
  label: string;
  options: Option[];
  value?: string;
  onChange?: (value: string) => void;
}) {
  const [localValue, setLocalValue] = useState(options[0].value);
  const current = value ?? localValue;
  return (
    <div
      role="radiogroup"
      aria-label={label}
      className="flex rounded-md bg-gray-100 p-0.5"
    >
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          role="radio"
          aria-checked={current === option.value}
          onClick={() => (onChange ?? setLocalValue)(option.value)}
          className="cursor-pointer rounded px-3 py-1 text-sm text-gray-600 hover:text-gray-900 aria-checked:bg-white aria-checked:font-medium aria-checked:text-gray-900 aria-checked:shadow-sm"
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}

function Toggle({
  label,
  defaultOn = false,
  checked,
  onChange,
}: {
  label: string;
  defaultOn?: boolean;
  checked?: boolean;
  onChange?: (checked: boolean) => void;
}) {
  const [localOn, setLocalOn] = useState(defaultOn);
  const isOn = checked ?? localOn;
  return (
    <button
      type="button"
      role="switch"
      aria-checked={isOn}
      aria-label={label}
      onClick={() => (onChange ?? setLocalOn)(!isOn)}
      className="relative h-6 w-11 cursor-pointer rounded-full bg-gray-300 transition-colors aria-checked:bg-accent"
    >
      <span
        aria-hidden
        className={`absolute top-0.5 left-0.5 size-5 rounded-full bg-white shadow transition-transform ${
          isOn ? "translate-x-5" : ""
        }`}
      />
    </button>
  );
}
