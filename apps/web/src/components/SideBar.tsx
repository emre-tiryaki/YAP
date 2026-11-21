import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ComponentType,
  type DragEvent,
  type FocusEvent,
  type MouseEvent,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { Link, useLocation } from "react-router";
import {
  FiBell,
  FiCalendar,
  FiCheckSquare,
  FiChevronUp,
  FiCommand,
  FiFile,
  FiFolder,
  FiHardDrive,
  FiHelpCircle,
  FiHome,
  FiLogOut,
  FiMoon,
  FiSettings,
  FiUser,
  FiX,
} from "react-icons/fi";
import {
  TbLayoutSidebarLeftCollapse,
  TbLayoutSidebarLeftExpand,
  TbPin,
} from "react-icons/tb";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import { addPin, unpinWhere, usePins } from "../hooks/usePins";
import { useProjects } from "../hooks/useProjects";
import { useShortcut } from "../hooks/useShortcut";
import { useWorkspace } from "../hooks/useWorkspace";
import type { Pin } from "../types/pin";
import type { SettingsSection } from "../types/settings";
import { hasPinDrag, readPinDrag, setPinDragData } from "../util/pinDrag";
import { SettingsModal } from "./SettingsModal";
import { useDismiss } from "../hooks/useDismiss";

type IconType = ComponentType<{ className?: string; "aria-hidden"?: boolean }>;

const COLLAPSED_KEY = "yap.sidebar.collapsed";

// True while something pinnable is being dragged anywhere in the window, so
// the Pinned section can show itself as a drop target before the pointer
// reaches it.
function usePinDragActive() {
  const [isActive, setIsActive] = useState(false);
  useEffect(() => {
    function handleStart(event: globalThis.DragEvent) {
      if (hasPinDrag(event)) setIsActive(true);
    }
    function handleEnd() {
      setIsActive(false);
    }
    window.addEventListener("dragstart", handleStart);
    window.addEventListener("dragend", handleEnd);
    window.addEventListener("drop", handleEnd);
    return () => {
      window.removeEventListener("dragstart", handleStart);
      window.removeEventListener("dragend", handleEnd);
      window.removeEventListener("drop", handleEnd);
    };
  }, []);
  return isActive;
}

// Shows `label` right of the hovered or focused element at once (no native
// title delay). Rendered in a portal with fixed position so the scrolling nav
// does not clip it.
function useSideLabel(label: string, isEnabled: boolean) {
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null);

  const show = (event: MouseEvent<HTMLElement> | FocusEvent<HTMLElement>) => {
    const rect = event.currentTarget.getBoundingClientRect();
    setPos({ top: rect.top + rect.height / 2, left: rect.right + 8 });
  };
  const hide = () => setPos(null);

  const handlers = isEnabled
    ? { onMouseEnter: show, onMouseLeave: hide, onFocus: show, onBlur: hide }
    : {};

  const tip =
    isEnabled && pos
      ? createPortal(
          <span
            role="tooltip"
            style={{ top: pos.top, left: pos.left }}
            className="pointer-events-none fixed z-50 -translate-y-1/2 rounded-md bg-gray-900 px-2 py-1 text-xs font-medium whitespace-nowrap text-white shadow-lg"
          >
            {label}
          </span>,
          document.body,
        )
      : null;

  return { handlers, tip };
}

