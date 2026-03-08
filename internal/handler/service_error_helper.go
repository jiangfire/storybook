package handler

import (
	"errors"

	"git.neolidy.top/neo/storybook/internal/api"
	"git.neolidy.top/neo/storybook/internal/service"
)

func serviceValidationItems(err error) ([]api.ErrorItem, bool) {
	var vErr *service.ValidationError
	if !errors.As(err, &vErr) {
		return nil, false
	}

	items := make([]api.ErrorItem, 0, len(vErr.Issues))
	for _, it := range vErr.Issues {
		items = append(items, api.ErrorItem{Field: it.Field, Message: it.Message})
	}
	return items, true
}
