import type { IconType } from "react-icons";
import {
    LuFile,
    LuFileAudio,
    LuFileCode,
    LuFileImage,
    LuFileText,
    LuFileVideo,
    LuFolder,
    LuFolderOpen,
} from "react-icons/lu";
import {
    fileCategory,
    type FileCategory,
    type ResourceLook,
} from "../util/resources";

const icons: Record<FileCategory, IconType> = {
    folder: LuFolder,
    image: LuFileImage,
    pdf: LuFileText,
    markdown: LuFileCode,
    text: LuFileText,
    audio: LuFileAudio,
    video: LuFileVideo,
    other: LuFile,
};

const colors: Record<FileCategory, string> = {
    folder: "text-amber-500",
    image: "text-emerald-600",
    pdf: "text-red-600",
    markdown: "text-sky-600",
    text: "text-gray-500",
    audio: "text-fuchsia-600",
    video: "text-violet-600",
    other: "text-gray-400",
};

type ResourceIconProps = {
    resource: ResourceLook;
    isOpen?: boolean;
    className?: string;
};

export function ResourceIcon({
    resource,
    isOpen = false,
    className = "",
}: ResourceIconProps) {
    const category = fileCategory(resource);
    const Icon = category === "folder" && isOpen ? LuFolderOpen : icons[category];
    return (
        <Icon aria-hidden className={`shrink-0 ${colors[category]} ${className}`} />
    );
}
