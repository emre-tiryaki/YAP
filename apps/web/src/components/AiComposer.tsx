import { useRef, useState, type KeyboardEvent } from "react";
import { Link } from "react-router";
import { FiArrowUp, FiPlus, FiSquare } from "react-icons/fi";
import { aiErrorKey } from "../hooks/useAi";
import type { AiModelChoice } from "../hooks/useAiModelChoice";
import { useAiMentions, type AiMentionHit } from "../hooks/useAiMentions";
import { useResources } from "../hooks/useResources";
import { useTranslate } from "../hooks/useTranslate";
import type { AiAttachmentInput } from "../types/ai";
import { attachmentIcons } from "../util/aiAttachmentIcons";
import { indexResources } from "../util/resources";
import { AiAttachmentChip } from "./AiAttachmentChip";
import { AiMentionPopover } from "./AiMentionPopover";
import { AiModelSelect } from "./AiModelSelect";
import { Button } from "./Button";
import { FormError } from "./FormError";
import { Menu } from "./Menu";
import { MenuItem } from "./MenuItem";
import { PlanPickerDialog } from "./PlanPickerDialog";
import { RepoFilePickerDialog } from "./RepoFilePickerDialog";
import { ResourcePickerDialog } from "./ResourcePickerDialog";
import { TaskPickerDialog } from "./TaskPickerDialog";

const maxAttachments = 20;
const mentionPattern = /(?:^|\s)@(\S*)$/;

type Picker = "task" | "plan" | "project" | "file" | "repo" | null;

type AiComposerProps = {
    choice: AiModelChoice | null;
    onChoiceChange: (choice: AiModelChoice) => void;
    // Sends the message; rejects when the server refuses it.
    onSend: (content: string, attachments: AiAttachmentInput[]) => Promise<void>;
    onStop: () => void;
    // An answer is being written for the chat shown here.
    generating: boolean;
    // Another chat is being answered; sending is blocked until it ends.
    busyChatId: string | null;
    error: unknown;
    // Text to put in the field (a suggestion); n changes on every request.
    draft?: { text: string; n: number } | null;
};

