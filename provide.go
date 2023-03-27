package goCommsMultiDialer

import (
	"context"
	"github.com/bhbosman/goConnectionManager"
	"github.com/bhbosman/gocommon/GoFunctionCounter"
	"github.com/bhbosman/gocommon/services/interfaces"
	"github.com/cskr/pubsub"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func Provide() fx.Option {
	return fx.Options(
		fx.Provide(
			func(
				params struct {
					fx.In
				},
			) (func() (INetMultiDialerData, error), error) {
				return func() (INetMultiDialerData, error) {
					return newData()
				}, nil
			},
		),
		fx.Provide(
			func(
				params struct {
					fx.In
					PubSub                 *pubsub.PubSub  `name:"Application"`
					ApplicationContext     context.Context `name:"Application"`
					OnData                 func() (INetMultiDialerData, error)
					Lifecycle              fx.Lifecycle
					Logger                 *zap.Logger
					UniqueReferenceService interfaces.IUniqueReferenceService
					GoFunctionCounter      GoFunctionCounter.IService
					ConnectionManager      goConnectionManager.IService
					UniqueSessionNumber    interfaces.IUniqueReferenceService
				},
			) (INetMultiDialerService, error) {
				serviceInstance, err := newService(
					params.ApplicationContext,
					params.OnData,
					params.Logger,
					params.PubSub,
					params.GoFunctionCounter,
					params.ConnectionManager,
					params.UniqueSessionNumber,
				)
				if err != nil {
					return nil, err
				}
				params.Lifecycle.Append(
					fx.Hook{
						OnStart: serviceInstance.OnStart,
						OnStop:  serviceInstance.OnStop,
					},
				)
				return serviceInstance, nil
			},
		),
	)
}
