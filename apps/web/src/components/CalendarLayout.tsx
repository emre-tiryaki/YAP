import { useRef, useState, type FormEvent } from "react";
import {
    DndContext,
    DragOverlay,
    PointerSensor,
    useDraggable,
    useDroppable,
    useSensor,
    useSensors,
    type DragEndEvent,
    type DragStartEvent,
} from "@dnd-kit/core";
import {
    FiChevronLeft,
    FiChevronRight,
    FiChevronsRight,
    FiPlus,
} from "react-icons/fi";
import { useDateFormat } from "../hooks/useDateFormat";
import { useCreateTask, useUpdateTask } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import type { Status } from "../types/status";
import type { PlanTask } from "../types/task";
import {
    addDays,
    addMonths,
    calendarDays,
    isSameDay,
    weekDays,
} from "../util/calendar";
import { fromDateInputValue, toDateInputValue } from "../util/date";
import { Button } from "./Button";
import { InlineError } from "./InlineError";
import { Popover } from "./Popover";
import { ViewToggle } from "./ViewToggle";

type CalendarLayoutProps = {
    projectId: string;
    planId: string;
    statuses: Status[];
    // Filtered and ordered.
    tasks: PlanTask[];
    onOpenTask: (task: PlanTask) => void;
};

type Mode = "month" | "week";

const MAX_CHIPS = 3;

function dayId(date: Date) {
    return `day:${toDateInputValue(date)}`;
}

export function CalendarLayout({
    projectId,
    planId,
    statuses,
    tasks,
    onOpenTask,
}: CalendarLayoutProps) {
    const t = useTranslate();
    const format = useDateFormat();
    const update = useUpdateTask();
    const [cursor, setCursor] = useState(() => new Date());
    const [mode, setMode] = useState<Mode>("month");
    const [trayOpen, setTrayOpen] = useState(true);
    const [activeId, setActiveId] = useState<string | null>(null);
    const sensors = useSensors(
        useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    );

    const days =
        mode === "month"
            ? calendarDays(cursor, format.weekStart)
            : weekDays(cursor, format.weekStart);
    const today = new Date();
    const undated = tasks.filter((task) => task.dueDate === null);
    const activeTask = tasks.find((task) => task.id === activeId);

    function step(direction: 1 | -1) {
        setCursor(
            mode === "month"
                ? addMonths(cursor, direction)
                : addDays(cursor, 7 * direction),
        );
    }

    const title =
        mode === "month"
            ? cursor.toLocaleDateString(format.locale, {
                  month: "long",
                  year: "numeric",
              })
            : `${format.date(days[0])} – ${format.date(days[6])}`;

    function handleDragStart({ active }: DragStartEvent) {
        setActiveId(String(active.id));
    }

    function handleDragEnd({ active, over }: DragEndEvent) {
        setActiveId(null);
        const task = tasks.find((x) => x.id === active.id);
        if (!task || !over) return;
        const target = String(over.id);
        if (target === "tray") {
            if (task.dueDate !== null)
                update.mutate({ taskId: task.id, input: { dueDate: null } });
        } else if (target.startsWith("day:")) {
            const date = fromDateInputValue(target.slice(4));
            if (date && (!task.dueDate || !isSameDay(task.dueDate, date)))
                update.mutate({ taskId: task.id, input: { dueDate: date } });
        }
    }

    return (
        <DndContext
            sensors={sensors}
            onDragStart={handleDragStart}
            onDragEnd={handleDragEnd}
            onDragCancel={() => setActiveId(null)}
        >
            <div className="flex min-h-0 flex-1 flex-col gap-3">
                <div className="flex flex-wrap items-center gap-2">
                    <h2 className="mr-2 text-sm font-semibold text-fg capitalize">
                        {title}
                    </h2>
                    <Button
                        variant="secondary"
                        aria-label={t("calendar.prev")}
                        onClick={() => step(-1)}
                    >
                        <FiChevronLeft aria-hidden />
                    </Button>
                    <Button variant="secondary" onClick={() => setCursor(new Date())}>
                        {t("calendar.today")}
                    </Button>
                    <Button
                        variant="secondary"
                        aria-label={t("calendar.next")}
                        onClick={() => step(1)}
                    >
                        <FiChevronRight aria-hidden />
                    </Button>
                    <div className="ml-auto flex items-center gap-2">
                        <ViewToggle<Mode>
                            value={mode}
                            onChange={setMode}
                            label={t("calendar.range")}
                            options={[
                                { view: "month", label: t("calendar.month") },
                                { view: "week", label: t("calendar.week") },
                            ]}
                        />
                        <Button
                            variant="ghost"
                            aria-pressed={trayOpen}
                            onClick={() => setTrayOpen(!trayOpen)}
                        >
                            {trayOpen
                                ? t("calendar.tray.hide")
                                : t("calendar.tray.show")}
                            <span className="text-fg-subtle">{undated.length}</span>
                        </Button>
                    </div>
                </div>
                {update.isError && (
                    <InlineError message={t("calendar.moveError")} />
                )}

                <div className="flex min-h-0 flex-1 gap-3">
                    <div
                        role="grid"
                        aria-label={title}
                        className="min-h-0 min-w-0 flex-1 overflow-auto rounded-lg border-t border-l border-line"
                    >
                        <div role="row" className="grid grid-cols-7">
                            {days.slice(0, 7).map((day) => (
                                <div
                                    key={day.getDay()}
                                    role="columnheader"
                                    className="border-r border-b border-line bg-layer-1 px-2 py-1 text-caption font-medium text-fg-subtle"
                                >
                                    {day.toLocaleDateString(format.locale, {
                                        weekday: "short",
                                    })}
                                </div>
                            ))}
                        </div>
                        <div className="grid grid-cols-7">
                            {days.map((day) => (
                                <DayCell
                                    key={dayId(day)}
                                    day={day}
                                    isToday={isSameDay(day, today)}
                                    isOutside={
                                        mode === "month" &&
                                        day.getMonth() !== cursor.getMonth()
                                    }
                                    tall={mode === "week"}
                                    tasks={tasks.filter(
                                        (task) =>
                                            task.dueDate !== null &&
                                            isSameDay(task.dueDate, day),
                                    )}
                                    statuses={statuses}
                                    projectId={projectId}
                                    planId={planId}
                                    onOpenTask={onOpenTask}
                                />
                            ))}
                        </div>
                    </div>
                    {trayOpen && (
                        <Tray tasks={undated} statuses={statuses} onOpenTask={onOpenTask} />
                    )}
                </div>
            </div>
            <DragOverlay>
                {activeTask && (
                    <Chip task={activeTask} statuses={statuses} overlay />
                )}
            </DragOverlay>
        </DndContext>
    );
}

