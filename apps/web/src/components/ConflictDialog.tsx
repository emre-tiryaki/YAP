import { useState } from "react";
import { useTranslate } from "../hooks/useTranslate";
import { Modal } from "./Modal";
import { Button } from "./Button";

export type ConflictChoice = "replace" | "keep_both" | "skip";

type ConflictDialogProps = {
    // The name that is already taken in the target folder.
    name: string;
    canReplace: boolean;
    // Offers "apply to all" when more items are waiting.
    hasMore: boolean;
    onChoose: (choice: ConflictChoice, applyToAll: boolean) => void;
};

// Asks what to do when an uploaded, moved or pasted item has the name of an
// item already in the folder, like a file manager does.
export function ConflictDialog({
    name,
    canReplace,
    hasMore,
    onChoose,
}: ConflictDialogProps) {
    const t = useTranslate();
    const [applyToAll, setApplyToAll] = useState(false);
    const choose = (choice: ConflictChoice) => onChoose(choice, applyToAll);

    return (
        <Modal
            isOpen
            title={t("resources.conflict.title")}
            onClose={() => choose("skip")}
        >
            <p className="text-sm text-gray-600">
                {t("resources.conflict.message").replace("{name}", name)}
            </p>
            {hasMore && (
                <label className="mt-4 flex items-center gap-2 text-sm text-gray-700">
                    <input
                        type="checkbox"
                        checked={applyToAll}
                        onChange={(event) => setApplyToAll(event.target.checked)}
                    />
                    {t("resources.conflict.applyToAll")}
                </label>
            )}
            <div className="mt-6 flex flex-wrap justify-end gap-2">
                <button
                    type="button"
                    onClick={() => choose("skip")}
                    className="cursor-pointer rounded-md border border-gray-300 px-4 py-2 text-sm hover:bg-gray-100"
                >
                    {t("resources.conflict.skip")}
                </button>
                <button
                    type="button"
                    onClick={() => choose("keep_both")}
                    className="cursor-pointer rounded-md border border-gray-300 px-4 py-2 text-sm hover:bg-gray-100"
                >
                    {t("resources.conflict.keepBoth")}
                </button>
                {canReplace && (
                    <Button variant="primary" size="lg"
                        type="button"
                        autoFocus
                        onClick={() => choose("replace")}
                    >
                        {t("resources.conflict.replace")}
                    </Button>
                )}
            </div>
        </Modal>
    );
}
