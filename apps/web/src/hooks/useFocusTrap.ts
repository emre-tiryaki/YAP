import { useEffect, type RefObject } from "react";

const focusable =
    'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

// While `active`, focus moves into `ref`, Tab cycles inside it, and focus goes
// back to the previously focused element on deactivation.
export function useFocusTrap(
    ref: RefObject<HTMLElement | null>,
    active: boolean,
) {
    useEffect(() => {
        const container = ref.current;
        if (!active || !container) return;

        const previous = document.activeElement as HTMLElement | null;
        if (!container.contains(document.activeElement)) {
            const first = container.querySelector<HTMLElement>(focusable);
            if (first) first.focus();
            else container.focus();
        }

        function handleKeyDown(event: KeyboardEvent) {
            if (event.key !== "Tab" || !container) return;
            const items = Array.from(
                container.querySelectorAll<HTMLElement>(focusable),
            );
            if (items.length === 0) {
                event.preventDefault();
                return;
            }
            const first = items[0];
            const last = items[items.length - 1];
            const current = document.activeElement;
            if (event.shiftKey && (current === first || current === container)) {
                event.preventDefault();
                last.focus();
            } else if (!event.shiftKey && current === last) {
                event.preventDefault();
                first.focus();
            }
        }

        container.addEventListener("keydown", handleKeyDown);
        return () => {
            container.removeEventListener("keydown", handleKeyDown);
            previous?.focus();
        };
    }, [ref, active]);
}
