import type { StatusCategory } from "./task";

export type Status = {
    id: string;
    title: string;
    color: string;
    category: StatusCategory;
    position: string;
};

// Partial update: omitted fields are kept.
export type UpdateStatusInput = {
    title?: string;
    color?: string;
};

export type CreateStatusInput = {
    title: string;
    color: string;
    category: StatusCategory;
};
