import type { ResourceKind } from "./resource";

export type GitHubAccount = {
    id: number;
    login: string;
    name: string;
    avatarUrl: string;
    scopes: string;
    connectedAt: Date;
};

export type GitHubStatus = {
    // False when the backend has no GITHUB_CLIENT_ID; nothing can be connected.
    configured: boolean;
    // Whether repositories can be cloned to this computer.
    gitAvailable: boolean;
    account: GitHubAccount | null;
};

// Shown while connecting: open verificationUri and enter userCode.
export type DeviceFlow = {
    userCode: string;
    verificationUri: string;
    expiresAt: Date;
    intervalSeconds: number;
};

export type ConnectState = "pending" | "connected" | "expired" | "denied";

export type ConnectResult = {
    state: ConnectState;
    account: GitHubAccount | null;
};

export type GitHubRepo = {
    id: number;
    fullName: string;
    description: string;
    htmlUrl: string;
    defaultBranch: string;
    private: boolean;
    pushedAt: Date | null;
};

export type CheckoutState = "pending" | "fetching" | "ready";

// Why the last fetch or update attempt failed.
export type CheckoutError =
    | "offline"
    | "unauthorized"
    | "not_found"
    | "too_large"
    | "failed";

// A resource folder filled with a GitHub repository, with the state of this
// device's read-only copy (the files of the newest commit, from a ZIP).
export type Repository = {
    resourceId: string;
    githubId: number;
    fullName: string;
    htmlUrl: string;
    branch: string;
    private: boolean;
    state: CheckoutState;
    // The commit of the copy on disk.
    commitSha: string | null;
    fetchedAt: Date | null;
    // The last time GitHub answered.
    checkedAt: Date | null;
    attemptedAt: Date | null;
    error: CheckoutError | null;
};

// A file or folder inside a repository's copy.
export type RepoEntry = {
    name: string;
    // "/"-separated from the top of the repository.
    path: string;
    kind: ResourceKind;
    size: number;
    mimeType: string;
    isText: boolean;
    modifiedAt: Date;
};

export type AddRepositoryInput = {
    parentId: string | null;
    fullName: string;
};

// A git clone the user made into a folder of their choice. YAP never changes
// it afterwards.
export type RepositoryClone = {
    id: string;
    path: string;
    createdAt: Date;
    // False when the folder was deleted or moved since.
    exists: boolean;
};

// A folder on this computer, listed to pick where to clone.
export type LocalFolder = {
    path: string;
    // null at the root of the file system.
    parent: string | null;
    home: string;
    folders: string[];
};

export type CloneRepositoryInput = {
    resourceId: string;
    parentPath: string;
    name: string;
};

// One commit of the repository's branch, newest first.
export type RepoCommit = {
    sha: string;
    parents: string[];
    message: string;
    authorName: string;
    // Empty when GitHub cannot match the author to an account.
    authorLogin: string;
    avatarUrl: string;
    date: Date;
    htmlUrl: string;
};
