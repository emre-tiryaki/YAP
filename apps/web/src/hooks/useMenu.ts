import { createContext, useContext } from "react";

// Lets a MenuItem close the Menu it sits in after it was selected.
export const MenuContext = createContext<{ close: () => void }>({
    close: () => {},
});

export function useMenu() {
    return useContext(MenuContext);
}
