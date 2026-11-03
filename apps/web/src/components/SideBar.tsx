import { useState } from "react";
import { useTranslate } from "../hooks/useTranslate";
import type { TranslationKey } from "../hooks/useTranslate";
import { TbLayoutSidebarLeftCollapse, TbLayoutSidebarRightCollapse } from "react-icons/tb";


type NavItem = {
  labelKey: TranslationKey;
  href: string;
};

const navItems: NavItem[] = [
  { labelKey: "sidebar.home", href: "/" },
  { labelKey: "sidebar.projects", href: "/projects" },
  { labelKey: "sidebar.tasks", href: "/tasks" },
  { labelKey: "sidebar.teams", href: "/teams" },
  { labelKey: "sidebar.settings", href: "/settings" },
];

type SidebarProps = {
  activeHref: string;
};

export function Sidebar({ activeHref }: SidebarProps) {
  const t = useTranslate();
  const [isCollapsed, setIsCollapsed] = useState(false);

  return (
    <aside className="flex w-64 shrink-0 flex-col border-r border-gray-200 bg-gray-50 p-4">
      <div className="">
        <span className="mb-6 px-3 text-xl font-bold">{t("app.name")}</span>
        <button
          onClick={() => setIsCollapsed(!isCollapsed)}
          className="cursor-pointer">
          {isCollapsed ? <TbLayoutSidebarLeftCollapse /> : <TbLayoutSidebarRightCollapse />}
        </button>
      </div>
      <nav className="flex flex-col gap-1">
        {navItems.map((item) => (
          <a
            key={item.href}
            href={item.href}
            aria-current={item.href === activeHref ? "page" : undefined}
            className="rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-200 aria-[current=page]:bg-gray-200 aria-[current=page]:font-semibold"
          >
            {t(item.labelKey)}
          </a>
        ))}
      </nav>
    </aside>
  );
}
