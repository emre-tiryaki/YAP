// Card styling that follows the accent preference (`--color-accent`, see
// index.css), so cards share the color used for primary actions and active
// items instead of a palette of their own.
export const cardAccent = {
    // Initials tile shown when a project/plan has no emoji icon.
    tile: "bg-accent/10 text-accent",
    // Progress bar fill.
    bar: "bg-accent",
    hoverBorder: "hover:border-accent/50",
} as const;

// First letter of the first two words, e.g. "Mobile App" -> "MA".
export function initials(name: string): string {
    const words = name.trim().split(/\s+/).filter(Boolean);
    const letters = words.slice(0, 2).map((w) => Array.from(w)[0]);
    return letters.join("").toLocaleUpperCase() || "?";
}
