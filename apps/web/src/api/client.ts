import type { ImportCounts, ImportResult } from "../types/account";
import type { Comment } from "../types/comment";
import type { RecentComment } from "../types/notification";
import type { SearchResults } from "../types/search";
import type { CreatePlanInput, Plan, UpdatePlanInput } from "../types/plan";
import type {
    CreateProjectInput,
    Project,
    UpdateProjectInput,
} from "../types/project";
import type {
    CreateStatusInput,
    Status,
    UpdateStatusInput,
} from "../types/status";
import type {
    CreateTaskInput,
    MoveTaskInput,
    PlanTask,
    StatusCategory,
    Subtask,
    Task,
    UpdateTaskInput,
} from "../types/task";
import type { UpdateWorkspaceInput, Workspace } from "../types/workspace";
import type {
    LinkedTask,
    OnConflict,
    Resource,
    ResourceAction,
    ResourceEvent,
    ResourceEventScope,
    ResourceKind,
} from "../types/resource";
import type {
    Reminder,
    UpdateUserSettingsInput,
    UserSettings,
} from "../types/userSettings";

type WorkspaceDto = {
    id: string;
    name: string;
    user_name: string;
};

type SearchResultsDto = {
    projects: { id: string; name: string; icon: string }[];
    plans: {
        id: string;
        project_id: string;
        project_name: string;
        name: string;
        icon: string;
    }[];
    tasks: {
        id: string;
        project_id: string;
        project_name: string;
        plan_id: string;
        plan_name: string;
        parent_id: string | null;
        title: string;
        done: boolean;
    }[];
    resources: {
        id: string;
        parent_id: string | null;
        parent_name: string | null;
        kind: ResourceKind;
        name: string;
        mime_type: string;
        is_text: boolean;
    }[];
};

type ProjectDto = {
    id: string;
    name: string;
    description: string;
    icon: string;
    task_count: number;
};

type TaskDto = {
    id: string;
    title: string;
    description: string;
    project_id: string;
    project_name: string;
    plan_id: string;
    status_id: string;
    position: string;
    due_date: string | null;
    status_category: StatusCategory;
    completed_at: string | null;
};

type PlanDto = {
    id: string;
    project_id: string;
    team_id: string | null;
    team_name: string | null;
    name: string;
    description: string;
    icon: string;
    task_count: number;
    done_task_count: number;
};

type StatusDto = {
    id: string;
    plan_id: string;
    title: string;
    description: string;
    color: string;
    category: StatusCategory;
    position: string;
};

type PlanTaskDto = {
    id: string;
    title: string;
    description: string;
    status_id: string;
    status_category: StatusCategory;
    position: string;
    due_date: string | null;
    completed_at: string | null;
    // Sent for plan tasks; the upcoming list leaves them out.
    subtask_count?: number;
    done_subtask_count?: number;
    comment_count?: number;
};

type SubtaskDto = {
    id: string;
    parent_id: string;
    title: string;
    plan_id: string;
    status_id: string;
    status_category: StatusCategory;
    completed_at: string | null;
};

type CommentDto = {
    id: string;
    task_id: string;
    author_id: string;
    author_name: string;
    body: string;
    created_at: string;
    updated_at: string;
    is_own: boolean;
};

type ResourceDto = {
    id: string;
    parent_id: string | null;
    kind: ResourceKind;
    name: string;
    mime_type: string;
    size: number;
    is_text: boolean;
    revision: number;
    created_at: string;
    updated_at: string;
    trashed_at: string | null;
    trashed_root_id: string | null;
};

type ResourceEventDto = {
    id: string;
    resource_id: string;
    resource_kind: ResourceKind;
    actor_id: string;
    actor_name: string;
    action: ResourceAction;
    task_id: string | null;
    details: {
        name?: string;
        from?: string;
        to?: string;
        size?: number;
        replaced?: boolean;
        task_title?: string;
    };
    created_at: string;
    updated_at: string;
};

type LinkedTaskDto = {
    id: string;
    title: string;
    project_id: string;
    project_name: string;
    plan_id: string;
    completed_at: string | null;
};