export function Sidebar() {
  const t = useTranslate();
  const { pathname } = useLocation();
  const { pins } = usePins();
  const projects = useProjects();
  const [isCollapsed, setIsCollapsed] = useState(
    () => localStorage.getItem(COLLAPSED_KEY) === "true",
  );

  useEffect(() => {
    localStorage.setItem(COLLAPSED_KEY, String(isCollapsed));
  }, [isCollapsed]);

  useShortcut({ key: "b", mod: true }, () => setIsCollapsed((c) => !c));
  const toggleTip = useSideLabel(t("sidebar.expand"), isCollapsed);
  const isPinDragActive = usePinDragActive();
  const [isPinDropOver, setIsPinDropOver] = useState(false);

  const pinDropProps = {
    onDragOver: (event: DragEvent) => {
      if (!hasPinDrag(event)) return;
      event.preventDefault();
      event.dataTransfer.dropEffect = "copy";
      setIsPinDropOver(true);
    },
    onDragLeave: (event: DragEvent) => {
      // Moving onto a child element also fires dragleave; ignore that.
      if (!event.currentTarget.contains(event.relatedTarget as Node | null))
        setIsPinDropOver(false);
    },
    onDrop: (event: DragEvent) => {
      setIsPinDropOver(false);
      const pin = readPinDrag(event);
      if (!pin) return;
      event.preventDefault();
      addPin(pin);
    },
  };

  const projectItems = projects.data ?? [];
  // Show the live project name and icon, so an edit elsewhere is picked up at once.
  const projectsById = new Map(projectItems.map((p) => [p.id, p]));
  const pinLabel = (pin: Pin) =>
    (pin.kind === "project" && projectsById.get(pin.id)?.name) || pin.label;
  const pinEmoji = (pin: Pin) =>
    pin.kind === "project" ? projectsById.get(pin.id)?.icon : undefined;

  return (
    <aside
      className={`sticky top-0 flex h-screen shrink-0 flex-col border-r border-gray-200 app-sidebar transition-[width] duration-200 ${
        isCollapsed ? "w-16" : "w-64"
      }`}
    >
      <div
        className={`flex h-14 shrink-0 items-center border-b border-gray-200 px-3 ${
          isCollapsed ? "justify-center" : "justify-between"
        }`}
      >
        {!isCollapsed && (
          <Link to="/home" className="px-2 text-xl font-bold">
            {t("app.name")}
          </Link>
        )}
        <button
          type="button"
          onClick={() => {
            toggleTip.handlers.onMouseLeave?.();
            setIsCollapsed(!isCollapsed);
          }}
          {...toggleTip.handlers}
          aria-label={t(isCollapsed ? "sidebar.expand" : "sidebar.collapse")}
          title={isCollapsed ? undefined : t("sidebar.collapse")}
          aria-expanded={!isCollapsed}
          className="cursor-pointer rounded-md p-1.5 text-xl text-gray-500 hover:bg-gray-200 hover:text-gray-900"
        >
          {isCollapsed ? (
            <TbLayoutSidebarLeftExpand aria-hidden />
          ) : (
            <TbLayoutSidebarLeftCollapse aria-hidden />
          )}
        </button>
        {toggleTip.tip}
      </div>

      <nav className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto p-3">
        <ul className="flex flex-col gap-1">
          <NavLink
            href="/home"
            label={t("sidebar.home")}
            Icon={FiHome}
            isActive={pathname === "/home"}
            isCollapsed={isCollapsed}
          />
          <NavLink
            href="/tasks"
            label={t("workspace.tasks.title")}
            Icon={FiCalendar}
            isActive={pathname === "/tasks"}
            isCollapsed={isCollapsed}
          />
          <NavLink
            href="/resources"
            label={t("sidebar.resources")}
            Icon={FiHardDrive}
            isActive={
              pathname === "/resources" || pathname.startsWith("/resources/")
            }
            isCollapsed={isCollapsed}
          />
        </ul>

        {(pins.length > 0 || !isCollapsed || isPinDragActive) && (
          <Section
            title={t("sidebar.pinned")}
            isCollapsed={isCollapsed}
            dropProps={pinDropProps}
            dropState={
              isPinDropOver ? "over" : isPinDragActive ? "active" : undefined
            }
          >
            {pins.length === 0 && (
              <li
                className={`flex items-center gap-2 py-1 text-xs ${
                  isCollapsed ? "justify-center px-2 py-2" : "px-3"
                } ${isPinDropOver ? "text-accent" : "text-gray-400"}`}
              >
                <TbPin aria-hidden className="shrink-0" />
                {!isCollapsed &&
                  t(
                    isPinDragActive
                      ? "sidebar.pinned.drop"
                      : "sidebar.pinned.empty",
                  )}
              </li>
            )}
            {pins.map((pin) => (
              <NavLink
                key={`${pin.kind}-${pin.id}`}
                href={pin.href}
                label={pinLabel(pin)}
                Icon={
                  pin.kind === "project"
                    ? FiFolder
                    : pin.kind === "resource"
                      ? FiFile
                      : FiCheckSquare
                }
                emoji={pinEmoji(pin)}
                isActive={false}
                isCollapsed={isCollapsed}
                onUnpin={() =>
                  unpinWhere((p) => p.kind === pin.kind && p.id === pin.id)
                }
              />
            ))}
          </Section>
        )}

        <Section
          title={t("sidebar.projects")}
          href="/projects"
          isCollapsed={isCollapsed}
        >
          {projects.isLoading && !isCollapsed && (
            <li className="px-3 py-1 text-xs text-gray-400">
              {t("common.loading")}
            </li>
          )}
          {projects.isError && !isCollapsed && (
            <li className="px-3 py-1 text-xs text-red-600">
              {t("common.error")}
            </li>
          )}
          {projects.isSuccess && projectItems.length === 0 && !isCollapsed && (
            <li className="px-3 py-1 text-xs text-gray-400">
              {t("sidebar.projects.empty")}
            </li>
          )}
          {projectItems.map((project) => {
            const href = `/projects/${project.id}`;
            return (
              <NavLink
                key={project.id}
                href={href}
                label={project.name}
                Icon={FiFolder}
                emoji={project.icon}
                isActive={pathname === href || pathname.startsWith(`${href}/`)}
                isCollapsed={isCollapsed}
                dragPin={{
                  kind: "project",
                  id: project.id,
                  label: project.name,
                  href,
                }}
              />
            );
          })}
        </Section>
      </nav>

      <ProfileMenu isCollapsed={isCollapsed} />
    </aside>
  );
}

