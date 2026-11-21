import { useSyncExternalStore } from "react";

// True while the CSS media query matches, e.g. "(max-width: 767px)".
export function useMediaQuery(query: string): boolean {
    return useSyncExternalStore(
        (listener) => {
            const list = window.matchMedia(query);
            list.addEventListener("change", listener);
            return () => list.removeEventListener("change", listener);
        },
        () => window.matchMedia(query).matches,
    );
}
