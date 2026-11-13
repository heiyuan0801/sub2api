package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDocsImageReferencesExcludeCodeAndText(t *testing.T) {
	image := uuid.New()
	unused := uuid.New()
	body := "![Screenshot][shot]\n\n[shot]: /api/v1/docs/images/" + image.String() +
		"\n\n`![Code sample](/api/v1/docs/images/" + unused.String() + ")`" +
		"\n\n![Screenshot](/api/v1/docs/images/" + image.String() + ")"
	ids, err := docImageIDs(body)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{image}, ids)
	_, err = docImageIDs("![Bad reference](/api/v1/docs/images/invalid)")
	require.ErrorIs(t, err, ErrDocInvalid)
}

func TestDocsSlugReservesBatchImageGuide(t *testing.T) {
	require.True(t, ValidDocSlug("api-v2"))
	for _, invalid := range []string{"batch-image", "", "../api", "API", "api--guide", "api/guide"} {
		require.False(t, ValidDocSlug(invalid), invalid)
	}
}
