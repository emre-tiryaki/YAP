export type Comment = {
  id: string
  taskId: string
  authorId: string
  authorName: string
  body: string
  createdAt: Date
  updatedAt: Date
  // Whether the current member wrote it (and so may edit or delete it).
  isOwn: boolean
}
