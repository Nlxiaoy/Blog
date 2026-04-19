package v1

import (
	"net/http"
	sharedresp "server/internal/controller/http/shared"

	"github.com/gofiber/fiber/v3"
	"server/internal/controller/http/V1/response"
)

// @Summary 友链列表
// @Tags V1.Links
// @Produce json
// @Success 200 {object} sharedresp.Envelope{data=[]response.LinkDetail}
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/links [get]
func (r *V1) listLinks(ctx fiber.Ctx) error {
	result, err := r.link.GetAllPublicLinks(ctx.Context())
	if err != nil {
		r.logger.Error(err, "http - v1 - link - listLinks - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListLinksFailed, "failed to list links")
	}
	list := make([]response.LinkDetail, 0, len(result.Items))
	for _, l := range result.Items {
		list = append(list, response.LinkDetail{
			ID:          l.ID,
			Name:        l.Name,
			URL:         l.URL,
			Description: l.Description,
			Logo:        l.Logo,
			SortOrder:   l.SortOrder,
			Status:      l.Status,
			CreatedAt:   l.CreatedAt,
			UpdatedAt:   l.UpdatedAt,
		})
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(list))
}
