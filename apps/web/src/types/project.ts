export type Project = {
  id: string;
  name: string;
  description: string;
  // A single emoji, or "" for none.
  icon: string;
  taskCount: number;
};

export type CreateProjectInput = {
  name: string
  description: string
  icon: string
}

export type UpdateProjectInput = {
  name?: string
  description?: string
  // "" removes the icon.
  icon?: string
}
