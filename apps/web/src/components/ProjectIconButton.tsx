import { useUpdateProject } from "../hooks/useProjects";
import type { Project } from "../types/project";
import { IconButton } from "./IconButton";

type ProjectIconButtonProps = {
    project: Project;
};

export function ProjectIconButton({ project }: ProjectIconButtonProps) {
    const update = useUpdateProject();
    return (
        <IconButton
            name={project.name}
            icon={project.icon}
            error={update.error}
            reset={update.reset}
            onSelect={(icon, onSaved) =>
                update.mutate(
                    { projectId: project.id, input: { icon } },
                    { onSuccess: onSaved },
                )
            }
        />
    );
}
