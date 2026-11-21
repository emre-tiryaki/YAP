import type { ButtonHTMLAttributes, ComponentType } from "react";
import { FiLoader } from "react-icons/fi";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: "primary" | "secondary" | "ghost" | "danger";
    size?: "sm" | "md" | "lg";
    leadingIcon?: ComponentType<{ "aria-hidden"?: boolean }>;
    loading?: boolean;
};

const sizes = {
    sm: "h-6 px-2 text-caption",
    md: "h-7 px-2.5 text-body-xs",
    lg: "h-8 px-3 text-sm",
};

const variants = {
    primary: "bg-accent text-accent-fg hover:opacity-90",
    secondary: "border border-line bg-layer-2 text-fg hover:bg-layer-2-hover",
    ghost: "text-fg-muted hover:bg-layer-1-hover hover:text-fg",
    danger: "bg-danger text-white hover:opacity-90",
};

export function Button({
    variant = "secondary",
    size = "md",
    leadingIcon: LeadingIcon,
    loading,
    className = "",
    type = "button",
    disabled,
    children,
    ...rest
}: ButtonProps) {
    return (
        <button
            type={type}
            disabled={disabled || loading}
            className={`inline-flex cursor-pointer items-center justify-center gap-1.5 rounded-md font-medium whitespace-nowrap focus-visible:outline-2 focus-visible:outline-accent disabled:cursor-default disabled:opacity-50 ${sizes[size]} ${variants[variant]} ${className}`}
            {...rest}
        >
            {loading ? (
                <FiLoader className="animate-spin" aria-hidden />
            ) : (
                LeadingIcon && <LeadingIcon aria-hidden />
            )}
            {children}
        </button>
    );
}
