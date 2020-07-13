package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/harmony-one/harmony/internal/blsgen"
	nodeconfig "github.com/harmony-one/harmony/internal/configs/node"
	"github.com/harmony-one/harmony/multibls"
)

var (
	blsKeyFile        = flag.String("blskey_file", "", "The encrypted file of bls serialized private key by passphrase.")
	blsFolder         = flag.String("blsfolder", ".hmy/blskeys", "The folder that stores the bls keys and corresponding passphrases; e.g. <blskey>.key and <blskey>.pass; all bls keys mapped to same shard")
	maxBLSKeysPerNode = flag.Int("max_bls_keys_per_node", 4, "maximum number of bls keys allowed per node (default 4)")

	// TODO(jacky): rename it to a better name with cobra alias
	blsPass         = flag.String("blspass", "default", "The source for bls passphrases. (default, no-prompt, prompt, file:$CONFIG_FILE, none)")
	awsConfigSource = flag.String("aws-config-source", "default", "the source for aws config. (default, prompt, none, file:$CONFIG_FILE)")
)

// setupConsensusKeys load bls keys and add the keys to nodeConfig. Return the loaded public keys.
func setupConsensusKeys(config *nodeconfig.ConfigType) multibls.PublicKeys {
	multiBLSPriKey, err := loadBLSKeys()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR when loading bls key: %v\n", err)
		os.Exit(100)
	}
	config.ConsensusPriKey = multiBLSPriKey
	return multiBLSPriKey.GetPublicKeys()
}

func loadBLSKeys() (multibls.PrivateKeys, error) {
	config, err := parseBLSLoadingConfig()
	if err != nil {
		return nil, err
	}
	keys, err := blsgen.LoadKeys(config)
	if err != nil {
		return nil, err
	}
	if len(keys) >= *maxBLSKeysPerNode {
		return nil, fmt.Errorf("keys exceed maximum count %v", *maxBLSKeysPerNode)
	}
	return keys, err
}

func parseBLSLoadingConfig() (blsgen.Config, error) {
	var (
		config blsgen.Config
		err    error
	)

	config, err = parseBLSPass(config, *blsPass)
	if err != nil {
		return blsgen.Config{}, err
	}
	config, err = parseAwsConfigSrc(config, *awsConfigSource)
	if err != nil {
		return blsgen.Config{}, err
	}
	return config, nil
}

func parseBLSPass(config blsgen.Config, src string) (blsgen.Config, error) {
	methodArgs := strings.SplitN(src, ":", 2)
	method := methodArgs[0]
	switch method {
	case "default":
		config.PassSrcType = blsgen.PassSrcAuto
	case "file":
		config.PassSrcType = blsgen.PassSrcFile
		if len(methodArgs) < 2 {
			return blsgen.Config{}, errors.New("must specify passphrase file")
		}
		config.PassFile = &methodArgs[1]
	case "no-prompt":
		config.PassSrcType = blsgen.PassSrcFile
	case "prompt":
		config.PassSrcType = blsgen.PassSrcPrompt
	case "none":
		config.PassSrcType = blsgen.PassSrcNil
	}
	return config, nil
}

func parseAwsConfigSrc(config blsgen.Config, src string) (blsgen.Config, error) {
	methodArgs := strings.SplitN(src, ":", 2)
	method := methodArgs[0]
	switch method {
	case "default":
		config.AwsCfgSrcType = blsgen.AwsCfgSrcShared
	case "file":
		config.AwsCfgSrcType = blsgen.AwsCfgSrcFile
		if len(methodArgs) < 2 {
			return blsgen.Config{}, errors.New("must specify aws config file")
		}
		config.AwsConfigFile = &methodArgs[1]
	case "prompt":
		config.AwsCfgSrcType = blsgen.AwsCfgSrcPrompt
	case "none":
		config.AwsCfgSrcType = blsgen.AwsCfgSrcNil
	}
	return config, nil
}
