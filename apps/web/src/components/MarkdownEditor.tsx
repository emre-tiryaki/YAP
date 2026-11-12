import { Markdown } from "@tiptap/markdown";
import { TaskItem, TaskList } from "@tiptap/extension-list";
import { Placeholder } from "@tiptap/extensions";
import {
    EditorContent,
    useEditor,
    useEditorState,
    type Editor,
} from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import {
    useEffect,
    useLayoutEffect,
    useRef,
    useState,
    type FocusEvent,
} from "react";
import type { IconType } from "react-icons";
import {
    LuBold,
    LuCode,
    LuHeading1,
    LuHeading2,
    LuHeading3,
    LuItalic,
    LuList,
    LuListOrdered,
    LuListTodo,
    LuMinus,
    LuSquareCode,
    LuStrikethrough,
    LuTextQuote,
} from "react-icons/lu";
import { useTranslate, type TranslationKey } from "../hooks/useTranslate";
import {
    applyMarkdownCommand,
    type MarkdownCommand,
    type TextEdit,
} from "../util/markdown";

type EditorMode = "rich" | "markdown";

const modeStorageKey = "yap.editorMode";

function readStoredMode(): EditorMode {
    return localStorage.getItem(modeStorageKey) === "markdown"
        ? "markdown"
        : "rich";
}

// Toolbar buttons; `null` separates groups.
const toolbar: ({
    command: MarkdownCommand;
    label: TranslationKey;
    Icon: IconType;
} | null)[] = [
    { command: "h1", label: "editor.heading1", Icon: LuHeading1 },
    { command: "h2", label: "editor.heading2", Icon: LuHeading2 },
    { command: "h3", label: "editor.heading3", Icon: LuHeading3 },
    null,
    { command: "bold", label: "editor.bold", Icon: LuBold },
    { command: "italic", label: "editor.italic", Icon: LuItalic },
    { command: "strike", label: "editor.strike", Icon: LuStrikethrough },
    { command: "code", label: "editor.code", Icon: LuCode },
    null,
    { command: "bulletList", label: "editor.bulletList", Icon: LuList },
    { command: "orderedList", label: "editor.orderedList", Icon: LuListOrdered },
    { command: "taskList", label: "editor.taskList", Icon: LuListTodo },
    null,
    { command: "quote", label: "editor.quote", Icon: LuTextQuote },
    { command: "codeBlock", label: "editor.codeBlock", Icon: LuSquareCode },
    { command: "divider", label: "editor.divider", Icon: LuMinus },
];

type ActiveState = Partial<Record<MarkdownCommand, boolean>>;

type MarkdownEditorProps = {
    id?: string;
    value: string;
    onChange: (markdown: string) => void;
    // Fires once focus leaves the whole editor (toolbar included).
    onBlur?: () => void;
    placeholder?: string;
};

// Markdown text editor with two modes that both read and write markdown:
// "rich" edits formatted text (typing "# " makes a heading, "- " a list),
// "markdown" shows the raw source. The chosen mode is remembered.
export function MarkdownEditor({
    id,
    value,
    onChange,
    onBlur,
    placeholder,
}: MarkdownEditorProps) {
    const [mode, setMode] = useState<EditorMode>(readStoredMode);

    function changeMode(next: EditorMode) {
        setMode(next);
        localStorage.setItem(modeStorageKey, next);
    }

    function handleBlur(event: FocusEvent<HTMLDivElement>) {
        if (!event.currentTarget.contains(event.relatedTarget)) onBlur?.();
    }

    const shared = { id, value, onChange, placeholder, mode, changeMode };
    return (
        <div
            onBlur={handleBlur}
            className="flex flex-1 flex-col rounded-md border border-gray-200 focus-within:border-gray-400"
        >
            {mode === "rich" ? (
                <RichEditor {...shared} />
            ) : (
                <SourceEditor {...shared} />
            )}
        </div>
    );
}

type ModeEditorProps = {
    id?: string;
    value: string;
    onChange: (markdown: string) => void;
    placeholder?: string;
    mode: EditorMode;
    changeMode: (mode: EditorMode) => void;
};

function runRichCommand(editor: Editor, command: MarkdownCommand) {
    const chain = editor.chain().focus();
    switch (command) {
        case "h1":
        case "h2":
        case "h3":
            return chain
                .toggleHeading({ level: Number(command[1]) as 1 | 2 | 3 })
                .run();
        case "bold":
            return chain.toggleBold().run();
        case "italic":
            return chain.toggleItalic().run();
        case "strike":
            return chain.toggleStrike().run();
        case "code":
            return chain.toggleCode().run();
        case "bulletList":
            return chain.toggleBulletList().run();
        case "orderedList":
            return chain.toggleOrderedList().run();
        case "taskList":
            return chain.toggleTaskList().run();
        case "quote":
            return chain.toggleBlockquote().run();
        case "codeBlock":
            return chain.toggleCodeBlock().run();
        case "divider":
            return chain.setHorizontalRule().run();
    }
}

function richActiveState(editor: Editor): ActiveState {
    return {
        h1: editor.isActive("heading", { level: 1 }),
        h2: editor.isActive("heading", { level: 2 }),
        h3: editor.isActive("heading", { level: 3 }),
        bold: editor.isActive("bold"),
        italic: editor.isActive("italic"),
        strike: editor.isActive("strike"),
        code: editor.isActive("code"),
        bulletList: editor.isActive("bulletList"),
        orderedList: editor.isActive("orderedList"),
        taskList: editor.isActive("taskList"),
        quote: editor.isActive("blockquote"),
        codeBlock: editor.isActive("codeBlock"),
    };
}

