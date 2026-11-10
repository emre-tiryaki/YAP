export type Plan = {
    id: string;
    projectId: string;
    name: string;
    description: string;
    teamName: string | null;
    taskCount: number;
    doneTaskCount: number;
};

export type CreatePlanInput = {
    name: string;
    description: string;
};

export type UpdatePlanInput = {
    name?: string;
    description?: string;
};
