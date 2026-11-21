import { useNavigate } from "react-router";
import { openPalette } from "../hooks/useCommandPalette";
import { openSettings } from "../hooks/useSettingsDialog";
import { openShortcutsDialog } from "../hooks/useShortcutsDialog";
import { useKeySequence, useShortcut } from "../hooks/useShortcut";

// Registers the app-wide shortcuts; renders nothing. Mounted once in the shell.
export function GlobalShortcuts() {
    const navigate = useNavigate();

    useShortcut({ key: "k", mod: true }, () => openPalette());
    useShortcut({ key: "/", mod: true }, openShortcutsDialog);
    useShortcut({ key: ",", mod: true }, () => openSettings("general"));

    useKeySequence(["g", "h"], () => void navigate("/home"));
    useKeySequence(["g", "t"], () => void navigate("/tasks"));
    useKeySequence(["g", "p"], () => void navigate("/projects"));
    useKeySequence(["g", "r"], () => void navigate("/resources"));
    useKeySequence(["g", "s"], () => openSettings("general"));

    return null;
}
