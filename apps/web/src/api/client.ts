import type { CreateProjectInput, Project } from "../types/project"
import type { CreateTaskInput, StatusCategory, Task } from "../types/task"
import type { Workspace } from "../types/workspace"

type WorkspaceDto = {
    id: string
    name: string
    user_name: string
}

type ProjectDto = {
    id: string
    name: string
    description: string
    task_count: number
}

type TaskDto = {
    id: string
    title: string
    project_id: string
    project_name: string
    due_date: string | null
    status_category: StatusCategory
}

export class ApiError extends Error {
    status: number

    constructor(status: number, message: string) {
        super(message)
        this.status = status
    }
}

async function request<T>(path:string, init?: RequestInit): Promise<T> {
    const response = await fetch(path, {
        ...init,
        headers: { 'Content-Type': 'application/json' },
    })

    if (!response.ok) {
        const body: unknown = await response.json().catch(() => null)
        const code = typeof body === 'object' && body !== null && 'error' in body && typeof body.error === 'string' ? body.error : 'unknown'
        throw new ApiError(response.status, code)
    }

    return (await response.json()) as T
}

function toProject(dto: ProjectDto): Project {
    return {
        id: dto.id,
        name: dto.name,
        description: dto.description,
        taskCount: dto.task_count
    }
}

function toTask(dto: TaskDto): Task {
    return {
        id: dto.id,
        projectName: dto.project_name,
        title: dto.title,
        dueDate: dto.due_date ? new Date(dto.due_date) : null,
        statusCategory: dto.status_category,
    };
}

export async function getWorkspace(): Promise<Workspace> {
    const dto = await request<WorkspaceDto>("/api/workspace")
    return {id: dto.id, name: dto.name, username: dto.user_name}
}

export async function listProjects(): Promise<Project[]> {
    const dtos = await request<ProjectDto[]>("/api/projects")
    return dtos.map(toProject)
}

export async function createProject(input: CreateProjectInput): Promise<Project> {
    const dto = await request<ProjectDto>("/api/projects",{
        method: 'POST',
        body: JSON.stringify(input),
    })
    return toProject(dto)
}

export async function listUpcomingTasks(): Promise<Task[]> {
    const dtos = await request<TaskDto[]>(`/api/tasks/upcoming`)
    return dtos.map(toTask)
}

export async function createTask(projectId: string, input: CreateTaskInput): Promise<Task> {
    const dto = await request<TaskDto>(`/api/projects/${projectId}/tasks`, {
        method: "POST",
        body: JSON.stringify({
            title: input.title,
            due_date: input.dueDate ? input.dueDate.toISOString : null,
        }),
    })
    return toTask(dto)
}