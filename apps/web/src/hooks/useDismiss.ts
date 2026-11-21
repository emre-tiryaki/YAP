import { useEffect, useRef, type RefObject } from "react";

// While `open`, a pointerdown outside `ref` (and `ignoreRefs`) or the Escape
// key calls `onDismiss`. Capture phase, so it also fires when the click lands
// on an element that stops propagation.
export function useDismiss(
    ref: RefObject<HTMLElement | null>,
    open: boolean,
    onDismiss: () => void,
    ignoreRefs: RefObject<HTMLElement | null>[] = [],
) {
    const onDismissRef = useRef(onDismiss);
    const ignoreRef = useRef(ignoreRefs);
    useEffect(() => {
        onDismissRef.current = onDismiss;
        ignoreRef.current = ignoreRefs;
    });

    useEffect(() => {
        if (!open) return;

        function handlePointerDown(event: PointerEvent) {
            const target = event.target as Node;
            if (ref.current?.contains(target)) return;
            if (ignoreRef.current.some((r) => r.current?.contains(target)))
                return;
            onDismissRef.current();
        }
        function handleKeyDown(event: KeyboardEvent) {
            if (event.key !== "Escape") return;
            event.stopPropagation();
            onDismissRef.current();
        }

        document.addEventListener("pointerdown", handlePointerDown, true);
        document.addEventListener("keydown", handleKeyDown, true);
        return () => {
            document.removeEventListener(
                "pointerdown",
                handlePointerDown,
                true,
            );
            document.removeEventListener("keydown", handleKeyDown, true);
        };
    }, [open, ref]);
}
