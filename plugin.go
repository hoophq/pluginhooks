package pluginhooks

import (
	"net/rpc"
	"os"
	"strconv"

	hcplugin "github.com/hashicorp/go-plugin"
)

const (
	RPCPluginMethodOnConnect RPCPluginMethod = "OnConnect"
	RPCPluginMethodOnReceive RPCPluginMethod = "OnReceive"
	RPCPluginMethodOnSend    RPCPluginMethod = "OnSend"
)

// RPCPluginMethod is the name of the RPC method to be called.
// It must match exactly the name of methods in the Plugin interface
type RPCPluginMethod string
type Empty struct{}
type Config struct {
	SessionID         string
	UserID            string
	Config            map[string]any
	ConnectionName    string
	ConnectionType    string
	ConnectionEnvVars map[string]any
	ConnectionCommand []string
	ClientArgs        []string
	ClientVerb        string
}

type Request struct {
	PacketType string
	Payload    []byte
}

type Response struct {
	// mutate | stop | log (default)
	Payload []byte
	Err     error
}

type plugin struct {
	Plugin
}

func (p *plugin) Server(*hcplugin.MuxBroker) (interface{}, error)              { return p, nil }
func (p *plugin) Client(*hcplugin.MuxBroker, *rpc.Client) (interface{}, error) { return nil, nil }

type Plugin interface {
	// OnConnect phase will initialize the configuration in memory
	// that could be used to other phases. It's recommended to
	// return an error in case any pre-condition doesn't match
	OnConnect(*Config, *Empty) error
	// OnReceive phase process each received packet
	// the response object should be used to mutate the request packet or
	// returning an error and stopping processing further packets
	OnReceive(*Request, *Response) error
	// OnSend phase will trigger when a packet will be sent
	// to the client. The request will contain the packet and type
	// and the response could mutate the payload or return an error
	// if a condiction is not met.
	OnSend(*Request, *Response) error
}

// Server starts the plugin, it should be called in the main() function.
// The following environment variables are required when using this function:
//
// MagicCookieKey and value are used as a very basic verification
// that a plugin is intended to be launched. This is not a security
// measure, just a UX feature. If the magic cookie doesn't match,
// we show human-friendly output.
//
// * MAGIC_COOKIE_KEY
// * MAGIC_COOKIE_VAL
// * PLUGIN_NAME
// * PLUGIN_VERSION (uint)
//
// Serve will panic for unexpected conditions where a user's fix is unknown.
func Serve(pl Plugin) {
	cookieKey := os.Getenv("MAGIC_COOKIE_KEY")
	cookieVal := os.Getenv("MAGIC_COOKIE_VAL")
	pluginName := os.Getenv("PLUGIN_NAME")
	pluginVersion, _ := strconv.Atoi(os.Getenv("PLUGIN_VERSION"))
	if cookieKey == "" || cookieVal == "" || pluginName == "" || pluginVersion == 0 {
		panic("missing required env vars [MAGIC_COOKIE_KEY, MAGIC_COOKIE_VAL, PLUGIN_NAME, PLUGIN_VERSION]")
	}

	hcplugin.Serve(&hcplugin.ServeConfig{
		HandshakeConfig: hcplugin.HandshakeConfig{
			ProtocolVersion:  uint(pluginVersion),
			MagicCookieKey:   cookieKey,
			MagicCookieValue: cookieVal,
		},
		Plugins: map[string]hcplugin.Plugin{
			pluginName: &plugin{pl},
		},
	})
}
