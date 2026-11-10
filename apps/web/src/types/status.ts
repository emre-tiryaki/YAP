import type { StatusCategory } from "./task";

export type Status = {
    id: string;
    title: string;
    color: string;
    category: StatusCategory;
    position: string;
};

export type CreateStatusInput = {
    title: string;
    color: string;
    category: StatusCategory;
};
