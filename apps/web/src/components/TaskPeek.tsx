import { useEffect, useId, useRef, useState } from "react";
import { usePlanStatuses } from "../hooks/usePlans";
import { useFocusTrap } from "../hooks/useFocusTrap";
import { usePreferences } from "../hooks/usePreferences";
import { usePlanTasks } from "../hooks/useTasks";
import { useTranslate } from "../hooks/useTranslate";
import { TaskBody } from "./TaskBody";
import { TaskDialogs } from "./TaskDialogs";
import { TaskPeekHeader } from "./TaskPeekHeader";

type TaskPeekProps = {
    projectId: string;
    planId: string;
    taskId: string;
    onClose: () => void;
};

const frames = {
    side: "fixed inset-y-2 right-2 z-50 w-[min(50vw,48rem)] min-w-[min(28rem,calc(100vw-1rem))] animate-drawer-in",
    modal: "h-[85vh] w-full max-w-4xl",
    full: "fixed inset-4 z-50",
};

// A task opened over a plan: as a side panel that leaves the list clickable, a
// centred dialog, or nearly full screen (the peek mode preference). Loads the
// plan's statuses and tasks itself, so any page can open one.
export function TaskPeek({ projectId, planId, taskId, onClose }: TaskPeekProps) {
    const t = useTranslate();
    const { taskPeekMode: mode } = usePreferences();
    const statuses = usePlanStatuses(projectId, planId);
    const tasks = usePlanTasks(projectId, planId);
    const task = tasks.data?.find((other) => other.id === taskId);
    const titleId = useId();
    const panelRef = useRef<HTMLDivElement>(null);
    const [isDeleting, setIsDeleting] = useState(false);
    const onCloseRef = useRef(onClose);
    const isTrapped = mode !== "side";

    useEffect(() => {
        onCloseRef.current = onClose;
    });
    useFocusTrap(panelRef, isTrapped && task !== undefined);

    // Esc closes; in the side mode focus goes back to where it was (the row).
    useEffect(() => {
        const previous = document.activeElement as HTMLElement | null;
        function handleKeyDown(event: KeyboardEvent) {
            if (event.key !== "Escape") return;
            // Blurring first saves a field that is still being edited.
            if (document.activeElement instanceof HTMLElement)
                document.activeElement.blur();
            onCloseRef.current();
        }
        document.addEventListener("keydown", handleKeyDown);
        return () => {
            document.removeEventListener("keydown", handleKeyDown);
            if (!isTrapped) previous?.focus?.();
        };
    }, [isTrapped]);

    // The task is gone (deleted, or a stale link): leave.
    const isMissing = tasks.isSuccess && task === undefined;
    useEffect(() => {
        if (isMissing) onCloseRef.current();
    }, [isMissing]);

    if (!task || !statuses.data) return null;

    const panel = (
        <div
            ref={panelRef}
            role="dialog"
            aria-modal={isTrapped}
            aria-labelledby={titleId}
            tabIndex={-1}
            className={`app-opaque flex flex-col rounded-lg border border-line bg-surface-1 shadow-overlay outline-none ${frames[mode]}`}
        >
            <TaskPeekHeader
                task={task}
                projectId={projectId}
                planId={planId}
                mode={mode}
                onClose={onClose}
                onDelete={() => setIsDeleting(true)}
            />
            <h2 id={titleId} className="sr-only">
                {task.title}
            </h2>
            <div className="min-h-0 flex-1 overflow-auto">
                {/* Keyed so the local field state resets when another task opens. */}
                <TaskBody
                    key={task.id}
                    projectId={projectId}
                    planId={planId}
                    task={task}
                    statuses={statuses.data}
                    full={mode === "full"}
                />
            </div>
        </div>
    );

    return (
        <>
            {mode === "side" ? (
                panel
            ) : (
                <div
                    aria-label={t("task.detail.label")}
                    className="fixed inset-0 z-50 flex animate-fade-in items-center justify-center bg-backdrop p-4"
                    onMouseDown={(event) => {
                        if (event.target === event.currentTarget) onClose();
                    }}
                >
                    {panel}
                </div>
            )}
            {isDeleting && (
                <TaskDialogs
                    task={task}
                    action="delete"
                    onClose={() => setIsDeleting(false)}
                />
            )}
        </>
    );
}
