import { useSyncExternalStore } from "react";
import type { Pin, PinKind } from "../types/pin";

// Pins are a per-device UI preference, not synchronized data, so they live in
// localStorage rather than the database.
const STORAGE_KEY = "yap.pins";

const listeners = new Set<() => void>();
// Cached so useSyncExternalStore gets the same array until the pins change.
let cache: Pin[] | null = null;

function isPin(value: unknown): value is Pin {
  if (typeof value !== "object" || value === null) return false;
  const pin = value as Record<string, unknown>;
  return (
    (pin.kind === "project" || pin.kind === "task") &&
    typeof pin.id === "string" &&
    typeof pin.label === "string" &&
    typeof pin.href === "string"
  );
}

function read(): Pin[] {
  if (cache) return cache;
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "[]");
    cache = Array.isArray(parsed) ? parsed.filter(isPin) : [];
  } catch {
    cache = [];
  }
  return cache;
}

function write(pins: Pin[]) {
  cache = pins;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(pins));
  listeners.forEach((listener) => listener());
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

function samePin(pin: Pin, kind: PinKind, id: string) {
  return pin.kind === kind && pin.id === id;
}

export function togglePin(pin: Pin) {
  const pins = read();
  write(
    pins.some((p) => samePin(p, pin.kind, pin.id))
      ? pins.filter((p) => !samePin(p, pin.kind, pin.id))
      : [...pins, pin],
  );
}

export function unpinWhere(match: (pin: Pin) => boolean) {
  const pins = read();
  if (pins.some(match)) write(pins.filter((pin) => !match(pin)));
}

export function relabelPin(kind: PinKind, id: string, label: string) {
  const pins = read();
  if (!pins.some((p) => samePin(p, kind, id) && p.label !== label)) return;
  write(pins.map((p) => (samePin(p, kind, id) ? { ...p, label } : p)));
}

export function usePins() {
  const pins = useSyncExternalStore(subscribe, read);
  return {
    pins,
    isPinned: (kind: PinKind, id: string) =>
      pins.some((pin) => samePin(pin, kind, id)),
  };
}
