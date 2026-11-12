import { useEffect, useRef, useState } from "react";
import { ApiError } from "../api/client";
import { useUpdateProject } from "../hooks/useProjects";
import { useTranslate } from "../hooks/useTranslate";
import type { Project } from "../types/project";
import { accentFor, initials } from "../util/accent";
import { FormError } from "./FormError";
import { ProjectIconPicker } from "./ProjectIconPicker";

type ProjectIconButtonProps = {
    project: Project;
};

// The project's icon tile (emoji, or initials when there is none). Clicking it
// opens a picker that saves the chosen icon right away.
export function ProjectIconButton({ project }: ProjectIconButtonProps) {
    const t = useTranslate();
    const update = useUpdateProject();
    const [isOpen, setIsOpen] = useState(false);
    const ref = useRef<HTMLDivElement>(null);
    const accent = accentFor(project.id);
    const { reset } = update;

    useEffect(() => {
        if (!isOpen) return;

        const close = () => {
            setIsOpen(false);
            reset();
        };
        function handlePointerDown(event: PointerEvent) {
            if (!ref.current?.contains(event.target as Node)) close();
        }
        function handleKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") close();
        }

        document.addEventListener("pointerdown", handlePointerDown, true);
        document.addEventListener("keydown", handleKeyDown, true);
        return () => {
            document.removeEventListener(
                "pointerdown",
                handlePointerDown,
                true,
            );
            document.removeEventListener("keydown", handleKeyDown, true);
        };
    }, [isOpen, reset]);

    function select(icon: string) {
        update.mutate(
            { projectId: project.id, input: { icon } },
            {
                onSuccess: () => {
                    setIsOpen(false);
                    reset();
                },
            },
        );
    }

    const errorMessage = !update.error
        ? null
        : update.error instanceof ApiError &&
            update.error.message === "invalid_input"
          ? t("project.icon.invalid")
          : t("common.error");

    return (
        <div ref={ref} className="relative shrink-0">
            <button
                type="button"
                onClick={() => {
                    setIsOpen(!isOpen);
                    reset();
                }}
                aria-label={t("project.icon.change")}
                title={t("project.icon.change")}
                aria-haspopup="dialog"
                aria-expanded={isOpen}
                className={`grid size-12 cursor-pointer place-items-center rounded-lg font-semibold transition hover:ring-2 hover:ring-gray-300 focus-visible:outline-2 focus-visible:outline-gray-900 ${
                    project.icon ? "text-3xl" : `text-base ${accent.tile}`
                }`}
            >
                {project.icon || initials(project.name)}
            </button>
            {isOpen && (
                <div
                    role="dialog"
                    aria-label={t("project.icon.change")}
                    className="absolute top-full left-0 z-40 mt-2 w-80 rounded-lg border border-gray-200 bg-white p-3 shadow-lg"
                >
                    <ProjectIconPicker
                        value={project.icon}
                        onChange={select}
                    />
                    {errorMessage && (
                        <div className="mt-2">
                            <FormError message={errorMessage} />
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
