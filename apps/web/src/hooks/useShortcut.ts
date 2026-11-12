import { useEffect, useRef } from "react";
import { usePreferences } from "./usePreferences";

type Shortcut = {
  // Compared with KeyboardEvent.key, case-insensitively.
  key: string;
  // Ctrl, or Cmd on macOS.
  mod?: boolean;
};

function isEditable(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  return (
    target.isContentEditable ||
    target.tagName === "INPUT" ||
    target.tagName === "TEXTAREA" ||
    target.tagName === "SELECT"
  );
}

// Registers a global keyboard shortcut. It is skipped when shortcuts are
// turned off in the settings, while a dialog is open, inside rich-text
// editors (which own their Ctrl shortcuts) and, for single keys, while typing.
export function useShortcut(
  shortcut: Shortcut,
  handler: () => void,
  enabled = true,
) {
  const { shortcutsEnabled } = usePreferences();
  const handlerRef = useRef(handler);
  const { key, mod = false } = shortcut;

  useEffect(() => {
    handlerRef.current = handler;
  });

  useEffect(() => {
    if (!enabled || !shortcutsEnabled) return;
    function handleKeyDown(event: KeyboardEvent) {
      if (event.defaultPrevented || event.repeat || event.altKey) return;
      const hasMod = event.ctrlKey || event.metaKey;
      if (hasMod !== mod || event.key.toLowerCase() !== key.toLowerCase())
        return;
      if (!mod && (event.shiftKey || isEditable(event.target))) return;
      if (event.target instanceof HTMLElement && event.target.isContentEditable)
        return;
      if (document.querySelector('[aria-modal="true"]')) return;
      event.preventDefault();
      handlerRef.current();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [enabled, shortcutsEnabled, key, mod]);
}
