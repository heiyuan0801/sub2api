package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type docsRepository struct{ db *sql.DB }

func NewDocsRepository(db *sql.DB) service.DocsRepository { return &docsRepository{db: db} }

const docPublicFields = "p.article_id, p.category_id, p.slug, p.title, p.description, p.visibility"
const docsWriteLock int64 = 581296024761532515

type docsScanner interface{ Scan(...any) error }

func docSummaryFields(summary *service.DocSummary) []any {
	return []any{&summary.ID, &summary.CategoryID, &summary.Slug, &summary.Title, &summary.Description, &summary.Visibility}
}

func docError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrDocNotFound
	}
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return service.ErrDocConflict
		case "23503":
			return service.ErrDocInvalid
		}
	}
	return err
}

func docRows[T any](rows *sql.Rows, err error, scan func(*T) []any) ([]T, error) {
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]T, 0)
	for rows.Next() {
		var item T
		if err := rows.Scan(scan(&item)...); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func docsCategories(ctx context.Context, tx *sql.Tx, public bool, signedIn bool) ([]service.DocCategory, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT c.id, c.name FROM docs_categories c
		WHERE NOT $1 OR EXISTS (SELECT 1 FROM docs_publications p WHERE p.category_id=c.id AND ($2 OR p.visibility='public'))
		ORDER BY c.position, c.id`, public, signedIn)
	return docRows(rows, err, func(c *service.DocCategory) []any { return []any{&c.ID, &c.Name} })
}

func (r *docsRepository) Navigation(ctx context.Context, signedIn bool) (*service.DocNavigation, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	categories, err := docsCategories(ctx, tx, true, signedIn)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT `+docPublicFields+`
		FROM docs_publications p JOIN docs_categories c ON c.id=p.category_id
		WHERE $1 OR p.visibility='public' ORDER BY c.position,c.id,p.position,p.article_id`, signedIn)
	articles, err := docRows(rows, err, docSummaryFields)
	if err != nil {
		return nil, err
	}
	return &service.DocNavigation{Categories: categories, Articles: articles}, tx.Commit()
}

func (r *docsRepository) Publication(ctx context.Context, slug string, signedIn bool) (*service.DocPublication, error) {
	var item service.DocPublication
	err := r.db.QueryRowContext(ctx, `SELECT `+docPublicFields+`, p.markdown, p.seo_title, p.seo_description, p.published_at
		FROM docs_publications p WHERE p.slug=$1 AND ($2 OR p.visibility='public')`, slug, signedIn).
		Scan(append(docSummaryFields(&item.DocSummary), &item.Markdown, &item.SEOTitle, &item.SEODescription, &item.PublishedAt)...)
	return &item, docError(err)
}

