package service

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

const MaxDocMarkdownBytes = 256 << 10

var (
	ErrDocNotFound         = infraerrors.NotFound("DOC_NOT_FOUND", "Document not found")
	ErrDocConflict         = infraerrors.Conflict("DOC_CONFLICT", "The name or URL is in use, or the document changed. Reload and try again.")
	ErrDocCategoryNotEmpty = infraerrors.Conflict("DOC_CATEGORY_NOT_EMPTY", "Move the articles out of this category before deleting it")
	ErrDocInvalid          = infraerrors.BadRequest("DOC_INVALID", "Check the documentation fields")
	docSlugPattern         = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	docImagePattern        = regexp.MustCompile(`^/api/v1/docs/images/([0-9a-fA-F-]{36})$`)
)

var docsMarkdownParser = goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser()

type DocCategory struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type DocSummary struct {
	ID          int64  `json:"id"`
	CategoryID  int64  `json:"category_id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type DocInput struct {
	CategoryID     int64  `json:"category_id"`
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Visibility     string `json:"visibility"`
	Markdown       string `json:"markdown"`
	SEOTitle       string `json:"seo_title"`
	SEODescription string `json:"seo_description"`
	Revision       int64  `json:"revision"`
}

type DocArticle struct {
	ID int64 `json:"id"`
	DocInput
	Status            string    `json:"status"`
	PublishedRevision int64     `json:"published_revision"`
	PublishedSlug     string    `json:"published_slug"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type DocAdminSummary struct {
	DocSummary
	Status            string `json:"status"`
	Revision          int64  `json:"revision"`
	PublishedRevision int64  `json:"published_revision"`
	PublishedSlug     string `json:"published_slug"`
}

type DocPublication struct {
	DocSummary
	Markdown       string    `json:"markdown"`
	SEOTitle       string    `json:"seo_title"`
	SEODescription string    `json:"seo_description"`
	PublishedAt    time.Time `json:"published_at"`
}

type DocNavigation struct {
	Categories []DocCategory `json:"categories"`
	Articles   []DocSummary  `json:"articles"`
}

type DocAdminNavigation struct {
	Categories []DocCategory     `json:"categories"`
	Articles   []DocAdminSummary `json:"articles"`
}

type DocSearchResult struct {
	DocSummary
	Snippet string `json:"snippet"`
}

type DocsRepository interface {
	Navigation(context.Context, bool) (*DocNavigation, error)
	Publication(context.Context, string, bool) (*DocPublication, error)
	PublicMetadata(context.Context, string) (string, string, error)
	Search(context.Context, string, bool) ([]DocSearchResult, error)
	AdminNavigation(context.Context) (*DocAdminNavigation, error)
	Article(context.Context, int64) (*DocArticle, error)
	SaveArticle(context.Context, int64, DocInput) (*DocArticle, error)
	Publish(context.Context, int64, int64, []uuid.UUID) (*DocArticle, error)
	Offline(context.Context, int64, int64) (*DocArticle, error)
	DeleteArticle(context.Context, int64, int64) ([]string, error)
	SaveCategory(context.Context, int64, string) (*DocCategory, error)
	DeleteCategory(context.Context, int64) error
	SortCategories(context.Context, []int64) error
	SortArticles(context.Context, int64, []int64) error
	SaveImage(context.Context, int64, DocImage) error
	AdminImage(context.Context, int64, uuid.UUID) (*DocImage, error)
	PublishedImage(context.Context, uuid.UUID, bool) (*DocImage, error)
}

type DocsService struct {
	repo      DocsRepository
	images    *ImageStorageSettingService
	encryptor SecretEncryptor
}

func NewDocsService(repo DocsRepository, images *ImageStorageSettingService, encryptor SecretEncryptor) *DocsService {
	return &DocsService{repo: repo, images: images, encryptor: encryptor}
}

func ValidDocSlug(slug string) bool {
	return slug != "batch-image" && len(slug) <= 96 && docSlugPattern.MatchString(slug)
}

func (s *DocsService) Navigation(ctx context.Context, signedIn bool) (*DocNavigation, error) {
	return s.repo.Navigation(ctx, signedIn)
}

func (s *DocsService) Publication(ctx context.Context, slug string, signedIn bool) (*DocPublication, error) {
	if !ValidDocSlug(slug) {
		return nil, ErrDocNotFound
	}
	return s.repo.Publication(ctx, slug, signedIn)
}

func (s *DocsService) PublicMetadata(ctx context.Context, slug string) (string, string, error) {
	if !ValidDocSlug(slug) {
		return "", "", ErrDocNotFound
	}
	return s.repo.PublicMetadata(ctx, slug)
}

func (s *DocsService) Search(ctx context.Context, query string, signedIn bool) ([]DocSearchResult, error) {
	query = strings.Join(strings.Fields(query), " ")
	if utf8.RuneCountInString(query) > 120 || strings.IndexByte(query, 0) >= 0 {
		return nil, ErrDocInvalid
	}
	if query == "" {
		return []DocSearchResult{}, nil
	}
	return s.repo.Search(ctx, query, signedIn)
}

func (s *DocsService) AdminNavigation(ctx context.Context) (*DocAdminNavigation, error) {
	return s.repo.AdminNavigation(ctx)
}

func (s *DocsService) Article(ctx context.Context, id int64) (*DocArticle, error) {
	return s.repo.Article(ctx, id)
}

func (s *DocsService) SaveArticle(ctx context.Context, id int64, in DocInput) (*DocArticle, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	in.Description = strings.TrimSpace(in.Description)
	in.SEOTitle = strings.TrimSpace(in.SEOTitle)
	in.SEODescription = strings.TrimSpace(in.SEODescription)
	if in.CategoryID < 1 || (id != 0 && in.Revision < 1) || !ValidDocSlug(in.Slug) ||
		in.Title == "" || len(in.Markdown) > MaxDocMarkdownBytes ||
		(in.Visibility != "public" && in.Visibility != "authenticated") {
		return nil, ErrDocInvalid
	}
	for _, field := range []struct {
		value string
		max   int
	}{
		{in.Title, 160}, {in.Description, 320}, {in.SEOTitle, 160}, {in.SEODescription, 320}, {in.Markdown, MaxDocMarkdownBytes},
	} {
		if strings.IndexByte(field.value, 0) >= 0 || !utf8.ValidString(field.value) || utf8.RuneCountInString(field.value) > field.max {
			return nil, ErrDocInvalid
		}
	}
	return s.repo.SaveArticle(ctx, id, in)
}

func (s *DocsService) Publish(ctx context.Context, id, revision int64) (*DocArticle, error) {
	article, err := s.repo.Article(ctx, id)
	if err != nil {
		return nil, err
	}
	if revision < 1 || article.Revision != revision {
		return nil, ErrDocConflict
	}
	if strings.TrimSpace(article.Markdown) == "" {
		return nil, ErrDocInvalid
	}
	images, err := docImageIDs(article.Markdown)
	if err != nil {
		return nil, err
	}
	return s.repo.Publish(ctx, id, revision, images)
}

func docImageIDs(markdown string) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]bool)
	images := make([]uuid.UUID, 0)
	tree := docsMarkdownParser.Parse(text.NewReader([]byte(markdown)))
	err := ast.Walk(tree, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		image, ok := node.(*ast.Image)
		if !entering || !ok || !strings.HasPrefix(string(image.Destination), "/api/v1/docs/images/") {
			return ast.WalkContinue, nil
		}
		match := docImagePattern.FindSubmatch(image.Destination)
		if match == nil {
			return ast.WalkStop, ErrDocInvalid
		}
		id, err := uuid.Parse(string(match[1]))
		if err != nil {
			return ast.WalkStop, ErrDocInvalid
		}
		if !seen[id] {
			seen[id] = true
			images = append(images, id)
		}
		return ast.WalkContinue, nil
	})
	return images, err
}

func (s *DocsService) Offline(ctx context.Context, id, revision int64) (*DocArticle, error) {
	return s.repo.Offline(ctx, id, revision)
}

func (s *DocsService) SaveCategory(ctx context.Context, id int64, name string) (*DocCategory, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.IndexByte(name, 0) >= 0 || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 100 {
		return nil, ErrDocInvalid
	}
	return s.repo.SaveCategory(ctx, id, name)
}

func (s *DocsService) DeleteCategory(ctx context.Context, id int64) error {
	return s.repo.DeleteCategory(ctx, id)
}

func (s *DocsService) SortCategories(ctx context.Context, ids []int64) error {
	return s.repo.SortCategories(ctx, ids)
}

func (s *DocsService) SortArticles(ctx context.Context, category int64, ids []int64) error {
	return s.repo.SortArticles(ctx, category, ids)
}
