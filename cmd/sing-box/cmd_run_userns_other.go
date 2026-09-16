//go:build !linux

package boxmain

import "github.com/sagernet/sing-box/option"

func runInUserNamespaceIfNeeded(options option.Options, optionsList []*OptionsEntry) error {
	return nil
}
