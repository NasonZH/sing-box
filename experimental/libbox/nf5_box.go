package libbox

import (
	"context"

	"github.com/sagernet/sing/service"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
)

// NewProxyBox 用平台接口构造一个可直接 Start/Close 的 sing-box 实例,
// 供外部持有内核句柄并自行管理生命周期。
// 构造流程与 NewCommandServer 内部保持一致: 复用 baseContext 与 platformInterfaceWrapper,
// box 启动时会自动取出并 Initialize 该平台接口。
// 返回的 cancel 必须在 box 关闭后调用, 以释放其 context。
func NewProxyBox(platformInterface PlatformInterface, configContent string) (*box.Box, context.CancelFunc, error) {
	ctx := baseContext(platformInterface)
	options, err := parseConfig(ctx, configContent)
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	service.MustRegister[adapter.PlatformInterface](ctx, &platformInterfaceWrapper{
		iif:       platformInterface,
		useProcFS: platformInterface.UseProcFS(),
	})

	instance, err := box.New(box.Options{
		Context: ctx,
		Options: options,
	})
	if err != nil {
		cancel()
		return nil, nil, err
	}
	return instance, cancel, nil
}