function RichEditor({
    id,
    value,
    onChange,
    placeholder,
    mode,
    changeMode,
}: ModeEditorProps) {
    // The editor keeps the callbacks it was created with, so it reads the
    // latest one through a ref.
    const onChangeRef = useRef(onChange);
    useEffect(() => {
        onChangeRef.current = onChange;
    }, [onChange]);

    const editor = useEditor({
        extensions: [
            StarterKit.configure({ link: { openOnClick: false } }),
            TaskList,
            TaskItem.configure({ nested: true }),
            Placeholder.configure({ placeholder: placeholder ?? "" }),
            Markdown,
        ],
        // Created once from the current markdown; later edits flow out only.
        content: value,
        contentType: "markdown",
        onUpdate: ({ editor }) => onChangeRef.current(editor.getMarkdown()),
        editorProps: {
            attributes: {
                ...(id && { id }),
                class: "markdown-body prose prose-sm dark:prose-invert max-w-none min-h-60 px-3 py-2 focus:outline-none",
            },
        },
    });

    const active = useEditorState({
        editor,
        selector: ({ editor }) => (editor ? richActiveState(editor) : {}),
    });

    return (
        <>
            <Toolbar
                active={active ?? {}}
                onCommand={(command) => {
                    if (editor) runRichCommand(editor, command);
                }}
                mode={mode}
                changeMode={changeMode}
            />
            <EditorContent
                editor={editor}
                className="flex-1 cursor-text overflow-x-auto"
                onClick={() => {
                    if (editor && !editor.isFocused) editor.commands.focus();
                }}
            />
        </>
    );
}

function SourceEditor({
    id,
    value,
    onChange,
    placeholder,
    mode,
    changeMode,
}: ModeEditorProps) {
    const ref = useRef<HTMLTextAreaElement>(null);
    // Selection to restore once a toolbar edit has been rendered.
    const [pending, setPending] = useState<TextEdit | null>(null);

    useLayoutEffect(() => {
        const textarea = ref.current;
        if (!pending || !textarea || textarea.value !== pending.text) return;
        textarea.focus();
        textarea.setSelectionRange(pending.selectionStart, pending.selectionEnd);
    }, [pending]);

    function runCommand(command: MarkdownCommand) {
        const textarea = ref.current;
        if (!textarea) return;
        const edit = applyMarkdownCommand(
            value,
            textarea.selectionStart,
            textarea.selectionEnd,
            command,
        );
        onChange(edit.text);
        setPending(edit);
    }

    return (
        <>
            <Toolbar
                active={{}}
                onCommand={runCommand}
                mode={mode}
                changeMode={changeMode}
            />
            <textarea
                ref={ref}
                id={id}
                value={value}
                placeholder={placeholder}
                spellCheck={false}
                onChange={(event) => onChange(event.target.value)}
                className="min-h-60 w-full flex-1 resize-none bg-transparent px-3 py-2 font-mono text-sm leading-relaxed text-gray-800 focus:outline-none"
            />
        </>
    );
}

type ToolbarProps = {
    active: ActiveState;
    onCommand: (command: MarkdownCommand) => void;
    mode: EditorMode;
    changeMode: (mode: EditorMode) => void;
};

function Toolbar({ active, onCommand, mode, changeMode }: ToolbarProps) {
    const t = useTranslate();

    return (
        <div className="flex flex-wrap items-center gap-1 border-b border-gray-200 bg-gray-50 px-2 py-1.5">
            <div
                role="toolbar"
                aria-label={t("editor.toolbar")}
                className="flex flex-wrap items-center gap-0.5"
            >
                {toolbar.map((item, i) =>
                    item === null ? (
                        <span
                            key={`sep-${i}`}
                            aria-hidden
                            className="mx-1 h-5 w-px bg-gray-300"
                        />
                    ) : (
                        <button
                            key={item.command}
                            type="button"
                            title={t(item.label)}
                            aria-label={t(item.label)}
                            aria-pressed={active[item.command] ?? false}
                            // Keeps the focus (and selection) in the editor.
                            onMouseDown={(event) => event.preventDefault()}
                            onClick={() => onCommand(item.command)}
                            className={`cursor-pointer rounded p-1.5 text-gray-600 hover:bg-gray-200 hover:text-gray-900 ${
                                active[item.command]
                                    ? "bg-gray-200 text-gray-900"
                                    : ""
                            }`}
                        >
                            <item.Icon aria-hidden />
                        </button>
                    ),
                )}
            </div>
            <div
                role="group"
                aria-label={t("editor.mode.label")}
                className="ml-auto flex rounded-md border border-gray-300 bg-white p-0.5 text-xs"
            >
                {(["rich", "markdown"] as const).map((option) => (
                    <button
                        key={option}
                        type="button"
                        aria-pressed={mode === option}
                        onClick={() => changeMode(option)}
                        className={`cursor-pointer rounded px-2 py-1 ${
                            mode === option
                                ? "bg-accent text-accent-fg"
                                : "text-gray-600 hover:text-gray-900"
                        }`}
                    >
                        {t(`editor.mode.${option}`)}
                    </button>
                ))}
            </div>
        </div>
    );
}
