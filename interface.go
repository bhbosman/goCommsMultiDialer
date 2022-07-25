package goCommsMultiDialer

import (
	"github.com/bhbosman/goCommsDefinitions"
	"github.com/bhbosman/gocommon/Services/IDataShutDown"
	"github.com/bhbosman/gocommon/Services/IFxService"
	"github.com/bhbosman/gocommon/Services/ISendMessage"
	"github.com/bhbosman/gocommon/messages"
	"go.uber.org/fx"
	"net/url"
)

type INetMultiDialer interface {
	ISendMessage.ISendMessage
}

type INetMultiDialerService interface {
	INetMultiDialer
	IFxService.IFxServices
	Dial(
		isSocksConnection bool,
		socksUrl *url.URL,
		connectionUrl *url.URL,
		releaseFunc func(),
		connectionName string,
		connectionPrefix string,
		options ...fx.Option,
	) (messages.IApp, goCommsDefinitions.ICancellationContext, error)
}

type INetMultiDialerData interface {
	INetMultiDialer
	IDataShutDown.IDataShutDown
}
