export type Plan = {
    id: string;
    name: string;
    description: string;
    teamName: string | null;
    taskCount: number;
};

export type CreatePlanInput = {
    name: string;
    description: string;
};
