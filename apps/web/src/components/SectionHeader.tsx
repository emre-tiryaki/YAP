import { useTranslate } from "../hooks/useTranslate";

type SectionHeaderProps = {
    title: string;
    onCreate: () => void;
};

export function SectionHeader({ title, onCreate }: SectionHeaderProps) {
    const t = useTranslate();

    return (
        <div className="mb-4 flex items-center justify-between">
            <h2 className="text-lg font-semibold">{title}</h2>
            <button
                type="button"
                onClick={onCreate}
                className="cursor-pointer rounded-md bg-gray-900 px-3 py-1.5 text-sm font-medium text-white hover:bg-gray-700"
            >
                + {t("common.new")}
            </button>
        </div>
    );
}
