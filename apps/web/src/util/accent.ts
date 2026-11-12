// Deterministic accent colors for cards, so each project/plan keeps the same
// color across renders and sessions. Class names are written out in full so
// Tailwind can detect them.
export type Accent = {
    tile: string;
    bar: string;
    hoverBorder: string;
};

const accents: Accent[] = [
    {
        tile: "bg-indigo-100 text-indigo-700",
        bar: "bg-indigo-500",
        hoverBorder: "hover:border-indigo-300",
    },
    {
        tile: "bg-sky-100 text-sky-700",
        bar: "bg-sky-500",
        hoverBorder: "hover:border-sky-300",
    },
    {
        tile: "bg-teal-100 text-teal-700",
        bar: "bg-teal-500",
        hoverBorder: "hover:border-teal-300",
    },
    {
        tile: "bg-amber-100 text-amber-700",
        bar: "bg-amber-500",
        hoverBorder: "hover:border-amber-300",
    },
    {
        tile: "bg-rose-100 text-rose-700",
        bar: "bg-rose-500",
        hoverBorder: "hover:border-rose-300",
    },
    {
        tile: "bg-violet-100 text-violet-700",
        bar: "bg-violet-500",
        hoverBorder: "hover:border-violet-300",
    },
    {
        tile: "bg-lime-100 text-lime-700",
        bar: "bg-lime-500",
        hoverBorder: "hover:border-lime-300",
    },
    {
        tile: "bg-fuchsia-100 text-fuchsia-700",
        bar: "bg-fuchsia-500",
        hoverBorder: "hover:border-fuchsia-300",
    },
];

// FNV-1a over the key; any stable string (an ID) works.
export function accentFor(key: string): Accent {
    let hash = 0x811c9dc5;
    for (let i = 0; i < key.length; i++) {
        hash ^= key.charCodeAt(i);
        hash = Math.imul(hash, 0x01000193);
    }
    return accents[(hash >>> 0) % accents.length];
}

// First letter of the first two words, e.g. "Mobile App" -> "MA".
export function initials(name: string): string {
    const words = name.trim().split(/\s+/).filter(Boolean);
    const letters = words.slice(0, 2).map((w) => Array.from(w)[0]);
    return letters.join("").toLocaleUpperCase() || "?";
}
