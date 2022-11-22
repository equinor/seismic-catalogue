package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pborman/getopt/v2"
	"github.com/gin-gonic/gin"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/equinor/seismic-catalogue/api/api"
	"github.com/equinor/seismic-catalogue/api/internal/auth"
	"github.com/equinor/seismic-catalogue/api/internal/postgres"
)

type opts struct {
	clientId     string
	clientSecret string
	tenantId     string
	keyVault     string
	port         int
}

func parseopts() opts {
	help := getopt.BoolLong("help", 0, "help")
	opts := opts {
		clientId:     os.Getenv("AZURE_CLIENT_ID"),
		clientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		tenantId:     os.Getenv("AZURE_TENANT_ID"),
		keyVault:     os.Getenv("AZURE_KEY_VAULT"),
		port:         8080,
	}

	getopt.FlagLong(
		&opts.clientId,
		"client-id",
		0,
		"Application client ID for azure Service Principal. " +
		"Defaults to the value of environment variable AZURE_CLIENT_ID",
		"string",
	)
	getopt.FlagLong(
		&opts.clientSecret,
		"client-secret",
		0,
		"Application client secret. " +
		"Defaults to the value of environment variable AZURE_CLIENT_SECRET",
		"string",
	)
	getopt.FlagLong(
		&opts.tenantId,
		"tenant-id",
		0,
		"Tenant ID. " +
		"Defaults to the value of environment variable AZURE_TENANT_ID",
		"string",
	)
	getopt.FlagLong(
		&opts.keyVault,
		"key-vault",
		0,
		"KeyVault URI. The key-vault should contain a secret named " +
		"'db-reader-connection-string', which contains a postgres connection-" +
		"string. The client should have read access on the key-vault. " +
		"Defaults to the value of environment variable AZURE_KEY_VAULT",
		"string",
	)
	getopt.FlagLong(
		&opts.port,
		"port",
		'p',
		"Port to start server on. Defaults to 8080",
		"int",
	)

	getopt.Parse()
	if *help {
		getopt.Usage()
		os.Exit(0)
	}

	return opts
}

func main() {
	opts := parseopts()

	servicePrincipal, err := azidentity.NewClientSecretCredential(
		opts.tenantId,
		opts.clientId,
		opts.clientSecret,
		nil,
	)

	if err != nil {
		log.Fatal(err)
	}

	connectionString := auth.GetSecretFromKeyVault(
		servicePrincipal,
		opts.keyVault,
	 	"db-reader-connection-string",
	)

	db, err := postgres.NewAdapter(connectionString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	
	udcProvider := auth.NewUdcCachingProvider(
		servicePrincipal,
		time.Duration(7 * 24 * time.Hour),
	)
	sasProvider := auth.NewUserDelegationSasProvider(
		udcProvider,
		time.Duration(24 * time.Hour),
		opts.tenantId,
		opts.clientId,
		opts.clientSecret,
	)

	catalogue := api.NewCatalogueAPI(db, sasProvider)

	app := gin.Default()
	
	authServer := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", opts.tenantId)
	provider := auth.NewJwksProvider(authServer)
	app.Use(auth.NewJwtTokenValidator(
		authServer,
		opts.clientId,
		provider.KeyFunc,
	))

	group := app.Group("/catalogue")
	group.GET( "", catalogue.Get)
	group.POST("", catalogue.Post)

	app.Run(fmt.Sprintf(":%d", opts.port))
}
