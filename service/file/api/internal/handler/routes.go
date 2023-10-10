package handler

import (
	"context"
	"fmt"
	"github.com/GoCloudstorage/GoCloudstorage/opt"
	"github.com/GoCloudstorage/GoCloudstorage/pb/file"
	"github.com/GoCloudstorage/GoCloudstorage/pb/storage"
	"github.com/GoCloudstorage/GoCloudstorage/pkg/token"
	"github.com/GoCloudstorage/GoCloudstorage/pkg/xrpc"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type API struct {
	storageRPCClient storage.StorageClient
	fileRPCClient    file.FileClient
}

func (a *API) InitGrpc() {
	// add local rpc client
	client, err := xrpc.GetGrpcClient(
		xrpc.Config{
			Domain:          opt.Cfg.StorageRPC.Domain,
			Endpoints:       opt.Cfg.StorageRPC.Endpoints,
			BackoffInterval: 0,
			MaxAttempts:     0,
		},
		storage.NewStorageClient,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		panic(err)
	}
	a.storageRPCClient = client.NewSession()

	// add file rpc client
	fileClient, err := xrpc.GetGrpcClient(
		xrpc.Config{
			Domain:          opt.Cfg.FileRPC.Domain,
			Endpoints:       opt.Cfg.FileRPC.Endpoints,
			BackoffInterval: 0,
			MaxAttempts:     0,
		},
		file.NewFileClient,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		panic(err)
	}
	a.fileRPCClient = fileClient.NewSession()
}

func (a *API) registerAPI() *fiber.App {
	app := fiber.New()

	api := app.Group("/file")
	api.Use(cors.New())
	api.Use(token.JWTMiddleware())
	{
		api.Post("/", a.preUpload)
		api.Post("/update", a.updateFile)
		api.Get("/", a.getAllFile)
		api.Get("/:id", a.preDownload)
	}
	return app
}

var api API

func InitAPI(ctx context.Context, name string, host string, port string) {
	var (
		addr = fmt.Sprintf("%s:%s", host, port)
		app  = api.registerAPI()
	)
	api.InitGrpc()
	go func() {
		logrus.Infof("Start fiber webserver, addr: %s", addr)
		if err := app.Listen(addr); err != nil {
			logrus.Panicf("%s listen address %v fail, err: %v", name, addr, err)
		}
	}()
	select {
	case <-ctx.Done():
		_ = app.Shutdown()
	}
}
