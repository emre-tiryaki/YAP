import en from "../i18n/en.json";

export type TranslationKey = keyof typeof en;

export function useTranslate() {
  return (key: TranslationKey): string => en[key];
}