type UserSettingsDto = {
    autosave_enabled: boolean;
    autosave_delay_ms: number;
    hide_done: boolean;
    done_to_bottom: boolean;
    status_names: { todo: string; in_progress: string; done: string };
    notifications: {
        due_soon: boolean;
        reminder: Reminder;
        overdue: boolean;
        comments: boolean;
        sound: boolean;
    };
};

type UpdateUserSettingsDto = Partial<
    Omit<UserSettingsDto, "status_names" | "notifications">
> & {
    status_names?: Partial<UserSettingsDto["status_names"]>;
    notifications?: Partial<UserSettingsDto["notifications"]>;
};

type RecentCommentDto = {
    id: string;
    task_id: string;
    task_title: string;
    project_id: string;
    plan_id: string;
    parent_id: string;
    author_name: string;
    body: string;
    created_at: string;
};

export class ApiError extends Error {
    status: number;

    constructor(status: number, message: string) {
        super(message);
        this.status = status;
    }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
    return send<T>(path, {
        ...init,
        headers: { "Content-Type": "application/json" },
    });
}

async function send<T>(path: string, init: RequestInit): Promise<T> {
    const response = await fetch(path, init);

    if (!response.ok) {
        const body: unknown = await response.json().catch(() => null);
        const code =
            typeof body === "object" &&
            body !== null &&
            "error" in body &&
            typeof body.error === "string"
                ? body.error
                : "unknown";
        throw new ApiError(response.status, code);
    }

    if (response.status === 204) return undefined as T;
    return (await response.json()) as T;
}

function toProject(dto: ProjectDto): Project {
    return {
        id: dto.id,
        name: dto.name,
        description: dto.description,
        icon: dto.icon,
        taskCount: dto.task_count,
    };
}

function toTask(dto: TaskDto): Task {
    return {
        ...toPlanTask(dto),
        projectId: dto.project_id,
        projectName: dto.project_name,
        planId: dto.plan_id,
    };
}

function toPlan(dto: PlanDto): Plan {
    return {
        id: dto.id,
        projectId: dto.project_id,
        name: dto.name,
        description: dto.description,
        icon: dto.icon,
        teamName: dto.team_name,
        taskCount: dto.task_count,
        doneTaskCount: dto.done_task_count,
    };
}

function toStatus(dto: StatusDto): Status {
    return {
        id: dto.id,
        title: dto.title,
        description: dto.description,
        color: dto.color,
        category: dto.category,
        position: dto.position,
    };
}

function toPlanTask(dto: PlanTaskDto): PlanTask {
    return {
        id: dto.id,
        title: dto.title,
        description: dto.description,
        statusId: dto.status_id,
        statusCategory: dto.status_category,
        position: dto.position,
        dueDate: dto.due_date ? new Date(dto.due_date) : null,
        completedAt: dto.completed_at ? new Date(dto.completed_at) : null,
        subtaskCount: dto.subtask_count ?? 0,
        doneSubtaskCount: dto.done_subtask_count ?? 0,
        commentCount: dto.comment_count ?? 0,
    };
}

function toSubtask(dto: SubtaskDto): Subtask {
    return {
        id: dto.id,
        parentId: dto.parent_id,
        title: dto.title,
        planId: dto.plan_id,
        statusId: dto.status_id,
        statusCategory: dto.status_category,
        completedAt: dto.completed_at ? new Date(dto.completed_at) : null,
    };
}

function toComment(dto: CommentDto): Comment {
    return {
        id: dto.id,
        taskId: dto.task_id,
        authorId: dto.author_id,
        authorName: dto.author_name,
        body: dto.body,
        createdAt: new Date(dto.created_at),
        updatedAt: new Date(dto.updated_at),
        isOwn: dto.is_own,
    };
}

export async function getWorkspace(): Promise<Workspace> {
    const dto = await request<WorkspaceDto>("/api/workspace");
    return { id: dto.id, name: dto.name, username: dto.user_name };
}

