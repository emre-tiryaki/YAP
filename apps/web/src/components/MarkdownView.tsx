import { TaskItem, TaskList } from "@tiptap/extension-list";
import { CodeBlockLowlight } from "@tiptap/extension-code-block-lowlight";
import { Markdown } from "@tiptap/markdown";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { common, createLowlight } from "lowlight";
import { usePreferences } from "../hooks/usePreferences";

const lowlight = createLowlight(common);

// Read-only rendering of a markdown document.
export function MarkdownView({ markdown }: { markdown: string }) {
    const { documentFontSize } = usePreferences();
    const editor = useEditor(
        {
            extensions: [
                StarterKit.configure({ codeBlock: false }),
                CodeBlockLowlight.configure({ lowlight }),
                TaskList,
                TaskItem.configure({ nested: true }),
                Markdown,
            ],
            content: markdown,
            contentType: "markdown",
            editable: false,
            editorProps: {
                attributes: {
                    class: "markdown-body prose prose-sm dark:prose-invert max-w-none px-6 py-4 focus:outline-none",
                },
            },
        },
        [markdown],
    );
    // The prose sizes are in em, so one font size scales the whole document.
    return (
        <EditorContent
            editor={editor}
            style={{ fontSize: documentFontSize }}
            className="h-full overflow-auto [&_.markdown-body]:text-[1em]"
        />
    );
}
