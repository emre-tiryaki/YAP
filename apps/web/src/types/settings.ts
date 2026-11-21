export type SettingsSection =
  | "general"
  | "account"
  | "appearance"
  | "categories"
  | "notifications"
  | "github"
  | "shortcuts";

export type Language = "en" | "tr";
export type WeekStart = "monday" | "sunday";
export type DateFormat = "dmy" | "mdy" | "iso";
export type TimeFormat = "24h" | "12h";
export type Theme = "light" | "dark" | "system";
// "neutral" is the default gray look; the rest map to palettes in index.css.
export type Accent =
  | "neutral"
  | "blue"
  | "indigo"
  | "violet"
  | "fuchsia"
  | "rose"
  | "teal"
  | "emerald";
// "compact" tightens the spacing scale (see index.css).
export type Density = "comfortable" | "compact";

// Id of a bundled background preset (src/generated/backgrounds.json, built
// by scripts/backgrounds.ts); validated against that list when read.
export type BackgroundImage = string;
// Solid background colors; each has a light and a dark shade in index.css.
export type BackgroundColor =
  | "slate"
  | "sand"
  | "sage"
  | "sky"
  | "teal"
  | "lavender"
  | "rose"
  | "peach";
// What shows behind the app. A "custom" image is uploaded by the user and kept
// in IndexedDB (util/backgroundStore.ts); `tint` is its average color, shown
// until the image has loaded, and `version` changes on every upload so other
// tabs reload it.
export type Background =
  | { kind: "none" }
  | { kind: "color"; color: BackgroundColor }
  // A color picked freely, as #rrggbb.
  | { kind: "customColor"; color: string }
  | { kind: "image"; image: BackgroundImage }
  | { kind: "custom"; tint: string; version: number };

// Per-device interface preferences.
export type Preferences = {
  language: Language;
  weekStart: WeekStart;
  dateFormat: DateFormat;
  timeFormat: TimeFormat;
  shortcutsEnabled: boolean;
  theme: Theme;
  accent: Accent;
  density: Density;
  background: Background;
  // Sidebar shown as an icon rail.
  sidebarCollapsed: boolean;
  // Width in px of the expanded sidebar.
  sidebarWidth: number;
};
