package service

import (
	"strings"

	"backend/internal/datapermission"
	myerrors "backend/internal/errors"
	"backend/model"
)

// DimensionProviderRequest carries only the relation facts required by a
// Dimension Provider. It contains no policy, grant, or executable filter data.
type DimensionProviderRequest struct {
	DimensionCode string
	Relation      string
	StructureCode string
}

func newDimensionProviderRequest(
	dimensionCode string,
	relation string,
	structureCode *string,
) (DimensionProviderRequest, error) {
	request := DimensionProviderRequest{
		DimensionCode: strings.TrimSpace(dimensionCode),
		Relation:      strings.TrimSpace(relation),
	}
	if structureCode != nil {
		request.StructureCode = strings.TrimSpace(*structureCode)
	}
	if err := request.validate(); err != nil {
		return DimensionProviderRequest{}, err
	}
	return request, nil
}

func (request DimensionProviderRequest) validate() error {
	if request.DimensionCode == "" {
		return myerrors.ErrDataPermissionDimensionUnsupported
	}
	switch request.Relation {
	case model.DataPolicyRelationExact:
		if request.StructureCode != "" {
			return myerrors.ErrDataPermissionDimensionUnsupported
		}
	case model.DataPolicyRelationSelfAndDescendants:
		if request.DimensionCode != datapermission.DimensionCodeManagementOrg ||
			request.StructureCode == "" {
			return myerrors.ErrDataPermissionDimensionUnsupported
		}
	default:
		return myerrors.ErrDataPermissionDimensionUnsupported
	}
	return nil
}

func (request DimensionProviderRequest) cacheKey() string {
	return request.DimensionCode + "\x00" + request.Relation + "\x00" + request.StructureCode
}
