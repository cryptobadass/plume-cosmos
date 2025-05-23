package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/plume-protocol/plume-db/config"
	"github.com/spf13/viper"
)

const DefaultConfigTemplate = `# This is a TOML config file.
# For more information, see https://github.com/toml-lang/toml

###############################################################################
###                           Base Configuration                            ###
###############################################################################

# The minimum gas prices a validator is willing to accept for processing a
# transaction. A transaction's fees must meet the minimum of any denomination
# specified in this config (e.g. 0.25token1;0.0001token2).
minimum-gas-prices = "{{ .BaseConfig.MinGasPrices }}"

# Pruning Strategies:
# - default: Keep the recent 362880 blocks and prune is triggered every 10 blocks
# - nothing: all historic states will be saved, nothing will be deleted (i.e. archiving node)
# - everything: all saved states will be deleted, storing only the recent 2 blocks; pruning at every block
# - custom: allow pruning options to be manually specified through 'pruning-keep-recent' and 'pruning-interval'
# Pruning strategy is completely ignored when plumedb is enabled
pruning = "{{ .BaseConfig.Pruning }}"

# These are applied if and only if the pruning strategy is custom, and plumedb is not enabled
pruning-keep-recent = "{{ .BaseConfig.PruningKeepRecent }}"
pruning-keep-every = "{{ .BaseConfig.PruningKeepEvery }}"
pruning-interval = "{{ .BaseConfig.PruningInterval }}"

# HaltHeight contains a non-zero block height at which a node will gracefully
# halt and shutdown that can be used to assist upgrades and testing.
#
# Note: Commitment of state will be attempted on the corresponding block.
halt-height = {{ .BaseConfig.HaltHeight }}

# HaltTime contains a non-zero minimum block time (in Unix seconds) at which
# a node will gracefully halt and shutdown that can be used to assist upgrades
# and testing.
#
# Note: Commitment of state will be attempted on the corresponding block.
halt-time = {{ .BaseConfig.HaltTime }}

# MinRetainBlocks defines the minimum block height offset from the current
# block being committed, such that all blocks past this offset are pruned
# from Tendermint. It is used as part of the process of determining the
# ResponseCommit.RetainHeight value during ABCI Commit. A value of 0 indicates
# that no blocks should be pruned.
#
# This configuration value is only responsible for pruning Tendermint blocks.
# It has no bearing on application state pruning which is determined by the
# "pruning-*" configurations.
#
# Note: Tendermint block pruning is dependant on this parameter in conunction
# with the unbonding (safety threshold) period, state pruning and state sync
# snapshot parameters to determine the correct minimum value of
# ResponseCommit.RetainHeight.
min-retain-blocks = {{ .BaseConfig.MinRetainBlocks }}

# InterBlockCache enables inter-block caching.
inter-block-cache = {{ .BaseConfig.InterBlockCache }}

# IndexEvents defines the set of events in the form {eventType}.{attributeKey},
# which informs Tendermint what to index. If empty, all events will be indexed.
#
# Example:
# ["message.sender", "message.recipient"]
index-events = {{ .BaseConfig.IndexEvents }}

# IavlCacheSize set the size of the iavl tree cache.
# Default cache size is 50mb.
iavl-cache-size = {{ .BaseConfig.IAVLCacheSize }}

# IAVLDisableFastNode enables or disables the fast node feature of IAVL.
# Default is true.
iavl-disable-fastnode = {{ .BaseConfig.IAVLDisableFastNode }}

# CompactionInterval sets (in seconds) the interval between forced levelDB
# compaction. A value of 0 means no forced levelDB.
# Default is 0.
compaction-interval = {{ .BaseConfig.CompactionInterval }}

# deprecated
no-versioning = {{ .BaseConfig.NoVersioning }}

# Whether to store orphan data (to-be-deleted data pointers) outside the main
# application LevelDB
separate-orphan-storage = {{ .BaseConfig.SeparateOrphanStorage }}

# if separate-orphan-storage is true, how many versions of orphan data to keep
separate-orphan-versions-to-keep = {{ .BaseConfig.SeparateOrphanVersionsToKeep }}

# if separate-orphan-storage is true, how many orphans to store in each file
num-orphan-per-file = {{ .BaseConfig.NumOrphanPerFile }}

# if separate-orphan-storage is true, where to store orphan data
orphan-dir = "{{ .BaseConfig.OrphanDirectory }}"

# concurrency-workers defines how many workers to run for concurrent transaction execution
# concurrency-workers = {{ .BaseConfig.ConcurrencyWorkers }}

# occ-enabled defines whether OCC is enabled or not for transaction execution
occ-enabled = {{ .BaseConfig.OccEnabled }}

###############################################################################
###                         Telemetry Configuration                         ###
###############################################################################

[telemetry]

# Prefixed with keys to separate services.
service-name = "{{ .Telemetry.ServiceName }}"

# Enabled enables the application telemetry functionality. When enabled,
# an in-memory sink is also enabled by default. Operators may also enabled
# other sinks such as Prometheus.
enabled = {{ .Telemetry.Enabled }}

# Enable prefixing gauge values with hostname.
enable-hostname = {{ .Telemetry.EnableHostname }}

# Enable adding hostname to labels.
enable-hostname-label = {{ .Telemetry.EnableHostnameLabel }}

# Enable adding service to labels.
enable-service-label = {{ .Telemetry.EnableServiceLabel }}

# PrometheusRetentionTime, when positive, enables a Prometheus metrics sink.
prometheus-retention-time = {{ .Telemetry.PrometheusRetentionTime }}

# GlobalLabels defines a global set of name/value label tuples applied to all
# metrics emitted using the wrapper functions defined in telemetry package.
#
# Example:
# [["chain_id", "cosmoshub-1"]]
global-labels = [{{ range $k, $v := .Telemetry.GlobalLabels }}
  ["{{index $v 0 }}", "{{ index $v 1}}"],{{ end }}
]

###############################################################################
###                           API Configuration                             ###
###############################################################################

[api]

# Enable defines if the API server should be enabled.
enable = {{ .API.Enable }}

# Swagger defines if swagger documentation should automatically be registered.
swagger = {{ .API.Swagger }}

# Address defines the API server to listen on.
address = "{{ .API.Address }}"

# MaxOpenConnections defines the number of maximum open connections.
max-open-connections = {{ .API.MaxOpenConnections }}

# RPCReadTimeout defines the Tendermint RPC read timeout (in seconds).
rpc-read-timeout = {{ .API.RPCReadTimeout }}

# RPCWriteTimeout defines the Tendermint RPC write timeout (in seconds).
rpc-write-timeout = {{ .API.RPCWriteTimeout }}

# RPCMaxBodyBytes defines the Tendermint maximum response body (in bytes).
rpc-max-body-bytes = {{ .API.RPCMaxBodyBytes }}

# EnableUnsafeCORS defines if CORS should be enabled (unsafe - use it at your own risk).
enabled-unsafe-cors = {{ .API.EnableUnsafeCORS }}

###############################################################################
###                           Rosetta Configuration                         ###
###############################################################################

[rosetta]

# Enable defines if the Rosetta API server should be enabled.
enable = {{ .Rosetta.Enable }}

# Address defines the Rosetta API server to listen on.
address = "{{ .Rosetta.Address }}"

# Network defines the name of the blockchain that will be returned by Rosetta.
blockchain = "{{ .Rosetta.Blockchain }}"

# Network defines the name of the network that will be returned by Rosetta.
network = "{{ .Rosetta.Network }}"

# Retries defines the number of retries when connecting to the node before failing.
retries = {{ .Rosetta.Retries }}

# Offline defines if Rosetta server should run in offline mode.
offline = {{ .Rosetta.Offline }}

###############################################################################
###                           gRPC Configuration                            ###
###############################################################################

[grpc]

# Enable defines if the gRPC server should be enabled.
enable = {{ .GRPC.Enable }}

# Address defines the gRPC server address to bind to.
address = "{{ .GRPC.Address }}"

###############################################################################
###                        gRPC Web Configuration                           ###
###############################################################################

[grpc-web]

# GRPCWebEnable defines if the gRPC-web should be enabled.
# NOTE: gRPC must also be enabled, otherwise, this configuration is a no-op.
enable = {{ .GRPCWeb.Enable }}

# Address defines the gRPC-web server address to bind to.
address = "{{ .GRPCWeb.Address }}"

# EnableUnsafeCORS defines if CORS should be enabled (unsafe - use it at your own risk).
enable-unsafe-cors = {{ .GRPCWeb.EnableUnsafeCORS }}

###############################################################################
###                        State Sync Configuration                         ###
###############################################################################

# State sync snapshots allow other nodes to rapidly join the network without replaying historical
# blocks, instead downloading and applying a snapshot of the application state at a given height.
[state-sync]

# snapshot-interval specifies the block interval at which local state sync snapshots are
# taken (0 to disable). Must be a multiple of pruning-keep-every.
snapshot-interval = {{ .StateSync.SnapshotInterval }}

# snapshot-keep-recent specifies the number of recent snapshots to keep and serve (0 to keep all).
snapshot-keep-recent = {{ .StateSync.SnapshotKeepRecent }}

# snapshot-directory sets the directory for where state sync snapshots are persisted.
# default is emtpy which will then store under the app home directory same as before.
snapshot-directory = "{{ .StateSync.SnapshotDirectory }}"

###############################################################################
###                         Genesis Configuration                           ###
###############################################################################


# Genesis config allows configuring whether to stream from an genesis json file in streamed form
[genesis]

# stream-import specifies whether to the stream the import from the genesis json file. The genesis
# file must be in stream form and exported in a streaming fashion.
stream-import = {{ .Genesis.StreamImport }}

# genesis-stream-file specifies the path of the genesis json file to stream from.
genesis-stream-file = "{{ .Genesis.GenesisStreamFile }}"
` + config.DefaultConfigTemplate

var configTemplate *template.Template

func init() {
	var err error

	tmpl := template.New("appConfigFileTemplate")

	if configTemplate, err = tmpl.Parse(DefaultConfigTemplate); err != nil {
		panic(err)
	}
}

// ParseConfig retrieves the default environment configuration for the
// application.
func ParseConfig(v *viper.Viper) (*Config, error) {
	conf := DefaultConfig()
	err := v.Unmarshal(conf)

	return conf, err
}

// SetConfigTemplate sets the custom app config template for
// the application
func SetConfigTemplate(customTemplate string) {
	var err error

	tmpl := template.New("appConfigFileTemplate")

	if configTemplate, err = tmpl.Parse(customTemplate); err != nil {
		panic(err)
	}
}

// WriteConfigFile renders config using the template and writes it to
// configFilePath.
func WriteConfigFile(configFilePath string, config interface{}) {
	var buffer bytes.Buffer

	if err := configTemplate.Execute(&buffer, config); err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(configFilePath, buffer.Bytes(), 0644); err != nil {
		fmt.Printf("MustWriteFile failed: %v\n", err)
		os.Exit(1)
	}
}