export async function search(
    query: string,
    signal?: AbortSignal,
): Promise<SearchResults> {
    const params = new URLSearchParams({ q: query });
    const dto = await request<SearchResultsDto>(`/api/search?${params}`, {
        signal,
    });
    return {
        projects: dto.projects.map((p) => ({
            id: p.id,
            name: p.name,
            icon: p.icon,
        })),
        plans: dto.plans.map((p) => ({
            id: p.id,
            projectId: p.project_id,
            projectName: p.project_name,
            name: p.name,
            icon: p.icon,
        })),
        tasks: dto.tasks.map((t) => ({
            id: t.id,
            projectId: t.project_id,
            projectName: t.project_name,
            planId: t.plan_id,
            planName: t.plan_name,
            parentId: t.parent_id,
            title: t.title,
            done: t.done,
        })),
        resources: dto.resources.map((r) => ({
            id: r.id,
            parentId: r.parent_id,
            parentName: r.parent_name,
            kind: r.kind,
            name: r.name,
            mimeType: r.mime_type,
            isText: r.is_text,
        })),
    };
}

export async function listProjects(): Promise<Project[]> {
    const dtos = await request<ProjectDto[]>("/api/projects");
    return dtos.map(toProject);
}

export async function createProject(
    input: CreateProjectInput,
): Promise<Project> {
    const dto = await request<ProjectDto>("/api/projects", {
        method: "POST",
        body: JSON.stringify(input),
    });
    return toProject(dto);
}

export async function getProject(projectId: string): Promise<Project> {
    const dto = await request<ProjectDto>(`/api/projects/${projectId}`)
    return toProject(dto)
}

export async function listPlans(projectId: string): Promise<Plan[]> {
    const dtos = await request<PlanDto[]>(`/api/projects/${projectId}/plans`)
    return dtos.map(toPlan)
}

export async function createPlan(
    projectId: string,
    input: CreatePlanInput,
): Promise<Plan> {
    const dto = await request<PlanDto>(`/api/projects/${projectId}/plans`, {
        method: "POST",
        body: JSON.stringify(input),
    });
    return toPlan(dto);
}

export async function listUpcomingTasks(): Promise<Task[]> {
    const dtos = await request<TaskDto[]>(`/api/tasks/upcoming`);
    return dtos.map(toTask);
}

export async function createTask(
    projectId: string,
    input: CreateTaskInput,
): Promise<Task> {
    const dto = await request<TaskDto>(`/api/projects/${projectId}/tasks`, {
        method: "POST",
        body: JSON.stringify({
            title: input.title,
            description: input.description ?? "",
            due_date: input.dueDate ? input.dueDate.toISOString() : null,
            plan_id: input.planId ?? "",
            status_id: input.statusId ?? "",
        }),
    });
    return toTask(dto);
}

function planPath(projectId: string, planId: string): string {
    return `/api/projects/${projectId}/plans/${planId}`;
}

export async function getPlan(projectId: string, planId: string): Promise<Plan> {
    const dto = await request<PlanDto>(planPath(projectId, planId));
    return toPlan(dto);
}

export async function listStatuses(
    projectId: string,
    planId: string,
): Promise<Status[]> {
    const dtos = await request<StatusDto[]>(
        `${planPath(projectId, planId)}/statuses`,
    );
    return dtos.map(toStatus);
}

export async function createStatus(
    projectId: string,
    planId: string,
    input: CreateStatusInput,
): Promise<Status> {
    const dto = await request<StatusDto>(
        `${planPath(projectId, planId)}/statuses`,
        { method: "POST", body: JSON.stringify(input) },
    );
    return toStatus(dto);
}

export async function updateStatus(
    projectId: string,
    planId: string,
    statusId: string,
    input: UpdateStatusInput,
): Promise<Status> {
    const dto = await request<StatusDto>(
        `${planPath(projectId, planId)}/statuses/${statusId}`,
        { method: "PATCH", body: JSON.stringify(input) },
    );
    return toStatus(dto);
}

