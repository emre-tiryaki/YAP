import {
    useEffect,
    useState,
    type DragEvent,
    type KeyboardEvent,
    type PointerEvent,
} from "react";
import { useLocation } from "react-router";
import {
    FiCalendar,
    FiCheckSquare,
    FiFile,
    FiFolder,
    FiHardDrive,
    FiHome,
    FiPlus,
} from "react-icons/fi";
import {
    TbLayoutSidebarLeftCollapse,
    TbLayoutSidebarLeftExpand,
    TbPin,
} from "react-icons/tb";
import { useTranslate } from "../hooks/useTranslate";
import { addPin, unpinWhere, usePins } from "../hooks/usePins";
import {
    getPreferences,
    SIDEBAR_DEFAULT_WIDTH,
    SIDEBAR_MAX_WIDTH,
    SIDEBAR_MIN_WIDTH,
    updatePreferences,
    usePreferences,
} from "../hooks/usePreferences";
import { useMediaQuery } from "../hooks/useMediaQuery";
import { useProjects } from "../hooks/useProjects";
import { useShortcut } from "../hooks/useShortcut";
import { useSidebarExpanded } from "../hooks/useSidebarExpanded";
import { useWorkspace } from "../hooks/useWorkspace";
import type { Pin } from "../types/pin";
import { hasPinDrag, readPinDrag } from "../util/pinDrag";
import { Button } from "./Button";
import { CreateTaskForm } from "./CreateTaskForm";
import { Modal } from "./Modal";
import { ProfileMenu } from "./ProfileMenu";
import { SidebarNavItem } from "./SidebarNavItem";
import { SidebarProjectItem } from "./SidebarProjectItem";
import { SidebarSection } from "./SidebarSection";
import { Tooltip } from "./Tooltip";

const RAIL_WIDTH = 48;
const RESIZE_STEP = 8;

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

function clampWidth(width: number) {
    return Math.min(SIDEBAR_MAX_WIDTH, Math.max(SIDEBAR_MIN_WIDTH, width));
}

