package boxmain

import (
	"context"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
)

// Create used in nekoray: build and start an instance from config JSON
// without touching command line flags.
func Create(nekoConfigContent []byte) (*box.Box, context.CancelFunc, error) {
	var options option.Options
	err := options.UnmarshalJSONContext(context.Background(), nekoConfigContent)
	if err != nil {
		return nil, nil, err
	}
	if disableColor {
		if options.Log == nil {
			options.Log = &option.LogOptions{}
		}
		options.Log.DisableColor = true
	}
	ctx, cancel := context.WithCancel(context.Background())
	instance, err := box.New(box.Options{
		Context: ctx,
		Options: options,
	})
	if err != nil {
		cancel()
		return nil, nil, E.Cause(err, "create service")
	}
	err = instance.Start()
	if err != nil {
		cancel()
		return nil, nil, E.Cause(err, "start service")
	}
	return instance, cancel, nil
}

func SetDisableColor(dc bool) {
	disableColor = dc
}
