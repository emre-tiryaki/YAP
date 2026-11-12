import { useEffect } from "react";
import { usePreferences } from "./usePreferences";

const darkQuery = "(prefers-color-scheme: dark)";

// Keeps the `dark` class on <html> (Tailwind's class-based dark variant, see
// index.css) in sync with the theme preference. "system" follows the OS and
// updates live when it changes. index.html applies the same rule before the
// first paint so a dark page never flashes light.
// The accent preference is mirrored to `data-accent` on <html>, which selects
// the `--color-accent` palette in index.css, and the density preference to
// `data-density`, which sets the `--spacing` scale there.
export function useTheme() {
  const { theme, accent, density } = usePreferences();

  useEffect(() => {
    document.documentElement.dataset.accent = accent;
  }, [accent]);

  useEffect(() => {
    document.documentElement.dataset.density = density;
  }, [density]);

  useEffect(() => {
    const media = window.matchMedia(darkQuery);
    function apply() {
      const isDark = theme === "dark" || (theme === "system" && media.matches);
      document.documentElement.classList.toggle("dark", isDark);
    }
    apply();
    if (theme !== "system") return;
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, [theme]);
}