export function AiComposer({
    choice,
    onChoiceChange,
    onSend,
    onStop,
    generating,
    busyChatId,
    error,
    draft = null,
}: AiComposerProps) {
    const t = useTranslate();
    const textRef = useRef<HTMLTextAreaElement>(null);
    const [text, setText] = useState("");
    const [caret, setCaret] = useState(0);
    const [attachments, setAttachments] = useState<AiAttachmentInput[]>([]);
    const [picker, setPicker] = useState<Picker>(null);
    const [active, setActive] = useState(0);
    const [dismissed, setDismissed] = useState<string | null>(null);
    const [sending, setSending] = useState(false);
    const resources = useResources();
    const [seenDraft, setSeenDraft] = useState(draft?.n ?? 0);
    if (draft && draft.n !== seenDraft) {
        setSeenDraft(draft.n);
        setText(draft.text);
        setCaret(draft.text.length);
    }

    const mention = mentionPattern.exec(text.slice(0, caret));
    const mentionQuery =
        mention && dismissed !== mention[1] + ":" + caret ? mention[1] : null;
    const { hits, isPending } = useAiMentions(mentionQuery);
    const mentionOpen = mentionQuery !== null && mentionQuery !== "";
    const activeIndex = Math.min(active, Math.max(hits.length - 1, 0));

    const full = attachments.length >= maxAttachments;
    const blocked = busyChatId !== null && !generating;
    const canSend =
        !generating && !blocked && !sending && text.trim() !== "" && choice !== null;

    function attach(input: AiAttachmentInput) {
        setAttachments((prev) =>
            prev.length >= maxAttachments ||
            prev.some(
                (a) =>
                    a.kind === input.kind &&
                    a.targetId === input.targetId &&
                    (a.path ?? "") === (input.path ?? ""),
            )
                ? prev
                : [...prev, input],
        );
        setPicker(null);
    }

    function pickMention(hit: AiMentionHit) {
        if (!mention) return;
        // Cut "@query" out of the text and attach the hit instead.
        const start = caret - mention[1].length - 1;
        setText(text.slice(0, start) + text.slice(caret));
        attach({ kind: hit.kind, targetId: hit.targetId, label: hit.label });
        textRef.current?.focus();
    }

    async function submit() {
        if (!canSend || !choice) return;
        setSending(true);
        try {
            await onSend(text.trim(), attachments);
            setText("");
            setAttachments([]);
        } catch {
            // The parent shows the error; the draft stays.
        } finally {
            setSending(false);
        }
    }

    function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
        if (mentionOpen && hits.length > 0) {
            if (event.key === "ArrowDown") {
                event.preventDefault();
                setActive((activeIndex + 1) % hits.length);
                return;
            }
            if (event.key === "ArrowUp") {
                event.preventDefault();
                setActive((activeIndex - 1 + hits.length) % hits.length);
                return;
            }
            if (event.key === "Enter" || event.key === "Tab") {
                event.preventDefault();
                pickMention(hits[activeIndex]);
                return;
            }
        }
        if (mentionOpen && event.key === "Escape") {
            event.preventDefault();
            setDismissed((mention?.[1] ?? "") + ":" + caret);
            return;
        }
        if (event.key === "Enter" && !event.shiftKey && !event.nativeEvent.isComposing) {
            event.preventDefault();
            void submit();
        }
    }

    return (
        <div className="shrink-0 px-4 pb-4">
            <div className="mx-auto flex max-w-3xl flex-col gap-2">
                {blocked && (
                    <p role="status" className="text-body-xs text-fg-muted">
                        {t("assistant.busy")}{" "}
                        <Link
                            to={`/assistant/${busyChatId}`}
                            className="underline hover:text-fg"
                        >
                            {t("assistant.busy.link")}
                        </Link>
                    </p>
                )}
                {error != null && <FormError message={t(aiErrorKey(error))} />}
                <div className="flex flex-col gap-2 rounded-xl border border-line bg-layer-2 p-2.5 shadow-raised transition-colors focus-within:border-line-strong">
                    {attachments.length > 0 && (
                        <div className="flex flex-wrap gap-1.5">
                            {attachments.map((a) => (
                                <AiAttachmentChip
                                    key={`${a.kind}:${a.targetId}:${a.path ?? ""}`}
                                    kind={a.kind}
                                    label={a.label}
                                    onRemove={() =>
                                        setAttachments((prev) =>
                                            prev.filter((x) => x !== a),
                                        )
                                    }
                                />
                            ))}
                        </div>
                    )}
                    <textarea
                        ref={textRef}
                        value={text}
                        rows={2}
                        aria-label={t("assistant.composer.label")}
                        placeholder={t("assistant.composer.placeholder")}
                        autoFocus
                        onChange={(event) => {
                            setText(event.target.value);
                            setCaret(event.target.selectionStart);
                            setActive(0);
                        }}
                        onSelect={(event) =>
                            setCaret(event.currentTarget.selectionStart)
                        }
                        onKeyDown={handleKeyDown}
                        className="max-h-48 min-h-12 w-full resize-none bg-transparent px-1 text-sm text-fg outline-none"
                    />
                    {mentionOpen && (
                        <AiMentionPopover
                            anchorRef={textRef}
                            hits={hits}
                            isPending={isPending}
                            activeIndex={activeIndex}
                            onPick={pickMention}
                            onClose={() =>
                                setDismissed((mention?.[1] ?? "") + ":" + caret)
                            }
                        />
                    )}
                    <div className="flex items-center justify-between gap-2">
                        <div className="flex items-center gap-1">
                            <Menu
                                placement="bottom-start"
                                trigger={(props) => (
                                    <Button
                                        variant="ghost"
                                        size="md"
                                        disabled={full}
                                        aria-label={t("assistant.composer.attach")}
                                        title={full ? t("assistant.attach.limit") : undefined}
                                        {...props}
                                    >
                                        <FiPlus aria-hidden />
                                    </Button>
                                )}
                            >
                                <MenuItem icon={<attachmentIcons.task aria-hidden />} label={t("assistant.attach.task")} onSelect={() => setPicker("task")} />
                                <MenuItem icon={<attachmentIcons.plan aria-hidden />} label={t("assistant.attach.plan")} onSelect={() => setPicker("plan")} />
                                <MenuItem icon={<attachmentIcons.project aria-hidden />} label={t("assistant.attach.project")} onSelect={() => setPicker("project")} />
                                <MenuItem icon={<attachmentIcons.resource aria-hidden />} label={t("assistant.attach.file")} onSelect={() => setPicker("file")} />
                                <MenuItem icon={<attachmentIcons.repo_file aria-hidden />} label={t("assistant.attach.repoFile")} onSelect={() => setPicker("repo")} />
                            </Menu>
                            <AiModelSelect
                                value={choice}
                                onChange={onChoiceChange}
                                disabled={generating}
                            />
                        </div>
                        {generating ? (
                            <Button variant="secondary" size="md" leadingIcon={FiSquare} onClick={onStop}>
                                {t("assistant.composer.stop")}
                            </Button>
                        ) : (
                            <Button
                                variant="primary"
                                size="md"
                                leadingIcon={FiArrowUp}
                                loading={sending}
                                disabled={!canSend}
                                title={choice === null ? t("assistant.noModel") : undefined}
                                onClick={() => void submit()}
                            >
                                {t("assistant.composer.send")}
                            </Button>
                        )}
                    </div>
                </div>
            </div>

            {picker === "task" && (
                <TaskPickerDialog
                    title={t("assistant.picker.task")}
                    submitLabel={t("assistant.picker.submit")}
                    onPick={(id, label) => attach({ kind: "task", targetId: id, label })}
                    onClose={() => setPicker(null)}
                />
            )}
            {(picker === "plan" || picker === "project") && (
                <PlanPickerDialog
                    mode={picker}
                    title={t(picker === "plan" ? "assistant.picker.plan" : "assistant.picker.project")}
                    onPick={(id, label) => attach({ kind: picker, targetId: id, label })}
                    onClose={() => setPicker(null)}
                />
            )}
            {picker === "file" && (
                <ResourcePickerDialog
                    title={t("assistant.picker.file")}
                    confirmLabel={t("assistant.picker.submit")}
                    index={indexResources(resources.data ?? [])}
                    mode="textFile"
                    onPick={(id) => {
                        const resource = resources.data?.find((r) => r.id === id);
                        if (resource)
                            attach({ kind: "resource", targetId: resource.id, label: resource.name });
                    }}
                    onClose={() => setPicker(null)}
                />
            )}
            {picker === "repo" && (
                <RepoFilePickerDialog
                    onPick={(id, path, label) =>
                        attach({ kind: "repo_file", targetId: id, path, label })
                    }
                    onClose={() => setPicker(null)}
                />
            )}
        </div>
    );
}
