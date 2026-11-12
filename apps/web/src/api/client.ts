import type { Comment } from "../types/comment";
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
import type { Workspace } from "../types/workspace";

type WorkspaceDto = {
    id: string;
    name: string;
    user_name: string;
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

export class ApiError extends Error {
    status: number;

    constructor(status: number, message: string) {
        super(message);
        this.status = status;
    }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
    const response = await fetch(path, {
        ...init,
        headers: { "Content-Type": "application/json" },
    });

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

// Moves the task to its plan's first done status, or back to the first todo one.
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
