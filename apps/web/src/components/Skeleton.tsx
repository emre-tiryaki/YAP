export function Skeleton({ className = "" }: { className?: string }) {
    return (
        <div
            aria-hidden
            className={`animate-pulse rounded-md bg-layer-3 ${className}`}
        />
    );
}
