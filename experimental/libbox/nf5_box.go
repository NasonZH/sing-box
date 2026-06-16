package libbox

import (
	"context"
	"sync"

	"github.com/sagernet/sing/service"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
)

// ProxyBox 是 NewProxyBox 返回的内核句柄, 持有一个 box 实例及其 context.
// 它嵌入 *box.Box, 因此拥有内核的全部能力, 例如 Start 与 Router,
// 并覆写 Close: 先关闭内核再释放 context, 调用方无需关心二者顺序,
// 重复调用也是安全的.
type ProxyBox struct {
	Ctx context.Context
	*box.Box
	cancel    context.CancelFunc
	closeOnce sync.Once
	closeErr  error
}

// NewProxyBox 用平台接口构造一个可直接 Start/Close 的内核句柄,
// 供外部持有内核并自行管理生命周期.
// 构造流程与 NewCommandServer 内部保持一致: 复用 baseContext 与 platformInterfaceWrapper,
// box.New 在构造阶段就会从 context 取出并 Initialize 该平台接口.
// 句柄的 Close 会在关闭内核后释放 context, 调用方只需持有句柄, 无需单独管理 cancel.
func NewProxyBox(platformInterface PlatformInterface, configContent string) (*ProxyBox, error) {
	ctx := baseContext(platformInterface)
	options, err := parseConfig(ctx, configContent)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return &ProxyBox{Ctx: ctx, Box: instance, cancel: cancel}, nil
}

// Close 关闭内核并释放其 context, 关闭顺序由内部保证, 可安全重复调用.
func (b *ProxyBox) Close() error {
	b.closeOnce.Do(func() {
		b.closeErr = b.Box.Close()
		b.cancel()
	})
	return b.closeErr
}
