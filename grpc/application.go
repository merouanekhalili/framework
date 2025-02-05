package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/goravel/framework/facades"
)

type Application struct {
	server                       *grpc.Server
	unaryClientInterceptorGroups map[string][]grpc.UnaryClientInterceptor
}

func NewApplication() *Application {
	return &Application{}
}

func (app *Application) Server() *grpc.Server {
	return app.server
}

func (app *Application) Client(ctx context.Context, name string) (*grpc.ClientConn, error) {
	host := facades.Config.GetString(fmt.Sprintf("grpc.clients.%s.host", name))
	if host == "" {
		return nil, errors.New("client host can't be empty")
	}
	if !strings.Contains(host, ":") {
		port := facades.Config.GetString(fmt.Sprintf("grpc.clients.%s.port", name))
		if port == "" {
			return nil, errors.New("client port can't be empty")
		}

		host += ":" + port
	}

	interceptors, ok := facades.Config.Get(fmt.Sprintf("grpc.clients.%s.interceptors", name)).([]string)
	if !ok {
		return nil, fmt.Errorf("the type of clients.%s.interceptors must be []string", name)
	}

	clientInterceptors := app.getClientInterceptors(interceptors)

	return grpc.DialContext(
		ctx,
		host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(clientInterceptors...),
	)
}

func (app *Application) Run(host ...string) error {
	if len(host) == 0 {
		defaultHost := facades.Config.GetString("grpc.host")
		if defaultHost == "" {
			return errors.New("host can't be empty")
		}

		if !strings.Contains(defaultHost, ":") {
			defaultPort := facades.Config.GetString("grpc.port")
			if defaultPort == "" {
				return errors.New("port can't be empty")
			}
			defaultHost += ":" + defaultPort
		}

		host = append(host, defaultHost)
	}

	listen, err := net.Listen("tcp", host[0])
	if err != nil {
		return err
	}
	if err := app.server.Serve(listen); err != nil {
		return err
	}

	return nil
}

func (app *Application) UnaryServerInterceptors(unaryServerInterceptors []grpc.UnaryServerInterceptor) {
	app.server = grpc.NewServer(grpc.UnaryInterceptor(
		grpcmiddleware.ChainUnaryServer(unaryServerInterceptors...),
	))
}

func (app *Application) UnaryClientInterceptorGroups(unaryClientInterceptorGroups map[string][]grpc.UnaryClientInterceptor) {
	app.unaryClientInterceptorGroups = unaryClientInterceptorGroups
}

func (app *Application) getClientInterceptors(interceptors []string) []grpc.UnaryClientInterceptor {
	var unaryClientInterceptors []grpc.UnaryClientInterceptor
	for _, interceptor := range interceptors {
		for client, clientInterceptors := range app.unaryClientInterceptorGroups {
			if interceptor == client {
				unaryClientInterceptors = append(unaryClientInterceptors, clientInterceptors...)
			}
		}
	}

	return unaryClientInterceptors
}