type SectionProps = {
  title: string;
  // Makes the title a link, e.g. to the page listing the section's items.
  href?: string;
  isCollapsed: boolean;
  // Makes the section a drop target; dropState styles it while a matching
  // item is dragged ("active") or held over it ("over").
  dropProps?: {
    onDragOver: (event: DragEvent) => void;
    onDragLeave: (event: DragEvent) => void;
    onDrop: (event: DragEvent) => void;
  };
  dropState?: "active" | "over";
  children: ReactNode;
};

function Section({
  title,
  href,
  isCollapsed,
  dropProps,
  dropState,
  children,
}: SectionProps) {
  return (
    <section
      aria-label={title}
      {...dropProps}
      className={
        dropProps
          ? `-m-1 rounded-lg border border-dashed p-1 transition-colors ${
              dropState === "over"
                ? "border-accent bg-accent/10"
                : dropState === "active"
                  ? "border-gray-300 bg-gray-100"
                  : "border-transparent"
            }`
          : undefined
      }
    >
      {isCollapsed ? (
        <hr className="mx-2 mb-3 border-gray-200" />
      ) : (
        <h2 className="mb-1 px-3 text-xs font-semibold tracking-wide text-gray-400 uppercase">
          {href ? (
            <Link to={href} className="rounded hover:text-gray-900">
              {title}
            </Link>
          ) : (
            title
          )}
        </h2>
      )}
      <ul className="flex flex-col gap-1">{children}</ul>
    </section>
  );
}

type NavLinkProps = {
  href: string;
  label: string;
  Icon: IconType;
  // Shown instead of Icon when set, e.g. a project's emoji.
  emoji?: string;
  isActive: boolean;
  isCollapsed: boolean;
  onUnpin?: () => void;
  // Dragging the link carries this pin, so it can be dropped on Pinned.
  dragPin?: Pin;
};

function NavLink({
  href,
  label,
  Icon,
  emoji,
  isActive,
  isCollapsed,
  onUnpin,
  dragPin,
}: NavLinkProps) {
  const t = useTranslate();
  const sideLabel = useSideLabel(label, isCollapsed);

  return (
    <li className="group relative">
      <Link
        to={href}
        title={isCollapsed ? undefined : label}
        {...sideLabel.handlers}
        onDragStart={
          dragPin
            ? (event: DragEvent) => {
                sideLabel.handlers.onMouseLeave?.();
                setPinDragData(event, dragPin);
              }
            : undefined
        }
        aria-current={isActive ? "page" : undefined}
        className={`flex items-center gap-3 rounded-md py-2 text-sm text-gray-700 hover:bg-gray-200 aria-[current=page]:bg-gray-200 aria-[current=page]:font-semibold ${
          isCollapsed ? "justify-center px-2" : "px-3"
        } ${onUnpin && !isCollapsed ? "pr-8" : ""}`}
      >
        {emoji ? (
          <span
            aria-hidden
            className="w-4 shrink-0 text-center text-base leading-none"
          >
            {emoji}
          </span>
        ) : (
          <Icon
            aria-hidden
            className={`shrink-0 text-base ${isActive ? "text-accent" : "text-gray-500"}`}
          />
        )}
        {isCollapsed ? (
          <span className="sr-only">{label}</span>
        ) : (
          <span className="truncate">{label}</span>
        )}
      </Link>
      {sideLabel.tip}
      {onUnpin && !isCollapsed && (
        <button
          type="button"
          onClick={onUnpin}
          aria-label={`${t("card.menu.unpin")}: ${label}`}
          title={t("card.menu.unpin")}
          className="absolute top-1/2 right-1.5 hidden -translate-y-1/2 cursor-pointer rounded p-1 text-gray-400 group-focus-within:block group-hover:block hover:bg-gray-300 hover:text-gray-900"
        >
          <FiX aria-hidden />
        </button>
      )}
    </li>
  );
}

