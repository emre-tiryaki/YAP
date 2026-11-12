import en from "../i18n/en.json";
import tr from "../i18n/tr.json";
import type { Language } from "../types/settings";
import { usePreferences } from "./usePreferences";

export type TranslationKey = keyof typeof en;

const dictionaries: Record<Language, Record<TranslationKey, string>> = {
  en,
  // A key missing from tr.json falls back to English below.
  tr: tr as Record<TranslationKey, string>,
};

export function useTranslate() {
  const { language } = usePreferences();
  const dictionary = dictionaries[language];
  return (key: TranslationKey): string => dictionary[key] ?? en[key];
}
