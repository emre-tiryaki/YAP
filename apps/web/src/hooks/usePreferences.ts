import { useSyncExternalStore } from "react";
import type {
  Accent,
  Background,
  BackgroundColor,
  BackgroundImage,
  DateFormat,
  Density,
  Language,
  Preferences,
  TaskPeekMode,
  Theme,
  TimeFormat,
  WeekStart,
} from "../types/settings";
import { backgroundColors, backgroundImages } from "../util/backgrounds";

// Read by the pre-paint script in index.html too; keep the key in sync.
// Preferences only shape this device's interface, so like pins they live in
// localStorage rather than the database.
const STORAGE_KEY = "yap.preferences";

export const SIDEBAR_MIN_WIDTH = 220;
export const SIDEBAR_MAX_WIDTH = 360;
export const SIDEBAR_DEFAULT_WIDTH = 248;

const languages: readonly Language[] = ["en", "tr"];
const weekStarts: readonly WeekStart[] = ["monday", "sunday"];
const dateFormats: readonly DateFormat[] = ["dmy", "mdy", "iso"];
const timeFormats: readonly TimeFormat[] = ["24h", "12h"];
const themes: readonly Theme[] = ["light", "dark", "system"];
const taskPeekModes: readonly TaskPeekMode[] = ["side", "modal", "full"];
const densities: readonly Density[] = ["comfortable", "compact"];
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
    density: "comfortable",
    background: { kind: "none" },
    // The sidebar used to keep this in its own key; read it once as the start value.
    sidebarCollapsed: localStorage.getItem("yap.sidebar.collapsed") === "true",
    sidebarWidth: SIDEBAR_DEFAULT_WIDTH,
    taskPeekMode: "side",
  };
}

function pick<T>(allowed: readonly T[], value: unknown, fallback: T): T {
  return allowed.includes(value as T) ? (value as T) : fallback;
}

const hexColor = /^#[0-9a-f]{6}$/i;

// Unknown or malformed values (e.g. a removed wallpaper) fall back to none.
function pickBackground(value: unknown): Background {
  const none: Background = { kind: "none" };
  if (typeof value !== "object" || value === null) return none;
  const v = value as Record<string, unknown>;
  if (v.kind === "color") {
    const color = pick<BackgroundColor | null>(backgroundColors, v.color, null);
    return color ? { kind: "color", color } : none;
  }
  if (
    v.kind === "customColor" &&
    typeof v.color === "string" &&
    hexColor.test(v.color)
  ) {
    return { kind: "customColor", color: v.color };
  }
  if (v.kind === "image") {
    const image = pick<BackgroundImage | null>(backgroundImages, v.image, null);
    return image ? { kind: "image", image } : none;
  }
  if (
    v.kind === "custom" &&
    typeof v.tint === "string" &&
    hexColor.test(v.tint) &&
    typeof v.version === "number"
  ) {
    return { kind: "custom", tint: v.tint, version: v.version };
  }
  return none;
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
    density: pick(densities, stored.density, base.density),
    background: pickBackground(stored.background),
    sidebarCollapsed:
      typeof stored.sidebarCollapsed === "boolean"
        ? stored.sidebarCollapsed
        : base.sidebarCollapsed,
    sidebarWidth:
      typeof stored.sidebarWidth === "number"
        ? Math.min(
            SIDEBAR_MAX_WIDTH,
            Math.max(SIDEBAR_MIN_WIDTH, Math.round(stored.sidebarWidth)),
          )
        : base.sidebarWidth,
    taskPeekMode: pick(taskPeekModes, stored.taskPeekMode, base.taskPeekMode),
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
