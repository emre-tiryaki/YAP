// The user's uploaded background image. It is a per-device preference like
// the rest, but too large for localStorage (~5 MB), so the blob lives in
// IndexedDB and the preference only records that it exists (see
// Background "custom" in types/settings.ts).

const DB_NAME = "yap";
const STORE = "backgrounds";
const KEY = "custom";

function open(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 1);
    request.onupgradeneeded = () => request.result.createObjectStore(STORE);
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

async function run<T>(
  mode: IDBTransactionMode,
  action: (store: IDBObjectStore) => IDBRequest,
): Promise<T> {
  const db = await open();
  try {
    return await new Promise<T>((resolve, reject) => {
      const tx = db.transaction(STORE, mode);
      const request = action(tx.objectStore(STORE));
      tx.oncomplete = () => resolve(request.result as T);
      tx.onerror = () => reject(tx.error);
      tx.onabort = () => reject(tx.error);
    });
  } finally {
    db.close();
  }
}

// Kept after switching to another background, so the upload stays one click
// away in the settings until it is removed.
export type CustomBackground = {
  blob: Blob;
  // Average color as #rrggbb (see util/backgroundImage.ts).
  tint: string;
};

export async function loadCustomBackground(): Promise<CustomBackground | null> {
  const value = await run<unknown>("readonly", (store) => store.get(KEY));
  if (typeof value !== "object" || value === null) return null;
  const { blob, tint } = value as Record<string, unknown>;
  if (!(blob instanceof Blob) || typeof tint !== "string") return null;
  return { blob, tint };
}

export function saveCustomBackground(background: CustomBackground): Promise<void> {
  return run("readwrite", (store) => store.put(background, KEY));
}

export function deleteCustomBackground(): Promise<void> {
  return run("readwrite", (store) => store.delete(KEY));
}
