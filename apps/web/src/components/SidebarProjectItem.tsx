import { useState } from "react";
import { Link, useNavigate } from "react-router";
import {
    FiChevronRight,
    FiEdit2,
    FiFolder,
    FiLink,
    FiMoreHorizontal,
    FiTrash2,
    FiType,
} from "react-icons/fi";
import { TbLayoutKanban } from "react-icons/tb";
import { toggleSidebarExpanded } from "../hooks/useSidebarExpanded";
import { useProjectPlans } from "../hooks/useProjects";
import { pushInfoToast } from "../hooks/useToasts";
import { useTranslate } from "../hooks/useTranslate";
import type { CardAction } from "../types/cardAction";
import type { Pin } from "../types/pin";
import type { Project } from "../types/project";
import { setPinDragData } from "../util/pinDrag";
import { Menu } from "./Menu";
import { MenuItem } from "./MenuItem";
import { MenuSeparator } from "./MenuSeparator";
import { ProjectDialogs } from "./ProjectDialogs";
import { SidebarNavItem } from "./SidebarNavItem";
import { Tooltip } from "./Tooltip";

type SidebarProjectItemProps = {
    project: Project;
    pathname: string;
    expanded: boolean;
    collapsed: boolean;
};

export function SidebarProjectItem({
    project,
    pathname,
    expanded,
    collapsed,
}: SidebarProjectItemProps) {
    const t = useTranslate();
    const navigate = useNavigate();
    const [action, setAction] = useState<CardAction | null>(null);
    const href = `/projects/${project.id}`;
    const active = pathname === href || pathname.startsWith(`${href}/`);
    const pin: Pin = {
        kind: "project",
        id: project.id,
        label: project.name,
        href,
    };

    // In the icon rail a project is a single link, with no plans below it.
    if (collapsed)
        return (
            <SidebarNavItem
                to={href}
                label={project.name}
                icon={FiFolder}
                emoji={project.icon}
                active={active}
                collapsed
                dragPin={pin}
            />
        );

    function copyLink() {
        void navigator.clipboard.writeText(`${window.location.origin}${href}`);
        pushInfoToast(t("sidebar.project.linkCopied"), project.name);
    }

    return (
        <li>
            <div className="group relative flex items-center">
                <Tooltip
                    label={t(
                        expanded
                            ? "sidebar.project.collapse"
                            : "sidebar.project.expand",
                    )}
                    side="right"
                >
                    <button
                        type="button"
                        onClick={() => toggleSidebarExpanded(project.id)}
                        aria-expanded={expanded}
                        aria-label={`${t(
                            expanded
                                ? "sidebar.project.collapse"
                                : "sidebar.project.expand",
                        )}: ${project.name}`}
                        className="absolute left-0.5 z-10 flex size-5 cursor-pointer items-center justify-center rounded text-fg-placeholder hover:bg-layer-3-hover hover:text-fg"
                    >
                        <FiChevronRight
                            aria-hidden
                            className={`transition-transform ${expanded ? "rotate-90" : ""}`}
                        />
                    </button>
                </Tooltip>
                <Link
                    to={href}
                    onDragStart={(event) => setPinDragData(event, pin)}
                    aria-current={pathname === href ? "page" : undefined}
                    title={project.name}
                    className="flex h-7 min-w-0 flex-1 items-center gap-2 rounded-md pr-8 pl-7 text-fg-muted hover:bg-layer-1-hover aria-[current=page]:bg-layer-1-hover aria-[current=page]:font-medium aria-[current=page]:text-fg"
                >
                    {project.icon ? (
                        <span
                            aria-hidden
                            className="w-4 shrink-0 text-center text-base leading-none"
                        >
                            {project.icon}
                        </span>
                    ) : (
                        <FiFolder
                            aria-hidden
                            className="shrink-0 text-base text-fg-subtle"
                        />
                    )}
                    <span className="truncate">{project.name}</span>
                </Link>
                <div className="absolute top-1/2 right-1 -translate-y-1/2 opacity-0 group-focus-within:opacity-100 group-hover:opacity-100">
                    <Menu
                        placement="bottom-end"
                        trigger={({ ref, ...props }) => (
                            <button
                                ref={ref}
                                type="button"
                                aria-label={`${t("sidebar.project.menu")}: ${project.name}`}
                                {...props}
                                className="flex size-5 cursor-pointer items-center justify-center rounded text-fg-subtle hover:bg-layer-3-hover hover:text-fg"
                            >
                                <FiMoreHorizontal aria-hidden />
                            </button>
                        )}
                    >
                        <MenuItem
                            icon={<FiType aria-hidden />}
                            label={t("card.menu.rename")}
                            onSelect={() => setAction("rename")}
                        />
                        <MenuItem
                            icon={<FiEdit2 aria-hidden />}
                            label={t("card.menu.edit")}
                            onSelect={() => setAction("edit")}
                        />
                        <MenuItem
                            icon={<FiLink aria-hidden />}
                            label={t("sidebar.project.copyLink")}
                            onSelect={copyLink}
                        />
                        <MenuSeparator />
                        <MenuItem
                            danger
                            icon={<FiTrash2 aria-hidden />}
                            label={t("card.menu.delete")}
                            onSelect={() => setAction("delete")}
                        />
                    </Menu>
                </div>
            </div>
            {expanded && (
                <SidebarProjectPlans projectId={project.id} pathname={pathname} />
            )}
            <ProjectDialogs
                project={project}
                action={action}
                onClose={() => setAction(null)}
                onDeleted={() => {
                    if (active) void navigate("/home", { replace: true });
                }}
            />
        </li>
    );
}

// Only mounted while the project is expanded, so collapsed projects do not
// each fetch their plans.
function SidebarProjectPlans({
    projectId,
    pathname,
}: {
    projectId: string;
    pathname: string;
}) {
    const t = useTranslate();
    const plans = useProjectPlans(projectId);
    const overview = `/projects/${projectId}`;

    return (
        <ul className="mt-0.5 flex flex-col gap-0.5">
            <SidebarNavItem
                nested
                to={overview}
                label={t("sidebar.project.overview")}
                icon={FiFolder}
                active={pathname === overview}
                collapsed={false}
            />
            {plans.isLoading && (
                <li className="py-1 pl-8 text-caption text-fg-placeholder">
                    {t("common.loading")}
                </li>
            )}
            {plans.isError && (
                <li className="py-1 pl-8 text-caption text-danger">
                    {t("common.error")}
                </li>
            )}
            {plans.data?.map((plan) => {
                const to = `${overview}/plans/${plan.id}`;
                return (
                    <SidebarNavItem
                        key={plan.id}
                        nested
                        to={to}
                        label={plan.icon ? `${plan.icon} ${plan.name}` : plan.name}
                        icon={TbLayoutKanban}
                        active={pathname === to}
                        collapsed={false}
                    />
                );
            })}
        </ul>
    );
}