function statusColor(statuses: Status[], task: PlanTask) {
    return statuses.find((s) => s.id === task.statusId)?.color ?? "#9ca3af";
}

function Chip({
    task,
    statuses,
    onOpen,
    overlay = false,
}: {
    task: PlanTask;
    statuses: Status[];
    onOpen?: () => void;
    overlay?: boolean;
}) {
    const isTemp = task.id.startsWith("temp-");
    const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
        id: task.id,
        disabled: overlay || isTemp,
    });
    const isDone = task.completedAt !== null;

    return (
        <button
            type="button"
            ref={overlay ? undefined : setNodeRef}
            onClick={onOpen}
            {...(overlay ? {} : { ...attributes, ...listeners })}
            className={`flex w-full cursor-pointer items-center gap-1.5 rounded border border-line bg-layer-2 px-1.5 py-0.5 text-left text-caption hover:bg-layer-2-hover focus-visible:outline-2 focus-visible:outline-accent ${
                isDragging ? "opacity-40" : ""
            } ${overlay ? "shadow-overlay" : ""}`}
        >
            <span
                aria-hidden
                className="size-2 shrink-0 rounded-full"
                style={{ backgroundColor: statusColor(statuses, task) }}
            />
            <span
                className={`truncate ${isDone ? "text-fg-subtle line-through" : "text-fg"}`}
            >
                {task.title}
            </span>
        </button>
    );
}

