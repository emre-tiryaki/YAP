import { useEffect, useRef } from "react";
import { usePreferences } from "./usePreferences";

type Shortcut = {
  // Compared with KeyboardEvent.key, case-insensitively.
  key: string;
  // Ctrl, or Cmd on macOS.
  mod?: boolean;
  // Also fires while a modal dialog is open. For shortcuts of the dialog's
  // own content (e.g. the task peek in its modal mode).
  inModal?: boolean;
};

// How long the first key of a sequence stays armed.
const SEQUENCE_MS = 1000;

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
  const { key, mod = false, inModal = false } = shortcut;

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
      if (!inModal && document.querySelector('[aria-modal="true"]')) return;
      event.preventDefault();
      handlerRef.current();
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [enabled, shortcutsEnabled, key, mod, inModal]);
}

// Registers a two-key sequence such as G then H. The first key arms it for a
// second; the second completes it. Same guards as useShortcut: skipped when
// shortcuts are off, while typing, while a dialog is open and with modifiers.
export function useKeySequence(keys: [string, string], handler: () => void) {
  const { shortcutsEnabled } = usePreferences();
  const handlerRef = useRef(handler);
  const [first, second] = keys;

  useEffect(() => {
    handlerRef.current = handler;
  });

  useEffect(() => {
    if (!shortcutsEnabled) return;
    let armedUntil = 0;
    function handleKeyDown(event: KeyboardEvent) {
      if (event.defaultPrevented || event.repeat) return;
      if (event.ctrlKey || event.metaKey || event.altKey || event.shiftKey)
        return;
      if (isEditable(event.target)) return;
      if (document.querySelector('[aria-modal="true"]')) return;
      const pressed = event.key.toLowerCase();
      if (Date.now() < armedUntil && pressed === second.toLowerCase()) {
        armedUntil = 0;
        event.preventDefault();
        handlerRef.current();
        return;
      }
      armedUntil =
        pressed === first.toLowerCase() ? Date.now() + SEQUENCE_MS : 0;
    }
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [shortcutsEnabled, first, second]);
}
