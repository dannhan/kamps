package kulon2

import (
	"context"
	"errors"
	"fmt"

	"github.com/ruyam/kamps/internal/library"
	"github.com/ruyam/kamps/internal/model"
)

// Download fetches a material's bytes through the authenticated session.
// External links are rejected: the caller opens those in a browser instead.
func (s *LiveSource) Download(ctx context.Context, material model.Material, destPath string, progress func(done, total int64)) (string, error) {
	if material.Kind == "url" {
		return "", errors.New("material is an external link, not a file")
	}
	if material.URL == "" {
		return "", fmt.Errorf("material %s has no download URL", material.ID)
	}
	if err := s.waitForRequestSlot(ctx); err != nil {
		return "", err
	}
	path, err := library.DownloadFile(ctx, s.client, material.URL, destPath, material.Size, progress)
	if errors.Is(err, library.ErrSessionExpiredHTML) || errors.Is(err, library.ErrUnauthorized) {
		return "", fmt.Errorf("%w: run `kamps login` again", err)
	}
	return path, err
}
