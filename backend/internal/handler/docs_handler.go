package handler

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DocsHandler struct {
	service *service.DocsService
}

func NewDocsHandler(docs *service.DocsService) *DocsHandler { return &DocsHandler{service: docs} }

func docsReply(c *gin.Context, data any, err error) {
	if !response.ErrorFrom(c, err) {
		response.Success(c, data)
	}
}

func (h *DocsHandler) Navigation(c *gin.Context) {
	_, signedIn := middleware.GetAuthSubjectFromContext(c)
	data, err := h.service.Navigation(c.Request.Context(), signedIn)
	docsReply(c, data, err)
}

func (h *DocsHandler) Search(c *gin.Context) {
	_, signedIn := middleware.GetAuthSubjectFromContext(c)
	data, err := h.service.Search(c.Request.Context(), c.Query("q"), signedIn)
	docsReply(c, data, err)
}

func (h *DocsHandler) Publication(c *gin.Context) {
	_, signedIn := middleware.GetAuthSubjectFromContext(c)
	data, err := h.service.Publication(c.Request.Context(), c.Param("slug"), signedIn)
	docsReply(c, data, err)
}

func (h *DocsHandler) AdminNavigation(c *gin.Context) {
	data, err := h.service.AdminNavigation(c.Request.Context())
	docsReply(c, data, err)
}

func docID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, service.ErrDocNotFound)
		return 0, false
	}
	return id, true
}

func docsBind(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 6*service.MaxDocMarkdownBytes+8192)
	if err := c.ShouldBindJSON(target); err != nil {
		response.ErrorFrom(c, service.ErrDocInvalid)
		return false
	}
	return true
}

func (h *DocsHandler) Article(c *gin.Context) {
	if id, ok := docID(c); ok {
		data, err := h.service.Article(c.Request.Context(), id)
		docsReply(c, data, err)
	}
}

func (h *DocsHandler) SaveArticle(c *gin.Context) {
	var id int64
	if c.Request.Method != http.MethodPost {
		var ok bool
		id, ok = docID(c)
		if !ok {
			return
		}
	}
	var input service.DocInput
	if docsBind(c, &input) {
		data, err := h.service.SaveArticle(c.Request.Context(), id, input)
		docsReply(c, data, err)
	}
}

func (h *DocsHandler) ChangePublication(c *gin.Context) {
	id, ok := docID(c)
	if !ok {
		return
	}
	var input struct {
		Revision  int64 `json:"revision" binding:"required"`
		Published *bool `json:"published" binding:"required"`
	}
	if !docsBind(c, &input) {
		return
	}
	var data *service.DocArticle
	var err error
	if *input.Published {
		data, err = h.service.Publish(c.Request.Context(), id, input.Revision)
	} else {
		data, err = h.service.Offline(c.Request.Context(), id, input.Revision)
	}
	docsReply(c, data, err)
}

func (h *DocsHandler) DeleteArticle(c *gin.Context) {
	id, ok := docID(c)
	if !ok {
		return
	}
	revision, err := strconv.ParseInt(c.Query("revision"), 10, 64)
	if err != nil || revision < 1 {
		response.ErrorFrom(c, service.ErrDocInvalid)
		return
	}
	docsReply(c, nil, h.service.DeleteArticle(c.Request.Context(), id, revision))
}

func (h *DocsHandler) SaveCategory(c *gin.Context) {
	var id int64
	if c.Request.Method != http.MethodPost {
		var ok bool
		id, ok = docID(c)
		if !ok {
			return
		}
	}
	var input struct {
		Name string `json:"name"`
	}
	if docsBind(c, &input) {
		data, err := h.service.SaveCategory(c.Request.Context(), id, input.Name)
		docsReply(c, data, err)
	}
}

func (h *DocsHandler) DeleteCategory(c *gin.Context) {
	if id, ok := docID(c); ok {
		docsReply(c, nil, h.service.DeleteCategory(c.Request.Context(), id))
	}
}

func (h *DocsHandler) Sort(c *gin.Context) {
	var input struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if !docsBind(c, &input) {
		return
	}
	if c.Param("id") == "" {
		docsReply(c, nil, h.service.SortCategories(c.Request.Context(), input.IDs))
	} else if id, ok := docID(c); ok {
		docsReply(c, nil, h.service.SortArticles(c.Request.Context(), id, input.IDs))
	}
}

func (h *DocsHandler) UploadImage(c *gin.Context) {
	id, ok := docID(c)
	if !ok {
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxDocImageBytes))
	if err != nil {
		response.Error(c, http.StatusRequestEntityTooLarge, "Image must be at most 5 MB")
		return
	}
	path, err := h.service.UploadImage(c.Request.Context(), id, data)
	docsReply(c, gin.H{"path": path}, err)
}

func (h *DocsHandler) Image(c *gin.Context) {
	var id int64
	if c.Param("id") != "" {
		var ok bool
		id, ok = docID(c)
		if !ok {
			return
		}
	}
	imageID, err := uuid.Parse(c.Param("image"))
	if err != nil {
		response.ErrorFrom(c, service.ErrDocNotFound)
		return
	}
	_, signedIn := middleware.GetAuthSubjectFromContext(c)
	asset, err := h.service.ReadImage(c.Request.Context(), id, imageID, signedIn)
	if !response.ErrorFrom(c, err) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Data(http.StatusOK, asset.ContentType, asset.Data)
	}
}

// PublicMetadata supplies only metadata for anonymously accessible publications.
func (h *DocsHandler) PublicMetadata(ctx context.Context, slug string) (title, description string, found bool) {
	title, description, err := h.service.PublicMetadata(ctx, slug)
	return title, description, err == nil
}
