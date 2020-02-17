package logger

import (
	"github.com/iotaledger/hive.go/node"
)

var PLUGIN = node.NewPlugin("Logger", node.Enabled, run)

func run(ctx *node.Plugin) {
	// do nothing; everything is handled in the init method
}
