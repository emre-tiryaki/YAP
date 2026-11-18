export type ResourceKind = "folder" | "file";

export type Resource = {
  id: string;
  // null at the top of the tree.
  parentId: string | null;
  kind: ResourceKind;
  name: string;
  mimeType: string;
  size: number;
  // Valid UTF-8 text small enough to edit.
  isText: boolean;
  // Counts content changes; a save must name the revision it was based on.
  revision: number;
  createdAt: Date;
  updatedAt: Date;
  // Set while the item (or a folder it sits in) is in the trash.
  trashedAt: Date | null;
  // The item that was moved to the trash, taking this one with it. Only an
  // item that is its own trash root can be restored.
  trashedRootId: string | null;
  // A read-only folder filled with a GitHub repository.
  isRepository: boolean;
};

// What to do when the target folder already has an item with the name.
export type OnConflict = "fail" | "keep_both" | "replace";

export type ResourceAction =
  | "created"
  | "uploaded"
  | "edited"
  | "renamed"
  | "moved"
  | "copied"
  | "trashed"
  | "restored"
  | "purged"
  | "linked"
  | "unlinked";

export type ResourceEventDetails = {
  name: string;
  // Previous name (renamed) or folder path (moved, copied, trashed).
  from?: string;
  // Target folder path (moved, restored).
  to?: string;
  size?: number;
  replaced?: boolean;
  taskTitle?: string;
};

export type ResourceEvent = {
  id: string;
  resourceId: string;
  resourceKind: ResourceKind;
  actorName: string;
  action: ResourceAction;
  taskId: string | null;
  details: ResourceEventDetails;
  createdAt: Date;
  // Later than createdAt when several edits were merged into the event.
  updatedAt: Date;
};

// Narrows the activity log; nothing set lists the whole workspace.
export type ResourceEventScope =
  | { kind: "workspace" }
  | { kind: "resource"; id: string }
  | { kind: "project"; id: string };

export type LinkedTask = {
  id: string;
  title: string;
  projectId: string;
  projectName: string;
  planId: string;
  completedAt: Date | null;
};

export type ResourceViewMode = "list" | "grid";
