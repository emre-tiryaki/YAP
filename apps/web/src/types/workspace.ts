export type Workspace = {
  id: string;
  name: string;
  username: string;
};

// Fields left undefined are kept.
export type UpdateWorkspaceInput = {
  name?: string;
  username?: string;
};
