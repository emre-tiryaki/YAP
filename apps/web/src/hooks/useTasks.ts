import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
    CreateTaskInput,
    MoveTaskInput,
    PlanTask,
    StatusCategory,
    Subtask,
    UpdateTaskInput,
} from "../types/task";
import {
    createSubtask,
    createTask,
    deleteTask,
    listPlanTasks,
    listUpcomingTasks,
    listSubtasks,
    moveTask,
    setTaskDone,
    updateTask,
} from "../api/client";
import { planKey } from "./usePlans";
import { relabelPin, unpinWhere } from "./usePins";

type CreateTaskVariables = {
    projectId: string;
    input: CreateTaskInput;
};

export type MoveTaskVariables = MoveTaskInput & {
    taskId: string;
    statusCategory: StatusCategory;
};

export function useUpcomingTasks() {
    return useQuery({
        queryKey: ["tasks", "upcoming"],
        queryFn: listUpcomingTasks,
    });
}

// Task counts show up on the workspace, project and plan pages, so every
// task write refreshes all project queries plus the upcoming list.
function invalidateTaskViews(queryClient: ReturnType<typeof useQueryClient>) {
    return Promise.all([
        queryClient.invalidateQueries({ queryKey: ["tasks"] }),
        queryClient.invalidateQueries({ queryKey: ["projects"] }),
    ]);
}

export function useCreateTask() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ projectId, input }: CreateTaskVariables) =>
            createTask(projectId, input),
        onSuccess: () => invalidateTaskViews(queryClient),
    });
}

function planTasksKey(projectId: string, planId: string) {
    return [...planKey(projectId, planId), "tasks"];
}

export function usePlanTasks(projectId: string, planId: string) {
    return useQuery({
        queryKey: planTasksKey(projectId, planId),
        queryFn: () => listPlanTasks(projectId, planId),
        enabled: projectId !== "" && planId !== "",
    });
}

// Places the task at `index` among the other tasks of the target status. The
// list keeps the server's order, so grouping by status needs no re-sorting.
// Applying it twice gives the same result.
function applyMove(tasks: PlanTask[], move: MoveTaskVariables): PlanTask[] {
    const task = tasks.find((t) => t.id === move.taskId);
    if (!task) return tasks;

    const rest = tasks.filter((t) => t.id !== move.taskId);
    const siblings = rest.filter((t) => t.statusId === move.statusId);
    const moved: PlanTask = {
        ...task,
        statusId: move.statusId,
        statusCategory: move.statusCategory,
    };
    const anchor = siblings[move.index];
    if (!anchor) return [...rest, moved];
    const at = rest.indexOf(anchor);
    return [...rest.slice(0, at), moved, ...rest.slice(at)];
}

export function useMoveTask(projectId: string, planId: string) {
    const queryClient = useQueryClient();
    const key = planTasksKey(projectId, planId);

    return useMutation({
        mutationFn: ({ taskId, statusId, index }: MoveTaskVariables) =>
            moveTask(taskId, { statusId, index }),
        onMutate: async (move) => {
            queryClient.setQueryData<PlanTask[]>(
                key,
                (old) => old && applyMove(old, move),
            );
            // An in-flight refetch would overwrite the optimistic order, so
            // cancel it and re-apply the (idempotent) move afterwards.
            await queryClient.cancelQueries({ queryKey: key });
            queryClient.setQueryData<PlanTask[]>(
                key,
                (old) => old && applyMove(old, move),
            );
        },
        // On failure the refetch restores the server's order. A status
        // change can move the task in or out of the upcoming list too.
        onSettled: () =>
            Promise.all([
                queryClient.invalidateQueries({
                    queryKey: planKey(projectId, planId),
                }),
                queryClient.invalidateQueries({
                    queryKey: ["tasks", "upcoming"],
                }),
            ]),
    });
}

export function useDeleteTask() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: deleteTask,
        onSuccess: (_data, taskId) => {
            unpinWhere((pin) => pin.kind === "task" && pin.id === taskId);
            return invalidateTaskViews(queryClient);
        },
    });
}

type UpdateTaskVariables = {
    taskId: string;
    input: UpdateTaskInput;
};

export function useUpdateTask() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: ({ taskId, input }: UpdateTaskVariables) =>
            updateTask(taskId, input),
        onSuccess: (task) => {
            relabelPin("task", task.id, task.title);
            return invalidateTaskViews(queryClient);
        },
    });
}

function subtasksKey(taskId: string) {
    return ["tasks", taskId, "subtasks"];
}

export function useSubtasks(taskId: string) {
    return useQuery({
        queryKey: subtasksKey(taskId),
        queryFn: () => listSubtasks(taskId),
        enabled: taskId !== "",
    });
}

export function useCreateSubtask(parentId: string) {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: (title: string) => createSubtask(parentId, title),
        onSuccess: () => invalidateTaskViews(queryClient),
    });
}

type SetDoneVariables = {
    taskId: string;
    done: boolean;
};

// Checks a subtask off (or back on). The checkbox flips at once; the refetch
// afterwards brings the real status and the parent's counts.
export function useSetSubtaskDone(parentId: string) {
    const queryClient = useQueryClient();
    const key = subtasksKey(parentId);

    return useMutation({
        mutationFn: ({ taskId, done }: SetDoneVariables) =>
            setTaskDone(taskId, done),
        onMutate: async ({ taskId, done }) => {
            await queryClient.cancelQueries({ queryKey: key });
            queryClient.setQueryData<Subtask[]>(key, (old) =>
                old?.map((sub) =>
                    sub.id === taskId
                        ? {
                              ...sub,
                              statusCategory: done ? "done" : "todo",
                          }
                        : sub,
                ),
            );
        },
        onSettled: () => invalidateTaskViews(queryClient),
    });
}
