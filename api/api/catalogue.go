package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
	
	"github.com/gin-gonic/gin"

	"github.com/equinor/seismic-catalogue/api/internal/auth"
	"github.com/equinor/seismic-catalogue/api/internal/database"
)

type cubeResponse struct {
	Country          string `json:"country"          example:"Norway"`
	Field            string `json:"field"            example:"Volve"`
	FilenameOnUpload string `json:"filenameOnUpload" example:"myfile.segy"`
	Url              string `json:"url"              example:"https://account.blob.core.windows.net/container/vds"`
	Sas              string `json:"sas,omitempty"    example:"<sas>"`
}

type Request struct {
	SasDuration uint `json:"sasDuration" example:"60"`
	UseServicePrincipal bool `json:"useServicePrincipal"` // for speed comparison testing: if true - our service principal would be used
	UseCachedServicePrincipalKey bool `json:"useCachedServicePrincipalKey"`
}

func makeContainerURL(storageAccount, container string) string {
	return fmt.Sprintf(
		"https://%s.blob.core.windows.net/%s",
		storageAccount,
		container,
	)
}

/* This helper simply aims at hiding the ugly lookup an cast needed to retrieve
 * values from the gin context
 */
func getRolesClaim(ctx *gin.Context) ([]string, error) {
	roles, exists := ctx.Get("jwtRolesClaim"); 
	if !exists {
		return []string{}, errors.New("expected to find roles claim")
	}
	return roles.([]string), nil
}

func execRequest(
	ctx          *gin.Context,
	request      Request,
	dbConnection database.Adapter,
	sasProvider  auth.SasTokenProvider,
) {
	start := time.Now()
	access, err := getRolesClaim(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	cubes, err := dbConnection.GetCubes(access)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	requestDuration := time.Since(start)
	fmt.Println("Cubes received", requestDuration)

	duration := time.Duration(request.SasDuration) * time.Minute
	accessToken, accessTokenExists := ctx.Get("jwtAccessToken")
	if !accessTokenExists {
		ctx.AbortWithError(
			http.StatusInternalServerError, 
			errors.New("Access token is not found in protected API!"),
		)
	}

	var response []cubeResponse
	for _, cube := range cubes {
		containerURL := makeContainerURL(cube.StorageAccount, cube.Container)
		
		var sas string
		if request.SasDuration != 0 {
			sas, err = sasProvider.ContainerSas(
				cube.StorageAccount,
				cube.Container,
				duration,
				request.UseServicePrincipal,
				request.UseCachedServicePrincipalKey,
				accessToken.(string),
			)
			if err != nil {
				ctx.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}

		response = append(response, cubeResponse{
			Country:          cube.Country,
			Field:            cube.Field,
			FilenameOnUpload: cube.FilenameOnUpload,
			Url:              fmt.Sprintf("%s/vds", containerURL),
			Sas:              sas,
		})
	}

	doc, err := json.Marshal(response)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	requestDuration = time.Since(start)
	fmt.Println("Full retrieval time", requestDuration)

	ctx.Data(http.StatusOK, "application/json", doc)
}
