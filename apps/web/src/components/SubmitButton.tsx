import type { ReactNode } from "react";
import { Button } from "./Button";

type SubmitButtonProps = {
    children: ReactNode;
    disabled?: boolean;
};

export function SubmitButton({ children, disabled }: SubmitButtonProps) {
    return (
        <Button variant="primary" size="lg" type="submit" disabled={disabled}>
            {children}
        </Button>
    );
}
