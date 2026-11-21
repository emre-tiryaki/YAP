import type { ReactNode } from "react";

export type BreadcrumbItem = {
    label: string;
    to?: string;
    icon?: ReactNode;
};