function DayCell({
    day,
    isToday,
    isOutside,
    tall,
    tasks,
    statuses,
    projectId,
    planId,
    onOpenTask,
}: {
    day: Date;
    isToday: boolean;
    isOutside: boolean;
    tall: boolean;
    tasks: PlanTask[];
    statuses: Status[];
    projectId: string;
    planId: string;
    onOpenTask: (task: PlanTask) => void;
}) {
    const t = useTranslate();
    const { setNodeRef, isOver } = useDroppable({ id: dayId(day) });
    const [adding, setAdding] = useState(false);
    const [title, setTitle] = useState("");
    const [moreOpen, setMoreOpen] = useState(false);
    const moreRef = useRef<HTMLButtonElement>(null);
    const create = useCreateTask();
    const shown = tasks.slice(0, MAX_CHIPS);
    const rest = tasks.length - shown.length;
    const first = statuses[0];

    function submit(event: FormEvent) {
        event.preventDefault();
        const trimmed = title.trim();
        if (!trimmed || !first) return;
        create.mutate({
            projectId,
            statusCategory: first.category,
            input: { title: trimmed, dueDate: day, planId, statusId: first.id },
        });
        setTitle("");
    }

    return (
        <div
            ref={setNodeRef}
            role="gridcell"
            className={`group relative flex flex-col gap-1 border-r border-b border-line p-1 ${
                tall ? "min-h-64" : "min-h-28"
            } ${isOutside ? "bg-layer-1/60" : ""} ${isOver ? "bg-accent/10" : ""}`}
        >
            <div className="flex items-center justify-between">
                <span
                    className={`flex size-5 items-center justify-center rounded-full text-caption ${
                        isToday
                            ? "bg-accent text-accent-fg"
                            : isOutside
                              ? "text-fg-placeholder"
                              : "text-fg-muted"
                    }`}
                >
                    {day.getDate()}
                </span>
                {first && (
                    <button
                        type="button"
                        aria-label={`${t("calendar.addTask")}: ${toDateInputValue(day)}`}
                        title={t("calendar.addTask")}
                        onClick={() => setAdding(true)}
                        className="cursor-pointer rounded p-0.5 text-fg-subtle opacity-0 group-focus-within:opacity-100 group-hover:opacity-100 hover:bg-layer-1-hover hover:text-fg"
                    >
                        <FiPlus aria-hidden />
                    </button>
                )}
            </div>
            {shown.map((task) => (
                <Chip
                    key={task.id}
                    task={task}
                    statuses={statuses}
                    onOpen={() => onOpenTask(task)}
                />
            ))}
            {rest > 0 && (
                <>
                    <button
                        type="button"
                        ref={moreRef}
                        aria-expanded={moreOpen}
                        onClick={() => setMoreOpen(!moreOpen)}
                        className="cursor-pointer rounded px-1 text-left text-caption text-fg-subtle hover:bg-layer-1-hover hover:text-fg"
                    >
                        {t("calendar.more").replace("{count}", String(rest))}
                    </button>
                    <Popover
                        open={moreOpen}
                        onClose={() => setMoreOpen(false)}
                        anchorRef={moreRef}
                        className="flex w-60 flex-col gap-1 p-2"
                    >
                        {tasks.map((task) => (
                            <Chip
                                key={task.id}
                                task={task}
                                statuses={statuses}
                                onOpen={() => {
                                    setMoreOpen(false);
                                    onOpenTask(task);
                                }}
                            />
                        ))}
                    </Popover>
                </>
            )}
            {adding && (
                <form onSubmit={submit}>
                    <input
                        autoFocus
                        value={title}
                        onChange={(event) => setTitle(event.target.value)}
                        onKeyDown={(event) => {
                            if (event.key === "Escape") {
                                event.stopPropagation();
                                setTitle("");
                                setAdding(false);
                            }
                        }}
                        onBlur={() => {
                            if (title.trim() === "") setAdding(false);
                        }}
                        placeholder={t("plan.task.create.name")}
                        aria-label={t("plan.task.create.name")}
                        className="w-full rounded border border-line-strong bg-surface-1 px-1.5 py-0.5 text-caption outline-none focus:border-accent"
                    />
                </form>
            )}
        </div>
    );
}

function Tray({
    tasks,
    statuses,
    onOpenTask,
}: {
    tasks: PlanTask[];
    statuses: Status[];
    onOpenTask: (task: PlanTask) => void;
}) {
    const t = useTranslate();
    const { setNodeRef, isOver } = useDroppable({ id: "tray" });

    return (
        <aside
            ref={setNodeRef}
            aria-label={t("calendar.noDate")}
            className={`flex w-60 shrink-0 flex-col gap-1 overflow-y-auto rounded-lg border border-line p-2 ${
                isOver ? "bg-accent/10" : "bg-layer-1"
            }`}
        >
            <h3 className="flex items-center gap-1 px-1 text-caption font-medium text-fg-subtle">
                <FiChevronsRight aria-hidden />
                {t("calendar.noDate")}
                <span>{tasks.length}</span>
            </h3>
            {tasks.map((task) => (
                <Chip
                    key={task.id}
                    task={task}
                    statuses={statuses}
                    onOpen={() => onOpenTask(task)}
                />
            ))}
        </aside>
    );
}
