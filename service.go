package goCommsMultiDialer

import (
	"context"
	"github.com/bhbosman/goCommsDefinitions"
	"github.com/bhbosman/goCommsNetDialer"
	"github.com/bhbosman/goConnectionManager"
	"github.com/bhbosman/gocommon"
	"github.com/bhbosman/gocommon/ChannelHandler"
	"github.com/bhbosman/gocommon/GoFunctionCounter"
	"github.com/bhbosman/gocommon/pubSub"
	"github.com/bhbosman/gocommon/services/IFxService"
	"github.com/bhbosman/gocommon/services/ISendMessage"
	"github.com/bhbosman/gocommon/services/interfaces"
	"github.com/cskr/pubsub"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"net/url"
)

type service struct {
	ctx                 context.Context
	cancelFunc          context.CancelFunc
	cmdChannel          chan interface{}
	onData              func() (INetMultiDialerData, error)
	Logger              *zap.Logger
	state               IFxService.State
	pubSub              *pubsub.PubSub
	goFunctionCounter   GoFunctionCounter.IService
	subscribeChannel    *pubsub.NextFuncSubscription
	connectionManager   goConnectionManager.IService
	UniqueSessionNumber interfaces.IUniqueReferenceService
}

func (self *service) Dial(
	isSocksConnection bool,
	socksUrl *url.URL,
	connectionUrl *url.URL,
	releaseFunc func(),
	CancellationContext gocommon.ICancellationContext,
	connectionName string,
	connectionPrefix string,
	options ...fx.Option,
) (gocommon.IApp, gocommon.ICancellationContext, string, error) {
	dialManager, err := goCommsNetDialer.NewMultiNetDialManager(
		isSocksConnection,
		socksUrl,
		connectionUrl,
		self.connectionManager,
		self.ctx,
		CancellationContext,
		self.Logger,
		self.UniqueSessionNumber,
		connectionName,
		connectionPrefix,
		func() fx.Option {
			return fx.Options(options...)
		}, self.goFunctionCounter,
	)
	if err != nil {
		return nil, nil, "", err
	}

	dial, g, s, err := dialManager.Dial(releaseFunc)
	if err != nil {
		return nil, nil, "", err
	}
	return dial, g, s, nil
}

func (self *service) Send(message interface{}) error {
	send, err := CallINetMultiDialerSend(self.ctx, self.cmdChannel, false, message)
	if err != nil {
		return err
	}
	return send.Args0
}

func (self *service) OnStart(ctx context.Context) error {
	err := self.start(ctx)
	if err != nil {
		return err
	}
	self.state = IFxService.Started
	return nil
}

func (self *service) OnStop(ctx context.Context) error {
	err := self.shutdown(ctx)
	close(self.cmdChannel)
	self.state = IFxService.Stopped
	return err
}

func (self *service) shutdown(_ context.Context) error {
	self.cancelFunc()
	return pubSub.Unsubscribe("", self.pubSub, self.goFunctionCounter, self.subscribeChannel)
}

func (self *service) start(_ context.Context) error {
	instanceData, err := self.onData()
	if err != nil {
		return err
	}

	return self.goFunctionCounter.GoRun(
		"Multi-Net Dialer Service",
		func() {
			self.goStart(instanceData)
		},
	)
}

func (self *service) goStart(instanceData INetMultiDialerData) {
	defer func(cmdChannel <-chan interface{}) {
		//flush
		for range cmdChannel {
		}
	}(self.cmdChannel)

	self.subscribeChannel = pubsub.NewNextFuncSubscription(goCommsDefinitions.CreateNextFunc(self.cmdChannel))
	self.pubSub.AddSub(self.subscribeChannel, self.ServiceName())

	channelHandlerCallback := ChannelHandler.CreateChannelHandlerCallback(
		self.ctx,
		instanceData,
		[]ChannelHandler.ChannelHandler{
			{
				Cb: func(next interface{}, message interface{}) (bool, error) {
					if unk, ok := next.(INetMultiDialer); ok {
						return ChannelEventsForINetMultiDialer(unk, message)
					}
					return false, nil

				},
			},
			{
				Cb: func(next interface{}, message interface{}) (bool, error) {
					if unk, ok := next.(ISendMessage.ISendMessage); ok {
						return true, unk.Send(message)
					}
					return false, nil
				},
			},
		},
		func() int {
			return len(self.cmdChannel) + self.subscribeChannel.Count()
		},
		goCommsDefinitions.CreateTryNextFunc(self.cmdChannel),
	)
loop:
	for {
		select {
		case <-self.ctx.Done():
			err := instanceData.ShutDown()
			if err != nil {
				self.Logger.Error(
					"error on done",
					zap.Error(err))
			}
			break loop
		case event, ok := <-self.cmdChannel:
			if !ok {
				return
			}
			breakLoop, err := channelHandlerCallback(event)
			if err != nil || breakLoop {
				break loop
			}
		}
	}
}

func (self *service) State() IFxService.State {
	return self.state
}

func (self *service) ServiceName() string {
	return "NetMultiDialer"
}

func newService(
	parentContext context.Context,
	onData func() (INetMultiDialerData, error),
	logger *zap.Logger,
	pubSub *pubsub.PubSub,
	goFunctionCounter GoFunctionCounter.IService,
	connectionManager goConnectionManager.IService,
	UniqueSessionNumber interfaces.IUniqueReferenceService,
) (INetMultiDialerService, error) {
	localCtx, localCancelFunc := context.WithCancel(parentContext)
	return &service{
		ctx:                 localCtx,
		cancelFunc:          localCancelFunc,
		cmdChannel:          make(chan interface{}, 32),
		onData:              onData,
		Logger:              logger,
		pubSub:              pubSub,
		goFunctionCounter:   goFunctionCounter,
		connectionManager:   connectionManager,
		UniqueSessionNumber: UniqueSessionNumber,
	}, nil
}