// Deletes the status together with the tasks in it.
export async function deleteStatus(
    projectId: string,
    planId: string,
    statusId: string,
): Promise<void> {
    await request<void>(`${planPath(projectId, planId)}/statuses/${statusId}`, {
        method: "DELETE",
    });
}

// Places the status at `index` among the other statuses of its plan.
export async function moveStatus(
    projectId: string,
    planId: string,
    statusId: string,
    index: number,
): Promise<void> {
    await request<void>(
        `${planPath(projectId, planId)}/statuses/${statusId}/move`,
        { method: "POST", body: JSON.stringify({ index }) },
    );
}

export async function listPlanTasks(
    projectId: string,
    planId: string,
): Promise<PlanTask[]> {
    const dtos = await request<PlanTaskDto[]>(
        `${planPath(projectId, planId)}/tasks`,
    );
    return dtos.map(toPlanTask);
}

export async function moveTask(
    taskId: string,
    input: MoveTaskInput,
): Promise<void> {
    await request<void>(`/api/tasks/${taskId}/move`, {
        method: "POST",
        body: JSON.stringify({ status_id: input.statusId, index: input.index }),
    });
}

export async function deleteTask(taskId: string): Promise<void> {
    await request<void>(`/api/tasks/${taskId}`, { method: "DELETE" });
}

export async function updateProject(
    projectId: string,
    input: UpdateProjectInput,
): Promise<Project> {
    const dto = await request<ProjectDto>(`/api/projects/${projectId}`, {
        method: "PATCH",
        body: JSON.stringify(input),
    });
    return toProject(dto);
}

export async function deleteProject(projectId: string): Promise<void> {
    await request<void>(`/api/projects/${projectId}`, { method: "DELETE" });
}

export async function updatePlan(
    projectId: string,
    planId: string,
    input: UpdatePlanInput,
): Promise<Plan> {
    const dto = await request<PlanDto>(planPath(projectId, planId), {
        method: "PATCH",
        body: JSON.stringify(input),
    });
    return toPlan(dto);
}

export async function deletePlan(
    projectId: string,
    planId: string,
): Promise<void> {
    await request<void>(planPath(projectId, planId), { method: "DELETE" });
}

// Only the fields present in `input` are sent; a null due date clears it.
export async function updateTask(
    taskId: string,
    input: UpdateTaskInput,
): Promise<PlanTask> {
    const body: Record<string, string | null> = {};
    if (input.title !== undefined) body.title = input.title;
    if (input.description !== undefined) body.description = input.description;
    if (input.dueDate !== undefined)
        body.due_date = input.dueDate ? input.dueDate.toISOString() : null;
    const dto = await request<PlanTaskDto>(`/api/tasks/${taskId}`, {
        method: "PATCH",
        body: JSON.stringify(body),
    });
    return toPlanTask(dto);
}

export async function listSubtasks(taskId: string): Promise<Subtask[]> {
    const dtos = await request<SubtaskDto[]>(`/api/tasks/${taskId}/subtasks`);
    return dtos.map(toSubtask);
}

export async function createSubtask(
    parentId: string,
    title: string,
): Promise<Subtask> {
    const dto = await request<SubtaskDto>(`/api/tasks/${parentId}/subtasks`, {
        method: "POST",
        body: JSON.stringify({ title }),
    });
    return toSubtask(dto);
}

// Marks the task finished or reopens it. Its status and position are kept.
export async function setTaskDone(
    taskId: string,
    done: boolean,
): Promise<Subtask> {
    const dto = await request<SubtaskDto>(`/api/tasks/${taskId}/done`, {
        method: "POST",
        body: JSON.stringify({ done }),
    });
    return toSubtask(dto);
}

export async function listComments(taskId: string): Promise<Comment[]> {
    const dtos = await request<CommentDto[]>(`/api/tasks/${taskId}/comments`);
    return dtos.map(toComment);
}

export async function createComment(
    taskId: string,
    body: string,
): Promise<Comment> {
    const dto = await request<CommentDto>(`/api/tasks/${taskId}/comments`, {
        method: "POST",
        body: JSON.stringify({ body }),
    });
    return toComment(dto);
}

