export type ImportCounts = {
    projects: number;
    plans: number;
    tasks: number;
    comments: number;
    resources: number;
    chats: number;
};

export type ImportResult = {
    // The workspace was empty, so names and settings came from the file too.
    restored: boolean;
    added: ImportCounts;
    // Items that were already here and were kept as they are.
    existing: number;
    // Files that got a free name because theirs was taken.
    renamed: number;
    missingFiles: string[];
};
