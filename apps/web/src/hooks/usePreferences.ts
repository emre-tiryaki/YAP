import { useSyncExternalStore } from "react";
import type {
  Accent,
  DateFormat,
  Language,
  Preferences,
  Theme,
  TimeFormat,
  WeekStart,
} from "../types/settings";

// Read by the pre-paint script in index.html too; keep the key in sync.
// Preferences only shape this device's interface, so like pins they live in
// localStorage rather than the database.
const STORAGE_KEY = "yap.preferences";

const languages: readonly Language[] = ["en", "tr"];
const weekStarts: readonly WeekStart[] = ["monday", "sunday"];
const dateFormats: readonly DateFormat[] = ["dmy", "mdy", "iso"];
const timeFormats: readonly TimeFormat[] = ["24h", "12h"];
const themes: readonly Theme[] = ["light", "dark", "system"];
export const accents: readonly Accent[] = [
  "neutral",
  "blue",
  "indigo",
  "violet",
  "fuchsia",
  "rose",
  "teal",
  "emerald",
];

function defaults(): Preferences {
  const isTurkish = navigator.language.toLowerCase().startsWith("tr");
  return {
    language: isTurkish ? "tr" : "en",
    weekStart: "monday",
    dateFormat: "dmy",
    timeFormat: "24h",
    shortcutsEnabled: true,
    theme: "system",
    accent: "neutral",
  };
}

function pick<T>(allowed: readonly T[], value: unknown, fallback: T): T {
  return allowed.includes(value as T) ? (value as T) : fallback;
}

const listeners = new Set<() => void>();
// Cached so useSyncExternalStore gets the same object until a change.
let cache: Preferences | null = null;

function read(): Preferences {
  if (cache) return cache;
  const base = defaults();
  let stored: Record<string, unknown> = {};
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "{}");
    if (typeof parsed === "object" && parsed !== null) {
      stored = parsed as Record<string, unknown>;
    }
  } catch {
    // Corrupt value: fall back to the defaults.
  }
  cache = {
    language: pick(languages, stored.language, base.language),
    weekStart: pick(weekStarts, stored.weekStart, base.weekStart),
    dateFormat: pick(dateFormats, stored.dateFormat, base.dateFormat),
    timeFormat: pick(timeFormats, stored.timeFormat, base.timeFormat),
    shortcutsEnabled:
      typeof stored.shortcutsEnabled === "boolean"
        ? stored.shortcutsEnabled
        : base.shortcutsEnabled,
    theme: pick(themes, stored.theme, base.theme),
    accent: pick(accents, stored.accent, base.accent),
  };
  return cache;
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  // Keeps other tabs in sync.
  function handleStorage(event: StorageEvent) {
    if (event.key !== STORAGE_KEY) return;
    cache = null;
    listener();
  }
  window.addEventListener("storage", handleStorage);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", handleStorage);
  };
}

export function updatePreferences(patch: Partial<Preferences>) {
  cache = { ...read(), ...patch };
  localStorage.setItem(STORAGE_KEY, JSON.stringify(cache));
  listeners.forEach((listener) => listener());
}

// For code outside React (event handlers registered once).
export function getPreferences(): Preferences {
  return read();
}

export function usePreferences(): Preferences {
  return useSyncExternalStore(subscribe, read);
}
