import { useId, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { useMatch, useNavigate } from "react-router";
import {
    FiAlertTriangle,
    FiCheck,
    FiCheckSquare,
    FiCommand,
    FiFlag,
    FiFolder,
    FiHash,
    FiHome,
    FiLink,
    FiMonitor,
    FiMoon,
    FiPlus,
    FiSearch,
    FiSettings,
    FiSidebar,
    FiSun,
    FiTag,
    FiTrash2,
    FiX,
    FiCalendar,
    FiHardDrive,
    FiMessageSquare,
} from "react-icons/fi";
import { useChangeTaskStatus } from "../hooks/useChangeTaskStatus";
import {
    closePalette,
    setPaletteContext,
    setPalettePage,
    useCommandPalette,
    type PaletteContext,
    type PalettePage,
} from "../hooks/useCommandPalette";
import { useFocusTrap } from "../hooks/useFocusTrap";
import { useLabels } from "../hooks/useLabels";
import { usePlanStatuses } from "../hooks/usePlans";
import { getPreferences, updatePreferences } from "../hooks/usePreferences";
import { useProjects } from "../hooks/useProjects";
import { useSearch } from "../hooks/useSearch";
import { openSettings } from "../hooks/useSettingsDialog";
import { openShortcutsDialog } from "../hooks/useShortcutsDialog";
import { usePlanTasks, useSetTaskDone, useUpdateTask } from "../hooks/useTasks";
import { pushInfoToast } from "../hooks/useToasts";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import { labelDotClass } from "../util/labelColors";
import { priorities } from "../util/priority";
import { taskKey } from "../util/taskKey";
import { toSearchGroups } from "../util/searchHits";
import { CreatePlanForm } from "./CreatePlanForm";
import { CreateProjectForm } from "./CreateProjectForm";
import { CreateTaskForm } from "./CreateTaskForm";
import { Kbd } from "./Kbd";
import { Modal } from "./Modal";
import { TaskDialogs } from "./TaskDialogs";

type PaletteItem = {
    id: string;
    group: string;
    label: string;
    icon: ReactNode;
    shortcut?: string[];
    checked?: boolean;
    // Keeps the palette open (e.g. to show another page).
    keepOpen?: boolean;
    run: () => void;
    muted?: boolean;
    detail?: string;
};

type Creating = "task" | "project" | "plan" | null;

// Cmd/Ctrl+K: commands, navigation and workspace search in one list. Mounted
// once in the app shell.
export function CommandPalette() {
    const t = useTranslate();
    const { isOpen, page, context } = useCommandPalette();
    const projects = useProjects();
    const projectMatch = useMatch("/projects/:projectId/*");
    const [creating, setCreating] = useState<Creating>(null);
    const [isDeleting, setIsDeleting] = useState(false);
    const tasks = usePlanTasks(context?.projectId ?? "", context?.planId ?? "");
    const contextTask = tasks.data?.find((task) => task.id === context?.taskId);

    return (
        <>
            {isOpen && (
                <PaletteDialog
                    key={page}
                    page={page}
                    context={context}
                    projectId={projectMatch?.params.projectId}
                    onCreate={setCreating}
                    onDelete={() => setIsDeleting(true)}
                />
            )}
            <Modal
                isOpen={creating === "task"}
                title={t("workspace.tasks.create.title")}
                onClose={() => setCreating(null)}
            >
                <CreateTaskForm
                    projects={projects.data ?? []}
                    onDone={() => setCreating(null)}
                />
            </Modal>
            <Modal
                isOpen={creating === "project"}
                title={t("workspace.projects.create.title")}
                onClose={() => setCreating(null)}
            >
                <CreateProjectForm onDone={() => setCreating(null)} />
            </Modal>
            <Modal
                isOpen={creating === "plan" && projectMatch !== null}
                title={t("project.plans.create.title")}
                onClose={() => setCreating(null)}
            >
                <CreatePlanForm
                    projectId={projectMatch?.params.projectId ?? ""}
                    onDone={() => setCreating(null)}
                />
            </Modal>
            {isDeleting && contextTask && (
                <TaskDialogs
                    task={contextTask}
                    action="delete"
                    onClose={() => setIsDeleting(false)}
                />
            )}
        </>
    );
}

const themeItems = [
    { value: "light", label: "palette.theme.light", Icon: FiSun },
    { value: "dark", label: "palette.theme.dark", Icon: FiMoon },
    { value: "system", label: "palette.theme.system", Icon: FiMonitor },
] as const;

type PaletteDialogProps = {
    page: PalettePage;
    context: PaletteContext | null;
    projectId: string | undefined;
    onCreate: (creating: Creating) => void;
    onDelete: () => void;
};

function PaletteDialog({ page, context, projectId, onCreate, onDelete }: PaletteDialogProps) {
    const t = useTranslate();
    const navigate = useNavigate();
    const listId = useId();
    const panelRef = useRef<HTMLDivElement>(null);
    const [query, setQuery] = useState("");
    const [active, setActive] = useState(0);
    const projects = useProjects();
    const statuses = usePlanStatuses(context?.projectId ?? "", context?.planId ?? "");
    const tasks = usePlanTasks(context?.projectId ?? "", context?.planId ?? "");
    const task = tasks.data?.find((other) => other.id === context?.taskId);
    const changeStatus = useChangeTaskStatus(
        context?.projectId ?? "",
        context?.planId ?? "",
        statuses.data ?? [],
    );
    const setDone = useSetTaskDone(context?.projectId ?? "", context?.planId ?? "");
    const update = useUpdateTask();
    const projectLabels = useLabels(context?.projectId ?? "");
    const contextKey = taskKey(
        projects.data?.find((p) => p.id === context?.projectId)?.identifier ?? "",
        task?.number ?? 0,
    );
    const needle = query.trim().toLowerCase();
    const search = useSearch(needle.length >= 2 ? query : "");

    useFocusTrap(panelRef, true);

    function go(to: string) {
        void navigate(to);
    }

    const items: PaletteItem[] = [];
    if (page === "labels") {
        const current = task?.labelIds ?? [];
        for (const label of projectLabels.data ?? []) {
            const isOn = current.includes(label.id);
            items.push({
                id: `label:${label.id}`,
                group: t("palette.group.labels"),
                label: label.name,
                icon: (
                    <span
                        aria-hidden
                        className={`size-2.5 rounded-full ${labelDotClass(label.color)}`}
                    />
                ),
                checked: isOn,
                keepOpen: true,
                run: () =>
                    task &&
                    update.mutate({
                        taskId: task.id,
                        input: {
                            labelIds: isOn
                                ? current.filter((id) => id !== label.id)
                                : [...current, label.id],
                        },
                    }),
            });
        }
    } else if (page === "priority") {
        for (const { value, label, Icon, color } of priorities) {
            items.push({
                id: `priority:${value}`,
                group: t("palette.group.priority"),
                label: t(label),
                icon: <Icon aria-hidden className={color} />,
                checked: task?.priority === value,
                run: () =>
                    task &&
                    update.mutate({ taskId: task.id, input: { priority: value } }),
            });
        }
    } else if (page === "status") {
        for (const status of statuses.data ?? []) {
            items.push({
                id: `status:${status.id}`,
                group: t("palette.group.status"),
                label: status.title,
                icon: (
                    <span
                        aria-hidden
                        className="size-2.5 rounded-full"
                        style={{ backgroundColor: status.color }}
                    />
                ),
                checked: task?.statusId === status.id,
                run: () => task && changeStatus.change(task, status.id),
            });
        }
    } else {
        const group = (key: TranslationKey) => t(key);
        if (context && task) {
            const isDone = task.completedAt !== null;
            items.push(
                {
                    id: "ctx:status",
                    group: group("palette.group.context"),
                    label: t("palette.changeStatus"),
                    icon: <FiFlag aria-hidden />,
                    shortcut: ["S"],
                    keepOpen: true,
                    run: () => setPalettePage("status"),
                },
                {
                    id: "ctx:priority",
                    group: group("palette.group.context"),
                    label: t("palette.changePriority"),
                    icon: <FiAlertTriangle aria-hidden />,
                    shortcut: ["P"],
                    keepOpen: true,
                    run: () => setPalettePage("priority"),
                },
                {
                    id: "ctx:labels",
                    group: group("palette.group.context"),
                    label: t("palette.changeLabels"),
                    icon: <FiTag aria-hidden />,
                    shortcut: ["L"],
                    keepOpen: true,
                    run: () => setPalettePage("labels"),
                },
                {
                    id: "ctx:done",
                    group: group("palette.group.context"),
                    label: t(isDone ? "palette.markOpen" : "palette.markDone"),
                    icon: <FiCheck aria-hidden />,
                    run: () => setDone.mutate({ taskId: task.id, done: !isDone }),
                },
                {
                    id: "ctx:link",
                    group: group("palette.group.context"),
                    label: t("palette.copyLink"),
                    icon: <FiLink aria-hidden />,
                    run: () => {
                        void navigator.clipboard.writeText(
                            `${window.location.origin}/projects/${context.projectId}/plans/${context.planId}?task=${task.id}`,
                        );
                        pushInfoToast(t("task.peek.linkCopied"), task.title);
                    },
                },
                ...(contextKey
                    ? [
                          {
                              id: "ctx:id",
                              group: group("palette.group.context"),
                              label: `${t("palette.copyId")} (${contextKey})`,
                              icon: <FiHash aria-hidden />,
                              run: () => {
                                  void navigator.clipboard.writeText(contextKey);
                                  pushInfoToast(t("task.key.copied"), contextKey);
                              },
                          },
                      ]
                    : []),
                {
                    id: "ctx:delete",
                    group: group("palette.group.context"),
                    label: t("palette.deleteTask"),
                    icon: <FiTrash2 aria-hidden />,
                    run: onDelete,
                },
            );
        }
        items.push(
            {
                id: "create:task",
                group: group("palette.group.create"),
                label: t("palette.newTask"),
                icon: <FiPlus aria-hidden />,
                shortcut: ["N"],
                run: () => onCreate("task"),
            },
            {
                id: "create:project",
                group: group("palette.group.create"),
                label: t("palette.newProject"),
                icon: <FiPlus aria-hidden />,
                run: () => onCreate("project"),
            },
        );
        if (projectId)
            items.push({
                id: "create:plan",
                group: group("palette.group.create"),
                label: t("palette.newPlan"),
                icon: <FiPlus aria-hidden />,
                run: () => onCreate("plan"),
            });
        const nav = group("palette.group.navigate");
        items.push(
            { id: "nav:home", group: nav, label: t("sidebar.home"), icon: <FiHome aria-hidden />, shortcut: ["G", "H"], run: () => go("/home") },
            { id: "nav:tasks", group: nav, label: t("workspace.tasks.title"), icon: <FiCalendar aria-hidden />, shortcut: ["G", "T"], run: () => go("/tasks") },
            { id: "nav:projects", group: nav, label: t("sidebar.projects"), icon: <FiFolder aria-hidden />, shortcut: ["G", "P"], run: () => go("/projects") },
            { id: "nav:resources", group: nav, label: t("sidebar.resources"), icon: <FiHardDrive aria-hidden />, shortcut: ["G", "R"], run: () => go("/resources") },
            { id: "nav:assistant", group: nav, label: t("sidebar.assistant"), icon: <FiMessageSquare aria-hidden />, run: () => go("/assistant") },
            { id: "nav:settings", group: nav, label: t("palette.settings"), icon: <FiSettings aria-hidden />, shortcut: ["G", "S"], run: () => openSettings("general") },
        );
        for (const project of projects.data ?? [])
            items.push({
                id: `project:${project.id}`,
                group: nav,
                label: project.name,
                icon: project.icon ? (
                    <span aria-hidden className="w-4 text-center leading-none">
                        {project.icon}
                    </span>
                ) : (
                    <FiFolder aria-hidden />
                ),
                run: () => go(`/projects/${project.id}`),
            });
        const prefs = group("palette.group.preferences");
        for (const { value, label, Icon } of themeItems)
            items.push({
                id: `theme:${value}`,
                group: prefs,
                label: t(label),
                icon: <Icon aria-hidden />,
                run: () => updatePreferences({ theme: value }),
            });
        items.push(
            {
                id: "pref:sidebar",
                group: prefs,
                label: t("palette.toggleSidebar"),
                icon: <FiSidebar aria-hidden />,
                shortcut: ["mod", "B"],
                run: () =>
                    updatePreferences({
                        sidebarCollapsed: !getPreferences().sidebarCollapsed,
                    }),
            },
            {
                id: "help:shortcuts",
                group: group("palette.group.help"),
                label: t("palette.shortcuts"),
                icon: <FiCommand aria-hidden />,
                shortcut: ["mod", "/"],
                run: openShortcutsDialog,
            },
        );
    }

    const filtered = items.filter((item) => item.label.toLowerCase().includes(needle));
    // Workspace search results come after the commands.
    const hits =
        page === "root" && needle.length >= 2 && search.data
            ? toSearchGroups(search.data, t)
            : [];
    const hitItems: PaletteItem[] = hits.flatMap((group) =>
        group.hits.map((hit) => ({
            id: `hit:${hit.key}`,
            group: group.title,
            label: hit.label,
            icon: hit.icon,
            detail: hit.detail,
            muted: hit.muted,
            run: () => go(hit.href),
        })),
    );
    const all = [...filtered, ...hitItems];
    const activeIndex = all.length === 0 ? -1 : Math.min(active, all.length - 1);
    const optionId = (index: number) => `${listId}-${index}`;

    function runItem(item: PaletteItem) {
        if (!item.keepOpen) closePalette();
        item.run();
    }

    function handleKeyDown(event: KeyboardEvent) {
        if (event.key === "Escape") {
            event.preventDefault();
            event.stopPropagation();
            closePalette();
        } else if (event.key === "ArrowDown" && all.length > 0) {
            event.preventDefault();
            setActive((activeIndex + 1) % all.length);
        } else if (event.key === "ArrowUp" && all.length > 0) {
            event.preventDefault();
            setActive((activeIndex - 1 + all.length) % all.length);
        } else if (event.key === "Enter" && activeIndex >= 0) {
            event.preventDefault();
            runItem(all[activeIndex]);
        } else if (event.key === "Backspace" && query === "" && page !== "root") {
            event.preventDefault();
            setPalettePage("root");
        }
    }

    let status: string | null = null;
    if (page === "root" && needle.length >= 2 && (search.isPending || search.isPlaceholderData) && hitItems.length === 0)
        status = t("palette.searching");
    else if (all.length === 0) status = t("palette.empty");

    let lastGroup = "";
    return (
        <div
            className="fixed inset-0 z-50 animate-fade-in bg-backdrop"
            onMouseDown={(event) => {
                if (event.target === event.currentTarget) closePalette();
            }}
        >
            <div
                ref={panelRef}
                role="dialog"
                aria-modal="true"
                aria-label={t("palette.label")}
                tabIndex={-1}
                onKeyDown={handleKeyDown}
                className="app-opaque fixed top-[15vh] left-1/2 w-[min(40rem,92vw)] -translate-x-1/2 overflow-hidden rounded-lg border border-line bg-surface-1 shadow-overlay outline-none"
            >
                <div className="flex items-center gap-2 border-b border-line px-3">
                    <FiSearch aria-hidden className="shrink-0 text-fg-subtle" />
                    <input
                        autoFocus
                        role="combobox"
                        aria-expanded
                        aria-controls={listId}
                        aria-autocomplete="list"
                        aria-activedescendant={
                            activeIndex >= 0 ? optionId(activeIndex) : undefined
                        }
                        value={query}
                        onChange={(event) => {
                            setQuery(event.target.value);
                            setActive(0);
                        }}
                        placeholder={t(
                            page === "status"
                                ? "palette.placeholder.status"
                                : page === "priority"
                                  ? "palette.placeholder.priority"
                                  : page === "labels"
                                    ? "palette.placeholder.labels"
                                    : "palette.placeholder",
                        )}
                        aria-label={t("palette.search")}
                        autoComplete="off"
                        className="h-11 w-full bg-transparent text-body-xs text-fg outline-none placeholder:text-fg-placeholder"
                    />
                </div>
                {context && (
                    <div className="flex border-b border-line px-3 py-1.5">
                        <span className="inline-flex max-w-full items-center gap-1 rounded-md bg-layer-1 px-1.5 py-0.5 text-caption text-fg-muted">
                            <FiCheckSquare aria-hidden />
                            <span className="truncate">
                                {t("palette.taskContext")} ▸ {context.title}
                            </span>
                            <button
                                type="button"
                                aria-label={t("palette.clearContext")}
                                onClick={() => {
                                    setPaletteContext(null);
                                    setPalettePage("root");
                                }}
                                className="cursor-pointer text-fg-subtle hover:text-fg"
                            >
                                <FiX aria-hidden />
                            </button>
                        </span>
                    </div>
                )}
                <div
                    id={listId}
                    role="listbox"
                    aria-label={t("palette.label")}
                    className="max-h-[50vh] overflow-y-auto p-1"
                >
                    {status && (
                        <p role="status" className="px-2 py-3 text-body-xs text-fg-subtle">
                            {status}
                        </p>
                    )}
                    {all.map((item, index) => {
                        const heading = item.group !== lastGroup;
                        lastGroup = item.group;
                        return (
                            <div key={item.id} role="presentation">
                                {heading && (
                                    <p
                                        role="presentation"
                                        className="px-2 pt-2 text-caption text-fg-subtle"
                                    >
                                        {item.group}
                                    </p>
                                )}
                                <div
                                    id={optionId(index)}
                                    role="option"
                                    aria-selected={index === activeIndex}
                                    onMouseEnter={() => setActive(index)}
                                    onClick={() => runItem(item)}
                                    className={`flex h-9 cursor-pointer items-center gap-2 rounded-md px-2 text-body-xs ${
                                        index === activeIndex ? "bg-layer-1-hover" : ""
                                    } ${item.muted ? "text-fg-subtle line-through" : "text-fg"}`}
                                >
                                    <span className="flex shrink-0 items-center text-fg-subtle">
                                        {item.icon}
                                    </span>
                                    <span className="min-w-0 flex-1 truncate">
                                        {item.label}
                                    </span>
                                    {item.checked && (
                                        <FiCheck aria-hidden className="shrink-0 text-fg-subtle" />
                                    )}
                                    {item.detail && (
                                        <span className="max-w-[50%] shrink-0 truncate text-caption text-fg-placeholder">
                                            {item.detail}
                                        </span>
                                    )}
                                    {item.shortcut && <Kbd keys={item.shortcut} />}
                                </div>
                            </div>
                        );
                    })}
                </div>
            </div>
        </div>
    );
}
