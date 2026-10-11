package provider

import "abdanhafidz.com/event-booking-api/middleware"

type MiddlewareProvider interface {
	ProvideUserMiddleware() middleware.UserMiddleware
	ProvideRequestMiddleware() middleware.RequestMiddleware
}

type middlewareProvider struct {
	userMiddleware    middleware.UserMiddleware
	requestMiddleware middleware.RequestMiddleware
}

func NewMiddlewareProvider(repoProvider RepositoriesProvider, configProvider ConfigProvider) MiddlewareProvider {
	return &middlewareProvider{
		userMiddleware:    middleware.NewUserMiddleware(repoProvider.ProvideUserRepository()),
		requestMiddleware: middleware.NewRequestMiddleware(configProvider.ProvideEnvConfig().GetRequestTimeout()),
	}
}

func (p *middlewareProvider) ProvideUserMiddleware() middleware.UserMiddleware {
	return p.userMiddleware
}

func (p *middlewareProvider) ProvideRequestMiddleware() middleware.RequestMiddleware {
	return p.requestMiddleware
}
