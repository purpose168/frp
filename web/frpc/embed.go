package frpc

import (
	"embed"

	"github.com/purpose168/frp/assets"
)

//go:embed dist
var EmbedFS embed.FS

func init() {
	assets.Register(EmbedFS)
}