func (r *docsRepository) PublicMetadata(ctx context.Context, slug string) (title, description string, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT coalesce(nullif(seo_title,''),title),coalesce(nullif(seo_description,''),description)
		FROM docs_publications WHERE slug=$1 AND visibility='public'`, slug).Scan(&title, &description)
	return title, description, docError(err)
}

func (r *docsRepository) Search(ctx context.Context, query string, signedIn bool) ([]service.DocSearchResult, error) {
	// PostgreSQL full text covers word queries; the literal fallback handles
	// unsegmented CJK, partial identifiers and punctuation in code examples.
	rows, err := r.db.QueryContext(ctx, `
		WITH q AS (SELECT lower($1) AS literal, plainto_tsquery('simple', $1) AS terms),
		ranked AS (
			SELECT `+docPublicFields+`, p.markdown, q.literal,
			  strpos(lower(p.markdown), q.literal) AS hit,
			  CASE WHEN strpos(lower(p.title), q.literal)>0 THEN 2 ELSE 0 END + ts_rank(p.search_vector, q.terms) AS score
			FROM docs_publications p CROSS JOIN q
			WHERE ($2 OR p.visibility='public')
			  AND (p.search_vector @@ q.terms OR strpos(lower(p.title || ' ' || p.description || ' ' || p.markdown), q.literal)>0)
			ORDER BY score DESC,p.article_id LIMIT 30
		)
		SELECT article_id,category_id,slug,title,description,visibility,
		  CASE WHEN hit>0 THEN
		    CASE WHEN hit>55 THEN '…' ELSE '' END || substr(markdown, greatest(hit-55,1), 180)
		  ELSE coalesce(nullif(description,''), left(markdown,180)) END
		FROM ranked ORDER BY score DESC, article_id`, query, signedIn)
	return docRows(rows, err, func(result *service.DocSearchResult) []any {
		return append(docSummaryFields(&result.DocSummary), &result.Snippet)
	})
}

func (r *docsRepository) AdminNavigation(ctx context.Context) (*service.DocAdminNavigation, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	categories, err := docsCategories(ctx, tx, false, false)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT a.id,a.category_id,a.slug,a.title,a.description,a.visibility,a.status,a.revision,coalesce(p.revision,0),coalesce(p.slug,'')
		FROM docs_articles a JOIN docs_categories c ON c.id=a.category_id LEFT JOIN docs_publications p ON p.article_id=a.id
		ORDER BY c.position,c.id,a.position,a.id`)
	articles, err := docRows(rows, err, func(a *service.DocAdminSummary) []any {
		return append(docSummaryFields(&a.DocSummary), &a.Status, &a.Revision, &a.PublishedRevision, &a.PublishedSlug)
	})
	if err != nil {
		return nil, err
	}
	return &service.DocAdminNavigation{Categories: categories, Articles: articles}, tx.Commit()
}

const docAdminArticle = `
	SELECT a.id,a.category_id,a.slug,a.title,a.description,a.visibility,a.markdown,a.seo_title,a.seo_description,a.revision,
	       a.status,coalesce(p.revision,0),coalesce(p.slug,''),a.updated_at
	FROM docs_articles a LEFT JOIN docs_publications p ON p.article_id=a.id WHERE a.id=$1`

func scanDocArticle(row docsScanner) (*service.DocArticle, error) {
	var a service.DocArticle
	err := row.Scan(&a.ID, &a.CategoryID, &a.Slug, &a.Title, &a.Description, &a.Visibility, &a.Markdown, &a.SEOTitle, &a.SEODescription,
		&a.Revision, &a.Status, &a.PublishedRevision, &a.PublishedSlug, &a.UpdatedAt)
	return &a, docError(err)
}

func (r *docsRepository) Article(ctx context.Context, id int64) (*service.DocArticle, error) {
	return scanDocArticle(r.db.QueryRowContext(ctx, docAdminArticle, id))
}

func (r *docsRepository) write(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// Keep append, move, delete, publish, and reorder consistent across nodes.
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", docsWriteLock); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return docError(err)
	}
	return docError(tx.Commit())
}

func checkDocRevision(ctx context.Context, tx *sql.Tx, id, revision int64) error {
	var current int64
	if err := tx.QueryRowContext(ctx, `SELECT revision FROM docs_articles WHERE id=$1`, id).Scan(&current); err != nil {
		return err
	}
	if current != revision {
		return service.ErrDocConflict
	}
	return nil
}

