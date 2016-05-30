package config

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/relab/goxos/grp"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/golang/glog"
)

// The Config holds a map of config values by their keys/names. They
// are all stored as strings and parsed on read time only. Currently
// there are four built in types:
//
// `string`: (GetString) Returns the config value as a string. This
// can never fail.
//
// `int`: (GetInt) Uses strconv.Atoi to parse the value and return an
// int.
//
// `duration`: (GetDuration) Uses time.ParseDuration to parse the
// value and return a duration. That means that you should set
// duration configs like "xxx us/ms/s/h/etc."
//
// `bool`: (GetBool) Uses strconv.ParseBool to read config vars like
// true/t/1 or false/f/0
//
// `nodemap`: (GetNodeMap) Returns a grp.NodeMap parsed from a format
// like ([id:hostname:paxosPort:clientPort], ...)
//
// `nodelist`: (GetNodeList) Returns a []grp.Node parsed from a format
// similar to `nodemap`, just without the ids. (used for the config
// `nodeInitStandbys`)
//
// When a value COULD NOT BE PARSED at runtime, Cofig emits a warning
// (with glog) and returns the given DEFAULT VALUE.
//
// We should prefer to keep this Config-system as simple as
// possible. If you really need another type than the ones that Config
// gives you, you should rather use GetString and parse it where you
// use it. And if you need an uint, you should rather cast the int
// given from GetInt.
type Config struct {
	values map[string]string
}

// Returns a new empty Config.
func NewConfig() *Config {
	return &Config{
		values: make(map[string]string),
	}
}

func newConfigFromValues(values map[string]string) *Config {
	return &Config{
		values: values,
	}
}

// Sets a config to a value. All values can only be set as strings.
func (c *Config) Set(key, value string) {
	c.values[key] = value
}

// Gets a value as a string. This one will never emit an warning
// because all values per definition is available as strings.
func (c *Config) GetString(key, defaultVal string) string {
	cfgValue, found := c.values[key]

	if !found {
		return defaultVal
	}

	return cfgValue
}

// Returns the config as an int. If the config is not set, the
// supplied default value is returned. If the config is not possible
// to parse as an int (strconv.Atoi), the default value is returned
// and an warning message is written to glog.
func (c *Config) GetInt(key string, defaultVal int) int {
	cfgValue, found := c.values[key]

	if !found {
		return defaultVal
	}

	dur, err := strconv.Atoi(cfgValue)

	if err != nil {
		glog.Warningf("Could not parse config \"%s\": \"%s\" as int (see strconv.Atoi). Using default value: \"%s\".",
			key, cfgValue, defaultVal)
		return defaultVal
	}

	return dur
}

// Returns the config as an time.Duration. If the config is not set,
// the supplied default value is returned. If the config is not
// possible to parse as an int (time.ParseDuration), the default value
// is returned and an warning message is written to glog.
func (c *Config) GetDuration(key string, defaultVal time.Duration) time.Duration {
	cfgValue, found := c.values[key]

	if !found {
		return defaultVal
	}

	dur, err := time.ParseDuration(cfgValue)

	if err != nil {
		glog.Warningf("Could not parse config \"%s\": \"%s\" as duration (see time.ParseDuration). Using default value: \"%s\".",
			key, cfgValue, defaultVal.String())
		return defaultVal
	}

	return dur
}

// Returns the config as an bool. If the config is not set, the
// supplied default value is returned. If the config is not possible
// to parse as an int (strconv.ParseBool), the default value is
// returned and an warning message is written to glog.
func (c *Config) GetBool(key string, defaultVal bool) bool {
	cfgValue, found := c.values[key]

	if !found {
		return defaultVal
	}

	bool, err := strconv.ParseBool(cfgValue)

	if err != nil {
		glog.Warningf("Could not parse config \"%s\": \"%s\" as boolean (see strconv.ParseBool). Using default value: \"%s\".",
			key, cfgValue, defaultVal)
		return defaultVal
	}

	return bool
}

// Parses a node map: ([id:hostname:paxosPort:clientPort], ...) into
// a grp.NodeMap.
//
// This one takes no default values. The node map in the config
// `node` is always required anyway, and the function returns an
// descriptive error instead.
func (c *Config) GetNodeMap(key string) (grp.NodeMap, error) {
	nodeMap := make(map[grp.ID]grp.Node)

	nodesCfg := c.GetString(key, "")
	if nodesCfg == "" {
		return grp.NodeMap{}, errors.New("Config `" + key + "` need to be set! Should be in the format [id:hostname:paxos-port:client-port], ...")
	}

	nodes := strings.Split(nodesCfg, ",")

	for _, node := range nodes {
		node = strings.TrimSpace(node)
		parts := strings.Split(node, ":")

		if len(parts) != 4 {
			return grp.NodeMap{}, errors.New("Could not understand `" + node + "` in `" + key + "` config. Should be in the format [id:hostname:paxos-port:client-port], ...")
		}

		id, err := strconv.Atoi(parts[0])
		if err != nil {
			return grp.NodeMap{}, errors.New("Could not understand `" + node + "` in `" + key + "` config (id format not integer). Should be in the format [id:hostname:paxos-port:client-port], ...")
		}

		n := grp.NewNode(parts[1], parts[2], parts[3], true, true, true)
		nodeMap[grp.NewID(grp.PaxosID(id), grp.Epoch(0))] = n
	}

	return *grp.NewNodeMap(nodeMap), nil
}

// Parses a node list: ([hostname:paxosPort:clientPort], ...) into
// a []grp.Node.
//
// This one takes no default values. The function returns an
// descriptive error if the value is unreadable. But an empty value
// gives an empty list.
func (c *Config) GetNodeList(key string) ([]grp.Node, error) {
	var nodeList []grp.Node

	nodesCfg := c.GetString(key, "")
	if nodesCfg == "" {
		return []grp.Node{}, nil
	}

	nodes := strings.Split(nodesCfg, ",")

	for _, node := range nodes {
		node = strings.TrimSpace(node)
		parts := strings.Split(node, ":")

		if len(parts) != 3 {
			return nil, errors.New("Could not understand `" + node + "` in `" + key + "` config. Should be in the format [hostname:paxos-port:client-port], ...")
		}

		n := grp.NewNode(parts[0], parts[1], parts[2], true, true, true)
		nodeList = append(nodeList, n)
	}

	return nodeList, nil
}

// Clones the config with all the values.
func (c *Config) CloneToKeyValueMap() map[string]string {
	clonedMap := make(map[string]string, len(c.values))
	for k, v := range c.values {
		clonedMap[k] = v
	}

	return clonedMap
}
