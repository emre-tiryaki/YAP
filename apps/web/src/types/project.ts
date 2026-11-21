export type Project = {
  id: string;
  name: string;
  description: string;
  // A single emoji, or "" for none.
  icon: string;
  // The project code of task keys, e.g. "ABC" in ABC-12.
  identifier: string;
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
  // 2 to 5 letters or digits, starting with a letter.
  identifier?: string
}
