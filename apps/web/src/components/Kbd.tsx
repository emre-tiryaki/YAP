import { isMac } from "../util/platform";

type KbdProps = {
    // "mod" renders as ⌘ on Mac and Ctrl elsewhere.
    keys: string[];
};

export function Kbd({ keys }: KbdProps) {
    return (
        <span className="inline-flex gap-0.5">
            {keys.map((key) => (
                <kbd
                    key={key}
                    className="rounded border border-line bg-layer-1 px-1 font-sans text-caption-sm text-fg-subtle"
                >
                    {key === "mod" ? (isMac ? "⌘" : "Ctrl") : key}
                </kbd>
            ))}
        </span>
    );
}
