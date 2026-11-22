type ProgressRingProps = {
    done: number;
    total: number;
    size?: number;
};

// A circular progress bar with "done/total" in the middle.
export function ProgressRing({ done, total, size = 56 }: ProgressRingProps) {
    const stroke = 5;
    const radius = (size - stroke) / 2;
    const circumference = 2 * Math.PI * radius;
    const ratio = total === 0 ? 0 : Math.min(done / total, 1);

    return (
        <div
            role="progressbar"
            aria-valuemin={0}
            aria-valuemax={total}
            aria-valuenow={done}
            aria-valuetext={`${done}/${total}`}
            className="relative shrink-0"
            style={{ width: size, height: size }}
        >
            <svg width={size} height={size} className="-rotate-90">
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    fill="none"
                    strokeWidth={stroke}
                    className="stroke-line-strong"
                />
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    fill="none"
                    strokeWidth={stroke}
                    strokeLinecap="round"
                    strokeDasharray={circumference}
                    strokeDashoffset={circumference * (1 - ratio)}
                    className="stroke-accent transition-[stroke-dashoffset] duration-300"
                />
            </svg>
            <span className="absolute inset-0 grid place-items-center text-xs font-semibold text-fg tabular-nums">
                {done}/{total}
            </span>
        </div>
    );
}
