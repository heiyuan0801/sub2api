package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
	"golang.org/x/image/webp"
)

const MaxDocImageBytes = 5 << 20

type DocImage struct {
	ID          uuid.UUID
	ContentType string
	Data        []byte
	ObjectKey   string
}

// Store ciphertext in the existing image bucket. The documentation API owns
// access checks, even on deployments whose image bucket serves public URLs.
type privateDocImageStorage interface {
	PutPrivate(context.Context, string, []byte) error
	OpenPrivate(context.Context, string) (io.ReadCloser, error)
	DeletePrivate(context.Context, string) error
}

func (s *DocsService) imageStore(ctx context.Context) (privateDocImageStorage, string, error) {
	if s.images == nil {
		return nil, "", nil
	}
	cfg, err := s.images.effectiveConfig(ctx)
	if err != nil {
		return nil, "", err
	}
	// An already uploaded asset stays readable when async generation is disabled.
	if !cfg.IsConfigured() {
		return nil, "", nil
	}
	storage, err := s.images.factory(ctx, cfg)
	if err != nil {
		return nil, "", err
	}
	private, ok := storage.(privateDocImageStorage)
	if !ok {
		return nil, "", errors.New("image storage does not support private document images")
	}
	return private, path.Join(cfg.Prefix, "docs"), nil
}

func docImageType(data []byte) (string, bool) {
	if len(data) == 0 || len(data) > MaxDocImageBytes {
		return "", false
	}
	contentType := http.DetectContentType(data)
	var cfg image.Config
	var err error
	switch contentType {
	case "image/webp":
		cfg, err = webp.DecodeConfig(bytes.NewReader(data))
	case "image/png", "image/jpeg", "image/gif":
		cfg, _, err = image.DecodeConfig(bytes.NewReader(data))
	default:
		return "", false
	}
	return contentType, err == nil && cfg.Width > 0 && cfg.Height > 0 && cfg.Width <= 12000 && cfg.Height <= 12000 &&
		int64(cfg.Width)*int64(cfg.Height) <= 40_000_000
}

func (s *DocsService) UploadImage(ctx context.Context, articleID int64, data []byte) (string, error) {
	contentType, ok := docImageType(data)
	if !ok {
		return "", ErrDocInvalid
	}
	if _, err := s.repo.Article(ctx, articleID); err != nil {
		return "", err
	}
	asset := DocImage{ID: uuid.New(), ContentType: contentType, Data: data}
	store, prefix, err := s.imageStore(ctx)
	if err != nil {
		return "", err
	}
	if store != nil {
		asset.ObjectKey = path.Join(prefix, asset.ID.String())
		encrypted, err := s.encryptor.Encrypt(base64.StdEncoding.EncodeToString(data))
		if err != nil {
			return "", err
		}
		if err := store.PutPrivate(ctx, asset.ObjectKey, []byte(encrypted)); err != nil {
			return "", err
		}
		asset.Data = nil
	}
	if err := s.repo.SaveImage(ctx, articleID, asset); err != nil {
		if store != nil {
			s.removeImageObjects(ctx, []string{asset.ObjectKey})
		}
		return "", err
	}
	return "/api/v1/docs/images/" + asset.ID.String(), nil
}

func (s *DocsService) ReadImage(ctx context.Context, articleID int64, id uuid.UUID, signedIn bool) (*DocImage, error) {
	var asset *DocImage
	var err error
	if articleID > 0 {
		asset, err = s.repo.AdminImage(ctx, articleID, id)
	} else {
		asset, err = s.repo.PublishedImage(ctx, id, signedIn)
	}
	if err != nil || asset.ObjectKey == "" {
		return asset, err
	}
	store, _, err := s.imageStore(ctx)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, errors.New("document image storage is unavailable")
	}
	reader, err := store.OpenPrivate(ctx, asset.ObjectKey)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	const maxCiphertext = 2 * MaxDocImageBytes
	ciphertext, err := io.ReadAll(io.LimitReader(reader, maxCiphertext+1))
	if err != nil {
		return nil, err
	}
	if len(ciphertext) > maxCiphertext {
		return nil, errors.New("stored document image is too large")
	}
	plain, err := s.encryptor.Decrypt(string(ciphertext))
	if err != nil {
		return nil, err
	}
	asset.Data, err = base64.StdEncoding.DecodeString(plain)
	if err != nil || len(asset.Data) > MaxDocImageBytes {
		return nil, errors.New("invalid stored document image")
	}
	return asset, nil
}

func (s *DocsService) DeleteArticle(ctx context.Context, id, revision int64) error {
	keys, err := s.repo.DeleteArticle(ctx, id, revision)
	if err == nil {
		s.removeImageObjects(ctx, keys)
	}
	return err
}

func (s *DocsService) removeImageObjects(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	store, _, err := s.imageStore(ctx)
	if err != nil || store == nil {
		logger.LegacyPrintf("service.docs", "Document image storage unavailable for cleanup")
		return
	}
	for _, key := range keys {
		if err := store.DeletePrivate(ctx, key); err != nil {
			logger.LegacyPrintf("service.docs", "Could not remove a document image from object storage")
		}
	}
}