func (r *docsRepository) SaveArticle(ctx context.Context, id int64, in service.DocInput) (item *service.DocArticle, err error) {
	err = r.write(ctx, func(tx *sql.Tx) error {
		var used bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM docs_publications WHERE slug=$1 AND article_id<>$2)`, in.Slug, id).Scan(&used); err != nil {
			return err
		}
		if used {
			return service.ErrDocConflict
		}
		args := []any{in.CategoryID, in.Slug, in.Title, in.Description, in.Visibility, in.Markdown, in.SEOTitle, in.SEODescription}
		if id == 0 {
			err := tx.QueryRowContext(ctx, `INSERT INTO docs_articles (category_id,slug,title,description,visibility,markdown,seo_title,seo_description,position)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,(SELECT coalesce(max(position)+1,0) FROM docs_articles WHERE category_id=$1)) RETURNING id`, args...).Scan(&id)
			if err != nil {
				return err
			}
		} else {
			if err := checkDocRevision(ctx, tx, id, in.Revision); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `UPDATE docs_articles SET
				position=CASE WHEN category_id=$1 THEN position ELSE (SELECT coalesce(max(position)+1,0) FROM docs_articles WHERE category_id=$1) END,
				category_id=$1, slug=$2,title=$3,description=$4,visibility=$5,markdown=$6,seo_title=$7,seo_description=$8,
				revision=revision+1,updated_at=now()
				WHERE id=$9`, append(args, id)...)
			if err != nil {
				return err
			}
		}
		item, err = scanDocArticle(tx.QueryRowContext(ctx, docAdminArticle, id))
		return err
	})
	return item, err
}

func (r *docsRepository) Publish(ctx context.Context, id, revision int64, images []uuid.UUID) (item *service.DocArticle, err error) {
	err = r.write(ctx, func(tx *sql.Tx) error {
		if err := checkDocRevision(ctx, tx, id, revision); err != nil {
			return err
		}
		var repeat bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM docs_publications WHERE article_id=$1 AND revision=$2)`, id, revision).Scan(&repeat); err != nil {
			return err
		}
		if !repeat {
			if _, err := tx.ExecContext(ctx, `DELETE FROM docs_publications WHERE article_id=$1`, id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO docs_publications(article_id,category_id,slug,title,description,visibility,markdown,seo_title,seo_description,revision,position)
				SELECT id,category_id,slug,title,description,visibility,markdown,seo_title,seo_description,revision,position FROM docs_articles WHERE id=$1`, id); err != nil {
				return err
			}
			for _, image := range images {
				if _, err := tx.ExecContext(ctx, `INSERT INTO docs_publication_images(article_id,image_id) VALUES ($1,$2)`, id, image); err != nil {
					return err
				}
			}
			if _, err := tx.ExecContext(ctx, `UPDATE docs_articles SET status='published' WHERE id=$1`, id); err != nil {
				return err
			}
		}
		item, err = scanDocArticle(tx.QueryRowContext(ctx, docAdminArticle, id))
		return err
	})
	return item, err
}

func (r *docsRepository) Offline(ctx context.Context, id, revision int64) (item *service.DocArticle, err error) {
	err = r.write(ctx, func(tx *sql.Tx) error {
		if err := checkDocRevision(ctx, tx, id, revision); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM docs_publications WHERE article_id=$1`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE docs_articles SET status='offline',revision=revision+1,updated_at=now() WHERE id=$1`, id); err != nil {
			return err
		}
		item, err = scanDocArticle(tx.QueryRowContext(ctx, docAdminArticle, id))
		return err
	})
	return item, err
}

func (r *docsRepository) DeleteArticle(ctx context.Context, id, revision int64) (keys []string, err error) {
	err = r.write(ctx, func(tx *sql.Tx) error {
		if err := checkDocRevision(ctx, tx, id, revision); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT object_key FROM docs_images WHERE article_id=$1 AND object_key IS NOT NULL`, id)
		keys, err = docRows(rows, err, func(s *string) []any { return []any{s} })
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM docs_articles WHERE id=$1`, id)
		return err
	})
	return keys, err
}

func (r *docsRepository) SaveCategory(ctx context.Context, id int64, name string) (*service.DocCategory, error) {
	item := &service.DocCategory{Name: name}
	err := r.write(ctx, func(tx *sql.Tx) error {
		if id == 0 {
			return tx.QueryRowContext(ctx, `INSERT INTO docs_categories (name,position) VALUES ($1,(SELECT coalesce(max(position)+1,0) FROM docs_categories)) RETURNING id`, name).Scan(&item.ID)
		}
		return tx.QueryRowContext(ctx, `UPDATE docs_categories SET name=$2 WHERE id=$1 RETURNING id`, id, name).Scan(&item.ID)
	})
	return item, err
}

func (r *docsRepository) DeleteCategory(ctx context.Context, id int64) error {
	return r.write(ctx, func(tx *sql.Tx) error {
		var deleted int64
		err := tx.QueryRowContext(ctx, `DELETE FROM docs_categories WHERE id=$1 RETURNING id`, id).Scan(&deleted)
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return service.ErrDocCategoryNotEmpty
		}
		return err
	})
}

func (r *docsRepository) SortCategories(ctx context.Context, ids []int64) error {
	return r.sort(ctx, 0, ids)
}

func (r *docsRepository) SortArticles(ctx context.Context, category int64, ids []int64) error {
	if category <= 0 {
		return service.ErrDocInvalid
	}
	return r.sort(ctx, category, ids)
}

func (r *docsRepository) sort(ctx context.Context, category int64, ids []int64) error {
	if len(ids) > 5000 {
		return service.ErrDocInvalid
	}
	return r.write(ctx, func(tx *sql.Tx) error {
		var count int
		table, filter, args := "docs_categories", "", []any{}
		if category != 0 {
			table, filter, args = "docs_articles", " WHERE category_id=$1", []any{category}
		}
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM "+table+filter, args...).Scan(&count); err != nil {
			return err
		}
		if count != len(ids) {
			return service.ErrDocConflict
		}
		query := `UPDATE ` + table + ` t SET position=ordered.pos-1
			FROM unnest($1::bigint[]) WITH ORDINALITY AS ordered(id,pos) WHERE t.id=ordered.id`
		params := []any{pq.Array(ids)}
		if category != 0 {
			query += " AND t.category_id=$2"
			params = append(params, category)
		}
		result, err := tx.ExecContext(ctx, query, params...)
		if err != nil {
			return err
		}
		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != int64(count) {
			return service.ErrDocConflict
		}
		if category != 0 {
			_, err = tx.ExecContext(ctx, `UPDATE docs_publications p SET position=a.position
				FROM docs_articles a WHERE p.article_id=a.id AND p.category_id=$1 AND a.category_id=$1`, category)
		}
		return err
	})
}

func (r *docsRepository) SaveImage(ctx context.Context, articleID int64, img service.DocImage) error {
	return r.write(ctx, func(tx *sql.Tx) error {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM docs_articles WHERE id=$1)`, articleID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return service.ErrDocNotFound
		}
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM docs_images WHERE article_id=$1`, articleID).Scan(&count); err != nil {
			return err
		}
		if count >= 40 {
			return service.ErrDocInvalid
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO docs_images(id,article_id,content_type,data,object_key) VALUES ($1,$2,$3,$4,NULLIF($5,''))`,
			img.ID, articleID, img.ContentType, img.Data, img.ObjectKey)
		return err
	})
}

func scanDocImage(row docsScanner) (*service.DocImage, error) {
	var img service.DocImage
	err := row.Scan(&img.ID, &img.ContentType, &img.Data, &img.ObjectKey)
	return &img, docError(err)
}

func (r *docsRepository) AdminImage(ctx context.Context, articleID int64, id uuid.UUID) (*service.DocImage, error) {
	return scanDocImage(r.db.QueryRowContext(ctx, `SELECT id,content_type,data,coalesce(object_key,'') FROM docs_images WHERE article_id=$1 AND id=$2`, articleID, id))
}

func (r *docsRepository) PublishedImage(ctx context.Context, id uuid.UUID, signedIn bool) (*service.DocImage, error) {
	return scanDocImage(r.db.QueryRowContext(ctx, `
		SELECT i.id,i.content_type,i.data,coalesce(i.object_key,'') FROM docs_images i
		JOIN docs_publication_images a ON a.image_id=i.id AND a.article_id=i.article_id
		JOIN docs_publications p ON p.article_id=a.article_id
		WHERE i.id=$1 AND ($2 OR p.visibility='public')`, id, signedIn))
}
