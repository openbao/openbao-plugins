package ui

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/hashicorp/go-hclog"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

//go:embed static/*
var embedFs embed.FS

var webFs fs.FS

func init() {
	path, ok := os.LookupEnv("BAO_FIDO2_DEV_UI") // FIXME: I guess I just created an arbitrary file read primitive for anyone with mount permissions (plugin env can be configured there): turn this into a built time flag?
	if ok {
		webFs = os.DirFS(path)
		return
	}

	subFs, err := fs.Sub(embedFs, "static")
	if err != nil {
		log.Fatal(err)
	}

	webFs = subFs
}

func getContentType(x string) string {
	switch path.Ext(x) {
	case ".html":
		return "text/html"
	case ".js":
		return "application/javascript"
	case ".css":
		return "text/css"
	default:
		return "application/octet-stream"
	}
}

func PathGet(ctx context.Context, req *logical.Request, data *framework.FieldData, logger hclog.Logger) (*logical.Response, error) {
	path := data.Get("path").(string)

	body, err := fs.ReadFile(webFs, path)
	if err != nil {
		logger.Trace("ui path not found", "fs", webFs, "path", path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, logical.CodedError(http.StatusNotFound, "path %q could not be found", path)
		}
		return nil, logical.CodedError(http.StatusInternalServerError, "internal server error while reading path")
	}

	return &logical.Response{
		Data: map[string]any{
			logical.HTTPStatusCode:  200,
			logical.HTTPRawBody:     body,
			logical.HTTPContentType: getContentType(path),
			//logical.HTTPCacheControlHeader: "max-age=3600",
		}}, nil
}
