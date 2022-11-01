package api

import (
	"encoding/json"
	"net/http"
	
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/equinor/seismic-catalogue/api/internal/auth"
	"github.com/equinor/seismic-catalogue/api/internal/database"
)

type CatalogueAPI struct {
	dbConnection database.Adapter
	sasProvider  auth.SasTokenProvider
}

func (c *CatalogueAPI) Post(ctx *gin.Context) {
	var request Request
	if err := ctx.ShouldBind(&request); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	execRequest(ctx, request, c.dbConnection, c.sasProvider)
}

func (c *CatalogueAPI) Get(ctx *gin.Context) {
	var request Request
	query := []byte(ctx.Query("query"))
	if err := json.Unmarshal(query, &request); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if err := binding.Validator.ValidateStruct(request); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	execRequest(ctx, request, c.dbConnection, c.sasProvider)
}

func NewCatalogueAPI(
	dbConnection database.Adapter,
	sasProvider  auth.SasTokenProvider,
) *CatalogueAPI {
	return &CatalogueAPI{ dbConnection: dbConnection, sasProvider: sasProvider }
}
