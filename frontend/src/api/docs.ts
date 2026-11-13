import { apiClient } from './client'

export interface DocCategory { id: number; name: string }
export interface DocSummary {
  id: number
  category_id: number
  slug: string
  title: string
  description: string
  visibility: 'public' | 'authenticated'
}
export interface DocInput extends Omit<DocSummary, 'id'> {
  markdown: string
  seo_title: string
  seo_description: string
  revision: number
}
interface DocState {
  status: 'draft' | 'published' | 'offline'
  revision: number
  published_revision: number
  published_slug: string
}
export interface DocArticle extends DocInput, DocState { id: number; updated_at: string }
export interface DocPublication extends DocSummary {
  markdown: string
  seo_title: string
  seo_description: string
  published_at: string
}
export interface DocAdminSummary extends DocSummary, DocState {}
export interface DocNavigation<T = DocSummary> { categories: DocCategory[]; articles: T[] }
export interface DocSearchResult extends DocSummary { snippet: string }

const publicBase = '/docs'
const adminBase = '/admin/docs'
const adminArticle = (id: number) => `${adminBase}/articles/${id}`

export const docsAPI = {
  navigation: (signal?: AbortSignal) => apiClient.get<DocNavigation>(publicBase, { signal }).then(r => r.data),
  article: (slug: string, signal?: AbortSignal) => apiClient.get<DocPublication>(`${publicBase}/articles/${encodeURIComponent(slug)}`, { signal }).then(r => r.data),
  search: (q: string, signal?: AbortSignal) => apiClient.get<DocSearchResult[]>(`${publicBase}/search`, { params: { q }, signal }).then(r => r.data),
  image: (id: string, signal: AbortSignal, articleID?: number) =>
    apiClient.get<Blob>(`${articleID ? adminArticle(articleID) : publicBase}/images/${id}`, { responseType: 'blob', signal }).then(r => r.data),
  adminNavigation: () => apiClient.get<DocNavigation<DocAdminSummary>>(adminBase).then(r => r.data),
  adminArticle: (id: number) => apiClient.get<DocArticle>(adminArticle(id)).then(r => r.data),
  saveArticle: (input: DocInput, id?: number) => (id
    ? apiClient.put<DocArticle>(adminArticle(id), input)
    : apiClient.post<DocArticle>(`${adminBase}/articles`, input)).then(r => r.data),
  publish: (id: number, revision: number, published: boolean) =>
    apiClient.put<DocArticle>(`${adminArticle(id)}/publication`, { published, revision }).then(r => r.data),
  deleteArticle: (id: number, revision: number) => apiClient.delete(adminArticle(id), { params: { revision } }),
  saveCategory: (name: string, id?: number) => (id
    ? apiClient.put<DocCategory>(`${adminBase}/categories/${id}`, { name })
    : apiClient.post<DocCategory>(`${adminBase}/categories`, { name })).then(r => r.data),
  deleteCategory: (id: number) => apiClient.delete(`${adminBase}/categories/${id}`),
  sortCategories: (ids: number[]) => apiClient.put(`${adminBase}/categories/order`, { ids }),
  sortArticles: (category: number, ids: number[]) => apiClient.put(`${adminBase}/categories/${category}/articles/order`, { ids }),
  uploadImage: (article: number, file: File) =>
    apiClient.post<{ path: string }>(`${adminArticle(article)}/images`, file, {
      headers: { 'Content-Type': 'application/octet-stream' },
    }).then(r => r.data.path),
}