export async function updateComment(
    commentId: string,
    body: string,
): Promise<Comment> {
    const dto = await request<CommentDto>(`/api/comments/${commentId}`, {
        method: "PATCH",
        body: JSON.stringify({ body }),
    });
    return toComment(dto);
}

export async function deleteComment(commentId: string): Promise<void> {
    await request<void>(`/api/comments/${commentId}`, { method: "DELETE" });
}

export async function updateWorkspace(
    input: UpdateWorkspaceInput,
): Promise<Workspace> {
    const dto = await request<WorkspaceDto>("/api/workspace", {
        method: "PATCH",
        body: JSON.stringify({ name: input.name, user_name: input.username }),
    });
    return { id: dto.id, name: dto.name, username: dto.user_name };
}

function toResource(dto: ResourceDto): Resource {
    return {
        id: dto.id,
        parentId: dto.parent_id,
        kind: dto.kind,
        name: dto.name,
        mimeType: dto.mime_type,
        size: dto.size,
        isText: dto.is_text,
        revision: dto.revision,
        createdAt: new Date(dto.created_at),
        updatedAt: new Date(dto.updated_at),
        trashedAt: dto.trashed_at ? new Date(dto.trashed_at) : null,
        trashedRootId: dto.trashed_root_id,
    };
}

function toResourceEvent(dto: ResourceEventDto): ResourceEvent {
    return {
        id: dto.id,
        resourceId: dto.resource_id,
        resourceKind: dto.resource_kind,
        actorName: dto.actor_name,
        action: dto.action,
        taskId: dto.task_id,
        details: {
            name: dto.details.name ?? "",
            from: dto.details.from,
            to: dto.details.to,
            size: dto.details.size,
            replaced: dto.details.replaced,
            taskTitle: dto.details.task_title,
        },
        createdAt: new Date(dto.created_at),
        updatedAt: new Date(dto.updated_at),
    };
}

function toLinkedTask(dto: LinkedTaskDto): LinkedTask {
    return {
        id: dto.id,
        title: dto.title,
        projectId: dto.project_id,
        projectName: dto.project_name,
        planId: dto.plan_id,
        completedAt: dto.completed_at ? new Date(dto.completed_at) : null,
    };
}

// Every live resource of the workspace; the tree is built from parentId.
export async function listResources(): Promise<Resource[]> {
    const dtos = await request<ResourceDto[]>("/api/resources");
    return dtos.map(toResource);
}

export async function getResource(id: string): Promise<Resource> {
    return toResource(await request<ResourceDto>(`/api/resources/${id}`));
}

export async function listTrash(): Promise<Resource[]> {
    const dtos = await request<ResourceDto[]>("/api/resources/trash");
    return dtos.map(toResource);
}

export async function emptyTrash(): Promise<void> {
    await request<void>("/api/resources/trash", { method: "DELETE" });
}

export async function createFolder(
    parentId: string | null,
    name: string,
): Promise<Resource> {
    const dto = await request<ResourceDto>("/api/resources/folders", {
        method: "POST",
        body: JSON.stringify({ parent_id: parentId, name }),
    });
    return toResource(dto);
}

// Creates an empty file, ready to be edited.
export async function createFile(
    parentId: string | null,
    name: string,
): Promise<Resource> {
    const dto = await request<ResourceDto>("/api/resources/files", {
        method: "POST",
        body: JSON.stringify({ parent_id: parentId, name }),
    });
    return toResource(dto);
}

export async function uploadFile(
    parentId: string | null,
    file: File,
    onConflict: OnConflict,
    name = file.name,
): Promise<Resource> {
    const query = new URLSearchParams({ name, on_conflict: onConflict });
    if (parentId) query.set("parent_id", parentId);
    const dto = await send<ResourceDto>(`/api/resources/upload?${query}`, {
        method: "POST",
        headers: file.type ? { "Content-Type": file.type } : undefined,
        body: file,
    });
    return toResource(dto);
}

