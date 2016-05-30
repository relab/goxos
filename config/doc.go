/*

Package config has three parts:

config.go: A really simple key-value store for configs. Stores all
configs as string values internally, but has getters like GetInt,
GetDuration, etc.

defaults.go: For now it is concidered a best practise to set all
default values here and write documentation for them.

dtoconf.go: Interface and functions used to pass over configuration
and state when replacing a node.


The application can chose to provide a configuration-system on top of
this simple config. The easiest would be to have a ini-based
config-system like the one found in goxos/kvs/kvsd

*/
package config
