package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDocsPublicationPostgres(t *testing.T) {
	dsn := os.Getenv("SUB2API_DOCS_TEST_DSN")
	if dsn == "" {
		t.Skip("set SUB2API_DOCS_TEST_DSN for PostgreSQL integration")
	}
	ctx := context.Background()
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	schema := "docs_test_" + uuid.NewString()[:8]
	_, err = db.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	defer func() { _, _ = db.Exec("DROP SCHEMA " + schema + " CASCADE") }()
	_, err = db.Exec("SET search_path TO " + schema)
	require.NoError(t, err)
	migration, err := migrations.FS.ReadFile("252_docs_center.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(migration))
	require.NoError(t, err)

	repo := NewDocsRepository(db)
	docs := service.NewDocsService(repo, nil, nil)
	nav, err := docs.Navigation(ctx, false)
	require.NoError(t, err)
	require.Len(t, nav.Categories, 5)
	require.Len(t, nav.Articles, 5)

	group, err := docs.SaveCategory(ctx, 0, "Protected documents")
	require.NoError(t, err)
	input := service.DocInput{CategoryID: group.ID, Slug: "docs-test", Title: "Public reference",
		Markdown: "Example text. 缓存时间可配置", Visibility: "public", SEOTitle: "Initial SEO"}
	draft, err := docs.SaveArticle(ctx, 0, input)
	require.NoError(t, err)
	_, err = docs.Publication(ctx, input.Slug, true)
	require.ErrorIs(t, err, service.ErrDocNotFound)

	id := uuid.New()
	require.NoError(t, repo.SaveImage(ctx, draft.ID, service.DocImage{ID: id, ContentType: "image/png", Data: []byte{1, 2, 3}}))
	_, err = repo.PublishedImage(ctx, id, true)
	require.ErrorIs(t, err, service.ErrDocNotFound)
	input.Revision = draft.Revision
	input.Markdown += "\n\n![Sample][img]\n\n[img]: /api/v1/docs/images/" + id.String()
	draft, err = docs.SaveArticle(ctx, draft.ID, input)
	require.NoError(t, err)
	draft, err = docs.Publish(ctx, draft.ID, draft.Revision)
	require.NoError(t, err)
	result, err := docs.Search(ctx, "缓存时间", false)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Contains(t, result[0].Snippet, "缓存时间")
	img, err := repo.PublishedImage(ctx, id, false)
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3}, img.Data)

	_, err = docs.SaveArticle(ctx, draft.ID, input)
	require.ErrorIs(t, err, service.ErrDocConflict)
	input.Revision = draft.Revision
	input.Markdown = "unpublished-secret phrase"
	input.Slug = "protected-reference"
	input.Visibility = "authenticated"
	draft, err = docs.SaveArticle(ctx, draft.ID, input)
	require.NoError(t, err)
	title, _, err := docs.PublicMetadata(ctx, "docs-test")
	require.NoError(t, err)
	require.Equal(t, "Initial SEO", title)
	result, err = docs.Search(ctx, "unpublished-secret", true)
	require.NoError(t, err)
	require.Empty(t, result)
	_, err = docs.Publication(ctx, input.Slug, true)
	require.ErrorIs(t, err, service.ErrDocNotFound)

	draft, err = docs.Publish(ctx, draft.ID, draft.Revision)
	require.NoError(t, err)
	_, err = docs.Publication(ctx, "docs-test", true)
	require.ErrorIs(t, err, service.ErrDocNotFound)
	_, err = docs.Publication(ctx, input.Slug, false)
	require.ErrorIs(t, err, service.ErrDocNotFound)
	_, _, err = docs.PublicMetadata(ctx, input.Slug)
	require.ErrorIs(t, err, service.ErrDocNotFound)
	_, err = repo.PublishedImage(ctx, id, true)
	require.ErrorIs(t, err, service.ErrDocNotFound)
	result, err = docs.Search(ctx, "unpublished-secret", false)
	require.NoError(t, err)
	require.Empty(t, result)
	result, err = docs.Search(ctx, "unpublished-secret", true)
	require.NoError(t, err)
	require.Len(t, result, 1)
	nav, err = docs.Navigation(ctx, false)
	require.NoError(t, err)
	require.Len(t, nav.Categories, 5)

	require.ErrorIs(t, docs.DeleteCategory(ctx, group.ID), service.ErrDocCategoryNotEmpty)
	require.NoError(t, docs.SortArticles(ctx, group.ID, []int64{draft.ID}))
	require.ErrorIs(t, docs.SortArticles(ctx, group.ID, []int64{nav.Articles[0].ID}), service.ErrDocConflict)
	all, err := docs.AdminNavigation(ctx)
	require.NoError(t, err)
	ids := make([]int64, len(all.Categories))
	for i, cat := range all.Categories {
		ids[len(ids)-1-i] = cat.ID
	}
	require.NoError(t, docs.SortCategories(ctx, ids))
	all, err = docs.AdminNavigation(ctx)
	require.NoError(t, err)
	require.Equal(t, group.ID, all.Categories[0].ID)

	draft, err = docs.Offline(ctx, draft.ID, draft.Revision)
	require.NoError(t, err)
	_, err = docs.Publication(ctx, input.Slug, true)
	require.ErrorIs(t, err, service.ErrDocNotFound)
	require.NoError(t, docs.DeleteArticle(ctx, draft.ID, draft.Revision))
	require.NoError(t, docs.DeleteCategory(ctx, group.ID))
}