// URL of a file's bytes, for previews (inline) and downloads. The revision
// busts the browser cache once the content changes.
export function resourceContentUrl(
    resource: Pick<Resource, "id" | "revision">,
    download = false,
): string {
    const query = new URLSearchParams({ v: String(resource.revision) });
    if (download) query.set("download", "1");
    return `/api/resources/${resource.id}/content?${query}`;
}

export async function readResourceText(
    resource: Pick<Resource, "id" | "revision">,
): Promise<string> {
    const response = await fetch(resourceContentUrl(resource));
    if (!response.ok) {
        const body: unknown = await response.json().catch(() => null);
        const code =
            typeof body === "object" &&
            body !== null &&
            "error" in body &&
            typeof body.error === "string"
                ? body.error
                : "unknown";
        throw new ApiError(response.status, code);
    }
    return response.text();
}

// Saves the text of a file. Rejected with 409 "stale" when the file changed
// since `revision` was read.
export async function saveResourceText(
    id: string,
    revision: number,
    text: string,
): Promise<Resource> {
    const dto = await send<ResourceDto>(
        `/api/resources/${id}/content?revision=${revision}`,
        {
            method: "PUT",
            headers: { "Content-Type": "text/plain; charset=utf-8" },
            body: text,
        },
    );
    return toResource(dto);
}

export async function renameResource(
    id: string,
    name: string,
): Promise<Resource> {
    const dto = await request<ResourceDto>(`/api/resources/${id}`, {
        method: "PATCH",
        body: JSON.stringify({ name }),
    });
    return toResource(dto);
}

export async function moveResource(
    id: string,
    parentId: string | null,
    onConflict: OnConflict,
): Promise<Resource> {
    const dto = await request<ResourceDto>(`/api/resources/${id}/move`, {
        method: "POST",
        body: JSON.stringify({ parent_id: parentId, on_conflict: onConflict }),
    });
    return toResource(dto);
}

export async function copyResource(
    id: string,
    parentId: string | null,
    onConflict: OnConflict,
): Promise<Resource> {
    const dto = await request<ResourceDto>(`/api/resources/${id}/copy`, {
        method: "POST",
        body: JSON.stringify({ parent_id: parentId, on_conflict: onConflict }),
    });
    return toResource(dto);
}

export async function trashResource(id: string): Promise<void> {
    await request<void>(`/api/resources/${id}/trash`, { method: "POST" });
}

export async function restoreResource(id: string): Promise<Resource> {
    const dto = await request<ResourceDto>(`/api/resources/${id}/restore`, {
        method: "POST",
    });
    return toResource(dto);
}

// Deletes a trashed item for good.
export async function purgeResource(id: string): Promise<void> {
    await request<void>(`/api/resources/${id}`, { method: "DELETE" });
}

function eventsPath(scope: ResourceEventScope): string {
    switch (scope.kind) {
        case "workspace":
            return "/api/resources/events";
        case "resource":
            return `/api/resources/${scope.id}/events`;
        case "project":
            return `/api/projects/${scope.id}/resource-events`;
    }
}

// A page of the activity log, newest first; `before` is the id of the oldest
// event already shown.
export async function listResourceEvents(
    scope: ResourceEventScope,
    before: string | null,
    limit: number,
): Promise<ResourceEvent[]> {
    const query = new URLSearchParams({ limit: String(limit) });
    if (before) query.set("before", before);
    const dtos = await request<ResourceEventDto[]>(
        `${eventsPath(scope)}?${query}`,
    );
    return dtos.map(toResourceEvent);
}

export async function listResourceTasks(id: string): Promise<LinkedTask[]> {
    const dtos = await request<LinkedTaskDto[]>(`/api/resources/${id}/tasks`);
    return dtos.map(toLinkedTask);
}

// Resources linked to a task; trashed ones included (trashedAt is set).
export async function listTaskResources(taskId: string): Promise<Resource[]> {
    const dtos = await request<ResourceDto[]>(`/api/tasks/${taskId}/resources`);
    return dtos.map(toResource);
}

