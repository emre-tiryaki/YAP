type ToggleProps = {
    label: string;
    checked: boolean;
    onChange: (checked: boolean) => void;
};

export function Toggle({ label, checked, onChange }: ToggleProps) {
    return (
        <button
            type="button"
            role="switch"
            aria-checked={checked}
            aria-label={label}
            onClick={() => onChange(!checked)}
            className="relative h-6 w-11 shrink-0 cursor-pointer rounded-full bg-line-strong transition-colors focus-visible:outline-2 focus-visible:outline-accent aria-checked:bg-accent"
        >
            <span
                aria-hidden
                className={`absolute top-0.5 left-0.5 size-5 rounded-full bg-white shadow transition-transform ${
                    checked ? "translate-x-5" : ""
                }`}
            />
        </button>
    );
}