export function Sidebar() {
    const t = useTranslate();
    const { pathname } = useLocation();
    const { pins } = usePins();
    const projects = useProjects();
    const workspace = useWorkspace();
    const expandedIds = useSidebarExpanded();
    const { sidebarCollapsed, sidebarWidth } = usePreferences();
    // Narrow screens always get the icon rail.
    const isNarrow = useMediaQuery("(max-width: 767px)");
    const collapsed = sidebarCollapsed || isNarrow;
    // The width while the handle is dragged; saved on release.
    const [dragWidth, setDragWidth] = useState<number | null>(null);
    const [dragStart, setDragStart] = useState<{ x: number; width: number } | null>(
        null,
    );
    const [isTaskModalOpen, setIsTaskModalOpen] = useState(false);
    const isPinDragActive = usePinDragActive();
    const [isPinDropOver, setIsPinDropOver] = useState(false);

    useShortcut({ key: "b", mod: true }, () =>
        updatePreferences({ sidebarCollapsed: !getPreferences().sidebarCollapsed }),
    );

    const width = collapsed ? RAIL_WIDTH : (dragWidth ?? sidebarWidth);

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

    function handleResizeKey(event: KeyboardEvent) {
        if (event.key === "ArrowLeft")
            updatePreferences({ sidebarWidth: clampWidth(sidebarWidth - RESIZE_STEP) });
        else if (event.key === "ArrowRight")
            updatePreferences({ sidebarWidth: clampWidth(sidebarWidth + RESIZE_STEP) });
        else return;
        event.preventDefault();
    }

    const collapseLabel = t(collapsed ? "sidebar.expand" : "sidebar.collapse");

    return (
        <aside
            style={{ width }}
            className={`app-sidebar relative flex shrink-0 flex-col text-body-xs ${
                dragStart ? "" : "transition-[width] duration-200"
            }`}
        >
            <div
                className={`flex h-10 shrink-0 items-center gap-1 px-2 ${
                    collapsed ? "justify-center" : "justify-between"
                }`}
            >
                {!collapsed && (
                    <span className="truncate px-1 text-body-xs font-semibold text-fg">
                        {workspace.data?.name ?? t("app.name")}
                    </span>
                )}
                <Tooltip
                    label={collapseLabel}
                    shortcut={["mod", "B"]}
                    side={collapsed ? "right" : "bottom"}
                >
                    <button
                        type="button"
                        onClick={() =>
                            updatePreferences({
                                sidebarCollapsed: !sidebarCollapsed,
                            })
                        }
                        aria-label={collapseLabel}
                        aria-expanded={!collapsed}
                        className="cursor-pointer rounded-md p-1.5 text-lg text-fg-subtle hover:bg-layer-1-hover hover:text-fg"
                    >
                        {collapsed ? (
                            <TbLayoutSidebarLeftExpand aria-hidden />
                        ) : (
                            <TbLayoutSidebarLeftCollapse aria-hidden />
                        )}
                    </button>
                </Tooltip>
            </div>

            <div className="shrink-0 px-2 pb-2">
                {collapsed ? (
                    <Tooltip label={t("sidebar.newTask")} side="right">
                        <Button
                            variant="secondary"
                            className="w-full"
                            aria-label={t("sidebar.newTask")}
                            onClick={() => setIsTaskModalOpen(true)}
                        >
                            <FiPlus aria-hidden />
                        </Button>
                    </Tooltip>
                ) : (
                    <Button
                        variant="secondary"
                        leadingIcon={FiPlus}
                        className="w-full"
                        onClick={() => setIsTaskModalOpen(true)}
                    >
                        {t("sidebar.newTask")}
                    </Button>
                )}
            </div>

            <nav
                aria-label={t("sidebar.nav.label")}
                className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-2 pb-2"
            >
                <ul className="flex flex-col gap-0.5">
                    <SidebarNavItem
                        to="/home"
                        label={t("sidebar.home")}
                        icon={FiHome}
                        active={pathname === "/home"}
                        collapsed={collapsed}
                    />
                    <SidebarNavItem
                        to="/tasks"
                        label={t("workspace.tasks.title")}
                        icon={FiCalendar}
                        active={pathname === "/tasks"}
                        collapsed={collapsed}
                    />
                    <SidebarNavItem
                        to="/resources"
                        label={t("sidebar.resources")}
                        icon={FiHardDrive}
                        active={
                            pathname === "/resources" ||
                            pathname.startsWith("/resources/")
                        }
                        collapsed={collapsed}
                    />
                </ul>

                {(pins.length > 0 || !collapsed || isPinDragActive) && (
                    <SidebarSection
                        title={t("sidebar.pinned")}
                        collapsed={collapsed}
                        dropProps={pinDropProps}
                        dropState={
                            isPinDropOver ? "over" : isPinDragActive ? "active" : undefined
                        }
                    >
                        {pins.length === 0 && (
                            <li
                                className={`flex items-center gap-2 py-1 text-caption ${
                                    collapsed ? "justify-center px-2 py-2" : "px-2"
                                } ${isPinDropOver ? "text-accent" : "text-fg-placeholder"}`}
                            >
                                <TbPin aria-hidden className="shrink-0" />
                                {!collapsed &&
                                    t(
                                        isPinDragActive
                                            ? "sidebar.pinned.drop"
                                            : "sidebar.pinned.empty",
                                    )}
                            </li>
                        )}
                        {pins.map((pin) => (
                            <SidebarNavItem
                                key={`${pin.kind}-${pin.id}`}
                                to={pin.href}
                                label={pinLabel(pin)}
                                icon={
                                    pin.kind === "project"
                                        ? FiFolder
                                        : pin.kind === "resource"
                                          ? FiFile
                                          : FiCheckSquare
                                }
                                emoji={pinEmoji(pin)}
                                active={false}
                                collapsed={collapsed}
                                onUnpin={() =>
                                    unpinWhere(
                                        (p) => p.kind === pin.kind && p.id === pin.id,
                                    )
                                }
                            />
                        ))}
                    </SidebarSection>
                )}

                <SidebarSection
                    title={t("sidebar.projects")}
                    to="/projects"
                    collapsed={collapsed}
                >
                    {projects.isLoading && !collapsed && (
                        <li className="px-2 py-1 text-caption text-fg-placeholder">
                            {t("common.loading")}
                        </li>
                    )}
                    {projects.isError && !collapsed && (
                        <li className="px-2 py-1 text-caption text-danger">
                            {t("common.error")}
                        </li>
                    )}
                    {projects.isSuccess && projectItems.length === 0 && !collapsed && (
                        <li className="px-2 py-1 text-caption text-fg-placeholder">
                            {t("sidebar.projects.empty")}
                        </li>
                    )}
                    {projectItems.map((project) => (
                        <SidebarProjectItem
                            key={project.id}
                            project={project}
                            pathname={pathname}
                            expanded={expandedIds.includes(project.id)}
                            collapsed={collapsed}
                        />
                    ))}
                </SidebarSection>
            </nav>

            <ProfileMenu collapsed={collapsed} />

            {!collapsed && (
                <div
                    role="separator"
                    aria-orientation="vertical"
                    aria-label={t("sidebar.resize")}
                    aria-valuemin={SIDEBAR_MIN_WIDTH}
                    aria-valuemax={SIDEBAR_MAX_WIDTH}
                    aria-valuenow={width}
                    tabIndex={0}
                    onPointerDown={(event: PointerEvent<HTMLDivElement>) => {
                        event.currentTarget.setPointerCapture(event.pointerId);
                        setDragStart({ x: event.clientX, width });
                    }}
                    onPointerMove={(event: PointerEvent<HTMLDivElement>) => {
                        if (dragStart)
                            setDragWidth(
                                clampWidth(dragStart.width + event.clientX - dragStart.x),
                            );
                    }}
                    onPointerUp={() => {
                        if (dragWidth !== null)
                            updatePreferences({ sidebarWidth: dragWidth });
                        setDragStart(null);
                        setDragWidth(null);
                    }}
                    onDoubleClick={() =>
                        updatePreferences({ sidebarWidth: SIDEBAR_DEFAULT_WIDTH })
                    }
                    onKeyDown={handleResizeKey}
                    className="absolute inset-y-0 -right-1 z-20 w-1 cursor-col-resize hover:bg-accent/40 focus-visible:bg-accent/60 focus-visible:outline-none"
                />
            )}

            <Modal
                isOpen={isTaskModalOpen}
                title={t("workspace.tasks.create.title")}
                onClose={() => setIsTaskModalOpen(false)}
            >
                <CreateTaskForm
                    projects={projectItems}
                    onDone={() => setIsTaskModalOpen(false)}
                />
            </Modal>
        </aside>
    );
}
