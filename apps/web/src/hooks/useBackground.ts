import { useEffect } from "react";
import { usePreferences } from "./usePreferences";
import { loadCustomBackground } from "../util/backgroundStore";

// Mirrors the background preference onto <html>, where index.css paints it:
// `data-bg` (color/customColor/image/custom) turns on the translucent surfaces, and
// `data-bg-color` / `data-bg-image` pick a preset. A custom upload is read from
// IndexedDB and set as `--app-bg-image`; until then its average color
// (`--app-bg-tint`) shows, which index.html also applies before first paint.
// A freely picked color is that same `--app-bg-tint`.
export function useBackground() {
  const { background } = usePreferences();

  const kind = background.kind;
  const color = background.kind === "color" ? background.color : undefined;
  const image = background.kind === "image" ? background.image : undefined;
  // A picked color is painted the same way as an upload's placeholder tint.
  const tint =
    background.kind === "custom"
      ? background.tint
      : background.kind === "customColor"
        ? background.color
        : undefined;
  const version = background.kind === "custom" ? background.version : undefined;

  useEffect(() => {
    const root = document.documentElement;
    if (kind === "none") delete root.dataset.bg;
    else root.dataset.bg = kind;
    if (color) root.dataset.bgColor = color;
    else delete root.dataset.bgColor;
    if (image) root.dataset.bgImage = image;
    else delete root.dataset.bgImage;
    if (tint) root.style.setProperty("--app-bg-tint", tint);
    else root.style.removeProperty("--app-bg-tint");
  }, [kind, color, image, tint]);

  useEffect(() => {
    if (version === undefined) return;
    const root = document.documentElement;
    let url: string | null = null;
    let cancelled = false;
    loadCustomBackground()
      .then((custom) => {
        if (cancelled || !custom) return;
        url = URL.createObjectURL(custom.blob);
        root.style.setProperty("--app-bg-image", `url("${url}")`);
      })
      .catch(() => {
        // Storage unavailable: the tint stays as a plain color.
      });
    return () => {
      cancelled = true;
      root.style.removeProperty("--app-bg-image");
      if (url) URL.revokeObjectURL(url);
    };
  }, [version]);
}
