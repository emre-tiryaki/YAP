import type { LabelColor } from "../types/label";

// Written out in full so Tailwind sees every class.
export const labelColors: readonly {
    value: LabelColor;
    dot: string;
}[] = [
    { value: "gray", dot: "bg-gray-500" },
    { value: "red", dot: "bg-red-500" },
    { value: "orange", dot: "bg-orange-500" },
    { value: "amber", dot: "bg-amber-500" },
    { value: "emerald", dot: "bg-emerald-500" },
    { value: "teal", dot: "bg-teal-500" },
    { value: "blue", dot: "bg-blue-500" },
    { value: "indigo", dot: "bg-indigo-500" },
    { value: "violet", dot: "bg-violet-500" },
    { value: "pink", dot: "bg-pink-500" },
];

export function labelDotClass(color: LabelColor): string {
    return labelColors.find((c) => c.value === color)?.dot ?? "bg-gray-500";
}
