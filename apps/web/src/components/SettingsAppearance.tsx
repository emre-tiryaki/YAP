import { ViewToggle } from "./ViewToggle";
import { useEffect, useRef, useState } from "react";
import {
    CODE_FONT_DEFAULT,
    CODE_FONT_MAX,
    CODE_FONT_MIN,
    DOCUMENT_FONT_DEFAULT,
    DOCUMENT_FONT_MAX,
    DOCUMENT_FONT_MIN,
    accents,
    getPreferences,
    updatePreferences,
    usePreferences,
} from "../hooks/usePreferences";
import { type TranslationKey, useTranslate } from "../hooks/useTranslate";
import { type Accent, type BackgroundColor, type Density, type Theme } from "../types/settings";
import { useHighlightedHtml } from "../hooks/useHighlightedHtml";
import { UnsupportedImageError, prepareBackground } from "../util/backgroundImage";
import { deleteCustomBackground, loadCustomBackground, saveCustomBackground } from "../util/backgroundStore";
import { backgroundColors, backgroundName, backgroundPresets, backgroundThumbUrls } from "../util/backgrounds";
import { SettingsGroup } from "./SettingsGroup";
import { SettingRow } from "./SettingRow";
import { Toggle } from "./Toggle";

// Previews use literal colors so they look the same in either theme (the
// theme variables like `bg-white` are redefined in dark mode).
const themes: { value: Theme; label: TranslationKey; preview: string }[] = [
    { value: "light", label: "settings.appearance.theme.light", preview: "bg-[#fff]" },
    { value: "dark", label: "settings.appearance.theme.dark", preview: "bg-[#1b1d22]" },
    {
        value: "system",
        label: "settings.appearance.theme.system",
        preview: "bg-linear-to-r from-[#fff] from-50% to-[#1b1d22] to-50%",
    },
];

// Must match the `data-accent` palettes in index.css.
const accentSwatches: Record<Accent, string> = {
    neutral: "var(--color-gray-900)",
    blue: "var(--color-blue-600)",
    indigo: "var(--color-indigo-600)",
    violet: "var(--color-violet-600)",
    fuchsia: "var(--color-fuchsia-600)",
    rose: "var(--color-rose-600)",
    teal: "var(--color-teal-700)",
    emerald: "var(--color-emerald-700)",
};

export function SettingsAppearance() {
    const t = useTranslate();
    const { theme, accent, density, sidebarCollapsed } = usePreferences();

    return (
        <div className="flex flex-col gap-6">
            <SettingsGroup title={t("settings.appearance.theme")}>
                <div
                    role="radiogroup"
                    aria-label={t("settings.appearance.theme")}
                    className="grid grid-cols-3 gap-3 px-5 py-4"
                >
                    {themes.map((option) => (
                        <button
                            key={option.value}
                            type="button"
                            role="radio"
                            aria-checked={theme === option.value}
                            onClick={() => updatePreferences({ theme: option.value })}
                            className="cursor-pointer rounded-lg border-2 border-gray-200 p-2 text-left hover:border-gray-300 aria-checked:border-accent"
                        >
                            <span
                                aria-hidden
                                className={`block h-16 rounded-md border border-gray-200 ${option.preview}`}
                            />
                            <span className="mt-2 block text-sm font-medium text-gray-700">
                                {t(option.label)}
                            </span>
                        </button>
                    ))}
                </div>
            </SettingsGroup>

            <BackgroundSettings />

            <ReadingSettings />

            <SettingsGroup>
                <SettingRow
                    label={t("settings.appearance.accent")}
                    hint={t("settings.appearance.accent.hint")}
                >
                    <div
                        role="radiogroup"
                        aria-label={t("settings.appearance.accent")}
                        className="flex gap-1.5"
                    >
                        {accents.map((option) => (
                            <button
                                key={option}
                                type="button"
                                role="radio"
                                aria-checked={accent === option}
                                aria-label={t(`settings.appearance.accent.${option}`)}
                                title={t(`settings.appearance.accent.${option}`)}
                                onClick={() => updatePreferences({ accent: option })}
                                style={{ backgroundColor: accentSwatches[option] }}
                                className="size-6 cursor-pointer rounded-full ring-offset-2 ring-offset-white aria-checked:ring-2 aria-checked:ring-gray-900"
                            />
                        ))}
                    </div>
                </SettingRow>
                <SettingRow label={t("settings.appearance.density")}>
                    <ViewToggle
                        label={t("settings.appearance.density")}
                        value={density}
                        onChange={(density) =>
                            updatePreferences({ density: density as Density })
                        }
                        options={[
                            {
                                view: "comfortable",
                                label: t("settings.appearance.density.comfortable"),
                            },
                            {
                                view: "compact",
                                label: t("settings.appearance.density.compact"),
                            },
                        ]}
                    />
                </SettingRow>
                <SettingRow
                    label={t("settings.appearance.sidebarCollapsed")}
                    hint={t("settings.appearance.sidebarCollapsed.hint")}
                >
                    <Toggle
                        label={t("settings.appearance.sidebarCollapsed")}
                        checked={sidebarCollapsed}
                        onChange={(collapsed) =>
                            updatePreferences({ sidebarCollapsed: collapsed })
                        }
                    />
                </SettingRow>
            </SettingsGroup>
        </div>
    );
}

