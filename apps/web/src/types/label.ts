export type LabelColor =
    | "gray"
    | "red"
    | "orange"
    | "amber"
    | "emerald"
    | "teal"
    | "blue"
    | "indigo"
    | "violet"
    | "pink";

export type Label = {
    id: string;
    projectId: string;
    name: string;
    color: LabelColor;
    position: string;
};

export type CreateLabelInput = {
    name: string;
    color: LabelColor;
};

// Omitted fields are kept.
export type UpdateLabelInput = {
    name?: string;
    color?: LabelColor;
};
