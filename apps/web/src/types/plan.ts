export type Plan = {
    id: string;
    projectId: string;
    name: string;
    description: string;
    // A single emoji, or "" for none.
    icon: string;
    teamName: string | null;
    taskCount: number;
    doneTaskCount: number;
};

export type CreatePlanInput = {
    name: string;
    description: string;
    icon: string;
};

export type UpdatePlanInput = {
    name?: string;
    description?: string;
    // "" removes the icon.
    icon?: string;
};
