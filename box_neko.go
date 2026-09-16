package box

import (
	"context"
	"io"
	"net"
	"reflect"
	"unsafe"

	M "github.com/sagernet/sing/common/metadata"
)

// SetLogWritter used in nekoray, change log writer without providing platform interface
func (s *Box) SetLogWritter(w io.Writer) {
	writer_ := reflect.Indirect(reflect.ValueOf(s.logFactory)).FieldByName("writer")
	writer_ = reflect.NewAt(writer_.Type(), unsafe.Pointer(writer_.UnsafeAddr())).Elem()
	writer_.Set(reflect.ValueOf(w))
}

// NekoDialContext dials through the default outbound of the instance.
func (s *Box) NekoDialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return s.outbound.Default().DialContext(ctx, network, destination)
}

// NekoDialUDP listens packets through the default outbound of the instance.
func (s *Box) NekoDialUDP(ctx context.Context) (net.PacketConn, error) {
	return s.outbound.Default().ListenPacket(ctx, M.Socksaddr{})
}
