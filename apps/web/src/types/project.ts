export type Project = {
  id: string;
  name: string;
  description: string;
  taskCount: number;
};

export type CreateProjectInput = {
  name: string
  description: string
}

export type UpdateProjectInput = {
  name?: string
  description?: string
}
