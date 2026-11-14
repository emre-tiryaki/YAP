import {
    FiClipboard,
    FiCopy,
    FiDownload,
    FiEdit3,
    FiExternalLink,
    FiFolderPlus,
    FiLink,
    FiMove,
    FiScissors,
    FiTrash2,
} from "react-icons/fi";
import { TbPin, TbPinnedOff } from "react-icons/tb";
import { useNavigate } from "react-router";
import type { ResourceMenuItem } from "../components/ResourceMenu";
import type { Resource } from "../types/resource";
import { togglePin, usePins } from "./usePins";
import type { ResourceOperations } from "./useResourceOperations";
import { useTranslate } from "./useTranslate";

// Builds the ⋮ menu of a resource. `selection` is what the operations act
// on: the clicked item, or every selected item when it is part of the
// selection.
export function useResourceMenuItems(ops: ResourceOperations) {
    const t = useTranslate();
    const navigate = useNavigate();
    const { isPinned } = usePins();

    return (resource: Resource, selection: string[] = [resource.id]): ResourceMenuItem[] => {
        const ids = selection.includes(resource.id) ? selection : [resource.id];
        const single = ids.length === 1;
        const pinned = isPinned("resource", resource.id);
        const items: ResourceMenuItem[] = [];
        if (single)
            items.push({
                key: "open",
                label: t("resources.open"),
                Icon: FiExternalLink,
                onSelect: () => void navigate(`/resources/${resource.id}`),
            });
        if (single && resource.kind === "folder")
            items.push({
                key: "newFolder",
                label: t("resources.newFolder"),
                Icon: FiFolderPlus,
                onSelect: () => ops.startCreate("folder", resource.id),
            });
        if (single)
            items.push({
                key: "rename",
                label: t("resources.rename"),
                Icon: FiEdit3,
                onSelect: () => ops.startRename(resource),
            });
        items.push(
            {
                key: "cut",
                label: t("resources.cut"),
                Icon: FiScissors,
                onSelect: () => ops.cut(ids),
                startsGroup: true,
            },
            {
                key: "copy",
                label: t("resources.copy"),
                Icon: FiCopy,
                onSelect: () => ops.copy(ids),
            },
        );
        if (single && resource.kind === "folder" && ops.clipboard)
            items.push({
                key: "paste",
                label: t("resources.paste"),
                Icon: FiClipboard,
                onSelect: () => void ops.paste(resource.id),
            });
        items.push({
            key: "move",
            label: t("resources.moveTo"),
            Icon: FiMove,
            onSelect: () => ops.startMove(ids),
        });
        if (single && resource.kind === "file")
            items.push({
                key: "download",
                label: t("resources.download"),
                Icon: FiDownload,
                onSelect: () => ops.download(resource),
                startsGroup: true,
            });
        if (single)
            items.push(
                {
                    key: "link",
                    label: t("resources.linkToTask"),
                    Icon: FiLink,
                    onSelect: () => ops.startLink(resource),
                    startsGroup: resource.kind !== "file",
                },
                {
                    key: "pin",
                    label: t(pinned ? "card.menu.unpin" : "card.menu.pin"),
                    Icon: pinned ? TbPinnedOff : TbPin,
                    onSelect: () =>
                        togglePin({
                            kind: "resource",
                            id: resource.id,
                            label: resource.name,
                            href: `/resources/${resource.id}`,
                        }),
                },
            );
        items.push({
            key: "trash",
            label: t("resources.moveToTrash"),
            Icon: FiTrash2,
            onSelect: () => ops.trash(ids),
            isDanger: true,
            startsGroup: true,
        });
        return items;
    };
}
