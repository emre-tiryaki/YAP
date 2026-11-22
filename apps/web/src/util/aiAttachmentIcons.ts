import type { ComponentType } from "react";
import { FiCheckSquare, FiFileText, FiFolder, FiGithub } from "react-icons/fi";
import { TbLayoutKanban } from "react-icons/tb";
import type { AiAttachmentKind } from "../types/ai";

type IconType = ComponentType<{ className?: string; "aria-hidden"?: boolean }>;

export const attachmentIcons: Record<AiAttachmentKind, IconType> = {
    task: FiCheckSquare,
    plan: TbLayoutKanban,
    project: FiFolder,
    resource: FiFileText,
    repo_file: FiGithub,
};
