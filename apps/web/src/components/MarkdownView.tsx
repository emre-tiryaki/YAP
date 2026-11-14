import { TaskItem, TaskList } from "@tiptap/extension-list";
import { Markdown } from "@tiptap/markdown";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";

// Read-only rendering of a markdown document.
export function MarkdownView({ markdown }: { markdown: string }) {
    const editor = useEditor(
        {
            extensions: [StarterKit, TaskList, TaskItem.configure({ nested: true }), Markdown],
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
    return <EditorContent editor={editor} className="h-full overflow-auto" />;
}
