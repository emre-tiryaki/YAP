import type { ButtonHTMLAttributes, ComponentType } from "react";
import { Button } from "./Button";
import { Tooltip } from "./Tooltip";

type HeaderActionProps = Omit<
    ButtonHTMLAttributes<HTMLButtonElement>,
    "children"
> & {
    icon: ComponentType<{ "aria-hidden"?: boolean }>;
    label: string;
    variant?: "primary" | "secondary" | "ghost";
};

// A PageHeader button whose label hides when the header is narrow; the label
// then stays as its accessible name and tooltip.
export function HeaderAction({
    icon,
    label,
    variant = "secondary",
    ...rest
}: HeaderActionProps) {
    return (
        <Tooltip label={label}>
            <Button
                variant={variant}
                size="md"
                leadingIcon={icon}
                aria-label={label}
                {...rest}
            >
                <span className="hidden @3xl:inline">{label}</span>
            </Button>
        </Tooltip>
    );
}
