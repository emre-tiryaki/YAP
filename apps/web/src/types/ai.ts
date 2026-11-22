export type AiProvider = "openai_compatible" | "anthropic";

export type AiPreset =
    | "ollama"
    | "lmstudio"
    | "llamacpp"
    | "openai"
    | "openrouter"
    | "gemini"
    | "groq"
    | "mistral"
    | "anthropic"
    | "custom";

export type AiConnection = {
    id: string;
    name: string;
    provider: AiProvider;
    preset: AiPreset;
    baseUrl: string;
    // An API key is stored in the OS keychain; it is never sent to the page.
    hasKey: boolean;
    createdAt: Date;
    updatedAt: Date;
};

export type CreateAiConnectionInput = {
    name: string;
    preset: AiPreset;
    baseUrl: string;
    apiKey?: string;
};

// A set field changes; apiKey "" removes the stored key.
export type UpdateAiConnectionInput = {
    name?: string;
    baseUrl?: string;
    apiKey?: string;
};

export type TestAiConnectionInput = {
    preset: AiPreset;
    baseUrl: string;
    apiKey?: string;
};

export type AiChat = {
    id: string;
    title: string;
    // null when the chat has none yet or its connection was removed.
    connectionId: string | null;
    model: string;
    generating: boolean;
    createdAt: Date;
    updatedAt: Date;
};

export type AiMessageStatus =
    | "complete"
    | "streaming"
    | "stopped"
    | "interrupted"
    | "error";

export type AiAttachmentKind =
    | "task"
    | "plan"
    | "project"
    | "resource"
    | "repo_file";

export type AiAttachmentStatus =
    | "included"
    | "truncated"
    | "omitted"
    | "missing"
    | "unreadable";

export type AiAttachment = {
    id: string;
    kind: AiAttachmentKind;
    targetId: string;
    // Only for repo_file: the file's path inside the repository.
    path: string | null;
    label: string;
    // How the target went into the latest request.
    lastStatus: AiAttachmentStatus;
    // The target is gone or in the trash.
    missing: boolean;
    projectId: string | null;
    planId: string | null;
};

export type AiMessage = {
    id: string;
    role: "user" | "assistant";
    content: string;
    status: AiMessageStatus;
    errorCode: string | null;
    connectionId: string | null;
    model: string | null;
    createdAt: Date;
    attachments: AiAttachment[];
};

export type AiChatDetail = {
    chat: AiChat;
    messages: AiMessage[];
};

// What the user attaches to a message; stored by reference.
export type AiAttachmentInput = {
    kind: AiAttachmentKind;
    targetId: string;
    path?: string;
    // Shown on the chip in the composer only; not sent.
    label: string;
};

export type AiSendInput = {
    connectionId: string;
    model: string;
    content: string;
    attachments: AiAttachmentInput[];
};

export type AiSendResult = {
    chat: AiChat;
    userMessageId: string;
    assistantMessageId: string;
};

export type AiStreamHandlers = {
    onSnapshot: (messageId: string, content: string) => void;
    onDelta: (text: string) => void;
    onDone: (messageId: string, status: AiMessageStatus) => void;
};