export async function linkResource(
    taskId: string,
    resourceId: string,
): Promise<void> {
    await request<void>(`/api/tasks/${taskId}/resources/${resourceId}`, {
        method: "PUT",
    });
}

export async function unlinkResource(
    taskId: string,
    resourceId: string,
): Promise<void> {
    await request<void>(`/api/tasks/${taskId}/resources/${resourceId}`, {
        method: "DELETE",
    });
}

function toUserSettings(dto: UserSettingsDto): UserSettings {
    const n = dto.notifications;
    return {
        autosaveEnabled: dto.autosave_enabled,
        autosaveDelayMs: dto.autosave_delay_ms,
        hideDone: dto.hide_done,
        doneToBottom: dto.done_to_bottom,
        statusNames: { ...dto.status_names },
        notifications: {
            dueSoon: n.due_soon,
            reminder: n.reminder,
            overdue: n.overdue,
            comments: n.comments,
            sound: n.sound,
        },
    };
}

export async function getUserSettings(): Promise<UserSettings> {
    return toUserSettings(await request<UserSettingsDto>("/api/settings"));
}

export async function updateUserSettings(
    input: UpdateUserSettingsInput,
): Promise<UserSettings> {
    const body: UpdateUserSettingsDto = {};
    if (input.autosaveEnabled !== undefined)
        body.autosave_enabled = input.autosaveEnabled;
    if (input.autosaveDelayMs !== undefined)
        body.autosave_delay_ms = input.autosaveDelayMs;
    if (input.hideDone !== undefined) body.hide_done = input.hideDone;
    if (input.doneToBottom !== undefined)
        body.done_to_bottom = input.doneToBottom;
    if (input.statusNames) body.status_names = { ...input.statusNames };
    const n = input.notifications;
    if (n) {
        body.notifications = {};
        if (n.dueSoon !== undefined) body.notifications.due_soon = n.dueSoon;
        if (n.reminder !== undefined) body.notifications.reminder = n.reminder;
        if (n.overdue !== undefined) body.notifications.overdue = n.overdue;
        if (n.comments !== undefined) body.notifications.comments = n.comments;
        if (n.sound !== undefined) body.notifications.sound = n.sound;
    }
    const dto = await request<UserSettingsDto>("/api/settings", {
        method: "PATCH",
        body: JSON.stringify(body),
    });
    return toUserSettings(dto);
}

// Comments other members wrote at or after `since`, oldest first.
export async function listRecentComments(since: Date): Promise<RecentComment[]> {
    const params = new URLSearchParams({ since: toSecondsISO(since) });
    const dtos = await request<RecentCommentDto[]>(
        `/api/comments/recent?${params}`,
    );
    return dtos.map((dto) => ({
        id: dto.id,
        taskId: dto.task_id,
        taskTitle: dto.task_title,
        projectId: dto.project_id,
        planId: dto.plan_id,
        parentId: dto.parent_id || null,
        authorName: dto.author_name,
        body: dto.body,
        createdAt: new Date(dto.created_at),
    }));
}

// RFC 3339 without fractional seconds, as the backend parses it.
function toSecondsISO(date: Date): string {
    return date.toISOString().replace(/\.\d{3}Z$/, "Z");
}

// URL of the data export: a ZIP with data.json and the resource files. It is
// a plain download, so it is opened as a link instead of fetched.
export const accountExportUrl = "/api/account/export";

type ImportResultDto = {
    restored: boolean;
    added: ImportCounts;
    existing: number;
    renamed: number;
    missing_files: string[];
};

// Imports an archive made by the export. Nothing is deleted: what is still
// here is kept, the rest is added.
export async function importAccount(file: File): Promise<ImportResult> {
    const dto = await send<ImportResultDto>("/api/account/import", {
        method: "POST",
        headers: { "Content-Type": "application/zip" },
        body: file,
    });
    return {
        restored: dto.restored,
        added: dto.added,
        existing: dto.existing,
        renamed: dto.renamed,
        missingFiles: dto.missing_files,
    };
}

export async function resetAccount(): Promise<void> {
    await request<void>("/api/account/reset", { method: "POST" });
}
