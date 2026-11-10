import type { StatusCategory } from "./task";

export type Status = {
    id: string;
    title: string;
    description: string;
    color: string;
    category: StatusCategory;
    position: string;
};

// Partial update: omitted fields are kept.
export type UpdateStatusInput = {
    title?: string;
    description?: string;
    color?: string;
};

export type CreateStatusInput = {
    title: string;
    description: string;
    color: string;
    category: StatusCategory;
};
