export interface PageQuery {
  search: string
  page: number
  pageSize: number
}

export interface PageResult<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
}