// Entries with a section open the settings modal there; the rest are
// placeholders that do nothing yet.
const profileItems: {
  label: TranslationKey;
  Icon: IconType;
  section?: SettingsSection;
}[] = [
  { label: "sidebar.profile.profile", Icon: FiUser, section: "account" },
  { label: "sidebar.profile.settings", Icon: FiSettings, section: "general" },
  { label: "sidebar.profile.appearance", Icon: FiMoon, section: "appearance" },
  {
    label: "sidebar.profile.notifications",
    Icon: FiBell,
    section: "notifications",
  },
  { label: "sidebar.profile.shortcuts", Icon: FiCommand, section: "shortcuts" },
  { label: "sidebar.profile.help", Icon: FiHelpCircle },
];

function ProfileMenu({ isCollapsed }: { isCollapsed: boolean }) {
  const t = useTranslate();
  const workspace = useWorkspace();
  const [isOpen, setIsOpen] = useState(false);
  const [settingsSection, setSettingsSection] =
    useState<SettingsSection | null>(null);
  const ref = useRef<HTMLDivElement>(null);

  useDismiss(ref, isOpen, () => setIsOpen(false));

  const username = workspace.data?.username ?? "";
  const profileTip = useSideLabel(
    username || t("sidebar.profile.label"),
    isCollapsed && !isOpen,
  );
  const close = () => setIsOpen(false);
  const closeSettings = useCallback(() => setSettingsSection(null), []);

  useShortcut({ key: ",", mod: true }, () => {
    setIsOpen(false);
    setSettingsSection("general");
  });

  return (
    <div ref={ref} className="relative shrink-0 border-t border-gray-200 p-3">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        aria-haspopup="menu"
        aria-expanded={isOpen}
        aria-label={t("sidebar.profile.label")}
        title={isCollapsed ? undefined : username || t("sidebar.profile.label")}
        {...profileTip.handlers}
        className={`flex w-full cursor-pointer items-center gap-3 rounded-md p-2 text-left hover:bg-gray-200 ${
          isCollapsed ? "justify-center" : ""
        } ${isOpen ? "bg-gray-200" : ""}`}
      >
        <span
          aria-hidden
          className="flex size-8 shrink-0 items-center justify-center rounded-full bg-gray-900 text-sm font-semibold text-white"
        >
          {username ? username[0].toUpperCase() : <FiUser />}
        </span>
        {!isCollapsed && (
          <>
            <span className="min-w-0 flex-1">
              <span className="block truncate text-sm font-medium text-gray-900">
                {username}
              </span>
              <span className="block truncate text-xs text-gray-500">
                {workspace.data?.name}
              </span>
            </span>
            <FiChevronUp
              aria-hidden
              className={`shrink-0 text-gray-400 transition-transform ${isOpen ? "" : "rotate-180"}`}
            />
          </>
        )}
      </button>
      {profileTip.tip}

      {isOpen && (
        <ul
          role="menu"
          className={`app-opaque absolute z-40 w-56 rounded-md border border-gray-200 bg-white py-1 text-sm shadow-lg ${
            isCollapsed
              ? "bottom-3 left-full ml-2"
              : "right-3 bottom-full left-3 mb-1 w-auto"
          }`}
        >
          {profileItems.map(({ label, Icon, section }) => (
            <li key={label} role="none">
              <button
                type="button"
                role="menuitem"
                onClick={() => {
                  close();
                  if (section) setSettingsSection(section);
                }}
                className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left text-gray-700 hover:bg-gray-100"
              >
                <Icon aria-hidden />
                {t(label)}
              </button>
            </li>
          ))}
          <li role="separator" className="my-1 border-t border-gray-200" />
          <li role="none">
            <button
              type="button"
              role="menuitem"
              onClick={close}
              className="flex w-full cursor-pointer items-center gap-2 px-3 py-2 text-left text-red-600 hover:bg-red-50"
            >
              <FiLogOut aria-hidden />
              {t("sidebar.profile.logout")}
            </button>
          </li>
        </ul>
      )}

      <SettingsModal
        isOpen={settingsSection !== null}
        initialSection={settingsSection ?? undefined}
        onClose={closeSettings}
      />
    </div>
  );
}
