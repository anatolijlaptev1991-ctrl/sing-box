package boxapi

import (
	"context"
	"net"

	box "github.com/sagernet/sing-box"
	M "github.com/sagernet/sing/common/metadata"
)

func DialContext(ctx context.Context, box *box.Box, network, addr string) (net.Conn, error) {
	return box.NekoDialContext(ctx, network, M.ParseSocksaddr(addr))
}

func DialUDP(ctx context.Context, box *box.Box) (net.PacketConn, error) {
	return box.NekoDialUDP(ctx)
}