type FontSizeControlProps = {
    label: string;
    value: number;
    min: number;
    max: number;
    defaultValue: number;
    onChange: (value: number) => void;
};

function FontSizeControl({ label, value, min, max, defaultValue, onChange }: FontSizeControlProps) {
    const t = useTranslate();
    return (
        <div className="flex items-center gap-3">
            <input
                type="range"
                aria-label={label}
                min={min}
                max={max}
                step={1}
                value={value}
                onChange={(event) => onChange(Number(event.target.value))}
                className="w-36 cursor-pointer accent-accent"
            />
            <span className="w-10 text-right text-sm text-gray-700 tabular-nums">{value}px</span>
            <button
                type="button"
                onClick={() => onChange(defaultValue)}
                disabled={value === defaultValue}
                className="cursor-pointer rounded-md px-2 py-1 text-xs font-medium text-gray-600 hover:bg-gray-100 disabled:cursor-default disabled:opacity-40 disabled:hover:bg-transparent"
            >
                {t("settings.appearance.reading.reset")}
            </button>
        </div>
    );
}

const codeSample = `function greet(name: string): string {
    // Say hello
    const count = 42;
    return "Hello, " + name + count;
}`;

// How documents and code in the resources are read; the sample shows the
// sizes before opening a file.
function ReadingSettings() {
    const t = useTranslate();
    const { documentFontSize, codeFontSize, codeWrap } = usePreferences();
    const html = useHighlightedHtml(codeSample, "typescript");
    return (
        <SettingsGroup title={t("settings.appearance.reading")}>
            <SettingRow
                label={t("settings.appearance.reading.documentSize")}
                hint={t("settings.appearance.reading.documentSize.hint")}
            >
                <FontSizeControl
                    label={t("settings.appearance.reading.documentSize")}
                    value={documentFontSize}
                    min={DOCUMENT_FONT_MIN}
                    max={DOCUMENT_FONT_MAX}
                    defaultValue={DOCUMENT_FONT_DEFAULT}
                    onChange={(size) => updatePreferences({ documentFontSize: size })}
                />
            </SettingRow>
            <SettingRow
                label={t("settings.appearance.reading.codeSize")}
                hint={t("settings.appearance.reading.codeSize.hint")}
            >
                <FontSizeControl
                    label={t("settings.appearance.reading.codeSize")}
                    value={codeFontSize}
                    min={CODE_FONT_MIN}
                    max={CODE_FONT_MAX}
                    defaultValue={CODE_FONT_DEFAULT}
                    onChange={(size) => updatePreferences({ codeFontSize: size })}
                />
            </SettingRow>
            <SettingRow
                label={t("settings.appearance.reading.wrap")}
                hint={t("settings.appearance.reading.wrap.hint")}
            >
                <Toggle
                    label={t("settings.appearance.reading.wrap")}
                    checked={codeWrap}
                    onChange={(wrap) => updatePreferences({ codeWrap: wrap })}
                />
            </SettingRow>
            <div aria-hidden className="flex flex-col gap-3 px-5 py-4">
                <p style={{ fontSize: documentFontSize }} className="text-gray-800">
                    {t("settings.appearance.reading.sample")}
                </p>
                <pre
                    style={{ fontSize: codeFontSize }}
                    className="m-0 overflow-x-auto rounded-md bg-gray-50 px-4 py-3 font-mono leading-[1.6] text-gray-900 [tab-size:4]"
                >
                    {html === null ? (
                        codeSample
                    ) : (
                        <code dangerouslySetInnerHTML={{ __html: html }} />
                    )}
                </pre>
            </div>
        </SettingsGroup>
    );
}

