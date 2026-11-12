export type SettingsSection =
  | "general"
  | "account"
  | "appearance"
  | "categories"
  | "notifications"
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
};
