package wire

import (
	"log/slog"

	appsample "github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/application/sample"
	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/infrastructure/storage/memory"
	httpserver "github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/interfaces/http"
	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/internal/interfaces/http/handler"

	"github.com/toheart/build-ai-apps-with-go/examples/04-go-react-fullstack/fullstack-starter/backend/conf"
)

func BuildHTTPServer(cfg conf.Config, logger *slog.Logger) *httpserver.Server {
	repo := memory.NewSampleRepository()
	service := appsample.NewService(repo)
	sampleHandler := handler.NewSampleHandler(service)

	return httpserver.New(cfg, logger, sampleHandler)
}