// Tiles share the theme tiles' look: a border that turns accent when chosen.
const backgroundTile =
    "block cursor-pointer rounded-lg border-2 border-transparent p-0.5 hover:border-gray-300 aria-checked:border-accent";

type UploadStatus = "idle" | "uploading" | "unsupported" | "failed";

// Swatches show each color's vivid hue in either theme, like the accent
// swatches; the background itself uses the soft theme shade from index.css
// (`--bg-swatch-*`), which would look dimmed and disabled as a swatch.
const backgroundSwatches: Record<BackgroundColor, string> = {
    slate: "#64748b",
    sand: "#c4a574",
    sage: "#7d9e72",
    sky: "#4f95d6",
    teal: "#2f9e94",
    lavender: "#9381d6",
    rose: "#dc6f86",
    peach: "#ec9a62",
};

function BackgroundSettings() {
    const t = useTranslate();
    const { background, language } = usePreferences();
    const inputRef = useRef<HTMLInputElement>(null);
    const colorInputRef = useRef<HTMLInputElement>(null);
    const [custom, setCustom] = useState<{ url: string; tint: string } | null>(
        null,
    );
    const [status, setStatus] = useState<UploadStatus>("idle");
    // Bumped after an upload or removal to reload the preview.
    const [revision, setRevision] = useState(0);

    useEffect(() => {
        let cancelled = false;
        let url: string | null = null;
        loadCustomBackground()
            .then((stored) => {
                if (cancelled) return;
                if (!stored) {
                    setCustom(null);
                    return;
                }
                url = URL.createObjectURL(stored.blob);
                setCustom({ url, tint: stored.tint });
            })
            .catch(() => {
                if (!cancelled) setCustom(null);
            });
        return () => {
            cancelled = true;
            if (url) URL.revokeObjectURL(url);
        };
    }, [revision]);

    async function upload(file: File) {
        setStatus("uploading");
        try {
            const prepared = await prepareBackground(file);
            await saveCustomBackground(prepared);
            updatePreferences({
                background: { kind: "custom", tint: prepared.tint, version: Date.now() },
            });
            setStatus("idle");
            setRevision((value) => value + 1);
        } catch (error) {
            setStatus(error instanceof UnsupportedImageError ? "unsupported" : "failed");
        }
    }

    async function remove() {
        try {
            await deleteCustomBackground();
        } catch {
            setStatus("failed");
            return;
        }
        if (getPreferences().background.kind === "custom") {
            updatePreferences({ background: { kind: "none" } });
        }
        setStatus("idle");
        setRevision((value) => value + 1);
    }

    let hint = t("settings.appearance.background.uploadHint");
    if (status === "uploading") hint = t("settings.appearance.background.uploading");
    if (status === "unsupported") hint = t("settings.appearance.background.unsupported");
    if (status === "failed") hint = t("settings.appearance.background.failed");

    const noneLabel = t("settings.appearance.background.none");
    const pickColorLabel = t("settings.appearance.background.pickColor");
    const pickedColor =
        background.kind === "customColor" ? background.color : null;
    const customLabel = t("settings.appearance.background.custom");

    return (
        <SettingsGroup title={t("settings.appearance.background")}>
            <div className="flex flex-col gap-4 px-5 py-4">
                <div
                    role="radiogroup"
                    aria-label={t("settings.appearance.background.images")}
                    className="grid grid-cols-4 gap-2"
                >
                    <button
                        type="button"
                        role="radio"
                        aria-checked={background.kind === "none"}
                        onClick={() => updatePreferences({ background: { kind: "none" } })}
                        title={noneLabel}
                        className={backgroundTile}
                    >
                        <span className="flex aspect-video items-center justify-center rounded-md border border-gray-200 bg-white text-xs font-medium text-gray-500">
                            {noneLabel}
                        </span>
                    </button>
                    {custom && (
                        <button
                            type="button"
                            role="radio"
                            aria-checked={background.kind === "custom"}
                            aria-label={customLabel}
                            title={customLabel}
                            onClick={() => {
                                if (background.kind === "custom") return;
                                updatePreferences({
                                    background: {
                                        kind: "custom",
                                        tint: custom.tint,
                                        version: Date.now(),
                                    },
                                });
                            }}
                            className={backgroundTile}
                        >
                            <img
                                src={custom.url}
                                alt=""
                                className="aspect-video w-full rounded-md object-cover"
                                style={{ backgroundColor: custom.tint }}
                            />
                        </button>
                    )}
                    {backgroundPresets.map((preset) => {
                        const image = preset.id;
                        const label = backgroundName(preset, language);
                        const thumbs = backgroundThumbUrls(preset);
                        return (
                            <button
                                key={image}
                                type="button"
                                role="radio"
                                aria-checked={
                                    background.kind === "image" && background.image === image
                                }
                                aria-label={label}
                                title={label}
                                onClick={() =>
                                    updatePreferences({ background: { kind: "image", image } })
                                }
                                className={backgroundTile}
                            >
                                {/* The variant the current theme will show. */}
                                <img
                                    src={thumbs.light}
                                    alt=""
                                    loading="lazy"
                                    className="aspect-video w-full rounded-md bg-gray-200 object-cover dark:hidden"
                                />
                                <img
                                    src={thumbs.dark}
                                    alt=""
                                    loading="lazy"
                                    className="hidden aspect-video w-full rounded-md bg-gray-200 object-cover dark:block"
                                />
                            </button>
                        );
                    })}
                </div>

                <div
                    role="radiogroup"
                    aria-label={t("settings.appearance.background.colors")}
                    className="flex flex-wrap gap-1.5"
                >
                    {backgroundColors.map((color) => {
                        const label = t(`settings.appearance.background.color.${color}`);
                        return (
                            <button
                                key={color}
                                type="button"
                                role="radio"
                                aria-checked={
                                    background.kind === "color" && background.color === color
                                }
                                aria-label={label}
                                title={label}
                                onClick={() =>
                                    updatePreferences({ background: { kind: "color", color } })
                                }
                                style={{ backgroundColor: backgroundSwatches[color] }}
                                className="size-7 cursor-pointer rounded-full ring-offset-2 ring-offset-white aria-checked:ring-2 aria-checked:ring-accent"
                            />
                        );
                    })}
                    {/* Any color: a rainbow swatch opening the native picker, filled
                            with the picked color once chosen. */}
                    <span className="relative">
                        <button
                            type="button"
                            role="radio"
                            aria-checked={pickedColor !== null}
                            aria-label={pickColorLabel}
                            title={pickColorLabel}
                            onClick={() => colorInputRef.current?.click()}
                            style={{
                                background:
                                    pickedColor ??
                                    "conic-gradient(#f87171, #fbbf24, #a3e635, #34d399, #38bdf8, #818cf8, #e879f9, #f87171)",
                            }}
                            className="block size-7 cursor-pointer rounded-full ring-offset-2 ring-offset-white aria-checked:ring-2 aria-checked:ring-accent"
                        />
                        <input
                            ref={colorInputRef}
                            type="color"
                            value={pickedColor ?? "#8a9bb0"}
                            onChange={(event) =>
                                updatePreferences({
                                    background: { kind: "customColor", color: event.target.value },
                                })
                            }
                            aria-hidden
                            tabIndex={-1}
                            className="pointer-events-none absolute inset-0 size-full opacity-0"
                        />
                    </span>
                </div>

                <div className="flex items-center justify-between gap-4 border-t border-gray-100 pt-4">
                    <p
                        role={status === "unsupported" || status === "failed" ? "alert" : undefined}
                        className={`text-xs ${
                            status === "unsupported" || status === "failed"
                                ? "text-red-600"
                                : "text-gray-500"
                        }`}
                    >
                        {hint}
                    </p>
                    <div className="flex shrink-0 gap-2">
                        <input
                            ref={inputRef}
                            type="file"
                            accept="image/*"
                            className="hidden"
                            aria-hidden
                            tabIndex={-1}
                            onChange={(event) => {
                                const file = event.target.files?.[0];
                                if (file) void upload(file);
                                // Lets the same file be picked again.
                                event.target.value = "";
                            }}
                        />
                        {custom && (
                            <button
                                type="button"
                                onClick={() => void remove()}
                                disabled={status === "uploading"}
                                className="cursor-pointer rounded-md px-3 py-1.5 text-sm font-medium text-red-600 hover:bg-red-50 disabled:opacity-50"
                            >
                                {t("settings.appearance.background.remove")}
                            </button>
                        )}
                        <button
                            type="button"
                            onClick={() => inputRef.current?.click()}
                            disabled={status === "uploading"}
                            className="cursor-pointer rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
                        >
                            {t(
                                custom
                                    ? "settings.appearance.background.replace"
                                    : "settings.appearance.background.upload",
                            )}
                        </button>
                    </div>
                </div>
            </div>
        </SettingsGroup>
    );
}
